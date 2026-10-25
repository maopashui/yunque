import { useEffect, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import { useAuth } from '@/app/auth-context'
import { YunQueBadge } from '@/components/brand/yunque-mark'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { api, ApiRequestError } from '@/lib/api'

export function LoginPage() {
  const { t } = useTranslation()
  const { user, loading, login } = useAuth()
  const navigate = useNavigate()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [registrationEnabled, setRegistrationEnabled] = useState(false)

  useEffect(() => {
    let cancelled = false
    void api
      .registrationStatus()
      .then((res) => {
        if (!cancelled) setRegistrationEnabled(res.enabled)
      })
      .catch(() => {
        if (!cancelled) setRegistrationEnabled(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  if (!loading && user) {
    return <Navigate to="/" replace />
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      await login(username.trim(), password)
      navigate('/', { replace: true })
    } catch (err) {
      if (err instanceof ApiRequestError) {
        setError(err.message)
      } else {
        setError(t('auth.loginFailed'))
      }
    } finally {
      setSubmitting(false)
    }
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
                {t('auth.loginSubtitle')}
              </div>
            </div>
          </div>
        </div>

        <div className="px-8 py-7">
          <form className="space-y-4" onSubmit={onSubmit}>
            <div className="space-y-2">
              <Label htmlFor="username">{t('auth.username')}</Label>
              <Input
                id="username"
                autoComplete="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
                minLength={3}
                className="h-10 rounded-lg border-zinc-200 bg-zinc-50 focus:bg-white"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">{t('auth.password')}</Label>
              <Input
                id="password"
                type="password"
                autoComplete="current-password"
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
              {submitting ? t('auth.loggingIn') : t('auth.enterWorkspace')}
            </Button>
          </form>

          {registrationEnabled ? (
            <p className="mt-6 text-center text-sm text-zinc-500">
              {t('auth.noAccount')}{' '}
              <Link
                className="font-medium text-zinc-900 underline-offset-4 hover:underline"
                to="/register"
              >
                {t('auth.register')}
              </Link>
            </p>
          ) : null}
        </div>
      </div>
    </div>
  )
}
