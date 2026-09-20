import { defineStore } from 'pinia'
import { api } from '../api'
import { getAll, getMeta, replaceAll, setMeta } from '../db'
import type { Department, ItemType, List, Supermarket, User } from '../types'

export const NO_DEPARTMENT = '__none__'

export const useCatalogStore = defineStore('catalog', {
  state: () => ({
    departments: [] as Department[],
    supermarkets: [] as Supermarket[],
    users: [] as User[],
    lists: [] as List[],
    /** kind -> selected list id (per device) */
    selectedLists: {} as Partial<Record<ItemType, string>>,
    /** supermarket id -> department ids in aisle order */
    orders: {} as Record<string, string[]>,
    selectedSupermarketId: '' as string,
    loaded: false,
  }),
  getters: {
    departmentById: (s) => (id: string | null) => s.departments.find((d) => d.id === id),
    userById: (s) => (id: string | null) => s.users.find((u) => u.id === id),
    selectedSupermarket: (s) => s.supermarkets.find((m) => m.id === s.selectedSupermarketId),
    listById: (s) => (id: string | null) => s.lists.find((l) => l.id === id),
    listsOfKind: (s) => (kind: ItemType) =>
      s.lists
        .filter((l) => l.kind === kind)
        .sort((a, b) => a.sort_order - b.sort_order || a.name.localeCompare(b.name)),
    /** The list currently shown for a kind, falling back to the first one when the stored choice is gone. */
    currentList(): (kind: ItemType) => List | undefined {
      return (kind) => {
        const ofKind = this.listsOfKind(kind)
        return ofKind.find((l) => l.id === this.selectedLists[kind]) ?? ofKind[0]
      }
    },
    /** Departments sorted by the selected supermarket's aisles, falling back to the default order. */
    orderedDepartments(s): Department[] {
      const byDefault = [...s.departments].sort(
        (a, b) => a.default_sort_order - b.default_sort_order || a.name.localeCompare(b.name),
      )
      const order = s.selectedSupermarketId ? s.orders[s.selectedSupermarketId] : undefined
      if (!order?.length) return byDefault
      const rank = new Map(order.map((id, i) => [id, i]))
      return byDefault.sort((a, b) => (rank.get(a.id) ?? 1e9) - (rank.get(b.id) ?? 1e9))
    },
  },
  actions: {
    async init() {
      const [departments, supermarkets, users, lists, orders, selected, selectedLists] = await Promise.all([
        getAll('departments'),
        getAll('supermarkets'),
        getAll('users'),
        getAll('lists'),
        getAll('orders'),
        getMeta<string>('selectedSupermarketId'),
        getMeta<Partial<Record<ItemType, string>>>('selectedLists'),
      ])
      this.departments = departments
      this.supermarkets = supermarkets
      this.users = users
      this.lists = lists
      this.selectedLists = selectedLists ?? {}
      this.orders = Object.fromEntries(orders.map((o) => [o.supermarket_id, o.department_ids]))
      this.selectedSupermarketId = selected ?? ''
      this.loaded = true
    },
    async refresh() {
      const [departments, supermarkets, users, lists] = await Promise.all([
        api.departments(),
        api.supermarkets(),
        api.users(),
        api.lists(),
      ])
      const orders = await Promise.all(
        supermarkets.map(async (m) => ({
          supermarket_id: m.id,
          department_ids: (await api.departmentOrder(m.id)).map((o) => o.department_id),
        })),
      )
      this.departments = departments
      this.supermarkets = supermarkets
      this.users = users
      // Summary figures are recomputed locally from the cached tasks; only the list itself is stored.
      // Keep a plain copy for IndexedDB: reactive proxies cannot be structured-cloned.
      const plainLists = lists.map((l): List => ({
        id: l.id,
        name: l.name,
        kind: l.kind,
        sort_order: l.sort_order,
        created_at: l.created_at,
        updated_at: l.updated_at,
      }))
      this.lists = plainLists
      this.orders = Object.fromEntries(orders.map((o) => [o.supermarket_id, o.department_ids]))
      if (this.selectedSupermarketId && !supermarkets.some((m) => m.id === this.selectedSupermarketId)) {
        await this.selectSupermarket('')
      }
      await Promise.all([
        replaceAll('departments', departments),
        replaceAll('supermarkets', supermarkets),
        replaceAll('users', users),
        replaceAll('lists', plainLists),
        replaceAll('orders', orders),
      ])
    },
    async selectList(kind: ItemType, id: string) {
      this.selectedLists = { ...this.selectedLists, [kind]: id }
      await setMeta('selectedLists', JSON.parse(JSON.stringify(this.selectedLists)))
    },
    async createList(name: string, kind: ItemType) {
      const last = this.listsOfKind(kind).at(-1)
      const created = await api.createList(name, kind, (last?.sort_order ?? 0) + 10)
      await this.refresh()
      await this.selectList(kind, created.id)
      return created
    },
    async renameList(id: string, name: string) {
      const l = this.listById(id)
      if (!l) return
      await api.updateList(id, name, l.sort_order)
      await this.refresh()
    },
    async deleteList(id: string) {
      await api.deleteList(id)
      await this.refresh()
    },
    async selectSupermarket(id: string) {
      this.selectedSupermarketId = id
      await setMeta('selectedSupermarketId', id)
    },

    // Catalogue edits are rare and done from the settings screen; they require connectivity.
    async createDepartment(d: Omit<Department, 'id'>) {
      await api.createDepartment(d)
      await this.refresh()
    },
    async updateDepartment(d: Department) {
      await api.updateDepartment(d)
      await this.refresh()
    },
    async deleteDepartment(id: string) {
      await api.deleteDepartment(id)
      await this.refresh()
    },
    async createSupermarket(name: string) {
      await api.createSupermarket(name)
      await this.refresh()
    },
    async renameSupermarket(id: string, name: string) {
      await api.updateSupermarket(id, name)
      await this.refresh()
    },
    async deleteSupermarket(id: string) {
      await api.deleteSupermarket(id)
      await this.refresh()
    },
    async setDepartmentOrder(id: string, departmentIds: string[]) {
      await api.setDepartmentOrder(id, departmentIds)
      await this.refresh()
    },
    /** Rewrites default_sort_order so that departments follow the given order. */
    async reorderDefaultDepartments(departmentIds: string[]) {
      const byId = new Map(this.departments.map((d) => [d.id, d]))
      await Promise.all(
        departmentIds.map((id, i) => {
          const d = byId.get(id)
          return d && d.default_sort_order !== (i + 1) * 10
            ? api.updateDepartment({ ...d, default_sort_order: (i + 1) * 10 })
            : Promise.resolve()
        }),
      )
      await this.refresh()
    },
    /**
     * Applies a new relative order for some departments (the visible groups) to the current
     * context: the selected supermarket's aisles, or the default order when none is selected.
     * Departments not listed keep their relative order after the listed ones.
     */
    async reorderVisibleDepartments(visibleIds: string[]) {
      const rest = this.orderedDepartments.map((d) => d.id).filter((id) => !visibleIds.includes(id))
      const full = [...visibleIds, ...rest]
      if (this.selectedSupermarketId) await this.setDepartmentOrder(this.selectedSupermarketId, full)
      else await this.reorderDefaultDepartments(full)
    },
  },
})
