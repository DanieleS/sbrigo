import { api, ApiError, NetworkError } from './api'
import { listOutbox, removeOutbox, type OutboxOp } from './db'
import { useAuthStore } from './stores/auth'
import { useTasksStore } from './stores/tasks'
import type { Task } from './types'

let flushing = false
let rerun = false

/**
 * Sends the offline queue to the backend in order. Stops at the first network failure and
 * resumes on the next `online` event or explicit kick. Server-side conflicts (last-write-wins)
 * are resolved by adopting the server row.
 */
export async function flush(): Promise<void> {
  if (flushing) {
    rerun = true
    return
  }
  if (typeof navigator !== 'undefined' && !navigator.onLine) return
  flushing = true
  const tasks = useTasksStore()
  const auth = useAuthStore()
  try {
    for (const op of await listOutbox()) {
      if (op.seq === undefined) continue
      try {
        await perform(op)
        await removeOutbox(op.seq)
      } catch (err) {
        if (err instanceof NetworkError) break
        if (err instanceof ApiError) {
          if (err.status === 401) {
            auth.markAnonymous()
            break
          }
          if (err.status === 409 && op.kind === 'update') {
            await tasks.adoptServer(err.body as Task)
            await removeOutbox(op.seq)
            continue
          }
          if (err.status === 404 && op.kind !== 'create') {
            await tasks.dropLocal(op.task_id)
            await removeOutbox(op.seq)
            continue
          }
          if (err.status >= 500) break
          // Any other 4xx: the operation can never succeed, drop it rather than block the queue.
          console.warn('dropping unprocessable operation', op, err.body)
          await removeOutbox(op.seq)
          continue
        }
        console.error('sync error', err)
        break
      }
    }
  } finally {
    flushing = false
    await tasks.refreshPending()
    if (rerun) {
      rerun = false
      void flush()
    }
  }
}

async function perform(op: OutboxOp): Promise<void> {
  switch (op.kind) {
    case 'create':
      await api.createTask(op.body)
      return
    case 'update':
      await api.patchTask(op.task_id, op.body)
      return
    case 'delete':
      await api.deleteTask(op.task_id)
      return
  }
}

/** Fire-and-forget flush. */
export function kick(): void {
  void flush().catch((err) => console.error(err))
}

if (typeof window !== 'undefined') {
  window.addEventListener('online', kick)
}
