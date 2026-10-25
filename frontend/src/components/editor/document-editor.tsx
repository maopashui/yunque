import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type MutableRefObject,
  type ReactNode,
} from 'react'
import { useTranslation } from 'react-i18next'
import { getSchema } from '@tiptap/core'
import { EditorContent, useEditor, type Editor } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import { Color } from '@tiptap/extension-color'
import Highlight from '@tiptap/extension-highlight'
import Link from '@tiptap/extension-link'
import Image from '@tiptap/extension-image'
import Placeholder from '@tiptap/extension-placeholder'
import { TextStyle } from '@tiptap/extension-text-style'
import Underline from '@tiptap/extension-underline'
import { CodeBlockWithLanguage } from '@/components/editor/code-block-extension'
import { Table } from '@tiptap/extension-table'
import { TableRow } from '@tiptap/extension-table-row'
import { TableCell } from '@tiptap/extension-table-cell'
import { TableHeader } from '@tiptap/extension-table-header'
import Collaboration from '@tiptap/extension-collaboration'
import CollaborationCaret from '@tiptap/extension-collaboration-caret'
import {
  prosemirrorJSONToYXmlFragment,
} from '@tiptap/y-tiptap'
import { BlockIdExtension } from '@/components/editor/block-id-extension'
import { EnterAsHardBreak } from '@/components/editor/enter-as-hard-break'
import { EditorToolbar } from '@/components/editor/editor-toolbar'
import { useAuth } from '@/app/auth-context'
import i18n from '@/i18n'
import { api, ApiRequestError, type Block, type Document, type TOCItem } from '@/lib/api'
import { blocksToTiptapDoc, tiptapDocToBlocks } from '@/lib/blocks-tiptap'
import {
  collectImageFiles,
  uploadDocumentImage,
} from '@/lib/editor-image-upload'
import {
  makeCollabUser,
  YunQueCollabProvider,
  type CollabStatus,
  type CollabUser,
  type PresencePeer,
} from '@/lib/collab/provider'
import {
  blocksHaveMeaningfulText,
  seedYjsFromBlocks,
  yFragmentPlainText,
} from '@/lib/collab/seed-yjs'

export type SaveStatus =
  | 'connecting'
  | 'synced'
  | 'offline'
  | 'error'
  | 'materializing'
  | 'saved'
  | 'dirty'
  | 'idle'

type Props = {
  documentId: string
  initialBlocks: Block[]
  initialRevision: number
  titleSlot?: ReactNode
  onDocumentChange?: (doc: Document) => void
  onStatusChange?: (status: SaveStatus, message?: string) => void
  onDirtyChange?: (dirty: boolean) => void
  onTocChange?: (items: TOCItem[]) => void
  onPeersChange?: (peers: PresencePeer[]) => void
  scrollToHeadingIndex?: number | null
  onScrollToHeadingHandled?: () => void
}

const MATERIALIZE_MS = 3500

/** Content extensions shared by TipTap editor and SQL→Yjs seeding (no Collaboration). */
function buildContentExtensions() {
  return [
    StarterKit.configure({
      heading: { levels: [1, 2, 3, 4, 5, 6] },
      undoRedo: false,
      codeBlock: false,
    }),
    CodeBlockWithLanguage,
    Underline,
    TextStyle,
    Color,
    Highlight.configure({ multicolor: true }),
    Link.configure({ openOnClick: false }),
    Image.configure({ allowBase64: false }),
    Placeholder.configure({ placeholder: () => i18n.t('editor.placeholder') }),
    Table.configure({ resizable: false }),
    TableRow,
    TableHeader,
    TableCell,
    BlockIdExtension,
    EnterAsHardBreak,
  ]
}

