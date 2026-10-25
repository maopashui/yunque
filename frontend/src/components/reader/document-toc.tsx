import { useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import type { TOCItem } from '@/lib/api'
import { cn } from '@/lib/utils'

type Props = {
  items: TOCItem[]
  activeId?: string | null
  onNavigate: (item: TOCItem) => void
  className?: string
}

export function DocumentTOC({ items, activeId, onNavigate, className }: Props) {
  const { t } = useTranslation()
  const navRef = useRef<HTMLElement | null>(null)

  useEffect(() => {
    if (!activeId || !navRef.current) return
    const el = navRef.current.querySelector<HTMLElement>(`[data-toc-id="${CSS.escape(activeId)}"]`)
    el?.scrollIntoView({ block: 'nearest', inline: 'nearest' })
  }, [activeId])

  return (
    <div className={cn(className)}>
      <div className="mb-3 text-sm font-bold tracking-tight text-zinc-900">
        {t('toc.title')}
      </div>
      {items.length === 0 ? (
        <p className="text-xs text-zinc-400">{t('toc.empty')}</p>
      ) : (
        <nav ref={navRef} className="flex flex-col gap-0.5">
          {items.map((item) => (
            <button
              key={item.id}
              type="button"
              data-toc-id={item.id}
              onClick={() => onNavigate(item)}
              className={cn(
                'rounded-lg px-2 py-1.5 text-left text-[13px] leading-5 text-zinc-500 transition-colors hover:bg-zinc-100 hover:text-zinc-900',
                activeId === item.id && 'bg-zinc-900 font-medium text-white hover:bg-zinc-900 hover:text-white',
              )}
              style={{ paddingLeft: `${0.5 + Math.max(0, item.level - 1) * 0.75}rem` }}
            >
              <span className="line-clamp-2">{item.text}</span>
            </button>
          ))}
        </nav>
      )}
    </div>
  )
}
