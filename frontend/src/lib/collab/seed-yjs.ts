import { getSchema, type Extensions } from '@tiptap/core'
import {
  prosemirrorJSONToYXmlFragment,
  yXmlFragmentToProseMirrorRootNode,
} from '@tiptap/y-tiptap'
import type { Block } from '@/lib/api'
import { blocksToTiptapDoc } from '@/lib/blocks-tiptap'
import type { YunQueCollabProvider } from '@/lib/collab/provider'

/** True when SQL/import blocks carry user-visible text (not just empty paragraphs). */
export function blocksHaveMeaningfulText(blocks: Block[]): boolean {
  for (const b of blocks) {
    const c = b.content ?? {}
    if (typeof c.text === 'string' && c.text.trim()) return true
    if (typeof c.markdown === 'string' && c.markdown.trim()) return true
    if (typeof c.src === 'string' && c.src.trim()) return true
    if (Array.isArray(c.inlines)) {
      for (const n of c.inlines as { text?: string; type?: string }[]) {
        if (n.type === 'image') return true
        if (typeof n.text === 'string' && n.text.trim()) return true
      }
    }
    if (Array.isArray(c.items) && c.items.length > 0) return true
    if (Array.isArray(c.rows) && c.rows.length > 0) return true
    if (Array.isArray(c.blocks) && c.blocks.length > 0) return true
  }
  return false
}

export function yFragmentPlainText(
  provider: YunQueCollabProvider,
  contentExtensions: Extensions,
): string {
  const fragment = provider.doc.getXmlFragment('default')
  if (fragment.length === 0) return ''
  try {
    const schema = getSchema(contentExtensions)
    const node = yXmlFragmentToProseMirrorRootNode(fragment, schema)
    return node.textContent.trim()
  } catch {
    return ''
  }
}

/**
 * When collab CRDT has no text but SQL blocks do (Markdown import / empty snapshot),
 * write blocks into Y.XmlFragment *before* TipTap Collaboration binds — setContent
 * alone does not reliably update Yjs under Collaboration.
 */
export function seedYjsFromBlocks(
  provider: YunQueCollabProvider,
  blocks: Block[],
  contentExtensions: Extensions,
): boolean {
  if (!provider.isSynced || !blocksHaveMeaningfulText(blocks)) return false
  if (yFragmentPlainText(provider, contentExtensions).length > 0) return false

  const schema = getSchema(contentExtensions)
  const fragment = provider.doc.getXmlFragment('default')
  const json = blocksToTiptapDoc(blocks)

  // Replace any empty placeholder structure left by a prior blank session.
  provider.doc.transact(() => {
    while (fragment.length > 0) {
      fragment.delete(0, fragment.length)
    }
    prosemirrorJSONToYXmlFragment(schema, json, fragment)
  })

  if (yFragmentPlainText(provider, contentExtensions).length === 0) {
    throw new Error('seedYjsFromBlocks: Y fragment still empty after write')
  }

  provider.pushSnapshot()
  return true
}
