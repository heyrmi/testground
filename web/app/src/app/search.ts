/**
 * Search params are the Phase 0 control surface: a challenge's timings are
 * tunable per page load rather than random. Values are clamped rather than
 * rejected so a mistyped URL still produces a working page.
 */
export function clampInt(raw: unknown, fallback: number, min: number, max: number): number {
  const parsed = typeof raw === 'number' ? raw : Number.parseInt(String(raw ?? ''), 10)
  if (!Number.isFinite(parsed)) return fallback
  return Math.min(max, Math.max(min, Math.trunc(parsed)))
}
