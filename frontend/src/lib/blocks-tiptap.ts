import type { JSONContent } from '@tiptap/react'
import type { Block } from '@/lib/api'

export type ContentBlockInput = {
  id?: string
  block_type: string
  content: Record<string, unknown>
  attrs: Record<string, unknown>
}

type Inline = {
  type: string
  text?: string
  marks?: { type: string; attrs?: Record<string, unknown> }[]
  attrs?: Record<string, unknown>
}

function inlineToTiptap(inlines: Inline[] | undefined): JSONContent[] {
  if (!inlines?.length) return []
  const out: JSONContent[] = []
  for (const node of inlines) {
    if (node.type === 'image') {
      out.push({
        type: 'image',
        attrs: {
          src: String(node.attrs?.src ?? ''),
          alt: String(node.attrs?.alt ?? ''),
        },
      })
      continue
    }
    const marks: JSONContent['marks'] = []
    for (const m of node.marks ?? []) {
      if (m.type === 'bold') marks.push({ type: 'bold' })
      else if (m.type === 'italic') marks.push({ type: 'italic' })
      else if (m.type === 'strike') marks.push({ type: 'strike' })
      else if (m.type === 'code') marks.push({ type: 'code' })
      else if (m.type === 'underline') marks.push({ type: 'underline' })
      else if (m.type === 'link') {
        marks.push({ type: 'link', attrs: { href: String(m.attrs?.href ?? '') } })
      } else if (m.type === 'textStyle' && m.attrs?.color) {
        marks.push({ type: 'textStyle', attrs: { color: String(m.attrs.color) } })
      } else if (m.type === 'highlight') {
        const color = m.attrs?.color != null ? String(m.attrs.color) : undefined
        marks.push({
          type: 'highlight',
          ...(color ? { attrs: { color } } : {}),
        })
      }
    }
    out.push({
      type: 'text',
      text: node.text ?? '',
      ...(marks.length ? { marks } : {}),
    })
  }
  return out
}

function tiptapToInlines(nodes: JSONContent[] | undefined): Inline[] {
  const out: Inline[] = []
  for (const n of nodes ?? []) {
    if (n.type === 'image') {
      out.push({
        type: 'image',
        attrs: {
          src: String(n.attrs?.src ?? ''),
          alt: String(n.attrs?.alt ?? ''),
        },
      })
      continue
    }
    if (n.type === 'hardBreak') {
      out.push({ type: 'text', text: '\n' })
      continue
    }
    if (n.type === 'text') {
      const marks: Inline['marks'] = []
      for (const m of n.marks ?? []) {
        if (m.type === 'bold') marks!.push({ type: 'bold' })
        else if (m.type === 'italic') marks!.push({ type: 'italic' })
        else if (m.type === 'strike') marks!.push({ type: 'strike' })
        else if (m.type === 'code') marks!.push({ type: 'code' })
        else if (m.type === 'underline') marks!.push({ type: 'underline' })
        else if (m.type === 'link') {
          marks!.push({ type: 'link', attrs: { href: String(m.attrs?.href ?? '') } })
        } else if (m.type === 'textStyle' && m.attrs?.color) {
          marks!.push({ type: 'textStyle', attrs: { color: String(m.attrs.color) } })
        } else if (m.type === 'highlight') {
          const color = m.attrs?.color != null ? String(m.attrs.color) : undefined
          marks!.push({
            type: 'highlight',
            ...(color ? { attrs: { color } } : {}),
          })
        }
      }
      out.push({
        type: 'text',
        text: n.text ?? '',
        ...(marks!.length ? { marks } : {}),
      })
    }
  }
  return out
}

function listItemsToTiptap(items: unknown, ordered: boolean): JSONContent {
  const list = (items as { inlines?: Inline[]; children?: unknown[] }[]) ?? []
  return {
    type: ordered ? 'orderedList' : 'bulletList',
    content: list.map((item) => {
      const children: JSONContent[] = [
        {
          type: 'paragraph',
          content: inlineToTiptap(item.inlines),
        },
      ]
      if (item.children?.length) {
        children.push(listItemsToTiptap(item.children, ordered))
      }
      return { type: 'listItem', content: children }
    }),
  }
}

