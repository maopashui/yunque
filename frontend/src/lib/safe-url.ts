/** Allow http(s)/mailto and relative/hash links; reject javascript:, data:, etc. */
export function safeLinkHref(raw: unknown): string | null {
  if (typeof raw !== 'string') return null
  const href = raw.trim()
  if (!href) return null
  if (
    href.startsWith('/') ||
    href.startsWith('#') ||
    href.startsWith('./') ||
    href.startsWith('../')
  ) {
    if (/[\r\n\0]/.test(href)) return null
    return href
  }
  try {
    const u = new URL(href)
    const scheme = u.protocol.replace(/:$/, '').toLowerCase()
    if (scheme === 'http' || scheme === 'https' || scheme === 'mailto') {
      return href
    }
  } catch {
    return null
  }
  return null
}

/** Allow http(s) and relative image paths. */
export function safeImageSrc(raw: unknown): string | null {
  if (typeof raw !== 'string') return null
  const src = raw.trim()
  if (!src) return null
  if (src.startsWith('/') || src.startsWith('./') || src.startsWith('../')) {
    if (/[\r\n\0]/.test(src)) return null
    return src
  }
  try {
    const u = new URL(src)
    const scheme = u.protocol.replace(/:$/, '').toLowerCase()
    if (scheme === 'http' || scheme === 'https') {
      return src
    }
  } catch {
    return null
  }
  return null
}
