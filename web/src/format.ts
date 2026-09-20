/** Small date helpers shared by the views. */

const dateFmt = new Intl.DateTimeFormat('it-IT', { day: 'numeric', month: 'short' })
const fullFmt = new Intl.DateTimeFormat('it-IT', { day: 'numeric', month: 'long', year: 'numeric' })

export function formatDue(iso: string | null): string {
  if (!iso) return ''
  const d = new Date(iso)
  const today = new Date()
  const diffDays = Math.round((startOfDay(d) - startOfDay(today)) / 86_400_000)
  if (diffDays === 0) return 'Oggi'
  if (diffDays === 1) return 'Domani'
  if (diffDays === -1) return 'Ieri'
  return d.getFullYear() === today.getFullYear() ? dateFmt.format(d) : fullFmt.format(d)
}

export function isOverdue(iso: string | null): boolean {
  return !!iso && startOfDay(new Date(iso)) < startOfDay(new Date())
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

function startOfDay(d: Date): number {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime()
}