function extractEditorTOC(editor: Editor): TOCItem[] {
  const items: TOCItem[] = []
  editor.state.doc.forEach((node, _offset, index) => {
    if (node.type.name === 'heading') {
      items.push({
        id: String(node.attrs.blockId ?? `heading-${index}`),
        order_key: '',
        index,
        level: Number(node.attrs.level ?? 1),
        text: node.textContent.trim() || i18n.t('common.untitled'),
      })
    }
  })
  return items
}

function findHeadingPos(editor: Editor, topLevelIndex: number): number | null {
  let found: number | null = null
  editor.state.doc.forEach((_node, offset, index) => {
    if (index === topLevelIndex) {
      found = offset
    }
  })
  return found
}

/** Outer shell: create collab provider, seed Yjs from SQL if needed, then mount TipTap. */
export function DocumentEditor(props: Props) {
  const { t } = useTranslation()
  const { user } = useAuth()
  const [provider, setProvider] = useState<YunQueCollabProvider | null>(null)
  const [status, setStatus] = useState<SaveStatus>('connecting')
  const [ready, setReady] = useState(false)
  const bootstrappedRef = useRef(false)
  const contentExtensions = useMemo(() => buildContentExtensions(), [])

  const collabUser = useMemo(
    () =>
      makeCollabUser(
        user?.id ?? 'anon',
        user?.display_name || user?.username || t('common.user'),
        user?.avatar_color,
      ),
    [user?.id, user?.display_name, user?.username, user?.avatar_color, t],
  )

  useEffect(() => {
    if (!user) return
    bootstrappedRef.current = false
    setReady(false)
    const p = new YunQueCollabProvider({
      documentId: props.documentId,
      user: collabUser,
      onStatus: (s: CollabStatus, message?: string) => {
        const mapped: SaveStatus =
          s === 'connecting'
            ? 'connecting'
            : s === 'synced'
              ? 'synced'
              : s === 'offline'
                ? 'offline'
                : 'error'
        setStatus(mapped)
        props.onStatusChange?.(mapped, message)
      },
      onPeers: (peers) => props.onPeersChange?.(peers),
    })
    setProvider(p)
    return () => {
      p.destroy()
      setProvider(null)
      setReady(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [props.documentId, user?.id])

  // Seed Yjs from SQL blocks before Collaboration binds (import / empty snapshot).
  useEffect(() => {
    if (!provider) return
    let cancelled = false
    const tryReady = () => {
      if (cancelled) return false
      if (!provider.isSynced) return false
      try {
        seedYjsFromBlocks(provider, props.initialBlocks, contentExtensions)
      } catch (err) {
        console.error('[yunque] SQL→Yjs seed failed', err)
        // Still mount the editor; inner bootstrap will retry. Never leave the UI stuck.
      }
      setReady(true)
      return true
    }
    if (tryReady()) return
    const id = window.setInterval(() => {
      if (tryReady()) window.clearInterval(id)
    }, 100)
    return () => {
      cancelled = true
      window.clearInterval(id)
    }
  }, [provider, props.initialBlocks, contentExtensions])

  if (!user || !provider || !ready) {
    return (
      <div className="space-y-3">
        <div className="sticky top-11 z-30 space-y-2 border-b border-border bg-background/95 pb-2 pt-1 backdrop-blur-md">
          {props.titleSlot}
        </div>
        <p className="font-mono text-[11px] text-muted" data-save-status="connecting">
          {t('editor.connecting')}
        </p>
      </div>
    )
  }

  return (
    <CollaborativeEditorInner
      {...props}
      provider={provider}
      collabUser={collabUser}
      contentExtensions={contentExtensions}
      status={status}
      setStatus={setStatus}
      bootstrappedRef={bootstrappedRef}
    />
  )
}

type InnerProps = Props & {
  provider: YunQueCollabProvider
  collabUser: CollabUser
  contentExtensions: ReturnType<typeof buildContentExtensions>
  status: SaveStatus
  setStatus: (s: SaveStatus) => void
  bootstrappedRef: MutableRefObject<boolean>
}

function pickMaterializeLeader(selfId: string, provider: YunQueCollabProvider): boolean {
  const ids = new Set<string>([selfId])
  provider.awareness.getStates().forEach((state) => {
    const u = state.user as CollabUser | undefined
    if (u?.id) ids.add(u.id)
  })
  const sorted = [...ids].sort()
  return sorted[0] === selfId
}

function CollaborativeEditorInner({
  documentId,
  initialBlocks,
  initialRevision,
  titleSlot,
  onDocumentChange,
  onStatusChange,
  onDirtyChange,
  onTocChange,
  scrollToHeadingIndex,
  onScrollToHeadingHandled,
  provider,
  collabUser,
  contentExtensions,
  status,
  setStatus,
  bootstrappedRef,
}: InnerProps) {
  const { t } = useTranslation()
  const revisionRef = useRef(Number(initialRevision) || 0)
  const dirtyRef = useRef(false)
  const materializingRef = useRef(false)
  const pendingMaterializeRef = useRef(false)
  const materializeTimerRef = useRef<number | null>(null)
  const editorRef = useRef<Editor | null>(null)
  const isLeaderRef = useRef(true)
  const readyToMaterializeRef = useRef(false)
  const initialBlocksRef = useRef(initialBlocks)
  initialBlocksRef.current = initialBlocks
  /** SQL/import had real content when this editor session opened — never wipe with empty PM. */
  const protectImportedContentRef = useRef(blocksHaveMeaningfulText(initialBlocks))

  useEffect(() => {
    const refreshLeader = () => {
      isLeaderRef.current = pickMaterializeLeader(collabUser.id, provider)
    }
    refreshLeader()
    provider.awareness.on('change', refreshLeader)
    return () => {
      provider.awareness.off('change', refreshLeader)
    }
  }, [provider, collabUser.id])

  const updateStatus = useCallback(
    (next: SaveStatus, message?: string) => {
      setStatus(next)
      onStatusChange?.(next, message)
    },
    [onStatusChange, setStatus],
  )

  const materialize = useCallback(
    async (editorInstance: Editor, opts?: { force?: boolean }) => {
      // Only one peer should write blocks; others rely on Yjs + leader's PUT.
      if (!opts?.force && !isLeaderRef.current) {
        dirtyRef.current = false
        onDirtyChange?.(false)
        updateStatus('synced')
        return
      }
      if (materializingRef.current) {
        pendingMaterializeRef.current = true
        return
      }

      const editorText = editorInstance.state.doc.textContent.trim()
      // Critical: after Markdown import, Collaboration may still show an empty PM doc
      // for a few frames while Y→PM binds. Materializing that empty JSON wipes SQL
      // (49 blocks → 1 empty paragraph within ~160ms). Refuse that wipe.
      if (
        editorText.length === 0 &&
        protectImportedContentRef.current &&
        blocksHaveMeaningfulText(initialBlocksRef.current)
      ) {
        dirtyRef.current = false
        onDirtyChange?.(false)
        updateStatus('synced')
        return
      }

      materializingRef.current = true
      updateStatus('materializing')
      try {
        do {
          pendingMaterializeRef.current = false
          const stillEmpty = editorInstance.state.doc.textContent.trim().length === 0
          if (
            stillEmpty &&
            protectImportedContentRef.current &&
            blocksHaveMeaningfulText(initialBlocksRef.current)
          ) {
            dirtyRef.current = false
            onDirtyChange?.(false)
            updateStatus('synced')
            return
          }
          const blocks = tiptapDocToBlocks(editorInstance.getJSON())
          const fresh = await api.getDocument(documentId)
          revisionRef.current = Number(fresh.document.revision) || 0

          const maxAttempts = 8
          let attempt = 0
          let done = false
          while (attempt < maxAttempts && !done) {
            attempt += 1
            try {
              const res = await api.syncContent(documentId, {
                base_revision: revisionRef.current,
                blocks,
              })
              revisionRef.current = Number(res.document.revision) || revisionRef.current
              dirtyRef.current = false
              onDirtyChange?.(false)
              onDocumentChange?.(res.document)
              updateStatus('saved')
              provider.pushSnapshot()
              done = true
            } catch (err) {
              const conflict =
                err instanceof ApiRequestError &&
                (err.code === 'version_conflict' || err.code === 'conflict')
              if (conflict) {
                // Peer likely materialized the same CRDT view — refresh and retry with jitter.
                await new Promise((r) =>
                  setTimeout(r, 80 * attempt + Math.floor(Math.random() * 120)),
                )
                const again = await api.getDocument(documentId)
                revisionRef.current = Number(again.document.revision) || 0
                onDocumentChange?.(again.document)
                if (attempt >= maxAttempts) {
                  // Soft success: another client won the race; reading view is still consistent enough.
                  dirtyRef.current = false
                  onDirtyChange?.(false)
                  updateStatus('saved')
                  done = true
                }
                continue
              }
              throw err
            }
          }
        } while (pendingMaterializeRef.current)
      } catch (err) {
        const message =
          err instanceof ApiRequestError
            ? err.message
            : err instanceof Error
              ? err.message
              : i18n.t('editor.materializeFailed')
        updateStatus('error', message)
      } finally {
        materializingRef.current = false
        if (pendingMaterializeRef.current && editorRef.current) {
          pendingMaterializeRef.current = false
          void materialize(editorRef.current, opts)
        }
      }
    },
    [documentId, onDirtyChange, onDocumentChange, provider, updateStatus],
  )

  const scheduleMaterialize = useCallback(
    (ed: Editor) => {
      onTocChange?.(extractEditorTOC(ed))
      // Avoid materializing the default empty doc before SQL→Yjs bootstrap finishes
      // (that would wipe imported Markdown from the reading store).
      if (!readyToMaterializeRef.current) return
      if (!isLeaderRef.current) {
        // Follower: Yjs already synced; skip PUT to avoid dual-writer 409 storms.
        dirtyRef.current = false
        onDirtyChange?.(false)
        updateStatus('synced')
        return
      }
      dirtyRef.current = true
      onDirtyChange?.(true)
      updateStatus('dirty')
      if (materializeTimerRef.current) window.clearTimeout(materializeTimerRef.current)
      const delay = MATERIALIZE_MS + Math.floor(Math.random() * 600)
      materializeTimerRef.current = window.setTimeout(() => {
        void materialize(ed)
      }, delay)
    },
    [materialize, onDirtyChange, onTocChange, updateStatus],
  )

  const extensions = useMemo(
    () => [
      ...contentExtensions,
      Collaboration.configure({
        document: provider.doc,
      }),
      CollaborationCaret.configure({
        provider,
        user: collabUser,
      }),
    ],
    [provider, collabUser, contentExtensions],
  )

  const insertUploadedImagesRef = useRef<(files: File[]) => void>(() => {})
  insertUploadedImagesRef.current = (files: File[]) => {
    const ed = editorRef.current
    if (!ed || files.length === 0) return
    void (async () => {
      for (const file of files) {
        try {
          updateStatus('materializing', t('editor.imageUploading'))
          const url = await uploadDocumentImage(documentId, file)
          ed.chain().focus().setImage({ src: url, alt: file.name || '' }).run()
          updateStatus('dirty')
        } catch (err) {
          const msg =
            err instanceof ApiRequestError
              ? err.message
              : t('editor.imageUploadFailed')
          updateStatus('error', msg)
        }
      }
    })()
  }

  const editor = useEditor({
    immediatelyRender: false,
    extensions,
    editorProps: {
      attributes: {
        class:
          'prose-editor min-h-[50vh] max-w-none px-1 py-2 text-[15px] leading-7 outline-none focus:outline-none',
      },
      handlePaste: (_view, event) => {
        const files = collectImageFiles(event.clipboardData)
        if (files.length === 0) return false
        event.preventDefault()
        insertUploadedImagesRef.current(files)
        return true
      },
      handleDrop: (_view, event) => {
        const files = collectImageFiles(event.dataTransfer)
        if (files.length === 0) return false
        event.preventDefault()
        insertUploadedImagesRef.current(files)
        return true
      },
    },
    onCreate: ({ editor: ed }) => {
      editorRef.current = ed
      onTocChange?.(extractEditorTOC(ed))
    },
    onUpdate: ({ editor: ed }) => {
      editorRef.current = ed
      scheduleMaterialize(ed)
    },
  })

  useEffect(() => {
    if (!editor) return
    readyToMaterializeRef.current = false
    let attempts = 0

    const finishBootstrap = (allowMaterialize: boolean) => {
      bootstrappedRef.current = true
      readyToMaterializeRef.current = allowMaterialize
      onTocChange?.(extractEditorTOC(editor))
    }

    const tryBootstrap = () => {
      if (bootstrappedRef.current) return true
      if (!provider.isSynced) return false
      attempts += 1

      const editorText = editor.state.doc.textContent.trim()
      const yText = yFragmentPlainText(provider, contentExtensions)
      const sqlHasContent = blocksHaveMeaningfulText(initialBlocks)

      // Prefer remote CRDT / bound editor when it already has real text.
      // Do NOT materialize here — SQL already has import content; wait for user edits.
      if (editorText.length > 0) {
        finishBootstrap(true)
        return true
      }

      if (yText.length > 0) {
        // Y has content but PM not bound yet — keep waiting briefly for Collaboration.
        if (attempts < 25) return false
        finishBootstrap(true)
        return true
      }

      if (!sqlHasContent) {
        finishBootstrap(true)
        return true
      }

      // Safety net: SQL has blocks but Y/editor still empty. Write Y first.
      // Never materialize until the editor actually shows text — that was wiping imports.
      const json = blocksToTiptapDoc(initialBlocks)
      try {
        provider.doc.transact(() => {
          const fragment = provider.doc.getXmlFragment('default')
          while (fragment.length > 0) fragment.delete(0, fragment.length)
          prosemirrorJSONToYXmlFragment(editor.schema, json, fragment)
        })
        provider.pushSnapshot()
      } catch (err) {
        console.error('[yunque] bootstrap Y seed failed', err)
        try {
          const schema = getSchema(contentExtensions)
          provider.doc.transact(() => {
            const fragment = provider.doc.getXmlFragment('default')
            while (fragment.length > 0) fragment.delete(0, fragment.length)
            prosemirrorJSONToYXmlFragment(schema, json, fragment)
          })
          provider.pushSnapshot()
        } catch (err2) {
          console.error('[yunque] bootstrap schema seed failed', err2)
        }
      }

      // Wait for Y→PM binding; if still empty after retries, stop looping but
      // keep materialize gated (guard in materialize() also blocks empty wipe).
      if (editor.state.doc.textContent.trim().length > 0) {
        finishBootstrap(true)
        return true
      }
      if (attempts < 25) return false
      finishBootstrap(false)
      return true
    }

    tryBootstrap()
    const id = window.setInterval(() => {
      if (tryBootstrap()) window.clearInterval(id)
    }, 100)
    return () => window.clearInterval(id)
  }, [
    editor,
    provider,
    initialBlocks,
    onTocChange,
    bootstrappedRef,
    contentExtensions,
  ])

  useEffect(() => {
    revisionRef.current = Number(initialRevision) || 0
  }, [documentId, initialRevision])

  useEffect(() => {
    return () => {
      if (materializeTimerRef.current) window.clearTimeout(materializeTimerRef.current)
    }
  }, [])

  useEffect(() => {
    const onBeforeUnload = (e: BeforeUnloadEvent) => {
      if (!dirtyRef.current) return
      e.preventDefault()
      e.returnValue = ''
    }
    window.addEventListener('beforeunload', onBeforeUnload)
    return () => window.removeEventListener('beforeunload', onBeforeUnload)
  }, [])

  useEffect(() => {
    if (scrollToHeadingIndex == null || !editor) return
    const pos = findHeadingPos(editor, scrollToHeadingIndex)
    if (pos != null) {
      editor.chain().focus().setTextSelection(pos + 1).scrollIntoView().run()
    }
    onScrollToHeadingHandled?.()
  }, [scrollToHeadingIndex, editor, onScrollToHeadingHandled])

  useEffect(() => {
    const w = window as unknown as {
      __yunqueFlushSave?: () => Promise<void>
      __yunqueGetFocusAnchor?: () => EditorFocusAnchor | null
    }
    w.__yunqueFlushSave = async () => {
      const ed = editorRef.current
      if (!ed) return
      if (materializeTimerRef.current) window.clearTimeout(materializeTimerRef.current)
      // Force on exit so reading view updates even if this client was a follower.
      await materialize(ed, { force: true })
    }
    w.__yunqueGetFocusAnchor = () => {
      const ed = editorRef.current
      if (!ed || ed.isDestroyed) return null
      const { $from } = ed.state.selection
      const index = Math.max(0, Math.min($from.index(0), ed.state.doc.childCount - 1))
      if (ed.state.doc.childCount === 0) return null
      const node = ed.state.doc.child(index)
      const blockId = typeof node.attrs?.blockId === 'string' ? node.attrs.blockId : undefined
      return { index, blockId: blockId || undefined }
    }
    return () => {
      delete w.__yunqueFlushSave
      delete w.__yunqueGetFocusAnchor
    }
  }, [documentId, materialize])

  if (!editor) {
    return (
      <div className="space-y-3">
        <div className="sticky top-11 z-30 space-y-2 border-b border-border bg-background/95 pb-2 pt-1 backdrop-blur-md">
          {titleSlot}
        </div>
        <p className="font-mono text-[11px] text-muted" data-save-status="connecting">
          {t('editor.initializing')}
        </p>
      </div>
    )
  }

  return (
    <div className="space-y-3">
      <div className="sticky top-11 z-30 space-y-2 overflow-visible border-b border-border bg-background/95 pb-2 pt-1 backdrop-blur-md">
        {titleSlot}
        <EditorToolbar
          editor={editor}
          documentId={documentId}
          onUploadError={(message) => updateStatus('error', message)}
        />
      </div>
      <div className="rounded-lg border border-border bg-surface px-0 py-2">
        <EditorContent editor={editor} />
      </div>
      <p className="font-mono text-[11px] text-muted" data-save-status={status}>
        {status === 'connecting' && t('editor.statusConnecting')}
        {status === 'synced' && t('editor.statusSynced')}
        {status === 'dirty' && t('editor.statusDirty')}
        {status === 'materializing' && t('editor.statusMaterializing')}
        {status === 'saved' && t('editor.statusSaved')}
        {status === 'offline' && t('editor.statusOffline')}
        {status === 'error' && t('editor.statusError')}
        {status === 'idle' && t('editor.statusIdle')}
      </p>
    </div>
  )
}

export type EditorFocusAnchor = {
  index: number
  blockId?: string
}

export async function flushDocumentEditorSave() {
  const fn = (window as unknown as { __yunqueFlushSave?: () => Promise<void> }).__yunqueFlushSave
  if (fn) await fn()
}

/** Top-level block under the caret; call before unmounting the editor. */
export function getDocumentEditorFocusAnchor(): EditorFocusAnchor | null {
  const fn = (window as unknown as { __yunqueGetFocusAnchor?: () => EditorFocusAnchor | null })
    .__yunqueGetFocusAnchor
  return fn ? fn() : null
}
