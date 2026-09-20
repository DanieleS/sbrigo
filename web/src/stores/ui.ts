import { defineStore } from 'pinia'
import { i18n } from '../i18n'

export type Theme = 'system' | 'light' | 'dark'

export interface Toast {
  id: number
  message: string
  tone: 'error' | 'info'
}

const THEME_KEY = 'sbrigo.theme'
let nextToastId = 1

function readTheme(): Theme {
  try {
    const v = localStorage.getItem(THEME_KEY)
    if (v === 'light' || v === 'dark' || v === 'system') return v
  } catch {
    // storage unavailable
  }
  return 'system'
}

export function applyTheme(theme: Theme): void {
  if (typeof document === 'undefined') return
  if (theme === 'system') delete document.documentElement.dataset.theme
  else document.documentElement.dataset.theme = theme
}

export const useUiStore = defineStore('ui', {
  state: () => ({ theme: readTheme(), toasts: [] as Toast[] }),
  actions: {
    setTheme(theme: Theme) {
      this.theme = theme
      applyTheme(theme)
      try {
        localStorage.setItem(THEME_KEY, theme)
      } catch {
        // storage unavailable
      }
    },
    toast(message: string, tone: Toast['tone'] = 'info') {
      this.toasts.push({ id: nextToastId++, message, tone })
    },
    dismiss(id: number) {
      this.toasts = this.toasts.filter((t) => t.id !== id)
    },
    /** Runs an online-only action and reports failures as a toast. */
    async guard(fn: () => Promise<void>): Promise<boolean> {
      try {
        await fn()
        return true
      } catch (err) {
        console.error(err)
        const offline = typeof navigator !== 'undefined' && !navigator.onLine
        this.toast(i18n.global.t(offline ? 'common.needsConnection' : 'common.error'), 'error')
        return false
      }
    },
  },
})
