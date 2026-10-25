import { useCallback, useRef, useState } from 'react'
import { api, type Block, type BlockMeta } from '@/lib/api'

const WINDOW = 40
const PREFETCH = 20

export function useBlockCache(documentId: string, index: BlockMeta[]) {
  const [cache, setCache] = useState<Record<string, Block>>({})
  const cacheRef = useRef(cache)
  cacheRef.current = cache
  const inflight = useRef(new Set<string>())
  const indexRef = useRef(index)
  indexRef.current = index

  const merge = useCallback((blocks: Block[]) => {
    if (!blocks.length) return
    setCache((prev) => {
      const next = { ...prev }
      let changed = false
      for (const b of blocks) {
        if (next[b.id]?.revision === b.revision) continue
        next[b.id] = b
        changed = true
      }
      return changed ? next : prev
    })
  }, [])

  const ensureRange = useCallback(
    async (start: number, end: number) => {
      const metas = indexRef.current
      if (!metas.length || !documentId) return

      const from = Math.max(0, start - PREFETCH)
      const to = Math.min(metas.length - 1, end + PREFETCH)
      const missing: string[] = []
      for (let i = from; i <= to; i++) {
        const id = metas[i]?.id
        if (!id) continue
        if (cacheRef.current[id] || inflight.current.has(id)) continue
        missing.push(id)
      }
      if (!missing.length) return

      for (const id of missing) inflight.current.add(id)
      try {
        for (let i = 0; i < missing.length; i += WINDOW) {
          const chunk = missing.slice(i, i + WINDOW)
          const res = await api.listBlocksBatch(documentId, chunk)
          merge(res.blocks)
        }
      } finally {
        for (const id of missing) inflight.current.delete(id)
      }
    },
    [documentId, merge],
  )

  const ensureAround = useCallback(
    async (anchorId: string, before = 30, after = 50) => {
      if (!documentId) return
      const res = await api.listBlocksWindow(documentId, {
        anchor_id: anchorId,
        before,
        after,
      })
      merge(res.blocks)
    },
    [documentId, merge],
  )

  return { cache, ensureRange, ensureAround }
}
