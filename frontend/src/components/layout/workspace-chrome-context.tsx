import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from 'react'

type WorkspaceChromeValue = {
  docCount: number | null
  docCountLoading: boolean
  setDocCount: (count: number | null, loading?: boolean) => void
}

const WorkspaceChromeContext = createContext<WorkspaceChromeValue | null>(null)

export function WorkspaceChromeProvider({ children }: { children: ReactNode }) {
  const [docCount, setDocCountState] = useState<number | null>(null)
  const [docCountLoading, setDocCountLoading] = useState(false)

  const setDocCount = useCallback((count: number | null, loading = false) => {
    setDocCountState(count)
    setDocCountLoading(loading)
  }, [])

  const value = useMemo(
    () => ({ docCount, docCountLoading, setDocCount }),
    [docCount, docCountLoading, setDocCount],
  )

  return (
    <WorkspaceChromeContext.Provider value={value}>
      {children}
    </WorkspaceChromeContext.Provider>
  )
}

export function useWorkspaceChrome() {
  const ctx = useContext(WorkspaceChromeContext)
  if (!ctx) {
    throw new Error('useWorkspaceChrome must be used within WorkspaceChromeProvider')
  }
  return ctx
}

/** Optional for pages that may render outside the provider. */
export function useWorkspaceChromeOptional() {
  return useContext(WorkspaceChromeContext)
}
