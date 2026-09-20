import { currentLocale, i18n } from './i18n'

/** Locale-aware date helpers shared by the views. */

function startOfDay(d: Date): number {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime()
}

/** Whole days between today and the given date (negative = past). */
export function daysFromToday(iso: string | null): number | null {
  if (!iso) return null
  return Math.round((startOfDay(new Date(iso)) - startOfDay(new Date())) / 86_400_000)
}

export function formatDue(iso: string | null): string {
  if (!iso) return ''
  const diff = daysFromToday(iso)
  const t = i18n.global.t
  if (diff === 0) return t('dates.today')
  if (diff === 1) return t('dates.tomorrow')
  if (diff === -1) return t('dates.yesterday')
  const d = new Date(iso)
  const locale = currentLocale()
  const sameYear = d.getFullYear() === new Date().getFullYear()
  return new Intl.DateTimeFormat(locale, {
    weekday: Math.abs(diff ?? 0) < 7 ? 'short' : undefined,
    day: 'numeric',
    month: 'short',
    year: sameYear ? undefined : 'numeric',
  }).format(d)
}

export function isOverdue(iso: string | null): boolean {
  const diff = daysFromToday(iso)
  return diff !== null && diff < 0
}

/** ISO date (yyyy-mm-dd) for <input type="date">, in local time. */
export function toDateInput(iso: string | null): string {
  if (!iso) return ''
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

/** Converts a yyyy-mm-dd input into an ISO timestamp at local noon, avoiding timezone day shifts. */
export function fromDateInput(value: string): string | null {
  if (!value) return null
  const [y, m, d] = value.split('-').map(Number)
  return new Date(y!, m! - 1, d!, 12, 0, 0).toISOString()
}

/** ISO timestamp at local noon, `offset` days from today. */
export function dateInDays(offset: number): string {
  const d = new Date()
  d.setDate(d.getDate() + offset)
  return new Date(d.getFullYear(), d.getMonth(), d.getDate(), 12, 0, 0).toISOString()
}

/** Two-letter initials for avatars. */
export function initials(name: string): string {
  const parts = name
    .trim()
    .split(/[\s@._-]+/)
    .filter(Boolean)
  const first = parts[0]?.[0] ?? '?'
  const second = parts[1]?.[0] ?? parts[0]?.[1] ?? ''
  return (first + second).toUpperCase()
}
