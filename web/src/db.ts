import { openDB, type DBSchema, type IDBPDatabase } from 'idb'
import type { Department, List, Supermarket, Task, TaskPatch, User } from './types'

/** One queued change waiting to reach the server. */
export type OutboxOp =
  | { seq?: number; kind: 'create'; task_id: string; body: Task; queued_at: string }
  | { seq?: number; kind: 'update'; task_id: string; body: TaskPatch; queued_at: string }
  | { seq?: number; kind: 'delete'; task_id: string; queued_at: string }

interface SbrigoDB extends DBSchema {
  tasks: { key: string; value: Task }
  departments: { key: string; value: Department }
  supermarkets: { key: string; value: Supermarket }
  users: { key: string; value: User }
  lists: { key: string; value: List }
  orders: { key: string; value: { supermarket_id: string; department_ids: string[] } }
  outbox: { key: number; value: OutboxOp }
  meta: { key: string; value: unknown }
}

type StoreName = 'tasks' | 'departments' | 'supermarkets' | 'users' | 'lists' | 'orders'

let dbPromise: Promise<IDBPDatabase<SbrigoDB>> | null = null

function db() {
  dbPromise ??= openDB<SbrigoDB>('sbrigo', 2, {
    upgrade(d, oldVersion) {
      if (oldVersion < 1) {
        d.createObjectStore('tasks', { keyPath: 'id' })
        d.createObjectStore('departments', { keyPath: 'id' })
        d.createObjectStore('supermarkets', { keyPath: 'id' })
        d.createObjectStore('users', { keyPath: 'id' })
        d.createObjectStore('orders', { keyPath: 'supermarket_id' })
        d.createObjectStore('outbox', { keyPath: 'seq', autoIncrement: true })
        d.createObjectStore('meta')
      }
      if (oldVersion < 2) {
        d.createObjectStore('lists', { keyPath: 'id' })
      }
    },
  })
  return dbPromise
}

export async function getAll<N extends StoreName>(name: N): Promise<SbrigoDB[N]['value'][]> {
  return (await db()).getAll(name)
}

/** Replaces the whole content of a store atomically. */
export async function replaceAll<N extends StoreName>(name: N, items: SbrigoDB[N]['value'][]): Promise<void> {
  const tx = (await db()).transaction(name, 'readwrite')
  await tx.store.clear()
  for (const item of items) await tx.store.put(item)
  await tx.done
}

export async function put<N extends StoreName>(name: N, item: SbrigoDB[N]['value']): Promise<void> {
  await (await db()).put(name, item)
}

export async function remove(name: StoreName, key: string): Promise<void> {
  await (await db()).delete(name, key)
}

export async function getMeta<T>(key: string): Promise<T | undefined> {
  return (await (await db()).get('meta', key)) as T | undefined
}

export async function setMeta(key: string, value: unknown): Promise<void> {
  await (await db()).put('meta', value, key)
}

export async function listOutbox(): Promise<OutboxOp[]> {
  return (await db()).getAll('outbox')
}

export async function outboxSize(): Promise<number> {
  return (await db()).count('outbox')
}

export async function removeOutbox(seq: number): Promise<void> {
  await (await db()).delete('outbox', seq)
}

export async function hasPendingFor(taskId: string): Promise<boolean> {
  return (await listOutbox()).some((op) => op.task_id === taskId)
}

/**
 * Queues a change, coalescing with earlier changes to the same task so the queue stays short:
 * updates fold into a pending create or update, and deleting a task the server never saw simply
 * drops its pending operations.
 */
export async function enqueue(op: OutboxOp): Promise<void> {
  const tx = (await db()).transaction('outbox', 'readwrite')
  const pending = (await tx.store.getAll()).filter((p) => p.task_id === op.task_id)
  const last = pending.at(-1)

  if (op.kind === 'update' && last && last.seq !== undefined) {
    if (last.kind === 'create') {
      const { updated_at, ...fields } = op.body
      await tx.store.put({ ...last, body: { ...last.body, ...fields, updated_at: updated_at ?? op.queued_at } })
      await tx.done
      return
    }
    if (last.kind === 'update') {
      await tx.store.put({ ...last, body: { ...last.body, ...op.body } })
      await tx.done
      return
    }
  }
  if (op.kind === 'delete') {
    for (const p of pending) if (p.seq !== undefined) await tx.store.delete(p.seq)
    if (pending.some((p) => p.kind === 'create')) {
      await tx.done
      return
    }
  }
  await tx.store.add(op)
  await tx.done
}
