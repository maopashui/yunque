import { useEffect, useRef, useState, type ChangeEvent, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import type { Editor } from '@tiptap/react'
import {
  Baseline,
  Bold,
  Code2,
  Columns3,
  Heading1,
  Heading2,
  Heading3,
  Highlighter,
  Image as ImageIcon,
  Italic,
  Link as LinkIcon,
  List,
  ListOrdered,
  Minus,
  Plus,
  Quote,
  Redo2,
  Rows3,
  Strikethrough,
  Table,
  Trash2,
  Underline,
  Undo2,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { PromptDialog } from '@/components/ui/prompt-dialog'
import { ApiRequestError } from '@/lib/api'
import { uploadDocumentImage } from '@/lib/editor-image-upload'
import { cn } from '@/lib/utils'

const TEXT_COLORS = [
  { value: '', labelKey: 'default' as const },
  { value: '#18181b', swatch: '#18181b' },
  { value: '#dc2626', swatch: '#dc2626' },
  { value: '#ea580c', swatch: '#ea580c' },
  { value: '#ca8a04', swatch: '#ca8a04' },
  { value: '#16a34a', swatch: '#16a34a' },
  { value: '#0891b2', swatch: '#0891b2' },
  { value: '#2563eb', swatch: '#2563eb' },
  { value: '#7c3aed', swatch: '#7c3aed' },
  { value: '#db2777', swatch: '#db2777' },
]

const BG_COLORS = [
  { value: '', labelKey: 'default' as const },
  { value: '#fef08a', swatch: '#fef08a' },
  { value: '#bbf7d0', swatch: '#bbf7d0' },
  { value: '#bfdbfe', swatch: '#bfdbfe' },
  { value: '#fecaca', swatch: '#fecaca' },
  { value: '#fde68a', swatch: '#fde68a' },
  { value: '#ddd6fe', swatch: '#ddd6fe' },
  { value: '#fbcfe8', swatch: '#fbcfe8' },
  { value: '#e4e4e7', swatch: '#e4e4e7' },
]

/** Insert empty code block after the current top-level node (like Shift-Enter then code). */
function insertCodeBlockAfter(editor: Editor) {
  if (editor.isActive('codeBlock')) {
    editor.chain().focus().toggleCodeBlock().run()
    return
  }
  const { $from } = editor.state.selection
  if ($from.depth < 1) {
    editor.chain().focus().setCodeBlock().run()
    return
  }
  const insertPos = $from.after(1)
  editor
    .chain()
    .focus()
    .insertContentAt(insertPos, {
      type: 'codeBlock',
      attrs: { language: null, title: '' },
    })
    .setTextSelection(insertPos + 1)
    .run()
}

type Props = {
  editor: Editor | null
  documentId: string
  onUploadError?: (message: string) => void
}

function ToolButton({
  active,
  disabled,
  onClick,
  label,
  children,
}: {
  active?: boolean
  disabled?: boolean
  onClick: () => void
  label: string
  children: ReactNode
}) {
  return (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      aria-label={label}
      disabled={disabled}
      onClick={onClick}
      className={cn(
        'h-7 w-7 rounded-md',
        active && 'bg-surface-hover text-foreground',
      )}
    >
      {children}
    </Button>
  )
}

function ColorMenu({
  label,
  icon,
  colors,
  current,
  onPick,
  clearLabel,
}: {
  label: string
  icon: ReactNode
  colors: { value: string; swatch?: string; labelKey?: 'default' }[]
  current: string
  onPick: (color: string | null) => void
  clearLabel: string
}) {
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    function onPointerDown(e: PointerEvent) {
      if (rootRef.current?.contains(e.target as Node)) return
      setOpen(false)
    }
    document.addEventListener('pointerdown', onPointerDown)
    return () => document.removeEventListener('pointerdown', onPointerDown)
  }, [open])

  return (
    <div ref={rootRef} className="relative shrink-0">
      <Button
        type="button"
        variant="ghost"
        size="icon"
        aria-label={label}
        aria-expanded={open}
        onClick={() => setOpen((v) => !v)}
        className={cn('h-7 w-7 rounded-md', open && 'bg-surface-hover')}
      >
        <span className="relative inline-flex flex-col items-center">
          {icon}
          <span
            className="mt-0.5 h-0.5 w-3.5 rounded-full"
            style={{
              backgroundColor: current || (colors[1]?.swatch ?? '#18181b'),
            }}
          />
        </span>
      </Button>
      {open ? (
        <div
          role="menu"
          className="absolute left-0 top-full z-[80] mt-1.5 box-border grid w-max grid-cols-5 gap-2 rounded-lg border border-border bg-white p-2.5 shadow-lg"
        >
          {colors.map((c) => (
            <button
              key={c.value || 'clear'}
              type="button"
              role="menuitem"
              title={c.value || clearLabel}
              aria-label={c.value || clearLabel}
              className={cn(
                'box-border h-7 w-7 shrink-0 rounded-md border border-zinc-200 transition-transform hover:scale-105',
                !c.value &&
                  'bg-[linear-gradient(135deg,#fff_46%,#ef4444_46%,#ef4444_54%,#fff_54%)] text-[10px] font-mono text-transparent',
                current === c.value && 'ring-2 ring-zinc-900 ring-offset-1',
              )}
              style={c.swatch ? { backgroundColor: c.swatch } : undefined}
              onClick={() => {
                onPick(c.value || null)
                setOpen(false)
              }}
            />
          ))}
        </div>
      ) : null}
    </div>
  )
}

