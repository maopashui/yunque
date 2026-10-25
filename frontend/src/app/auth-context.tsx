import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { api, ApiRequestError, type User } from '@/lib/api'

type AuthState = {
  user: User | null
  loading: boolean
  healthOk: boolean | null
  backendVersion: string | null
  refresh: () => Promise<void>
  login: (username: string, password: string) => Promise<void>
  register: (username: string, password: string, displayName?: string) => Promise<void>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [healthOk, setHealthOk] = useState<boolean | null>(null)
  const [backendVersion, setBackendVersion] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    try {
      const health = await api.health()
      setHealthOk(health.status === 'ok')
      setBackendVersion(health.version)
    } catch {
      setHealthOk(false)
      setBackendVersion(null)
    }

    try {
      const me = await api.me()
      setUser(me.user)
    } catch (err) {
      if (err instanceof ApiRequestError && err.status === 401) {
        setUser(null)
      } else {
        setUser(null)
      }
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void refresh()
  }, [refresh])

  const login = useCallback(async (username: string, password: string) => {
    const res = await api.login({ username, password })
    setUser(res.user)
  }, [])

  const register = useCallback(
    async (username: string, password: string, displayName?: string) => {
      await api.register({
        username,
        password,
        display_name: displayName,
      })
      const res = await api.login({ username, password })
      setUser(res.user)
    },
    [],
  )

  const logout = useCallback(async () => {
    await api.logout()
    setUser(null)
  }, [])

  const value = useMemo(
    () => ({
      user,
      loading,
      healthOk,
      backendVersion,
      refresh,
      login,
      register,
      logout,
    }),
    [user, loading, healthOk, backendVersion, refresh, login, register, logout],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error('useAuth must be used within AuthProvider')
  }
  return ctx
}
