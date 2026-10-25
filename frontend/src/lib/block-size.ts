import type { BlockMeta } from '@/lib/api'

export function estimateBlockSize(meta: BlockMeta): number {
  switch (meta.block_type) {
    case 'heading': {
      const level = Number(meta.attrs?.level ?? 1)
      if (level <= 1) return 64
      if (level === 2) return 52
      return 44
    }
    case 'code_block':
      return 220
    case 'table':
      return 260
    case 'blockquote':
      return 96
    case 'bullet_list':
    case 'ordered_list':
      return 88
    case 'thematic_break':
      return 40
    case 'image':
      return 220
    default:
      return 56
  }
}
