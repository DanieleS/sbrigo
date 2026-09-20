/** Position for an item dropped between two neighbours (either may be missing). */
export function positionBetween(prev: number | undefined, next: number | undefined): number {
  if (prev === undefined && next === undefined) return Date.now()
  if (prev === undefined) return next! - 1000
  if (next === undefined) return prev + 1000
  return (prev + next) / 2
}
