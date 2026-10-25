import { useEffect, useMemo, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { useVirtualizer } from '@tanstack/react-virtual'
import { BlockView } from '@/components/blocks/block-view'
import { useBlockCache } from '@/hooks/use-block-cache'
import { estimateBlockSize } from '@/lib/block-size'
import type { BlockMeta, TOCItem } from '@/lib/api'

type Props = {
  documentId: string
  index: BlockMeta[]
  scrollToIndex?: number | null
  onScrollToIndexHandled?: () => void
  onActiveHeadingChange?: (item: TOCItem | null) => void
  toc: TOCItem[]
}

/** Reading line just below the viewport top — last heading at/above this is active. */
const ACTIVE_HEADING_OFFSET_PX = 12

function resolveActiveHeading(
  virtualizer: {
    getOffsetForIndex: (
      index: number,
      align?: 'start' | 'center' | 'end' | 'auto',
    ) => readonly [number, string] | undefined
  },
  toc: TOCItem[],
  scrollOffset: number,
): TOCItem | null {
  if (!toc.length) return null
  const marker = scrollOffset + ACTIVE_HEADING_OFFSET_PX
  let active: TOCItem | null = null
  for (const item of toc) {
    const info = virtualizer.getOffsetForIndex(item.index, 'start')
    if (!info) continue
    const top = info[0]
    if (top <= marker) active = item
    else break
  }
  return active
}

export function VirtualReader({
  documentId,
  index,
  scrollToIndex,
  onScrollToIndexHandled,
  onActiveHeadingChange,
  toc,
}: Props) {
  const { t } = useTranslation()
  const parentRef = useRef<HTMLDivElement>(null)
  const lastActiveIdRef = useRef<string | null>(null)
  const { cache, ensureRange, ensureAround } = useBlockCache(documentId, index)

  const virtualizer = useVirtualizer({
    count: index.length,
    getScrollElement: () => parentRef.current,
    estimateSize: (i) => estimateBlockSize(index[i]!),
    overscan: 16,
    getItemKey: (i) => index[i]?.id ?? i,
  })

  const virtualItems = virtualizer.getVirtualItems()
  const scrollOffset = virtualizer.scrollOffset ?? 0

  useEffect(() => {
    if (!virtualItems.length) return
    const start = virtualItems[0]!.index
    const end = virtualItems[virtualItems.length - 1]!.index
    void ensureRange(start, end)
  }, [virtualItems, ensureRange])

  useEffect(() => {
    if (scrollToIndex == null || scrollToIndex < 0 || scrollToIndex >= index.length) return
    const meta = index[scrollToIndex]
    if (!meta) return
    void (async () => {
      await ensureAround(meta.id, 40, 60)
      virtualizer.scrollToIndex(scrollToIndex, { align: 'start' })
      onScrollToIndexHandled?.()
    })()
  }, [scrollToIndex, index, ensureAround, virtualizer, onScrollToIndexHandled])

  const tocOrdered = useMemo(
    () => [...toc].sort((a, b) => a.index - b.index),
    [toc],
  )

  useEffect(() => {
    if (!onActiveHeadingChange) return
    const active = resolveActiveHeading(virtualizer, tocOrdered, scrollOffset)
    const nextId = active?.id ?? null
    if (nextId === lastActiveIdRef.current) return
    lastActiveIdRef.current = nextId
    onActiveHeadingChange(active)
  }, [scrollOffset, tocOrdered, onActiveHeadingChange, virtualizer, virtualItems])

  if (index.length === 0) {
    return <p className="text-sm text-muted-foreground">{t('virtualReader.empty')}</p>
  }

  return (
    <div className="relative">
      <div
        ref={parentRef}
        className="h-[min(72vh,780px)] overflow-auto"
      >
        <div
          className="relative w-full py-1"
          style={{ height: `${virtualizer.getTotalSize()}px` }}
        >
          {virtualItems.map((item) => {
            const meta = index[item.index]!
            const block = cache[meta.id]
            return (
              <div
                key={item.key}
                data-index={item.index}
                data-block-id={meta.id}
                ref={virtualizer.measureElement}
                className="absolute left-0 top-0 w-full"
                style={{ transform: `translateY(${item.start}px)` }}
              >
                {block ? (
                  <BlockView block={block} />
                ) : (
                  <div className="my-3 h-10 animate-pulse rounded-md bg-surface-hover" />
                )}
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}
