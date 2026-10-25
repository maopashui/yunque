import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Navigate } from 'react-router-dom'
import { useAuth } from '@/app/auth-context'
import { useConfirmDialog } from '@/hooks/use-confirm-dialog'
import { api, ApiRequestError, type AdminUser } from '@/lib/api'
import { cn } from '@/lib/utils'

export function AdminPage() {
  const { t, i18n } = useTranslation()
  const { user, loading } = useAuth()
  const { confirm, dialog: confirmDialog } = useConfirmDialog()
  const [registrationEnabled, setRegistrationEnabled] = useState(true)
  const [users, setUsers] = useState<AdminUser[]>([])
  const [busy, setBusy] = useState(false)
  const [rowBusy, setRowBusy] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loaded, setLoaded] = useState(false)

  const load = useCallback(async () => {
    setError(null)
    try {
      const [settingsRes, usersRes] = await Promise.all([
        api.getAdminSettings(),
        api.listAdminUsers(),
      ])
      setRegistrationEnabled(settingsRes.settings.registration_enabled)
      setUsers(usersRes.users)
      setLoaded(true)
    } catch (err) {
      setError(
        err instanceof ApiRequestError ? err.message : t('common.loadFailed'),
      )
    }
  }, [t])

  useEffect(() => {
    if (!loading && user?.is_admin) {
      void load()
    }
  }, [loading, user?.is_admin, load])

  if (!loading && (!user || !user.is_admin)) {
    return <Navigate to="/" replace />
  }

  async function toggleRegistration(next: boolean) {
    setBusy(true)
    setError(null)
    try {
      const res = await api.patchAdminSettings({ registration_enabled: next })
      setRegistrationEnabled(res.settings.registration_enabled)
    } catch (err) {
      setError(
        err instanceof ApiRequestError ? err.message : t('common.updateFailed'),
      )
    } finally {
      setBusy(false)
    }
  }

  async function setDisabled(target: AdminUser, disabled: boolean) {
    const name = target.display_name || target.username
    const ok = await confirm({
      title: disabled ? t('admin.disableTitle') : t('admin.enableTitle'),
      description: disabled
        ? t('admin.confirmDisable', { name })
        : t('admin.confirmEnable', { name }),
      confirmLabel: disabled ? t('admin.disable') : t('admin.enable'),
      variant: disabled ? 'danger' : 'default',
    })
    if (!ok) return
    setRowBusy(target.id)
    setError(null)
    try {
      const res = await api.patchAdminUser(target.id, { disabled })
      setUsers((prev) =>
        prev.map((u) => (u.id === target.id ? res.user : u)),
      )
    } catch (err) {
      setError(
        err instanceof ApiRequestError ? err.message : t('admin.actionFailed'),
      )
    } finally {
      setRowBusy(null)
    }
  }

  async function removeUser(target: AdminUser) {
    const name = target.display_name || target.username
    const ok = await confirm({
      title: t('admin.deleteTitle'),
      description: t('admin.confirmDelete', { name }),
      confirmLabel: t('admin.delete'),
      variant: 'danger',
    })
    if (!ok) return
    setRowBusy(target.id)
    setError(null)
    try {
      await api.deleteAdminUser(target.id)
      setUsers((prev) => prev.filter((u) => u.id !== target.id))
    } catch (err) {
      setError(
        err instanceof ApiRequestError ? err.message : t('admin.actionFailed'),
      )
    } finally {
      setRowBusy(null)
    }
  }

  const dateLocale = i18n.language?.startsWith('zh') ? 'zh-CN' : 'en-US'

  return (
    <div className="mx-auto max-w-4xl px-4 py-8 lg:px-6">
      {confirmDialog}
      <h1 className="text-xl font-semibold tracking-tight text-zinc-900">
        {t('admin.title')}
      </h1>
      <p className="mt-1 text-sm text-zinc-500">{t('admin.subtitle')}</p>

      {error ? (
        <p className="mt-4 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900">
          {error}
        </p>
      ) : null}

      <section className="mt-8 rounded-xl border border-zinc-200 bg-white p-5 shadow-sm">
        <h2 className="text-sm font-semibold text-zinc-900">
          {t('admin.registration')}
        </h2>
        <p className="mt-1 text-sm text-zinc-500">{t('admin.registrationHint')}</p>
        <label className="mt-4 flex cursor-pointer items-center justify-between gap-4">
          <span className="text-sm text-zinc-800">
            {t('admin.allowRegistration')}
          </span>
          <button
            type="button"
            role="switch"
            aria-checked={registrationEnabled}
            disabled={busy || !loaded}
            onClick={() => void toggleRegistration(!registrationEnabled)}
            className={cn(
              'relative h-6 w-11 shrink-0 rounded-full transition-colors disabled:opacity-50',
              registrationEnabled ? 'bg-zinc-900' : 'bg-zinc-300',
            )}
          >
            <span
              className={cn(
                'absolute top-0.5 left-0.5 h-5 w-5 rounded-full bg-white shadow transition-transform',
                registrationEnabled && 'translate-x-5',
              )}
            />
          </button>
        </label>
        <p className="mt-2 font-mono text-[11px] text-zinc-500">
          {registrationEnabled
            ? t('admin.registrationOn')
            : t('admin.registrationOff')}
        </p>
      </section>

      <section className="mt-6 rounded-xl border border-zinc-200 bg-white shadow-sm">
        <div className="border-b border-zinc-100 px-5 py-3">
          <h2 className="text-sm font-semibold text-zinc-900">
            {t('admin.users')}
          </h2>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full min-w-[640px] text-left text-sm">
            <thead className="border-b border-zinc-100 bg-zinc-50/80 text-[11px] font-medium uppercase tracking-wide text-zinc-500">
              <tr>
                <th className="px-5 py-2.5">{t('admin.colUser')}</th>
                <th className="px-5 py-2.5">{t('admin.colRole')}</th>
                <th className="px-5 py-2.5">{t('admin.colStatus')}</th>
                <th className="px-5 py-2.5">{t('admin.colCreated')}</th>
                <th className="px-5 py-2.5 text-right">{t('admin.colActions')}</th>
              </tr>
            </thead>
            <tbody>
              {users.map((u) => {
                const isSelf = u.id === user?.id
                const disabled = Boolean(u.disabled_at)
                const pending = rowBusy === u.id
                return (
                  <tr
                    key={u.id}
                    className={cn(
                      'border-b border-zinc-50 last:border-0',
                      disabled && 'bg-zinc-50/80',
                    )}
                  >
                    <td className="px-5 py-3">
                      <div className="font-medium text-zinc-900">
                        {u.display_name || u.username}
                        {isSelf ? (
                          <span className="ml-2 rounded bg-zinc-100 px-1.5 py-0.5 text-[10px] font-medium text-zinc-500">
                            {t('admin.you')}
                          </span>
                        ) : null}
                      </div>
                      <div className="font-mono text-[11px] text-zinc-500">
                        @{u.username}
                      </div>
                    </td>
                    <td className="px-5 py-3 text-zinc-700">
                      {u.is_admin ? t('admin.roleAdmin') : t('admin.roleUser')}
                    </td>
                    <td className="px-5 py-3">
                      <span
                        className={cn(
                          'inline-flex rounded-md px-2 py-0.5 text-[11px] font-medium',
                          disabled
                            ? 'bg-amber-50 text-amber-800'
                            : 'bg-emerald-50 text-emerald-800',
                        )}
                      >
                        {disabled
                          ? t('admin.statusDisabled')
                          : t('admin.statusActive')}
                      </span>
                    </td>
                    <td className="px-5 py-3 font-mono text-[12px] text-zinc-500">
                      {new Date(u.created_at).toLocaleString(dateLocale)}
                    </td>
                    <td className="px-5 py-3 text-right">
                      {isSelf ? (
                        <span className="text-[12px] text-zinc-400">—</span>
                      ) : (
                        <div className="inline-flex items-center gap-2">
                          <button
                            type="button"
                            disabled={pending}
                            onClick={() => void setDisabled(u, !disabled)}
                            className="rounded-md border border-zinc-200 px-2 py-1 text-[12px] font-medium text-zinc-700 transition-colors hover:bg-zinc-50 disabled:opacity-50"
                          >
                            {disabled ? t('admin.enable') : t('admin.disable')}
                          </button>
                          <button
                            type="button"
                            disabled={pending}
                            onClick={() => void removeUser(u)}
                            className="rounded-md border border-red-200 px-2 py-1 text-[12px] font-medium text-red-700 transition-colors hover:bg-red-50 disabled:opacity-50"
                          >
                            {t('admin.delete')}
                          </button>
                        </div>
                      )}
                    </td>
                  </tr>
                )
              })}
              {loaded && users.length === 0 ? (
                <tr>
                  <td
                    colSpan={5}
                    className="px-5 py-8 text-center text-sm text-zinc-500"
                  >
                    {t('admin.noUsers')}
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  )
}
