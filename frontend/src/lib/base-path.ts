/** App mount path without trailing slash; empty string when hosted at domain root. */
export function appBase(): string {
  const base = import.meta.env.BASE_URL || '/'
  if (base === '/') return ''
  return base.replace(/\/$/, '')
}

/** Prefix an absolute app path (e.g. `/api/health`) with the build-time base. */
export function withBase(path: string): string {
  if (!path.startsWith('/')) return path
  return `${appBase()}${path}`
}
