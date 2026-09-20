import { createI18n } from 'vue-i18n'
import it from './it'
import en from './en'

export type Locale = 'it' | 'en'
export const LOCALES: Locale[] = ['it', 'en']
export const LOCALE_NAMES: Record<Locale, string> = { it: 'Italiano', en: 'English' }

const STORAGE_KEY = 'sbrigo.locale'

function detectLocale(): Locale {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored === 'it' || stored === 'en') return stored
  } catch {
    // storage unavailable
  }
  const nav = typeof navigator === 'undefined' ? 'en' : navigator.language
  return nav.toLowerCase().startsWith('it') ? 'it' : 'en'
}

export const i18n = createI18n({
  legacy: false,
  locale: detectLocale(),
  fallbackLocale: 'en',
  messages: { it, en },
})

export function currentLocale(): Locale {
  return i18n.global.locale.value as Locale
}

export function setLocale(locale: Locale): void {
  i18n.global.locale.value = locale
  try {
    localStorage.setItem(STORAGE_KEY, locale)
  } catch {
    // storage unavailable
  }
  if (typeof document !== 'undefined') document.documentElement.lang = locale
}

if (typeof document !== 'undefined') document.documentElement.lang = currentLocale()
