import { defineStore } from 'pinia'
import { api, ApiError, NetworkError } from '../api'
import { getMeta, setMeta } from '../db'
import type { User } from '../types'

export type AuthStatus = 'loading' | 'authenticated' | 'anonymous' | 'offline'

export const useAuthStore = defineStore('auth', {
  state: () => ({ status: 'loading' as AuthStatus, user: null as User | null }),
  actions: {
    async load() {
      try {
        const me = await api.me()
        this.user = me.user
        this.status = 'authenticated'
        await setMeta('me', me.user)
      } catch (err) {
        if (err instanceof ApiError && err.status === 401) {
          this.markAnonymous()
          return
        }
        if (err instanceof NetworkError) {
          // Offline: keep working with the cached profile and local data.
          this.user = (await getMeta<User | null>('me')) ?? null
          this.status = 'offline'
          return
        }
        throw err
      }
    },
    markAnonymous() {
      this.user = null
      this.status = 'anonymous'
    },
    login() {
      const returnTo = encodeURIComponent(location.pathname + location.search)
      location.href = `/auth/login?return_to=${returnTo}`
    },
    async logout() {
      try {
        await api.logout()
      } finally {
        await setMeta('me', null)
        this.markAnonymous()
      }
    },
    async logoutAll() {
      try {
        await api.logoutAll()
      } finally {
        await setMeta('me', null)
        this.markAnonymous()
      }
    },
  },
})
