import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom'
import {
  ArrowLeft,
  Download,
  History,
  List,
  Pencil,
  Share2,
  Star,
} from 'lucide-react'
import {
  DocumentEditor,
  flushDocumentEditorSave,
  getDocumentEditorFocusAnchor,
  type SaveStatus,
} from '@/components/editor/document-editor'
import { PresenceBar } from '@/components/editor/presence-bar'
import { HistoryLayout } from '@/components/history/history-layout'
import { SharePanel } from '@/components/share/share-panel'
import { DocumentTOC } from '@/components/reader/document-toc'
import { VirtualReader } from '@/components/reader/virtual-reader'
import { useAuth } from '@/app/auth-context'
import { useConfirmDialog } from '@/hooks/use-confirm-dialog'
import {
  api,
  ApiRequestError,
  type Block,
  type BlockMeta,
  type Document,
  type TOCItem,
} from '@/lib/api'
import type { PresencePeer } from '@/lib/collab/provider'
import { cn, formatDateTime } from '@/lib/utils'

export function DocumentReaderPage() {
  const { t } = useTranslation()
  const { id = '' } = useParams()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const { user } = useAuth()
  const { confirm, dialog: confirmDialog } = useConfirmDialog()
  const [doc, setDoc] = useState<Document | null>(null)
  const [index, setIndex] = useState<BlockMeta[]>([])
  const [toc, setToc] = useState<TOCItem[]>([])
  const [editToc, setEditToc] = useState<TOCItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [scrollToIndex, setScrollToIndex] = useState<number | null>(null)
  const [editScrollHeading, setEditScrollHeading] = useState<number | null>(null)
  const [activeHeadingId, setActiveHeadingId] = useState<string | null>(null)
  const [mobileTocOpen, setMobileTocOpen] = useState(false)

  const [editing, setEditing] = useState(false)
  const [editBlocks, setEditBlocks] = useState<Block[] | null>(null)
  const [editRevision, setEditRevision] = useState(0)
  const [editLoading, setEditLoading] = useState(false)
  const [dirty, setDirty] = useState(false)
  const [saveStatus, setSaveStatus] = useState<SaveStatus>('idle')
  const [peers, setPeers] = useState<PresencePeer[]>([])
  const [shareOpen, setShareOpen] = useState(false)
  const [historyMode, setHistoryMode] = useState(false)
  const [editTitle, setEditTitle] = useState('')
  const titleTimerRef = useRef<number | null>(null)
  const autoEditConsumedRef = useRef(false)

  function modeLabel(historyModeOn: boolean, editingOn: boolean) {
    if (historyModeOn) return t('reader.modeHistory')
    if (editingOn) return t('reader.modeEditing')
    return t('reader.modeReadonly')
  }

  function saveStatusLabel(status: SaveStatus) {
    switch (status) {
      case 'connecting':
        return t('reader.statusConnecting')
      case 'synced':
      case 'saved':
        return t('reader.statusSynced')
      case 'dirty':
        return t('reader.statusDirty')
      case 'materializing':
        return t('reader.statusMaterializing')
      case 'offline':
        return t('reader.statusOffline')
      case 'error':
        return t('reader.statusError')
      case 'idle':
        return t('reader.statusIdle')
      default:
        return ''
    }
  }

  const reloadIndex = useCallback(async () => {
    const [indexRes, tocRes] = await Promise.all([
      api.listBlockIndex(id),
      api.listTOC(id),
    ])
    setDoc(indexRes.document)
    setIndex(indexRes.blocks)
    setToc(tocRes.toc)
  }, [id])

  useEffect(() => {
    let cancelled = false
    async function load() {
      setLoading(true)
      setError(null)
      try {
        await reloadIndex()
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof ApiRequestError ? err.message : t('common.loadFailed'))
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    if (id) void load()
    return () => {
      cancelled = true
    }
  }, [id, reloadIndex, t])

  async function toggleFavorite() {
    if (!doc) return
    try {
      const res = await api.patchDocument(doc.id, { is_favorite: !doc.is_favorite })
      setDoc(res.document)
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : t('common.updateFailed'))
    }
  }

  async function exportMd() {
    if (!doc) return
    try {
      const md = await api.exportDocument(doc.id)
      const blob = new Blob([md], { type: 'text/markdown;charset=utf-8' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `${doc.title || 'document'}.md`
      a.click()
      URL.revokeObjectURL(url)
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : t('reader.exportFailed'))
    }
  }

  const enterEdit = useCallback(async () => {
    if (!doc) return
    if (historyMode) setHistoryMode(false)
    if (index.length > 8000) {
      const ok = await confirm({
        title: t('reader.largeDocTitle'),
        description: t('reader.largeDocConfirm', { count: index.length }),
        confirmLabel: t('common.continue'),
      })
      if (!ok) return
    }
    setEditLoading(true)
    setError(null)
    try {
      const [docRes, blocksRes] = await Promise.all([
        api.getDocument(doc.id),
        api.listBlocks(doc.id),
      ])
      setDoc(docRes.document)
      setEditTitle(docRes.document.title)
      setEditRevision(docRes.document.revision)
      setEditBlocks(blocksRes.blocks)
      setEditToc([])
      setPeers([])
      setEditing(true)
      setDirty(false)
      setSaveStatus('connecting')
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : t('reader.enterEditFailed'))
    } finally {
      setEditLoading(false)
    }
  }, [doc, historyMode, index.length, t, confirm])

  useEffect(() => {
    autoEditConsumedRef.current = false
  }, [id])

  useEffect(() => {
    if (searchParams.get('edit') !== '1') return
    if (autoEditConsumedRef.current || loading || !doc || editing || editLoading) return
    const role = doc.role
    const allowed = role === 'owner' || role === 'editor' || !role
    if (!allowed) return
    autoEditConsumedRef.current = true
    const next = new URLSearchParams(searchParams)
    next.delete('edit')
    setSearchParams(next, { replace: true })
    void enterEdit()
  }, [searchParams, setSearchParams, loading, doc, editing, editLoading, enterEdit])

  async function persistTitle(nextTitle: string) {
    if (!doc) return
    const title = nextTitle.trim() || '未命名文档'
    if (title === doc.title) {
      setEditTitle(title)
      return
    }
    try {
      const res = await api.patchDocument(doc.id, { title })
      setDoc(res.document)
      setEditTitle(res.document.title)
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : t('reader.titleSaveFailed'))
      setEditTitle(doc.title)
    }
  }

  function onTitleChange(value: string) {
    setEditTitle(value)
    if (titleTimerRef.current) window.clearTimeout(titleTimerRef.current)
    titleTimerRef.current = window.setTimeout(() => {
      void persistTitle(value)
    }, 800)
  }

  useEffect(() => {
    return () => {
      if (titleTimerRef.current) window.clearTimeout(titleTimerRef.current)
    }
  }, [])

  useEffect(() => {
    if (!editing) return
    const onKeyDown = (e: KeyboardEvent) => {
      if (!(e.metaKey || e.ctrlKey) || e.key.toLowerCase() !== 's') return
      e.preventDefault()
      void (async () => {
        if (titleTimerRef.current) {
          window.clearTimeout(titleTimerRef.current)
          titleTimerRef.current = null
        }
        await persistTitle(editTitle)
        await flushDocumentEditorSave()
      })()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [editing, editTitle, doc])

  async function exitEdit() {
    if (titleTimerRef.current) {
      window.clearTimeout(titleTimerRef.current)
      titleTimerRef.current = null
      await persistTitle(editTitle)
    }
    if (dirty) {
      const choice = await confirm({
        title: t('reader.unsavedTitle'),
        description: t('reader.unsavedLeaveEdit'),
        confirmLabel: t('reader.done'),
      })
      if (!choice) return
    }
    // Capture caret block before unmount; prefer blockId after materialize.
    const focusAnchor = getDocumentEditorFocusAnchor()
    // Always flush so SQL←editor stays aligned after CRDT bootstrap/seed
    // (dirty may be false if only Yjs was seeded from import).
    try {
      await flushDocumentEditorSave()
    } catch {
      if (dirty) {
        const force = await confirm({
          title: t('reader.saveFailedTitle'),
          description: t('reader.saveFailedLeaveEdit'),
          confirmLabel: t('common.leave'),
          variant: 'danger',
        })
        if (!force) return
      }
    }
    const anchorAfterFlush = getDocumentEditorFocusAnchor() ?? focusAnchor
    setEditing(false)
    setEditBlocks(null)
    setEditToc([])
    setPeers([])
    setDirty(false)
    try {
      const [indexRes, tocRes] = await Promise.all([
        api.listBlockIndex(id),
        api.listTOC(id),
      ])
      setDoc(indexRes.document)
      setIndex(indexRes.blocks)
      setToc(tocRes.toc)
      let target: number | null = null
      if (anchorAfterFlush?.blockId) {
        const byId = indexRes.blocks.findIndex((b) => b.id === anchorAfterFlush.blockId)
        if (byId >= 0) target = byId
      }
      if (target == null && anchorAfterFlush != null) {
        const i = anchorAfterFlush.index
        if (i >= 0 && i < indexRes.blocks.length) target = i
      }
      if (target != null) setScrollToIndex(target)
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : t('reader.refreshFailed'))
    }
  }

  async function enterHistory() {
    if (editing) {
      if (dirty) {
        const ok = await confirm({
          title: t('reader.unsavedTitle'),
          description: t('reader.unsavedEnterHistory'),
          confirmLabel: t('common.continue'),
        })
        if (!ok) return
        try {
          await flushDocumentEditorSave()
        } catch {
          const force = await confirm({
            title: t('reader.saveFailedTitle'),
            description: t('reader.saveFailedEnterHistory'),
            confirmLabel: t('common.continue'),
            variant: 'danger',
          })
          if (!force) return
        }
      }
      setEditing(false)
      setEditBlocks(null)
      setEditToc([])
      setPeers([])
      setDirty(false)
    }
    setHistoryMode(true)
  }

  async function backToList() {
    if (editing && dirty) {
      const ok = await confirm({
        title: t('reader.unsavedTitle'),
        description: t('reader.unsavedBackList'),
        confirmLabel: t('common.leave'),
        variant: 'danger',
      })
      if (!ok) return
    }
    navigate('/')
  }

  const onNavigateRead = useCallback((item: TOCItem) => {
    setScrollToIndex(item.index)
    setActiveHeadingId(item.id)
    setMobileTocOpen(false)
  }, [])

  const onNavigateEdit = useCallback((item: TOCItem) => {
    setEditScrollHeading(item.index)
    setActiveHeadingId(item.id)
    setMobileTocOpen(false)
  }, [])

  const canEdit = doc?.role === 'owner' || doc?.role === 'editor' || !doc?.role
  const canManage = doc?.role === 'owner'
  const outline = editing ? editToc : toc

  return (
    <div
      className={cn(
        'bg-background',
        historyMode ? 'flex h-screen flex-col overflow-hidden' : 'min-h-screen',
      )}
    >
      {confirmDialog}
      <header className="sticky top-0 z-40 flex h-11 shrink-0 items-center justify-between border-b border-border bg-white/90 px-4 backdrop-blur-md">
        <div className="flex min-w-0 items-center gap-2">
          {historyMode ? (
            <button
              type="button"
              className="shrink-0 rounded-lg p-1.5 text-zinc-500 transition-colors hover:bg-zinc-100 hover:text-zinc-900"
              aria-label={t('reader.backToRead')}
              onClick={() => setHistoryMode(false)}
            >
              <ArrowLeft className="h-4 w-4" />
            </button>
          ) : editing && dirty ? (
            <button
              type="button"
              className="shrink-0 rounded-lg p-1.5 text-zinc-500 transition-colors hover:bg-zinc-100 hover:text-zinc-900"
              aria-label={t('reader.backToList')}
              onClick={() => void backToList()}
            >
              <ArrowLeft className="h-4 w-4" />
            </button>
          ) : (
            <Link
              to="/"
              className="shrink-0 rounded-lg p-1.5 text-zinc-500 transition-colors hover:bg-zinc-100 hover:text-zinc-900"
              aria-label={t('reader.backToList')}
            >
              <ArrowLeft className="h-4 w-4" />
            </Link>
          )}
          <div className="flex min-w-0 items-center gap-1.5 rounded-md border border-zinc-200/80 bg-zinc-100 px-2.5 py-1 font-mono text-[11px] text-zinc-700">
            <span className="truncate font-medium text-zinc-900 max-w-[14rem] sm:max-w-xs">
              {doc?.title || t('common.loading')}
            </span>
            <span className="text-zinc-300">|</span>
            <span className="shrink-0 text-zinc-500">
              {modeLabel(historyMode, editing)}
            </span>
            {doc?.updated_at ? (
              <>
                <span className="text-zinc-300">·</span>
                <span className="shrink-0 text-zinc-500">
                  {formatDateTime(doc.updated_at)}
                </span>
              </>
            ) : null}
            {editing ? (
              <>
                <span className="text-zinc-300">·</span>
                <span className="shrink-0 text-zinc-500">
                  {saveStatusLabel(saveStatus)}
                </span>
              </>
            ) : null}
          </div>
        </div>

        <div className="flex items-center gap-3">
          {editing ? <PresenceBar peers={peers} selfId={user?.id} /> : null}

          <div className="flex items-center gap-0.5">
            {!historyMode ? (
              <button
                type="button"
                className="rounded-lg p-1.5 text-zinc-500 transition-colors hover:bg-zinc-100 hover:text-zinc-900 xl:hidden"
                aria-label={t('reader.toc')}
                onClick={() => setMobileTocOpen((v) => !v)}
              >
                <List className="h-[19px] w-[19px]" />
              </button>
            ) : null}
            {!historyMode ? (
              <>
                {!editing ? (
                  <button
                    type="button"
                    className="rounded-lg p-1.5 text-zinc-500 transition-colors hover:bg-zinc-100 hover:text-zinc-900"
                    aria-label={t('reader.export')}
                    onClick={() => void exportMd()}
                  >
                    <Download className="h-[19px] w-[19px]" />
                  </button>
                ) : null}
                <button
                  type="button"
                  className="rounded-lg p-1.5 text-zinc-500 transition-colors hover:bg-zinc-100 hover:text-zinc-900"
                  aria-label={t('reader.favorite')}
                  onClick={() => void toggleFavorite()}
                >
                  <Star
                    className={cn(
                      'h-[19px] w-[19px]',
                      doc?.is_favorite && 'fill-amber-400 text-amber-400',
                    )}
                  />
                </button>
                {canManage ? (
                  <button
                    type="button"
                    className="rounded-lg p-1.5 text-zinc-500 transition-colors hover:bg-zinc-100 hover:text-zinc-900"
                    aria-label={t('reader.share')}
                    onClick={() => setShareOpen(true)}
                  >
                    <Share2 className="h-[19px] w-[19px]" />
                  </button>
                ) : null}
              </>
            ) : null}
            <button
              type="button"
              className={cn(
                'rounded-lg p-1.5 transition-colors',
                historyMode
                  ? 'bg-zinc-900 text-white hover:bg-black'
                  : 'text-zinc-500 hover:bg-zinc-100 hover:text-zinc-900',
              )}
              aria-label={historyMode ? t('reader.exitHistory') : t('reader.history')}
              onClick={() => {
                if (historyMode) setHistoryMode(false)
                else void enterHistory()
              }}
            >
              <History className="h-[19px] w-[19px]" />
            </button>
            {canEdit && !editing && !historyMode ? (
              <button
                type="button"
                disabled={editLoading}
                className="ml-1 inline-flex h-[27px] items-center gap-1.5 rounded-md bg-zinc-900 px-2.5 py-1 font-mono text-[11px] font-medium text-white transition-colors hover:bg-black disabled:opacity-50"
                onClick={() => void enterEdit()}
              >
                <Pencil className="h-3 w-3" />
                {editLoading ? t('common.loading') : t('reader.edit')}
              </button>
            ) : null}
            {editing ? (
              <button
                type="button"
                className="ml-1 inline-flex h-[27px] items-center gap-1.5 rounded-md border border-zinc-200/80 bg-zinc-100 px-2.5 py-1 font-mono text-[11px] font-medium text-zinc-800 transition-colors hover:bg-zinc-200"
                onClick={() => void exitEdit()}
              >
                {t('reader.done')}
              </button>
            ) : null}
          </div>
        </div>
      </header>

      {historyMode && doc ? (
        <HistoryLayout
          documentId={doc.id}
          documentTitle={doc.title}
          currentRevision={doc.revision}
          canEdit={!!canEdit}
          onRestored={() => {
            void (async () => {
              setHistoryMode(false)
              setEditing(false)
              setEditBlocks(null)
              try {
                await reloadIndex()
              } catch (err) {
                setError(
                  err instanceof ApiRequestError ? err.message : t('reader.refreshFailed'),
                )
              }
            })()
          }}
        />
      ) : (
        <div className="px-5 py-5 lg:px-6">
          <div className="w-full">
            {loading ? (
              <p className="text-sm text-zinc-500">{t('reader.loadingIndex')}</p>
            ) : error ? (
              <p className="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900">
                {error}
              </p>
            ) : (
              <>
                {!editing ? (
                  <h1 className="text-3xl font-bold tracking-tight text-zinc-900">
                    {doc?.title}
                  </h1>
                ) : null}

                {mobileTocOpen ? (
                  <div className="mt-4 max-h-64 overflow-y-auto rounded-lg border border-zinc-200 bg-white p-3 xl:hidden">
                    <DocumentTOC
                      items={outline}
                      activeId={activeHeadingId}
                      onNavigate={editing ? onNavigateEdit : onNavigateRead}
                    />
                  </div>
                ) : null}

                {editing && editBlocks ? (
                  <div className="mt-1 flex gap-4">
                    <div className="min-w-0 flex-1">
                      <DocumentEditor
                        documentId={id}
                        initialBlocks={editBlocks}
                        initialRevision={editRevision}
                        titleSlot={
                          <input
                            type="text"
                            value={editTitle}
                            onChange={(e) => onTitleChange(e.target.value)}
                            onBlur={() => {
                              if (titleTimerRef.current) {
                                window.clearTimeout(titleTimerRef.current)
                                titleTimerRef.current = null
                              }
                              void persistTitle(editTitle)
                            }}
                            onKeyDown={(e) => {
                              if (e.key === 'Enter') {
                                e.preventDefault()
                                ;(e.target as HTMLInputElement).blur()
                              }
                            }}
                            maxLength={200}
                            aria-label={t('reader.docTitle')}
                            placeholder={t('reader.titlePlaceholder')}
                            className="w-full border-0 border-b border-transparent bg-transparent text-3xl font-bold tracking-tight text-zinc-900 outline-none placeholder:text-zinc-400 focus:border-zinc-200"
                          />
                        }
                        onDocumentChange={setDoc}
                        onDirtyChange={setDirty}
                        onStatusChange={setSaveStatus}
                        onTocChange={setEditToc}
                        onPeersChange={setPeers}
                        scrollToHeadingIndex={editScrollHeading}
                        onScrollToHeadingHandled={() => setEditScrollHeading(null)}
                      />
                    </div>
                    <aside className="sticky top-[9.5rem] hidden max-h-[calc(100vh-10.5rem)] w-60 shrink-0 overflow-y-auto xl:block">
                      <DocumentTOC
                        items={editToc}
                        activeId={activeHeadingId}
                        onNavigate={onNavigateEdit}
                      />
                    </aside>
                  </div>
                ) : (
                  <div className="mt-5 flex gap-4">
                    <div className="min-w-0 flex-1">
                      <VirtualReader
                        documentId={id}
                        index={index}
                        toc={toc}
                        scrollToIndex={scrollToIndex}
                        onScrollToIndexHandled={() => setScrollToIndex(null)}
                        onActiveHeadingChange={(item) =>
                          setActiveHeadingId(item?.id ?? null)
                        }
                      />
                    </div>
                    <aside className="sticky top-20 hidden max-h-[calc(100vh-6rem)] w-60 shrink-0 overflow-y-auto xl:block">
                      <DocumentTOC
                        items={toc}
                        activeId={activeHeadingId}
                        onNavigate={onNavigateRead}
                      />
                    </aside>
                  </div>
                )}
              </>
            )}
          </div>
        </div>
      )}

      {doc ? (
        <SharePanel
          documentId={doc.id}
          open={shareOpen}
          onClose={() => setShareOpen(false)}
          canManage={canManage}
        />
      ) : null}
    </div>
  )
}
