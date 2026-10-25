import { Extension } from '@tiptap/core'

/**
 * Enter inserts a hard break (same paragraph, no extra block gap).
 * Shift-Enter creates a new block/paragraph.
 * Lists and code blocks keep their default Enter behavior.
 */
export const EnterAsHardBreak = Extension.create({
  name: 'enterAsHardBreak',
  // After HardBreak / StarterKit so our Enter wins over default splitBlock.
  priority: 1010,
  addKeyboardShortcuts() {
    return {
      Enter: ({ editor }) => {
        if (editor.isActive('codeBlock') || editor.isActive('listItem')) {
          return false
        }
        // Leave headings via Enter → new paragraph (not a break inside the title).
        if (editor.isActive('heading')) {
          return editor.commands.splitBlock()
        }
        return editor.commands.setHardBreak()
      },
      'Shift-Enter': ({ editor }) => {
        if (editor.isActive('codeBlock')) {
          return false
        }
        if (editor.isActive('listItem')) {
          return editor.commands.setHardBreak()
        }
        return editor.commands.splitBlock()
      },
    }
  },
})
