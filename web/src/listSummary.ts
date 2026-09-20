import { computed, type ComputedRef } from 'vue'
import { daysFromToday } from './format'
import { useTasksStore } from './stores/tasks'
import type { List } from './types'

export interface Summary {
  open: number
  done: number
  overdue: number
  /** Earliest upcoming (today or later) due date among open tasks, ISO string. */
  nextDue: string | null
}

/** Preview figures for a list, derived from the locally cached tasks so they work offline. */
export function summarize(listId: string): Summary {
  const tasks = useTasksStore()
  const items = tasks.byList(listId)
  let open = 0
  let done = 0
  let overdue = 0
  let nextDue: string | null = null
  for (const t of items) {
    if (t.is_completed) {
      done++
      continue
    }
    open++
    const diff = daysFromToday(t.due_date)
    if (diff !== null && diff < 0) {
      overdue++
      continue
    }
    if (t.due_date && (!nextDue || t.due_date < nextDue)) nextDue = t.due_date
  }
  return { open, done, overdue, nextDue }
}

export function useSummaries(lists: ComputedRef<List[]>): ComputedRef<Record<string, Summary>> {
  return computed(() => Object.fromEntries(lists.value.map((l) => [l.id, summarize(l.id)])))
}
