import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { AuthProvider } from '@/app/auth-context'
import { ProtectedRoute } from '@/app/protected-route'
import { AppShell } from '@/components/layout/app-shell'
import { AdminPage } from '@/pages/admin'
import { DocumentReaderPage } from '@/pages/document-reader'
import { LoginPage } from '@/pages/login'
import { RegisterPage } from '@/pages/register'
import { WorkspacePage } from '@/pages/workspace'

export function AppRouter() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route element={<ProtectedRoute />}>
            <Route element={<AppShell />}>
              <Route index element={<WorkspacePage />} />
              <Route path="recent" element={<WorkspacePage />} />
              <Route path="favorites" element={<WorkspacePage />} />
              <Route path="trash" element={<WorkspacePage />} />
              <Route path="admin" element={<AdminPage />} />
              <Route path="docs/:id" element={<DocumentReaderPage />} />
            </Route>
          </Route>
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  )
}
