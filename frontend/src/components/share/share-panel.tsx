import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { UserPlus, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useConfirmDialog } from '@/hooks/use-confirm-dialog'
import {
  api,
  ApiRequestError,
  type DocumentMember,
} from '@/lib/api'
import { avatarStyle, resolveAvatarColor } from '@/lib/avatar-color'

type Props = {
  documentId: string
  open: boolean
  onClose: () => void
  canManage: boolean
}

export function SharePanel({ documentId, open, onClose, canManage }: Props) {
  const { t } = useTranslation()
  const { confirm, dialog: confirmDialog } = useConfirmDialog()
  const [members, setMembers] = useState<DocumentMember[]>([])
  const [username, setUsername] = useState('')
  const [role, setRole] = useState<'editor' | 'viewer'>('editor')
  const [loading, setLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const reload = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await api.listMembers(documentId)
      setMembers(res.members)
    } catch (err) {
      setError(
        err instanceof ApiRequestError ? err.message : t('share.loadMembersFailed'),
      )
    } finally {
      setLoading(false)
    }
  }, [documentId, t])

  useEffect(() => {
    if (open) void reload()
  }, [open, reload])

  async function invite(e: React.FormEvent) {
    e.preventDefault()
    if (!canManage || !username.trim()) return
    setSubmitting(true)
    setError(null)
    try {
      await api.addMember(documentId, { username: username.trim(), role })
      setUsername('')
      await reload()
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : t('share.inviteFailed'))
    } finally {
      setSubmitting(false)
    }
  }

  async function changeRole(userId: string, next: 'editor' | 'viewer') {
    setError(null)
    try {
      await api.updateMember(documentId, userId, { role: next })
      await reload()
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : t('common.updateFailed'))
    }
  }

  async function remove(userId: string) {
    const ok = await confirm({
      title: t('share.removeTitle'),
      description: t('share.confirmRemove'),
      confirmLabel: t('share.remove'),
      variant: 'danger',
    })
    if (!ok) return
    setError(null)
    try {
      await api.removeMember(documentId, userId)
      await reload()
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : t('share.removeFailed'))
    }
  }

  if (!open) return null

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-zinc-900/40 px-4 backdrop-blur-[2px]"
      onClick={onClose}
    >
      {confirmDialog}
      <div
        role="dialog"
        aria-labelledby="share-title"
        className="w-full max-w-md overflow-hidden rounded-lg border border-zinc-200 bg-white shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between border-b border-zinc-100 px-5 py-4">
          <div>
            <h2 id="share-title" className="text-base font-semibold text-zinc-900">
              {t('share.title')}
            </h2>
            <p className="mt-0.5 text-sm text-zinc-500">{t('share.hint')}</p>
          </div>
          <button
            type="button"
            aria-label={t('common.close')}
            className="rounded-lg p-1.5 text-zinc-400 transition-colors hover:bg-zinc-100 hover:text-zinc-900"
            onClick={onClose}
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="px-5 py-4">
          {canManage ? (
            <form onSubmit={(e) => void invite(e)} className="mb-5 space-y-3">
              <div className="space-y-1.5">
                <Label htmlFor="share-username">{t('share.username')}</Label>
                <Input
                  id="share-username"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  placeholder={t('share.usernamePlaceholder')}
                  autoComplete="off"
                  className="rounded-lg border-zinc-200 bg-zinc-50 focus:bg-white"
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="share-role">{t('share.role')}</Label>
                <select
                  id="share-role"
                  value={role}
                  onChange={(e) => setRole(e.target.value as 'editor' | 'viewer')}
                  className="h-10 w-full rounded-lg border border-zinc-200 bg-zinc-50 px-3 text-sm focus:bg-white focus:outline-none"
                >
                  <option value="editor">{t('share.roleEditor')}</option>
                  <option value="viewer">{t('share.roleViewer')}</option>
                </select>
              </div>
              <Button
                type="submit"
                size="sm"
                className="rounded-lg"
                disabled={submitting || !username.trim()}
              >
                <UserPlus className="h-4 w-4" />
                {submitting ? t('share.inviting') : t('share.invite')}
              </Button>
            </form>
          ) : (
            <p className="mb-4 text-sm text-zinc-500">{t('share.ownerOnly')}</p>
          )}

          {error ? (
            <p className="mb-3 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900">
              {error}
            </p>
          ) : null}

          <div className="space-y-2">
            <h3 className="font-mono text-[11px] font-medium uppercase tracking-wider text-zinc-500">
              {t('share.members')}
            </h3>
            {loading ? (
              <p className="text-sm text-zinc-500">{t('common.loading')}</p>
            ) : (
              <ul className="divide-y divide-zinc-100 overflow-hidden rounded-lg border border-zinc-200">
                {members.map((m) => (
                  <li key={m.user_id} className="flex items-center gap-2 px-3 py-2.5 text-sm">
                    <div
                      className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-[10px] font-bold text-white"
                      style={avatarStyle(
                        resolveAvatarColor(m.user_id, m.avatar_color),
                      )}
                    >
                      {(m.display_name || m.username || '?').slice(0, 1)}
                    </div>
                    <div className="min-w-0 flex-1">
                      <div className="truncate font-medium text-zinc-900">
                        {m.display_name || m.username}
                      </div>
                      <div className="font-mono text-[11px] text-zinc-500">@{m.username}</div>
                    </div>
                    {m.role === 'owner' || !canManage ? (
                      <span className="shrink-0 rounded border border-zinc-200 bg-zinc-50 px-2 py-0.5 text-[11px] text-zinc-600">
                        {m.role === 'owner'
                          ? t('share.roleOwner')
                          : m.role === 'editor'
                            ? t('share.roleEditor')
                            : t('share.roleViewer')}
                      </span>
                    ) : (
                      <select
                        value={m.role}
                        onChange={(e) =>
                          void changeRole(m.user_id, e.target.value as 'editor' | 'viewer')
                        }
                        className="h-8 shrink-0 rounded-lg border border-zinc-200 bg-white px-2 text-xs"
                      >
                        <option value="editor">{t('share.roleEditor')}</option>
                        <option value="viewer">{t('share.roleViewer')}</option>
                      </select>
                    )}
                    {canManage && m.role !== 'owner' ? (
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => void remove(m.user_id)}
                      >
                        {t('share.remove')}
                      </Button>
                    ) : null}
                  </li>
                ))}
                {members.length === 0 ? (
                  <li className="px-3 py-4 text-sm text-zinc-500">{t('share.empty')}</li>
                ) : null}
              </ul>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
