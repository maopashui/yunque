import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import {
  FileText,
  Search,
  Share2,
  Star,
  Trash2,
} from 'lucide-react'
import { useWorkspaceChrome } from '@/components/layout/workspace-chrome-context'
import { SharePanel } from '@/components/share/share-panel'
import { api, ApiRequestError, type Document } from '@/lib/api'
import { cn, formatDateTime } from '@/lib/utils'

const PAGE_SIZE = 20

function useListMode(): {
  trash: boolean
  favorite: boolean
  recent: boolean
} {
  const { pathname } = useLocation()
  if (pathname.startsWith('/trash')) {
    return { trash: true, favorite: false, recent: false }
  }
  if (pathname.startsWith('/favorites')) {
    return { trash: false, favorite: true, recent: false }
  }
  if (pathname.startsWith('/recent')) {
    return { trash: false, favorite: false, recent: true }
  }
  return { trash: false, favorite: false, recent: false }
}

export function WorkspacePage() {
  const { t } = useTranslation()
  const mode = useListMode()
  const navigate = useNavigate()
  const { setDocCount } = useWorkspaceChrome()
  const [docs, setDocs] = useState<Document[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [query, setQuery] = useState('')
  const [sort, setSort] = useState<'updated_at' | 'title'>('updated_at')
  const [busy, setBusy] = useState(false)
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null)
  const [shareDocId, setShareDocId] = useState<string | null>(null)
  const deleteConfirmRef = useRef<HTMLDivElement | null>(null)

  function documentStatusLabel(doc: Document, trash: boolean): string {
    if (trash) return t('workspace.statusDeleted')
    if (doc.role === 'viewer') return t('workspace.statusViewer')
    return t('workspace.statusEditor')
  }

  useEffect(() => {
    if (!deleteConfirmId) return
    function onPointerDown(e: PointerEvent) {
      if (deleteConfirmRef.current?.contains(e.target as Node)) return
      setDeleteConfirmId(null)
    }
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') setDeleteConfirmId(null)
    }
    document.addEventListener('pointerdown', onPointerDown)
    window.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('pointerdown', onPointerDown)
      window.removeEventListener('keydown', onKeyDown)
    }
  }, [deleteConfirmId])

  const load = useCallback(async () => {
    setLoading(true)
    setDocCount(null, true)
    setError(null)
    try {
      const res = await api.listDocuments({
        q: query.trim() || undefined,
        sort,
        order: sort === 'title' ? 'asc' : 'desc',
        trash: mode.trash,
        favorite: mode.favorite,
        limit: PAGE_SIZE,
        offset: (page - 1) * PAGE_SIZE,
      })
      const list = res.documents ?? []
      const nextTotal = Number(res.total) || 0
      setDocs(list)
      setTotal(nextTotal)
      setDocCount(nextTotal, false)
      const pageCount = Math.max(1, Math.ceil(nextTotal / PAGE_SIZE))
      if (page > pageCount) setPage(pageCount)
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : t('common.loadFailed'))
      setDocCount(null, false)
    } finally {
      setLoading(false)
    }
  }, [query, sort, mode.trash, mode.favorite, page, setDocCount, t])

  useEffect(() => {
    setPage(1)
  }, [query, sort, mode.trash, mode.favorite, mode.recent])

  useEffect(() => {
    void load()
  }, [load])

  useEffect(() => {
    return () => setDocCount(null, false)
  }, [setDocCount])

  async function createDoc() {
    setBusy(true)
    try {
      const res = await api.createDocument({ title: '未命名文档' })
      navigate(`/docs/${res.document.id}?edit=1`)
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : t('common.createFailed'))
    } finally {
      setBusy(false)
    }
  }

  async function toggleFavorite(doc: Document) {
    try {
      await api.patchDocument(doc.id, { is_favorite: !doc.is_favorite })
      await load()
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : t('common.updateFailed'))
    }
  }

  async function removeDoc(doc: Document) {
    try {
      await api.deleteDocument(doc.id)
      setDeleteConfirmId(null)
      await load()
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : t('common.deleteFailed'))
    }
  }

  async function restoreDoc(doc: Document) {
    try {
      await api.restoreDocument(doc.id)
      await load()
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : t('common.restoreFailed'))
    }
  }

  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const showPager = total > PAGE_SIZE

  return (
    <div className="px-5 py-5 lg:px-6">
      <div className="flex w-full flex-col gap-5">
        <div className="flex flex-col items-stretch justify-between gap-2 rounded-xl border border-zinc-200/80 bg-white p-2 shadow-sm md:flex-row md:items-center">
          <div className="relative min-w-0 flex-1">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-[18px] w-[18px] -translate-y-1/2 text-zinc-400" />
            <input
              type="search"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder={t('workspace.searchPlaceholder')}
              className="w-full rounded-lg border border-zinc-200 bg-zinc-50 py-2 pl-10 pr-3 text-sm text-zinc-900 placeholder:text-zinc-400 transition-all focus:border-zinc-400 focus:bg-white focus:outline-none"
            />
          </div>
          <div className="flex items-center gap-2 px-1">
            <select
              className="h-9 rounded-lg border border-zinc-200 bg-white px-2.5 text-sm text-zinc-700"
              value={sort}
              onChange={(e) => setSort(e.target.value as 'updated_at' | 'title')}
            >
              <option value="updated_at">{t('workspace.sortUpdated')}</option>
              <option value="title">{t('workspace.sortTitle')}</option>
            </select>
          </div>
        </div>

        {error ? (
          <p className="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900">
            {error}
          </p>
        ) : null}

        <section className="flex flex-col gap-3">
          <div className="flex items-center justify-between">
            <h2 className="text-base font-semibold text-zinc-900">
              {t('workspace.listTitle')}
            </h2>
            <div className="flex items-center gap-1.5 font-mono text-[11px] text-zinc-500">
              <span className="h-1.5 w-1.5 rounded-full bg-zinc-900" />
              {t('workspace.sortedByUpdate')}
            </div>
          </div>

          <div className="overflow-visible rounded-xl border border-zinc-200 bg-white shadow-sm">
            <div className="hidden grid-cols-12 items-center gap-2 border-b border-zinc-200 px-4 py-2 font-mono text-[11px] uppercase tracking-wider text-zinc-500 md:grid">
              <div className="col-span-1">{t('workspace.colIndex')}</div>
              <div className="col-span-4 xl:col-span-3">{t('workspace.colName')}</div>
              <div className="col-span-2">{t('workspace.colAuthor')}</div>
              <div className="col-span-2">{t('workspace.colUpdated')}</div>
              <div className="col-span-1 hidden xl:block">{t('workspace.colStatus')}</div>
              <div className="col-span-1">{t('workspace.colBlocks')}</div>
              <div className="col-span-2 text-right">{t('workspace.colActions')}</div>
            </div>

            {loading ? (
              <div className="px-4 py-16 text-center text-sm text-zinc-500">
                {t('common.loading')}
              </div>
            ) : docs.length === 0 ? (
              <div className="flex flex-col items-center justify-center gap-3 px-4 py-16 text-center">
                <div className="flex h-12 w-12 items-center justify-center rounded-xl border border-zinc-200 bg-zinc-50 text-zinc-400">
                  <FileText className="h-6 w-6" />
                </div>
                <p className="text-sm text-zinc-500">{t('workspace.empty')}</p>
                {!mode.trash ? (
                  <button
                    type="button"
                    disabled={busy}
                    onClick={() => void createDoc()}
                    className="rounded-lg bg-zinc-900 px-3 py-1.5 text-sm font-medium text-white hover:bg-black"
                  >
                    {t('workspace.createFirst')}
                  </button>
                ) : null}
              </div>
            ) : (
              docs.map((doc, index) => (
                <div
                  key={doc.id}
                  className="group grid grid-cols-1 items-center gap-1.5 border-b border-zinc-100 px-4 py-1.5 last:border-b-0 hover:bg-zinc-50 md:grid-cols-12 md:gap-2"
                >
                  <div className="col-span-1 hidden font-mono text-[12px] leading-5 text-zinc-400 md:block">
                    {(page - 1) * PAGE_SIZE + index + 1}
                  </div>
                  <div className="col-span-4 min-w-0 xl:col-span-3">
                    {mode.trash ? (
                      <span className="block truncate text-sm font-medium leading-5 text-zinc-900">
                        {doc.title}
                      </span>
                    ) : (
                      <Link
                        to={`/docs/${doc.id}`}
                        className="block truncate text-sm font-medium leading-5 text-zinc-900 hover:underline"
                      >
                        {doc.title}
                      </Link>
                    )}
                  </div>
                  <div className="col-span-2 hidden min-w-0 font-mono text-[12px] leading-5 text-zinc-500 md:block">
                    <span className="truncate" title={doc.owner_name || undefined}>
                      {doc.owner_name || '—'}
                    </span>
                  </div>
                  <div className="col-span-2 font-mono text-[12px] leading-5 text-zinc-500">
                    {formatDateTime(doc.updated_at)}
                  </div>
                  <div className="col-span-1 hidden min-w-0 items-center xl:flex">
                    <span className="truncate font-mono text-[11px] text-zinc-600">
                      {documentStatusLabel(doc, mode.trash)}
                    </span>
                  </div>
                  <div className="col-span-1 hidden font-mono text-[12px] leading-5 text-zinc-500 md:block">
                    {doc.block_count}
                  </div>
                  <div className="relative col-span-2 flex items-center justify-end gap-0.5">
                    {mode.trash ? (
                      <button
                        type="button"
                        onClick={() => void restoreDoc(doc)}
                        className="rounded-md border border-zinc-200 bg-white px-2 py-0.5 font-mono text-[11px] font-medium text-zinc-800 transition-colors hover:bg-zinc-100"
                      >
                        {t('common.restore')}
                      </button>
                    ) : (
                      <>
                        <button
                          type="button"
                          aria-label={t('workspace.favorite')}
                          className="rounded-md p-1 text-zinc-400 transition-colors hover:bg-zinc-100 hover:text-zinc-900"
                          onClick={() => void toggleFavorite(doc)}
                        >
                          <Star
                            className={cn(
                              'h-3.5 w-3.5',
                              doc.is_favorite && 'fill-amber-400 text-amber-400',
                            )}
                          />
                        </button>
                        {doc.role === 'owner' || !doc.role ? (
                          <button
                            type="button"
                            aria-label={t('workspace.share')}
                            className="rounded-md p-1 text-zinc-400 transition-colors hover:bg-zinc-100 hover:text-zinc-900"
                            onClick={() => setShareDocId(doc.id)}
                          >
                            <Share2 className="h-3.5 w-3.5" />
                          </button>
                        ) : null}
                        <div
                          className="relative"
                          ref={deleteConfirmId === doc.id ? deleteConfirmRef : undefined}
                        >
                          <button
                            type="button"
                            aria-label={t('common.delete')}
                            aria-expanded={deleteConfirmId === doc.id}
                            aria-haspopup="dialog"
                            onClick={() =>
                              setDeleteConfirmId((cur) => (cur === doc.id ? null : doc.id))
                            }
                            className="rounded-md p-1 text-red-500 transition-colors hover:bg-red-50 hover:text-red-600"
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </button>
                          {deleteConfirmId === doc.id ? (
                            <div
                              role="dialog"
                              aria-label={t('workspace.confirmDelete')}
                              className="absolute bottom-full right-0 z-20 mb-1.5 w-52 rounded-lg border border-zinc-200 bg-white p-3 shadow-lg"
                            >
                              <div className="absolute -bottom-1.5 right-3 h-3 w-3 rotate-45 border-b border-r border-zinc-200 bg-white" />
                              <p className="relative text-xs leading-relaxed text-zinc-700">
                                {t('workspace.confirmDeleteTitle', { title: doc.title })}
                                <span className="mt-0.5 block text-zinc-500">
                                  {t('workspace.confirmDeleteHint')}
                                </span>
                              </p>
                              <div className="relative mt-2.5 flex justify-end gap-1.5">
                                <button
                                  type="button"
                                  className="rounded-md px-2 py-1 text-[11px] font-medium text-zinc-600 hover:bg-zinc-100"
                                  onClick={() => setDeleteConfirmId(null)}
                                >
                                  {t('common.cancel')}
                                </button>
                                <button
                                  type="button"
                                  className="rounded-md bg-red-600 px-2 py-1 text-[11px] font-medium text-white hover:bg-red-700"
                                  onClick={() => void removeDoc(doc)}
                                >
                                  {t('common.delete')}
                                </button>
                              </div>
                            </div>
                          ) : null}
                        </div>
                      </>
                    )}
                  </div>
                </div>
              ))
            )}
          </div>

          {showPager ? (
            <div className="flex flex-wrap items-center justify-between gap-2 pt-1">
              <p className="font-mono text-[11px] text-zinc-500">
                {t('workspace.pageRange', {
                  from: (page - 1) * PAGE_SIZE + 1,
                  to: Math.min(page * PAGE_SIZE, total),
                  total,
                })}
              </p>
              <div className="flex items-center gap-1.5">
                <button
                  type="button"
                  disabled={page <= 1 || loading}
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  className="rounded-md border border-zinc-200 bg-white px-2.5 py-1 font-mono text-[11px] font-medium text-zinc-700 transition-colors hover:bg-zinc-50 disabled:opacity-40"
                >
                  {t('workspace.prevPage')}
                </button>
                <span className="min-w-[4.5rem] text-center font-mono text-[11px] text-zinc-600">
                  {t('workspace.pageOf', { page, pages: pageCount })}
                </span>
                <button
                  type="button"
                  disabled={page >= pageCount || loading}
                  onClick={() => setPage((p) => Math.min(pageCount, p + 1))}
                  className="rounded-md border border-zinc-200 bg-white px-2.5 py-1 font-mono text-[11px] font-medium text-zinc-700 transition-colors hover:bg-zinc-50 disabled:opacity-40"
                >
                  {t('workspace.nextPage')}
                </button>
              </div>
            </div>
          ) : null}
        </section>
      </div>

      {shareDocId ? (
        <SharePanel
          documentId={shareDocId}
          open
          onClose={() => setShareDocId(null)}
          canManage
        />
      ) : null}

    </div>
  )
}

export function PlaceholderPage({ title }: { title: string }) {
  const { t } = useTranslation()
  return (
    <div className="px-6 py-8 lg:px-10">
      <div className="mx-auto max-w-3xl">
        <h1 className="text-2xl font-bold tracking-tight text-zinc-900">{title}</h1>
        <p className="mt-2 text-sm text-zinc-500">{t('workspace.placeholderHint')}</p>
      </div>
    </div>
  )
}
