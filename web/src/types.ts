export interface User {
  id: string
  identity_id: string
  email: string
  display_name: string
  created_at: string
}

export interface Department {
  id: string
  name: string
  description: string
  default_sort_order: number
}

export interface Supermarket {
  id: string
  name: string
}

export interface DepartmentOrder {
  department_id: string
  sort_order: number
}

export type ItemType = 'grocery' | 'general_task'

export interface Task {
  id: string
  title: string
  notes: string | null
  is_completed: boolean
  item_type: ItemType
  department_id: string | null
  assignee_id: string | null
  due_date: string | null
  created_at: string
  updated_at: string
}

export type TaskFields = Pick<
  Task,
  'title' | 'notes' | 'is_completed' | 'item_type' | 'department_id' | 'assignee_id' | 'due_date'
>

/** Partial update; updated_at is the client-side timestamp used for last-write-wins. */
export type TaskPatch = Partial<TaskFields> & { updated_at?: string }

export interface Me {
  kind: 'user' | 'agent'
  user: User | null
}
