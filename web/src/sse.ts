import { useCatalogStore } from './stores/catalog'
import { useTasksStore } from './stores/tasks'
import type { Task } from './types'

let source: EventSource | null = null

/** Opens the Server-Sent Events stream; the browser reconnects automatically on drops. */
export function connect(): void {
  if (source) return
  const tasks = useTasksStore()
  const catalog = useCatalogStore()
  source = new EventSource('/api/v1/events', { withCredentials: true })

  // Emitted on every (re)connection: catch up on whatever happened while disconnected.
  source.addEventListener('connected', () => {
    void tasks.refreshFromServer()
    void catalog.refresh()
  })
  for (const type of ['task.created', 'task.updated']) {
    source.addEventListener(type, (ev) => void tasks.applyRemote(JSON.parse((ev as MessageEvent).data) as Task))
  }
  source.addEventListener('task.deleted', (ev) => {
    const { id } = JSON.parse((ev as MessageEvent).data) as { id: string }
    void tasks.applyRemoteDelete(id)
  })
  for (const type of [
    'department.created',
    'department.updated',
    'department.deleted',
    'supermarket.created',
    'supermarket.updated',
    'supermarket.deleted',
    'supermarket.order_updated',
    'list.created',
    'list.updated',
    'list.deleted',
  ]) {
    source.addEventListener(type, () => void catalog.refresh())
  }
}

export function disconnect(): void {
  source?.close()
  source = null
}
