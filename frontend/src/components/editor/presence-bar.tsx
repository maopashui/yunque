import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { PresencePeer } from '@/lib/collab/provider'
import { cn } from '@/lib/utils'

type Props = {
  peers: PresencePeer[]
  selfId?: string
  className?: string
}

const MAX_VISIBLE = 3

function initialOf(name: string) {
  const t = (name || '?').trim()
  return t.slice(0, 1) || '?'
}

export function PresenceBar({ peers, selfId, className }: Props) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)

  const list = useMemo(() => {
    const unique = new Map<string, PresencePeer>()
    for (const p of peers) {
      if (!p.user?.id) continue
      unique.set(p.user.id, p)
    }
    const arr = Array.from(unique.values())
    // Put self first for stable, recognizable stack
    arr.sort((a, b) => {
      const aSelf = selfId != null && a.user.id === selfId ? 0 : 1
      const bSelf = selfId != null && b.user.id === selfId ? 0 : 1
      return aSelf - bSelf
    })
    return arr
  }, [peers, selfId])

  if (list.length === 0) return null

  const visible = list.slice(0, MAX_VISIBLE)
  const overflow = list.length - visible.length

  return (
    <div
      className={cn('relative hidden md:block', className)}
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => setOpen(false)}
      onFocus={() => setOpen(true)}
      onBlur={() => setOpen(false)}
    >
      <button
        type="button"
        aria-label={t('presence.collaborating', { count: list.length })}
        className="inline-flex h-[27px] items-center gap-1.5 rounded-md border border-zinc-200/80 bg-zinc-100 py-1 pl-1 pr-2.5 font-mono text-[11px] text-zinc-700 transition-colors hover:bg-zinc-200"
      >
        <span className="flex items-center">
          {visible.map((p, i) => {
            const isSelf = selfId != null && p.user.id === selfId
            const isLast = i === visible.length - 1 && overflow === 0
            return (
              <span
                key={p.user.id}
                className={cn(
                  'relative flex h-[18px] w-[18px] items-center justify-center rounded-full border border-white text-[9px] font-bold text-white',
                  i > 0 && '-ml-1.5',
                )}
                style={{ background: p.user.color, zIndex: visible.length - i }}
                title={
                  isSelf ? `${p.user.name}${t('presence.me')}` : p.user.name
                }
              >
                {initialOf(p.user.name)}
                {isLast ? (
                  <span className="absolute -bottom-0.5 -right-0.5 h-1.5 w-1.5 rounded-full border border-white bg-emerald-500" />
                ) : null}
              </span>
            )
          })}
          {overflow > 0 ? (
            <span
              className="relative -ml-1.5 flex h-[18px] w-[18px] items-center justify-center rounded-full border border-white bg-zinc-800 text-[9px] font-bold text-white"
              style={{ zIndex: 0 }}
              aria-hidden
            >
              …
            </span>
          ) : null}
        </span>
        <span className="font-semibold text-zinc-700">
          {t('presence.collaborating', { count: list.length })}
        </span>
      </button>

      {open ? (
        <div
          role="tooltip"
          className="absolute right-0 top-full z-50 mt-1.5 min-w-[12rem] overflow-hidden rounded-lg border border-zinc-200 bg-white py-1.5 shadow-lg"
        >
          <div className="border-b border-zinc-100 px-3 py-1.5 font-mono text-[11px] font-medium text-zinc-500">
            {t('presence.allCollaborators', { count: list.length })}
          </div>
          <ul className="max-h-64 overflow-y-auto py-1">
            {list.map((p) => {
              const isSelf = selfId != null && p.user.id === selfId
              return (
                <li
                  key={p.user.id}
                  className="flex items-center gap-2 px-3 py-1.5 text-sm text-zinc-800"
                >
                  <span
                    className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-[10px] font-bold text-white"
                    style={{ background: p.user.color }}
                  >
                    {initialOf(p.user.name)}
                  </span>
                  <span className="min-w-0 truncate">
                    {p.user.name}
                    {isSelf ? t('presence.me') : ''}
                  </span>
                  <span className="ml-auto h-1.5 w-1.5 shrink-0 rounded-full bg-emerald-500" />
                </li>
              )
            })}
          </ul>
        </div>
      ) : null}
    </div>
  )
}
