/** Fixed palette — keep in sync with backend/internal/auth/avatar_color.go */
export const AVATAR_PALETTE = [
  '#0d9488', // teal-600
  '#059669', // emerald-600
  '#0284c7', // sky-600
  '#0891b2', // cyan-600
  '#d97706', // amber-600
  '#ea580c', // orange-600
  '#e11d48', // rose-600
  '#3f3f46', // zinc-700
] as const

function hashId(id: string): number {
  let h = 2166136261
  for (let i = 0; i < id.length; i++) {
    h ^= id.charCodeAt(i)
    h = Math.imul(h, 16777619)
  }
  return h >>> 0
}

export function avatarColorForId(id: string): string {
  return AVATAR_PALETTE[hashId(id) % AVATAR_PALETTE.length]!
}

export function resolveAvatarColor(id: string, stored?: string | null): string {
  if (stored && stored.trim()) return stored.trim()
  return avatarColorForId(id || '?')
}

export function avatarStyle(color: string): { backgroundColor: string } {
  return { backgroundColor: color }
}