export function EditorToolbar({ editor, documentId, onUploadError }: Props) {
  const { t } = useTranslation()
  const fileRef = useRef<HTMLInputElement>(null)
  const [, setTick] = useState(0)
  const [linkOpen, setLinkOpen] = useState(false)
  const [linkInitial, setLinkInitial] = useState('https://')

  useEffect(() => {
    if (!editor) return
    const bump = () => setTick((n) => n + 1)
    editor.on('selectionUpdate', bump)
    editor.on('transaction', bump)
    return () => {
      editor.off('selectionUpdate', bump)
      editor.off('transaction', bump)
    }
  }, [editor])

  if (!editor) return null

  const inTable = editor.isActive('table')
  const textColor = String(editor.getAttributes('textStyle').color ?? '')
  const highlightColor = String(editor.getAttributes('highlight').color ?? '')

  const openLinkDialog = () => {
    const prev = editor.getAttributes('link').href as string | undefined
    setLinkInitial(prev?.trim() ? prev : 'https://')
    setLinkOpen(true)
  }

  const applyLink = (href: string) => {
    const trimmed = href.trim()
    if (!trimmed) {
      editor.chain().focus().extendMarkRange('link').unsetLink().run()
    } else {
      editor.chain().focus().extendMarkRange('link').setLink({ href: trimmed }).run()
    }
    setLinkOpen(false)
  }

  const clearLink = () => {
    editor.chain().focus().extendMarkRange('link').unsetLink().run()
    setLinkOpen(false)
  }

  const pickImage = () => {
    fileRef.current?.click()
  }

  const onFileChange = (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (!file || !file.type.startsWith('image/')) return
    void (async () => {
      try {
        const url = await uploadDocumentImage(documentId, file)
        editor.chain().focus().setImage({ src: url, alt: file.name || '' }).run()
      } catch (err) {
        const msg =
          err instanceof ApiRequestError
            ? err.message
            : t('editor.imageUploadFailed')
        onUploadError?.(msg)
      }
    })()
  }

  const icon = 'h-3.5 w-3.5'

  return (
    <div className="relative z-40 flex flex-wrap items-center gap-px overflow-visible rounded-md border border-border bg-surface px-1 py-0.5">
      <PromptDialog
        open={linkOpen}
        title={t('editor.toolbar.linkTitle')}
        description={t('editor.toolbar.linkPrompt')}
        label={t('editor.toolbar.linkLabel')}
        initialValue={linkInitial}
        confirmLabel={t('editor.toolbar.applyLink')}
        clearLabel={t('editor.toolbar.removeLink')}
        allowClear={editor.isActive('link')}
        onConfirm={applyLink}
        onClear={clearLink}
        onCancel={() => setLinkOpen(false)}
      />
      <input
        ref={fileRef}
        type="file"
        accept="image/png,image/jpeg,image/gif,image/webp"
        className="hidden"
        onChange={onFileChange}
      />
      <ToolButton
        label={t('editor.toolbar.undo')}
        disabled={!editor.can().undo()}
        onClick={() => editor.chain().focus().undo().run()}
      >
        <Undo2 className={icon} />
      </ToolButton>
      <ToolButton
        label={t('editor.toolbar.redo')}
        disabled={!editor.can().redo()}
        onClick={() => editor.chain().focus().redo().run()}
      >
        <Redo2 className={icon} />
      </ToolButton>
      <span className="mx-0.5 h-3.5 w-px bg-border" />
      <ToolButton
        label={t('editor.toolbar.bold')}
        active={editor.isActive('bold')}
        onClick={() => editor.chain().focus().toggleBold().run()}
      >
        <Bold className={icon} />
      </ToolButton>
      <ToolButton
        label={t('editor.toolbar.italic')}
        active={editor.isActive('italic')}
        onClick={() => editor.chain().focus().toggleItalic().run()}
      >
        <Italic className={icon} />
      </ToolButton>
      <ToolButton
        label={t('editor.toolbar.strike')}
        active={editor.isActive('strike')}
        onClick={() => editor.chain().focus().toggleStrike().run()}
      >
        <Strikethrough className={icon} />
      </ToolButton>
      <ToolButton
        label={t('editor.toolbar.underline')}
        active={editor.isActive('underline')}
        onClick={() => editor.chain().focus().toggleUnderline().run()}
      >
        <Underline className={icon} />
      </ToolButton>
      <ToolButton label={t('editor.toolbar.link')} onClick={openLinkDialog}>
        <LinkIcon className={icon} />
      </ToolButton>
      <ColorMenu
        label={t('editor.toolbar.textColor')}
        icon={<Baseline className={icon} />}
        colors={TEXT_COLORS}
        current={textColor}
        clearLabel={t('editor.toolbar.colorDefault')}
        onPick={(color) => {
          if (!color) {
            editor.chain().focus().unsetColor().run()
            return
          }
          editor.chain().focus().setColor(color).run()
        }}
      />
      <ColorMenu
        label={t('editor.toolbar.highlight')}
        icon={<Highlighter className={icon} />}
        colors={BG_COLORS}
        current={highlightColor}
        clearLabel={t('editor.toolbar.colorDefault')}
        onPick={(color) => {
          if (!color) {
            editor.chain().focus().unsetHighlight().run()
            return
          }
          editor.chain().focus().toggleHighlight({ color }).run()
        }}
      />
      <span className="mx-0.5 h-3.5 w-px bg-border" />
      <ToolButton
        label={t('editor.toolbar.h1')}
        active={editor.isActive('heading', { level: 1 })}
        onClick={() => editor.chain().focus().toggleHeading({ level: 1 }).run()}
      >
        <Heading1 className={icon} />
      </ToolButton>
      <ToolButton
        label={t('editor.toolbar.h2')}
        active={editor.isActive('heading', { level: 2 })}
        onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}
      >
        <Heading2 className={icon} />
      </ToolButton>
      <ToolButton
        label={t('editor.toolbar.h3')}
        active={editor.isActive('heading', { level: 3 })}
        onClick={() => editor.chain().focus().toggleHeading({ level: 3 }).run()}
      >
        <Heading3 className={icon} />
      </ToolButton>
      <span className="mx-0.5 h-3.5 w-px bg-border" />
      <ToolButton
        label={t('editor.toolbar.bulletList')}
        active={editor.isActive('bulletList')}
        onClick={() => editor.chain().focus().toggleBulletList().run()}
      >
        <List className={icon} />
      </ToolButton>
      <ToolButton
        label={t('editor.toolbar.orderedList')}
        active={editor.isActive('orderedList')}
        onClick={() => editor.chain().focus().toggleOrderedList().run()}
      >
        <ListOrdered className={icon} />
      </ToolButton>
      <ToolButton
        label={t('editor.toolbar.quote')}
        active={editor.isActive('blockquote')}
        onClick={() => editor.chain().focus().toggleBlockquote().run()}
      >
        <Quote className={icon} />
      </ToolButton>
      <ToolButton
        label={t('editor.toolbar.codeBlock')}
        active={editor.isActive('codeBlock')}
        onClick={() => insertCodeBlockAfter(editor)}
      >
        <Code2 className={icon} />
      </ToolButton>
      <ToolButton
        label={t('editor.toolbar.table')}
        active={inTable}
        onClick={() =>
          editor.chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run()
        }
      >
        <Table className={icon} />
      </ToolButton>
      {inTable ? (
        <>
          <ToolButton
            label={t('editor.toolbar.tableAddRow')}
            disabled={!editor.can().addRowAfter()}
            onClick={() => editor.chain().focus().addRowAfter().run()}
          >
            <span className="relative inline-flex">
              <Rows3 className={icon} />
              <Plus className="absolute -right-1 -top-1 h-2.5 w-2.5" />
            </span>
          </ToolButton>
          <ToolButton
            label={t('editor.toolbar.tableDeleteRow')}
            disabled={!editor.can().deleteRow()}
            onClick={() => editor.chain().focus().deleteRow().run()}
          >
            <span className="relative inline-flex">
              <Rows3 className={icon} />
              <Minus className="absolute -right-1 -top-1 h-2.5 w-2.5" />
            </span>
          </ToolButton>
          <ToolButton
            label={t('editor.toolbar.tableAddColumn')}
            disabled={!editor.can().addColumnAfter()}
            onClick={() => editor.chain().focus().addColumnAfter().run()}
          >
            <span className="relative inline-flex">
              <Columns3 className={icon} />
              <Plus className="absolute -right-1 -top-1 h-2.5 w-2.5" />
            </span>
          </ToolButton>
          <ToolButton
            label={t('editor.toolbar.tableDeleteColumn')}
            disabled={!editor.can().deleteColumn()}
            onClick={() => editor.chain().focus().deleteColumn().run()}
          >
            <span className="relative inline-flex">
              <Columns3 className={icon} />
              <Minus className="absolute -right-1 -top-1 h-2.5 w-2.5" />
            </span>
          </ToolButton>
          <ToolButton
            label={t('editor.toolbar.tableDelete')}
            disabled={!editor.can().deleteTable()}
            onClick={() => editor.chain().focus().deleteTable().run()}
          >
            <Trash2 className={cn(icon, 'text-red-600')} />
          </ToolButton>
        </>
      ) : null}
      <ToolButton label={t('editor.toolbar.image')} onClick={pickImage}>
        <ImageIcon className={icon} />
      </ToolButton>
    </div>
  )
}