type ListItemModel = {
  inlines: Inline[]
  children?: ListItemModel[]
}

function tiptapListToItems(node: JSONContent): ListItemModel[] {
  const items: ListItemModel[] = []
  for (const li of node.content ?? []) {
    if (li.type !== 'listItem') continue
    let inlines: Inline[] = []
    let children: ListItemModel[] | undefined
    for (const child of li.content ?? []) {
      if (child.type === 'paragraph') {
        inlines = [...inlines, ...tiptapToInlines(child.content)]
      } else if (child.type === 'bulletList' || child.type === 'orderedList') {
        children = tiptapListToItems(child)
      }
    }
    items.push(children?.length ? { inlines, children } : { inlines })
  }
  return items
}

function blockToNode(block: Block): JSONContent | null {
  const content = block.content ?? {}
  const attrs = block.attrs ?? {}
  const blockId = block.id

  switch (block.block_type) {
    case 'heading':
      return {
        type: 'heading',
        attrs: { level: Number(attrs.level ?? 1), blockId },
        content: inlineToTiptap(content.inlines as Inline[]),
      }
    case 'paragraph':
      return {
        type: 'paragraph',
        attrs: { blockId },
        content: inlineToTiptap(content.inlines as Inline[]),
      }
    case 'bullet_list': {
      const node = listItemsToTiptap(content.items, false)
      node.attrs = { ...(node.attrs ?? {}), blockId }
      return node
    }
    case 'ordered_list': {
      const node = listItemsToTiptap(content.items, true)
      node.attrs = { ...(node.attrs ?? {}), blockId }
      return node
    }
    case 'code_block':
      return {
        type: 'codeBlock',
        attrs: {
          language: String(attrs.language ?? ''),
          title: String(attrs.title ?? ''),
          blockId,
        },
        content: content.text
          ? [{ type: 'text', text: String(content.text) }]
          : [],
      }
    case 'blockquote': {
      const inner = Array.isArray(content.blocks)
        ? (content.blocks as Record<string, unknown>[]).flatMap((b, i) => {
            const fake: Block = {
              id: `${block.id}-q-${i}`,
              document_id: block.document_id,
              order_key: String(i),
              block_type: String(b.block_type ?? 'paragraph'),
              content: (b.content as Record<string, unknown>) ?? {},
              attrs: (b.attrs as Record<string, unknown>) ?? {},
              revision: 1,
              created_at: block.created_at,
              updated_at: block.updated_at,
            }
            const n = blockToNode(fake)
            return n ? [n] : []
          })
        : []
      return {
        type: 'blockquote',
        attrs: { blockId },
        content: inner.length ? inner : [{ type: 'paragraph' }],
      }
    }
    case 'thematic_break':
      return { type: 'horizontalRule', attrs: { blockId } }
    case 'image':
      // TipTap Image defaults to a block node — do not wrap in paragraph
      // (invalid schema under Collaboration and breaks SQL→Yjs seed).
      return {
        type: 'image',
        attrs: {
          src: String(content.src ?? ''),
          alt: String(content.alt ?? ''),
          blockId,
        },
      }
    case 'table': {
      const rows = (content.rows as Inline[][][]) ?? []
      return {
        type: 'table',
        attrs: { blockId },
        content: rows.map((row) => ({
          type: 'tableRow',
          content: row.map((cell) => ({
            type: 'tableCell',
            content: [
              {
                type: 'paragraph',
                content: inlineToTiptap(cell),
              },
            ],
          })),
        })),
      }
    }
    case 'raw_markdown':
      return {
        type: 'codeBlock',
        attrs: { language: 'markdown', title: '', blockId },
        content: [{ type: 'text', text: String(content.markdown ?? '') }],
      }
    default:
      return {
        type: 'paragraph',
        attrs: { blockId },
        content: [{ type: 'text', text: `[unsupported:${block.block_type}]` }],
      }
  }
}

