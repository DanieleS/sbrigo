import { defineStore } from 'pinia'
import { api } from '../api'
import { getAll, getMeta, replaceAll, setMeta } from '../db'
import type { Department, Supermarket, User } from '../types'

export const NO_DEPARTMENT = '__none__'

export const useCatalogStore = defineStore('catalog', {
  state: () => ({
    departments: [] as Department[],
    supermarkets: [] as Supermarket[],
    users: [] as User[],
    /** supermarket id -> department ids in aisle order */
    orders: {} as Record<string, string[]>,
    selectedSupermarketId: '' as string,
    loaded: false,
  }),
  getters: {
    departmentById: (s) => (id: string | null) => s.departments.find((d) => d.id === id),
    userById: (s) => (id: string | null) => s.users.find((u) => u.id === id),
    selectedSupermarket: (s) => s.supermarkets.find((m) => m.id === s.selectedSupermarketId),
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
      const [departments, supermarkets, users, orders, selected] = await Promise.all([
        getAll('departments'),
        getAll('supermarkets'),
        getAll('users'),
        getAll('orders'),
        getMeta<string>('selectedSupermarketId'),
      ])
      this.departments = departments
      this.supermarkets = supermarkets
      this.users = users
      this.orders = Object.fromEntries(orders.map((o) => [o.supermarket_id, o.department_ids]))
      this.selectedSupermarketId = selected ?? ''
      this.loaded = true
    },
    async refresh() {
      const [departments, supermarkets, users] = await Promise.all([api.departments(), api.supermarkets(), api.users()])
      const orders = await Promise.all(
        supermarkets.map(async (m) => ({
          supermarket_id: m.id,
          department_ids: (await api.departmentOrder(m.id)).map((o) => o.department_id),
        })),
      )
      this.departments = departments
      this.supermarkets = supermarkets
      this.users = users
      this.orders = Object.fromEntries(orders.map((o) => [o.supermarket_id, o.department_ids]))
      if (this.selectedSupermarketId && !supermarkets.some((m) => m.id === this.selectedSupermarketId)) {
        await this.selectSupermarket('')
      }
      await Promise.all([
        replaceAll('departments', departments),
        replaceAll('supermarkets', supermarkets),
        replaceAll('users', users),
        replaceAll('orders', orders),
      ])
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
  },
})
