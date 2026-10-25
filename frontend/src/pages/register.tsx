import { useEffect, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import { useAuth } from '@/app/auth-context'
import { YunQueBadge } from '@/components/brand/yunque-mark'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { api, ApiRequestError } from '@/lib/api'

export function RegisterPage() {
  const { t } = useTranslation()
  const { user, loading, register } = useAuth()
  const navigate = useNavigate()
  const [username, setUsername] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [statusLoading, setStatusLoading] = useState(true)
  const [registrationEnabled, setRegistrationEnabled] = useState(false)

  useEffect(() => {
    let cancelled = false
    void api
      .registrationStatus()
      .then((res) => {
        if (!cancelled) {
          setRegistrationEnabled(res.enabled)
          setStatusLoading(false)
        }
      })
      .catch(() => {
        if (!cancelled) {
          setRegistrationEnabled(false)
          setStatusLoading(false)
        }
      })
    return () => {
      cancelled = true
    }
  }, [])

  if (!loading && user) {
    return <Navigate to="/" replace />
  }

  if (!statusLoading && !registrationEnabled) {
    return <Navigate to="/login" replace />
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      await register(username.trim(), password, displayName.trim() || undefined)
      navigate('/', { replace: true })
    } catch (err) {
      if (err instanceof ApiRequestError) {
        if (err.code === 'registration_closed') {
          setError(t('auth.registrationClosed'))
          setRegistrationEnabled(false)
        } else {
          setError(err.message)
        }
      } else {
        setError(t('auth.registerFailed'))
      }
    } finally {
      setSubmitting(false)
    }
  }

  if (statusLoading) {
    return (
      <div className="flex min-h-full items-center justify-center bg-[#fbfbfb] text-sm text-zinc-500">
        {t('common.loading')}
      </div>
    )
  }

  return (
    <div className="relative flex min-h-full items-center justify-center overflow-hidden bg-[#fbfbfb] px-4">
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_at_top,_#f4f4f5_0%,_transparent_55%),linear-gradient(to_bottom,#fbfbfb,#f8f8f9)]"
      />
      <div className="relative w-full max-w-md overflow-hidden rounded-xl border border-zinc-200 bg-white shadow-[0_16px_40px_rgba(0,0,0,0.06)]">
        <div className="border-b border-zinc-100 bg-zinc-50/80 px-8 py-6">
          <div className="flex items-center gap-3">
            <YunQueBadge className="h-10 w-10 rounded-lg" title={t('common.productName')} />
            <div>
              <div className="text-sm font-semibold text-zinc-900">
                {t('common.productName')}
              </div>
              <div className="font-mono text-[11px] text-zinc-500">
                {t('auth.registerSubtitle')}
              </div>
            </div>
          </div>
        </div>

        <div className="px-8 py-7">
          <h1 className="text-xl font-bold tracking-tight text-zinc-900">
            {t('auth.registerTitle')}
          </h1>
          <p className="mt-1 text-sm text-zinc-500">{t('auth.registerHint')}</p>

          <form className="mt-6 space-y-4" onSubmit={onSubmit}>
            <div className="space-y-2">
              <Label htmlFor="username">{t('auth.username')}</Label>
              <Input
                id="username"
                autoComplete="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
                minLength={3}
                maxLength={32}
                pattern="[A-Za-z0-9_-]+"
                title={t('auth.usernamePattern')}
                className="h-10 rounded-lg border-zinc-200 bg-zinc-50 focus:bg-white"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="displayName">{t('auth.displayName')}</Label>
              <Input
                id="displayName"
                autoComplete="nickname"
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
                maxLength={64}
                className="h-10 rounded-lg border-zinc-200 bg-zinc-50 focus:bg-white"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">{t('auth.password')}</Label>
              <Input
                id="password"
                type="password"
                autoComplete="new-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                minLength={8}
                className="h-10 rounded-lg border-zinc-200 bg-zinc-50 focus:bg-white"
              />
            </div>

            {error ? (
              <p className="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900">
                {error}
              </p>
            ) : null}

            <Button
              className="h-10 w-full rounded-lg bg-zinc-900 hover:bg-black"
              type="submit"
              disabled={submitting}
            >
              {submitting ? t('auth.registering') : t('auth.registerAndEnter')}
            </Button>
          </form>

          <p className="mt-6 text-center text-sm text-zinc-500">
            {t('auth.hasAccount')}{' '}
            <Link
              className="font-medium text-zinc-900 underline-offset-4 hover:underline"
              to="/login"
            >
              {t('auth.login')}
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}