export function blocksToTiptapDoc(blocks: Block[]): JSONContent {
  const content = blocks.map(blockToNode).filter(Boolean) as JSONContent[]
  return {
    type: 'doc',
    content: content.length ? content : [{ type: 'paragraph' }],
  }
}

function nodeToBlock(node: JSONContent): ContentBlockInput | null {
  const blockId = typeof node.attrs?.blockId === 'string' ? node.attrs.blockId : undefined

  switch (node.type) {
    case 'heading':
      return {
        id: blockId,
        block_type: 'heading',
        content: { inlines: tiptapToInlines(node.content) },
        attrs: { level: Number(node.attrs?.level ?? 1) },
      }
    case 'paragraph': {
      const inlines = tiptapToInlines(node.content)
      if (inlines.length === 1 && inlines[0]?.type === 'image') {
        return {
          id: blockId,
          block_type: 'image',
          content: {
            src: String(inlines[0].attrs?.src ?? ''),
            alt: String(inlines[0].attrs?.alt ?? ''),
          },
          attrs: {},
        }
      }
      return {
        id: blockId,
        block_type: 'paragraph',
        content: { inlines },
        attrs: {},
      }
    }
    case 'image':
      return {
        id: blockId,
        block_type: 'image',
        content: {
          src: String(node.attrs?.src ?? ''),
          alt: String(node.attrs?.alt ?? ''),
        },
        attrs: {},
      }
    case 'bulletList':
      return {
        id: blockId,
        block_type: 'bullet_list',
        content: { items: tiptapListToItems(node) },
        attrs: {},
      }
    case 'orderedList':
      return {
        id: blockId,
        block_type: 'ordered_list',
        content: { items: tiptapListToItems(node) },
        attrs: {},
      }
    case 'codeBlock': {
      const text = (node.content ?? []).map((n) => n.text ?? '').join('')
      const title = String(node.attrs?.title ?? '').trim()
      return {
        id: blockId,
        block_type: 'code_block',
        content: { text },
        attrs: {
          language: String(node.attrs?.language ?? ''),
          ...(title ? { title } : {}),
        },
      }
    }
    case 'blockquote': {
      const inner = (node.content ?? [])
        .map(nodeToBlock)
        .filter(Boolean)
        .map((b) => ({
          block_type: b!.block_type,
          content: b!.content,
          attrs: b!.attrs,
        }))
      return {
        id: blockId,
        block_type: 'blockquote',
        content: { blocks: inner },
        attrs: {},
      }
    }
    case 'horizontalRule':
      return {
        id: blockId,
        block_type: 'thematic_break',
        content: {},
        attrs: {},
      }
    case 'table': {
      const rows: Inline[][][] = []
      for (const row of node.content ?? []) {
        if (row.type !== 'tableRow') continue
        const cells: Inline[][] = []
        for (const cell of row.content ?? []) {
          const para = (cell.content ?? []).find((c) => c.type === 'paragraph')
          cells.push(tiptapToInlines(para?.content))
        }
        rows.push(cells)
      }
      return {
        id: blockId,
        block_type: 'table',
        content: { rows },
        attrs: {},
      }
    }
    default:
      return null
  }
}

function ensureBlockId(id?: string): string {
  if (id && id.trim()) return id.trim()
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `b-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`
}

export function tiptapDocToBlocks(doc: JSONContent): ContentBlockInput[] {
  const out: ContentBlockInput[] = []
  const seen = new Set<string>()
  for (const node of doc.content ?? []) {
    const b = nodeToBlock(node)
    if (!b) continue
    let id = ensureBlockId(b.id)
    // Split/Enter can leave duplicate blockIds; SQL ReplaceAllBlocks requires uniqueness.
    if (seen.has(id)) id = ensureBlockId()
    seen.add(id)
    out.push({ ...b, id })
  }
  if (!out.length) {
    out.push({
      id: ensureBlockId(),
      block_type: 'paragraph',
      content: { inlines: [] },
      attrs: {},
    })
  }
  return out
}
