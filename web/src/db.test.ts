import { beforeEach, describe, expect, it } from 'vitest'
import { deleteDB } from 'idb'
import { enqueue, hasPendingFor, listOutbox, type OutboxOp } from './db'
import type { Task } from './types'

const task = (id: string): Task => ({
  id,
  title: 'Latte',
  notes: null,
  is_completed: false,
  item_type: 'grocery',
  department_id: null,
  assignee_id: null,
  due_date: null,
  created_at: '2026-01-01T10:00:00.000Z',
  updated_at: '2026-01-01T10:00:00.000Z',
})

describe('outbox coalescing', () => {
  beforeEach(async () => {
    // The module caches the connection; clear rows instead of deleting the database.
    for (const op of await listOutbox()) if (op.seq !== undefined) await removeOp(op)
  })

  it('folds an update into a pending create', async () => {
    await enqueue({ kind: 'create', task_id: 'a', body: task('a'), queued_at: '2026-01-01T10:00:00.000Z' })
    await enqueue({
      kind: 'update',
      task_id: 'a',
      body: { is_completed: true, updated_at: '2026-01-01T10:01:00.000Z' },
      queued_at: '2026-01-01T10:01:00.000Z',
    })
    const ops = await listOutbox()
    expect(ops).toHaveLength(1)
    expect(ops[0]!.kind).toBe('create')
    expect((ops[0] as Extract<OutboxOp, { kind: 'create' }>).body.is_completed).toBe(true)
    expect((ops[0] as Extract<OutboxOp, { kind: 'create' }>).body.updated_at).toBe('2026-01-01T10:01:00.000Z')
  })

  it('merges consecutive updates keeping the latest timestamp', async () => {
    await enqueue({
      kind: 'update',
      task_id: 'b',
      body: { title: 'Pane', updated_at: '2026-01-01T10:00:00.000Z' },
      queued_at: 'x',
    })
    await enqueue({
      kind: 'update',
      task_id: 'b',
      body: { is_completed: true, updated_at: '2026-01-01T10:02:00.000Z' },
      queued_at: 'y',
    })
    const ops = await listOutbox()
    expect(ops).toHaveLength(1)
    expect((ops[0] as Extract<OutboxOp, { kind: 'update' }>).body).toEqual({
      title: 'Pane',
      is_completed: true,
      updated_at: '2026-01-01T10:02:00.000Z',
    })
  })

  it('drops everything when a task the server never saw is deleted', async () => {
    await enqueue({ kind: 'create', task_id: 'c', body: task('c'), queued_at: 'x' })
    await enqueue({ kind: 'update', task_id: 'c', body: { title: 'Uova' }, queued_at: 'y' })
    await enqueue({ kind: 'delete', task_id: 'c', queued_at: 'z' })
    expect(await listOutbox()).toHaveLength(0)
    expect(await hasPendingFor('c')).toBe(false)
  })

  it('replaces pending updates with a single delete for a known task', async () => {
    await enqueue({ kind: 'update', task_id: 'd', body: { title: 'Uova' }, queued_at: 'x' })
    await enqueue({ kind: 'delete', task_id: 'd', queued_at: 'y' })
    const ops = await listOutbox()
    expect(ops).toHaveLength(1)
    expect(ops[0]!.kind).toBe('delete')
  })

  it('keeps operations of different tasks independent and ordered', async () => {
    await enqueue({ kind: 'create', task_id: 'e', body: task('e'), queued_at: 'x' })
    await enqueue({ kind: 'create', task_id: 'f', body: task('f'), queued_at: 'y' })
    await enqueue({ kind: 'update', task_id: 'e', body: { title: 'Riso' }, queued_at: 'z' })
    const ops = await listOutbox()
    expect(ops.map((o) => o.task_id)).toEqual(['e', 'f'])
  })
})

async function removeOp(op: OutboxOp) {
  const { removeOutbox } = await import('./db')
  await removeOutbox(op.seq!)
}

// Silence the unused import warning when the helper is not needed by a runtime.
void deleteDB
