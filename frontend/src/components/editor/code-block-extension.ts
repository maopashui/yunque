import CodeBlockLowlight from '@tiptap/extension-code-block-lowlight'
import { ReactNodeViewRenderer } from '@tiptap/react'
import { CodeBlockNodeView } from '@/components/editor/code-block-view'
import { codeLowlight } from '@/lib/code-languages'

/** Code block with lowlight highlighting + title / language NodeView. */
export const CodeBlockWithLanguage = CodeBlockLowlight.extend({
  addOptions() {
    return {
      ...this.parent?.(),
      lowlight: codeLowlight,
      defaultLanguage: null,
      HTMLAttributes: {
        class: 'hljs',
      },
    }
  },

  addAttributes() {
    return {
      ...this.parent?.(),
      title: {
        default: '',
        parseHTML: (element) => element.getAttribute('data-title') ?? '',
        renderHTML: (attributes) => {
          const title = String(attributes.title ?? '').trim()
          if (!title) return {}
          return { 'data-title': title }
        },
      },
    }
  },

  addNodeView() {
    return ReactNodeViewRenderer(CodeBlockNodeView)
  },
})
