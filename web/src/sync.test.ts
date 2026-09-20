import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { enqueue, listOutbox, removeOutbox } from './db'
import { flush } from './sync'
import { useTasksStore } from './stores/tasks'
import type { Task } from './types'

const task = (id: string, extra: Partial<Task> = {}): Task => ({
  id,
  list_id: 'list-1',
  title: 'Latte',
  notes: null,
  is_completed: false,
  item_type: 'grocery',
  department_id: null,
  assignee_id: null,
  due_date: null,
  position: 1,
  created_at: '2026-01-01T10:00:00.000Z',
  updated_at: '2026-01-01T10:00:00.000Z',
  ...extra,
})

function jsonResponse(status: number, body: unknown) {
  return new Response(body === null ? null : JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

describe('flush', () => {
  beforeEach(async () => {
    setActivePinia(createPinia())
    for (const op of await listOutbox()) if (op.seq !== undefined) await removeOutbox(op.seq)
    vi.restoreAllMocks()
    Object.defineProperty(globalThis, 'navigator', { value: { onLine: true }, configurable: true })
  })

  it('sends queued operations in order and empties the outbox', async () => {
    const calls: string[] = []
    vi.stubGlobal(
      'fetch',
      vi.fn(async (url: string, init?: RequestInit) => {
        calls.push(`${init?.method} ${url}`)
        return jsonResponse(init?.method === 'POST' ? 201 : 200, task('a'))
      }),
    )
    await enqueue({ kind: 'create', task_id: 'a', body: task('a'), queued_at: 'x' })
    await enqueue({ kind: 'update', task_id: 'b', body: { is_completed: true }, queued_at: 'y' })
    await flush()
    expect(calls).toEqual(['POST /api/v1/tasks', 'PATCH /api/v1/tasks/b'])
    expect(await listOutbox()).toHaveLength(0)
  })

  it('stops at a network failure and keeps the queue for later', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => Promise.reject(new TypeError('Failed to fetch'))),
    )
    await enqueue({ kind: 'delete', task_id: 'a', queued_at: 'x' })
    await flush()
    expect(await listOutbox()).toHaveLength(1)
  })

  it('adopts the server row on a last-write-wins conflict', async () => {
    const tasks = useTasksStore()
    await tasks.adoptServer(task('a', { title: 'locale' }))
    const serverRow = task('a', { title: 'server', updated_at: '2026-01-01T12:00:00.000Z' })
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => jsonResponse(409, serverRow)),
    )
    await enqueue({
      kind: 'update',
      task_id: 'a',
      body: { title: 'locale', updated_at: '2026-01-01T11:00:00.000Z' },
      queued_at: 'x',
    })
    await flush()
    expect(await listOutbox()).toHaveLength(0)
    expect(tasks.tasks['a']?.title).toBe('server')
  })

  it('drops a local task that was deleted on the server', async () => {
    const tasks = useTasksStore()
    await tasks.adoptServer(task('gone'))
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => jsonResponse(404, { error: 'not found' })),
    )
    await enqueue({ kind: 'update', task_id: 'gone', body: { title: 'x' }, queued_at: 'x' })
    await flush()
    expect(await listOutbox()).toHaveLength(0)
    expect(tasks.tasks['gone']).toBeUndefined()
  })
})
