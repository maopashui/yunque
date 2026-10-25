import { useTranslation } from 'react-i18next'
import { Navigate, Outlet } from 'react-router-dom'
import { useAuth } from '@/app/auth-context'

export function ProtectedRoute() {
  const { t } = useTranslation()
  const { user, loading } = useAuth()

  if (loading) {
    return (
      <div className="flex min-h-full items-center justify-center bg-background text-sm text-muted">
        {t('common.loading')}
      </div>
    )
  }

  if (!user) {
    return <Navigate to="/login" replace />
  }

  return <Outlet />
}
