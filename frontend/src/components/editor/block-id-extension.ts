import { Extension } from '@tiptap/core'
import { Plugin, PluginKey } from '@tiptap/pm/state'

const BLOCK_TYPES = new Set([
  'paragraph',
  'heading',
  'bulletList',
  'orderedList',
  'codeBlock',
  'blockquote',
  'horizontalRule',
  'table',
  'image',
])

function newBlockId() {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `b-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`
}

export const BlockIdExtension = Extension.create({
  name: 'blockId',

  addGlobalAttributes() {
    return [
      {
        types: [...BLOCK_TYPES],
        attributes: {
          blockId: {
            default: null,
            parseHTML: (element) => element.getAttribute('data-block-id'),
            renderHTML: (attributes) => {
              if (!attributes.blockId) return {}
              return { 'data-block-id': attributes.blockId }
            },
          },
        },
      },
    ]
  },

  addProseMirrorPlugins() {
    return [
      new Plugin({
        key: new PluginKey('ensureBlockIds'),
        appendTransaction(_transactions, _oldState, newState) {
          const tr = newState.tr
          let modified = false
          // Enter/split copies attrs (including blockId); reassign duplicates
          // so materialize ReplaceAllBlocks does not hit UNIQUE conflicts.
          const seen = new Set<string>()
          newState.doc.forEach((node, offset) => {
            if (!BLOCK_TYPES.has(node.type.name)) return
            const current = node.attrs.blockId
            const valid = typeof current === 'string' && current.length > 0
            if (valid && !seen.has(current)) {
              seen.add(current)
              return
            }
            const id = newBlockId()
            tr.setNodeMarkup(offset, undefined, {
              ...node.attrs,
              blockId: id,
            })
            seen.add(id)
            modified = true
          })
          return modified ? tr : null
        },
      }),
    ]
  },
})
