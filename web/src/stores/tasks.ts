import { defineStore } from 'pinia'
import { api } from '../api'
import { enqueue, getAll, hasPendingFor, outboxSize, put, remove, replaceAll } from '../db'
import { kick } from '../sync'
import type { ItemType, Task, TaskFields, TaskPatch } from '../types'

function now(): string {
  return new Date().toISOString()
}

export const useTasksStore = defineStore('tasks', {
  state: () => ({
    tasks: {} as Record<string, Task>,
    pending: 0,
    loaded: false,
  }),
  getters: {
    all: (s) => Object.values(s.tasks),
    byType: (s) => (type: ItemType) => Object.values(s.tasks).filter((t) => t.item_type === type),
    byList: (s) => (listId: string) => Object.values(s.tasks).filter((t) => t.list_id === listId),
  },
  actions: {
    async init() {
      const stored = await getAll('tasks')
      this.tasks = Object.fromEntries(stored.map((t) => [t.id, t]))
      this.loaded = true
      await this.refreshPending()
    },
    async refreshPending() {
      this.pending = await outboxSize()
    },

    /** Pulls the full list from the server once the outbox is empty, so local edits never get overwritten. */
    async refreshFromServer() {
      if ((await outboxSize()) > 0) {
        kick()
        return
      }
      try {
        const list = await api.tasks()
        if ((await outboxSize()) > 0) return // an edit happened meanwhile; SSE will reconcile
        this.tasks = Object.fromEntries(list.map((t) => [t.id, t]))
        await replaceAll('tasks', list)
      } catch {
        // offline or unauthenticated: keep local data
      }
    },

    async add(fields: Partial<TaskFields> & { title: string; item_type: ItemType; list_id: string }): Promise<Task> {
      const ts = now()
      const task: Task = {
        id: crypto.randomUUID(),
        list_id: fields.list_id,
        title: fields.title.trim(),
        notes: fields.notes ?? null,
        is_completed: fields.is_completed ?? false,
        item_type: fields.item_type,
        department_id: fields.department_id ?? null,
        assignee_id: fields.assignee_id ?? null,
        due_date: fields.due_date ?? null,
        created_at: ts,
        updated_at: ts,
      }
      this.tasks[task.id] = task
      await put('tasks', task)
      await enqueue({ kind: 'create', task_id: task.id, body: task, queued_at: ts })
      await this.refreshPending()
      kick()
      return task
    },

    async patch(id: string, patch: Partial<TaskFields>) {
      const current = this.tasks[id]
      if (!current) return
      const ts = now()
      const updated: Task = { ...current, ...patch, updated_at: ts }
      this.tasks[id] = updated
      await put('tasks', updated)
      const body: TaskPatch = { ...patch, updated_at: ts }
      await enqueue({ kind: 'update', task_id: id, body, queued_at: ts })
      await this.refreshPending()
      kick()
    },

    async toggle(id: string) {
      const t = this.tasks[id]
      if (t) await this.patch(id, { is_completed: !t.is_completed })
    },

    async removeTask(id: string) {
      await this.dropLocal(id)
      await enqueue({ kind: 'delete', task_id: id, queued_at: now() })
      await this.refreshPending()
      kick()
    },

    async clearCompleted(listId: string) {
      const done = this.byList(listId).filter((t) => t.is_completed)
      for (const t of done) {
        await this.dropLocal(t.id)
        await enqueue({ kind: 'delete', task_id: t.id, queued_at: now() })
      }
      await this.refreshPending()
      kick()
    },

    /** Applies a row coming from the server (SSE or conflict resolution). */
    async adoptServer(task: Task) {
      this.tasks[task.id] = task
      await put('tasks', task)
    },
    async dropLocal(id: string) {
      this.tasks = Object.fromEntries(Object.entries(this.tasks).filter(([key]) => key !== id))
      await remove('tasks', id)
    },

    /** SSE handlers: a task with queued local edits is left alone until the queue is flushed. */
    async applyRemote(task: Task) {
      if (await hasPendingFor(task.id)) return
      const local = this.tasks[task.id]
      if (local && local.updated_at > task.updated_at) return
      await this.adoptServer(task)
    },
    async applyRemoteDelete(id: string) {
      if (await hasPendingFor(id)) return
      await this.dropLocal(id)
    },
  },
})
