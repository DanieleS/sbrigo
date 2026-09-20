import type {
  Department,
  DepartmentOrder,
  ItemType,
  List,
  ListSummary,
  Me,
  Supermarket,
  Task,
  TaskPatch,
  User,
} from './types'

export class ApiError extends Error {
  constructor(
    public status: number,
    public body: unknown,
  ) {
    super(`API error ${status}`)
  }
}

/** Raised when the request never reached the server (offline, DNS, proxy down). */
export class NetworkError extends Error {}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  let res: Response
  try {
    res = await fetch(path, {
      method,
      headers: body === undefined ? {} : { 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : JSON.stringify(body),
      credentials: 'same-origin',
    })
  } catch (err) {
    throw new NetworkError(err instanceof Error ? err.message : String(err))
  }
  const text = await res.text()
  const data: unknown = text ? safeJson(text) : null
  if (!res.ok) throw new ApiError(res.status, data)
  return data as T
}

function safeJson(text: string): unknown {
  try {
    return JSON.parse(text)
  } catch {
    return text
  }
}

export const api = {
  me: () => request<Me>('GET', '/api/v1/me'),
  logout: () => request<null>('POST', '/auth/logout'),
  logoutAll: () => request<null>('POST', '/auth/logout-all'),
  users: () => request<User[]>('GET', '/api/v1/users'),

  departments: () => request<Department[]>('GET', '/api/v1/departments'),
  createDepartment: (d: Omit<Department, 'id'>) => request<Department>('POST', '/api/v1/departments', d),
  updateDepartment: (d: Department) =>
    request<Department>('PUT', `/api/v1/departments/${d.id}`, {
      name: d.name,
      description: d.description,
      default_sort_order: d.default_sort_order,
    }),
  deleteDepartment: (id: string) => request<null>('DELETE', `/api/v1/departments/${id}`),

  supermarkets: () => request<Supermarket[]>('GET', '/api/v1/supermarkets'),
  createSupermarket: (name: string) => request<Supermarket>('POST', '/api/v1/supermarkets', { name }),
  updateSupermarket: (id: string, name: string) => request<Supermarket>('PUT', `/api/v1/supermarkets/${id}`, { name }),
  deleteSupermarket: (id: string) => request<null>('DELETE', `/api/v1/supermarkets/${id}`),
  departmentOrder: (id: string) => request<DepartmentOrder[]>('GET', `/api/v1/supermarkets/${id}/department-order`),
  setDepartmentOrder: (id: string, departmentIds: string[]) =>
    request<DepartmentOrder[]>('PUT', `/api/v1/supermarkets/${id}/department-order`, {
      department_ids: departmentIds,
    }),

  lists: () => request<ListSummary[]>('GET', '/api/v1/lists'),
  createList: (name: string, kind: ItemType, sortOrder: number) =>
    request<List>('POST', '/api/v1/lists', { name, kind, sort_order: sortOrder }),
  updateList: (id: string, name: string, sortOrder: number) =>
    request<List>('PUT', `/api/v1/lists/${id}`, { name, sort_order: sortOrder }),
  deleteList: (id: string) => request<null>('DELETE', `/api/v1/lists/${id}`),

  tasks: () => request<Task[]>('GET', '/api/v1/tasks'),
  createTask: (t: Task) => request<Task>('POST', '/api/v1/tasks', t),
  patchTask: (id: string, patch: TaskPatch) => request<Task>('PATCH', `/api/v1/tasks/${id}`, patch),
  deleteTask: (id: string) => request<null>('DELETE', `/api/v1/tasks/${id}`),
}
