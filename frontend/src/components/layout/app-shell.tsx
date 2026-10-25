import { useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Clock3,
  FilePlus2,
  FileText,
  LogOut,
  Settings,
  Star,
  Trash2,
  Upload,
} from 'lucide-react'
import { NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '@/app/auth-context'
import { YunQueBadge } from '@/components/brand/yunque-mark'
import {
  WorkspaceChromeProvider,
  useWorkspaceChrome,
} from '@/components/layout/workspace-chrome-context'
import { api, ApiRequestError } from '@/lib/api'
import { avatarStyle, resolveAvatarColor } from '@/lib/avatar-color'
import { cn } from '@/lib/utils'

export function AppShell() {
  return (
    <WorkspaceChromeProvider>
      <AppShellInner />
    </WorkspaceChromeProvider>
  )
}

function AppShellInner() {
  const { t } = useTranslation()
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const { docCount, docCountLoading } = useWorkspaceChrome()
  const initial = (user?.display_name || user?.username || '?').slice(0, 1)
  const isDocPage = location.pathname.startsWith('/docs/')
  const isTrashPage = location.pathname.startsWith('/trash')
  const isAdminPage = location.pathname.startsWith('/admin')
  const showWorkspaceActions = !isDocPage && !isTrashPage && !isAdminPage
  const [busy, setBusy] = useState(false)
  const [actionError, setActionError] = useState<string | null>(null)
  const importInputRef = useRef<HTMLInputElement | null>(null)

  const navItems = useMemo(() => {
    const items = [
      { to: '/', label: t('shell.allDocs'), icon: FileText, end: true },
      { to: '/recent', label: t('shell.recent'), icon: Clock3 },
      { to: '/favorites', label: t('shell.favorites'), icon: Star },
      { to: '/trash', label: t('shell.trash'), icon: Trash2 },
    ]
    if (user?.is_admin) {
      items.push({ to: '/admin', label: t('shell.admin'), icon: Settings, end: false })
    }
    return items
  }, [t, user?.is_admin])

  function pageLabel(pathname: string) {
    if (pathname.startsWith('/recent')) return t('shell.recent')
    if (pathname.startsWith('/favorites')) return t('shell.favorites')
    if (pathname.startsWith('/trash')) return t('shell.trash')
    if (pathname.startsWith('/admin')) return t('shell.admin')
    if (pathname.startsWith('/docs/')) return t('shell.document')
    return t('shell.allDocs')
  }

  async function createDocument() {
    setBusy(true)
    setActionError(null)
    try {
      const res = await api.createDocument({ title: '未命名文档' })
      navigate(`/docs/${res.document.id}?edit=1`)
    } catch (err) {
      setActionError(
        err instanceof ApiRequestError ? err.message : t('common.createFailed'),
      )
    } finally {
      setBusy(false)
    }
  }

  async function onImportFile(file: File) {
    setBusy(true)
    setActionError(null)
    try {
      const markdown = await file.text()
      if (!markdown.trim()) {
        setActionError(t('shell.emptyFile'))
        return
      }
      const title = file.name.replace(/\.(md|markdown|txt)$/i, '').trim()
      const res = await api.importDocument({
        title: title || undefined,
        markdown,
      })
      navigate(`/docs/${res.document.id}?edit=1`)
    } catch (err) {
      setActionError(
        err instanceof ApiRequestError ? err.message : t('shell.importFailed'),
      )
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="min-h-full bg-background text-foreground antialiased">
      <aside className="fixed inset-y-0 left-0 z-50 flex w-52 select-none flex-col justify-between border-r border-border bg-sidebar">
        <div className="flex min-h-0 flex-1 flex-col">
          <div className="flex h-11 items-center border-b border-border/80 bg-sidebar px-2.5">
            <div className="flex min-w-0 items-center gap-1.5">
              <YunQueBadge className="h-6 w-6" title={t('common.productName')} />
              <div className="min-w-0">
                <div className="truncate text-[13px] font-semibold leading-tight text-zinc-900">
                  {t('common.productName')}
                </div>
                <div className="truncate font-mono text-[10px] leading-tight text-zinc-500">
                  {t('shell.teamSpace')}
                </div>
              </div>
            </div>
          </div>

          <nav className="mt-1.5 flex flex-1 flex-col gap-0.5 overflow-y-auto px-1.5 py-0.5">
            {navItems.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.end}
                className={({ isActive }) =>
                  cn(
                    'flex items-center gap-1.5 rounded-md px-2 py-1 text-[13px] text-zinc-600 transition-colors hover:bg-zinc-200/50 hover:text-zinc-900',
                    isActive && 'bg-zinc-200/70 font-medium text-zinc-900',
                  )
                }
              >
                <item.icon className="h-4 w-4 shrink-0" />
                <span className="flex-1 truncate">{item.label}</span>
              </NavLink>
            ))}
          </nav>
        </div>

        <div className="border-t border-border bg-sidebar px-1.5 py-1">
          <div className="flex items-center justify-between gap-0.5 rounded-md px-1 py-0.5 transition-colors hover:bg-zinc-200/60">
            <div className="flex min-w-0 items-center gap-1.5">
              <div
                className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-[10px] font-bold text-white shadow-sm"
                style={avatarStyle(
                  resolveAvatarColor(user?.id ?? '', user?.avatar_color),
                )}
              >
                {initial}
              </div>
              <div className="min-w-0 leading-tight">
                <div className="truncate text-[12px] font-semibold text-zinc-900">
                  {user?.display_name || user?.username}
                </div>
                <div className="truncate font-mono text-[10px] text-zinc-500">
                  @{user?.username}
                </div>
              </div>
            </div>
            <button
              type="button"
              aria-label={t('shell.logout')}
              className="rounded-md p-0.5 text-zinc-500 transition-colors hover:text-zinc-900"
              onClick={() => void logout()}
            >
              <LogOut className="h-4 w-4" />
            </button>
          </div>
        </div>
      </aside>

      <div className="pl-52">
        {!isDocPage ? (
          <header className="fixed left-52 right-0 top-0 z-40 flex h-11 items-center justify-between gap-3 border-b border-border bg-white/90 px-4 backdrop-blur-md lg:pl-5 lg:pr-4">
            <div className="flex min-w-0 items-center gap-2 font-mono text-[11px] text-zinc-500">
              <span className="truncate">{t('shell.workspace')}</span>
              <span className="text-zinc-300">/</span>
              <span className="inline-flex items-center gap-1.5 truncate rounded-md border border-zinc-200/80 bg-zinc-100 px-2 py-0.5 font-medium text-zinc-900">
                <FileText className="h-3.5 w-3.5" />
                {pageLabel(location.pathname)}
              </span>
              <span className="rounded-full bg-zinc-900 px-2 py-0.5 font-mono text-[11px] font-semibold tracking-wide text-white">
                {docCountLoading || docCount == null
                  ? '…'
                  : t('shell.docCount', { count: docCount })}
              </span>
              {actionError ? (
                <span className="truncate text-red-600" title={actionError}>
                  {actionError}
                </span>
              ) : null}
            </div>
            {showWorkspaceActions ? (
              <div className="ml-auto flex shrink-0 flex-wrap items-center justify-end gap-1.5">
                <input
                  ref={importInputRef}
                  type="file"
                  accept=".md,.markdown,.txt,text/markdown,text/plain"
                  className="hidden"
                  onChange={(e) => {
                    const file = e.target.files?.[0]
                    e.target.value = ''
                    if (file) void onImportFile(file)
                  }}
                />
                <button
                  type="button"
                  disabled={busy}
                  onClick={() => importInputRef.current?.click()}
                  className="group inline-flex items-center gap-1.5 rounded-md border border-zinc-200 bg-white px-2.5 py-1 text-[13px] font-medium text-zinc-800 shadow-sm transition-all hover:border-zinc-300 hover:bg-zinc-50 disabled:opacity-50"
                >
                  <Upload className="h-3.5 w-3.5 text-zinc-500 transition-colors group-hover:text-zinc-900" />
                  {busy ? t('shell.processing') : t('shell.importMarkdown')}
                </button>
                <button
                  type="button"
                  disabled={busy}
                  onClick={() => void createDocument()}
                  className="inline-flex items-center gap-1.5 rounded-md bg-zinc-900 px-2.5 py-1 text-[13px] font-medium text-white shadow-sm transition-all hover:bg-black active:scale-[0.985] disabled:opacity-50"
                >
                  <FilePlus2 className="h-3.5 w-3.5" />
                  {t('shell.newDocument')}
                </button>
              </div>
            ) : null}
          </header>
        ) : null}
        <main className={cn('min-h-screen bg-background', !isDocPage && 'pt-11')}>
          <Outlet />
        </main>
      </div>
    </div>
  )
}
