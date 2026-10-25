export type ApiError = {
  error: {
    code: string
    message: string
  }
}

export type User = {
  id: string
  username: string
  display_name: string
  avatar_color?: string
  is_admin?: boolean
}

export type SiteSettings = {
  registration_enabled: boolean
}

export type AdminUser = {
  id: string
  username: string
  display_name: string
  avatar_color?: string
  is_admin: boolean
  disabled_at?: string | null
  created_at: string
}

export type Document = {
  id: string
  owner_id: string
  owner_name?: string
  folder_id?: string | null
  title: string
  format_version: number
  revision: number
  block_count: number
  is_favorite: boolean
  role?: string
  created_at: string
  updated_at: string
  deleted_at?: string | null
}

export type Block = {
  id: string
  document_id: string
  order_key: string
  block_type: string
  content: Record<string, unknown>
  attrs: Record<string, unknown>
  revision: number
  created_at: string
  updated_at: string
}

export type BlockMeta = {
  id: string
  order_key: string
  block_type: string
  attrs: Record<string, unknown>
  index: number
}

export type TOCItem = {
  id: string
  order_key: string
  index: number
  level: number
  text: string
}

export type DocumentMember = {
  user_id: string
  username: string
  display_name: string
  avatar_color?: string
  role: 'owner' | 'editor' | 'viewer'
  created_at: string
}

export type SnapshotMeta = {
  id: string
  document_id: string
  revision: number
  label: string
  created_by: string
  created_at: string
  block_count: number
  title: string
  creator_name?: string
}

export type SnapshotDetail = SnapshotMeta & {
  blocks: Block[]
}

export type DiffEntry = {
  id: string
  block_type: string
  preview: string
}

export type DiffChangeEntry = {
  id: string
  block_type: string
  from_preview: string
  to_preview: string
}

export type SnapshotDiff = {
  from_revision: number
  to_revision: number
  title_changed: boolean
  from_title: string
  to_title: string
  added: DiffEntry[]
  removed: DiffEntry[]
  changed: DiffChangeEntry[]
  unchanged: number
}

export type DocumentAsset = {
  id: string
  document_id: string
  uploader_id: string
  filename: string
  mime: string
  size: number
  url: string
  created_at: string
}

/** Normalize diff payload (supports legacy string[] entries from older backends). */
export function normalizeSnapshotDiff(raw: unknown): SnapshotDiff | null {
  if (!raw || typeof raw !== 'object') return null
  const d = raw as Record<string, unknown>

  const asEntry = (item: unknown, index: number): DiffEntry => {
    if (typeof item === 'string') {
      return { id: `legacy-${index}`, block_type: '', preview: item }
    }
    if (item && typeof item === 'object') {
      const o = item as Record<string, unknown>
      return {
        id: typeof o.id === 'string' && o.id ? o.id : `entry-${index}`,
        block_type: typeof o.block_type === 'string' ? o.block_type : '',
        preview: typeof o.preview === 'string' ? o.preview : '',
      }
    }
    return { id: `entry-${index}`, block_type: '', preview: '' }
  }

  const asChange = (item: unknown, index: number): DiffChangeEntry => {
    if (typeof item === 'string') {
      return {
        id: `legacy-change-${index}`,
        block_type: '',
        from_preview: item,
        to_preview: item,
      }
    }
    if (item && typeof item === 'object') {
      const o = item as Record<string, unknown>
      return {
        id: typeof o.id === 'string' && o.id ? o.id : `change-${index}`,
        block_type: typeof o.block_type === 'string' ? o.block_type : '',
        from_preview: typeof o.from_preview === 'string' ? o.from_preview : '',
        to_preview: typeof o.to_preview === 'string' ? o.to_preview : '',
      }
    }
    return {
      id: `change-${index}`,
      block_type: '',
      from_preview: '',
      to_preview: '',
    }
  }

  const addedRaw = Array.isArray(d.added) ? d.added : []
  const removedRaw = Array.isArray(d.removed) ? d.removed : []
  const changedRaw = Array.isArray(d.changed) ? d.changed : []

  return {
    from_revision: Number(d.from_revision ?? 0),
    to_revision: Number(d.to_revision ?? 0),
    title_changed: Boolean(d.title_changed),
    from_title: typeof d.from_title === 'string' ? d.from_title : '',
    to_title: typeof d.to_title === 'string' ? d.to_title : '',
    added: addedRaw.map(asEntry),
    removed: removedRaw.map(asEntry),
    changed: changedRaw.map(asChange),
    unchanged: Number(d.unchanged ?? 0),
  }
}

async function parseJSON<T>(res: Response): Promise<T> {
  const text = await res.text()
  if (!text) {
    return {} as T
  }
  return JSON.parse(text) as T
}

export class ApiRequestError extends Error {
  code: string
  status: number

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiRequestError'
    this.status = status
    this.code = code
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const headers: Record<string, string> = {
    ...(init?.headers as Record<string, string> | undefined),
  }
  const isFormData =
    typeof FormData !== 'undefined' && init?.body instanceof FormData
  if (init?.body && !headers['Content-Type'] && !isFormData) {
    headers['Content-Type'] = 'application/json'
  }

  const res = await fetch(path, {
    ...init,
    credentials: 'include',
    headers,
  })

  if (!res.ok) {
    let code = 'request_failed'
    let message = res.statusText || 'request failed'
    try {
      const body = await parseJSON<ApiError>(res)
      if (body.error?.code) code = body.error.code
      if (body.error?.message) message = body.error.message
    } catch {
      // ignore
    }
    throw new ApiRequestError(res.status, code, message)
  }

  const contentType = res.headers.get('Content-Type') || ''
  if (contentType.includes('text/markdown')) {
    return (await res.text()) as T
  }

  return parseJSON<T>(res)
}

export type ListDocumentsParams = {
  q?: string
  sort?: 'updated_at' | 'title' | 'created_at'
  order?: 'asc' | 'desc'
  trash?: boolean
  favorite?: boolean
  limit?: number
  offset?: number
}

export type DocumentListResponse = {
  documents: Document[]
  total: number
  limit: number
  offset: number
}

export const api = {
  health: () => request<{ status: string; version: string }>('/api/health'),
  me: () => request<{ user: User }>('/api/auth/me'),
  registrationStatus: () =>
    request<{ enabled: boolean }>('/api/auth/registration'),
  register: (body: { username: string; password: string; display_name?: string }) =>
    request<{ user: User }>('/api/auth/register', {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  login: (body: { username: string; password: string }) =>
    request<{ user: User }>('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  logout: () =>
    request<{ status: string }>('/api/auth/logout', {
      method: 'POST',
    }),
  getAdminSettings: () =>
    request<{ settings: SiteSettings }>('/api/admin/settings'),
  patchAdminSettings: (body: { registration_enabled: boolean }) =>
    request<{ settings: SiteSettings }>('/api/admin/settings', {
      method: 'PATCH',
      body: JSON.stringify(body),
    }),
  listAdminUsers: () => request<{ users: AdminUser[] }>('/api/admin/users'),
  patchAdminUser: (userId: string, body: { disabled: boolean }) =>
    request<{ user: AdminUser }>(`/api/admin/users/${userId}`, {
      method: 'PATCH',
      body: JSON.stringify(body),
    }),
  deleteAdminUser: (userId: string) =>
    request<{ status: string }>(`/api/admin/users/${userId}`, {
      method: 'DELETE',
    }),

  listDocuments: (params: ListDocumentsParams = {}) => {
    const q = new URLSearchParams()
    if (params.q) q.set('q', params.q)
    if (params.sort) q.set('sort', params.sort)
    if (params.order) q.set('order', params.order)
    if (params.trash) q.set('trash', '1')
    if (params.favorite) q.set('favorite', '1')
    if (params.limit != null) q.set('limit', String(params.limit))
    if (params.offset != null) q.set('offset', String(params.offset))
    const qs = q.toString()
    return request<DocumentListResponse>(`/api/documents${qs ? `?${qs}` : ''}`)
  },
  createDocument: (body: { title?: string } = {}) =>
    request<{ document: Document }>('/api/documents', {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  getDocument: (id: string) =>
    request<{ document: Document }>(`/api/documents/${id}`),
  patchDocument: (
    id: string,
    body: { title?: string; is_favorite?: boolean; clear_folder?: boolean },
  ) =>
    request<{ document: Document }>(`/api/documents/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(body),
    }),
  deleteDocument: (id: string) =>
    request<{ status: string }>(`/api/documents/${id}`, { method: 'DELETE' }),
  restoreDocument: (id: string) =>
    request<{ document: Document }>(`/api/documents/${id}/restore`, {
      method: 'POST',
    }),
  importDocument: (body: { title?: string; markdown: string }) =>
    request<{ document: Document }>('/api/documents/import', {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  exportDocument: async (id: string) => {
    const res = await fetch(`/api/documents/${id}/export.md`, {
      credentials: 'include',
    })
    if (!res.ok) {
      throw new ApiRequestError(res.status, 'export_failed', 'export failed')
    }
    return res.text()
  },
  listBlocks: (id: string) =>
    request<{ blocks: Block[] }>(`/api/documents/${id}/blocks`),
  listBlockIndex: (id: string) =>
    request<{ document: Document; revision: number; blocks: BlockMeta[] }>(
      `/api/documents/${id}/blocks/index`,
    ),
  listTOC: (id: string) =>
    request<{ toc: TOCItem[] }>(`/api/documents/${id}/toc`),
  listBlocksWindow: (
    id: string,
    params: { after_key?: string; limit?: number; anchor_id?: string; before?: number; after?: number },
  ) => {
    const q = new URLSearchParams()
    if (params.after_key != null) q.set('after_key', params.after_key)
    if (params.limit != null) q.set('limit', String(params.limit))
    if (params.anchor_id) q.set('anchor_id', params.anchor_id)
    if (params.before != null) q.set('before', String(params.before))
    if (params.after != null) q.set('after', String(params.after))
    return request<{ blocks: Block[] }>(`/api/documents/${id}/blocks/window?${q}`)
  },
  listBlocksBatch: (id: string, ids: string[]) =>
    request<{ blocks: Block[] }>(`/api/documents/${id}/blocks/batch`, {
      method: 'POST',
      body: JSON.stringify({ ids }),
    }),
  syncContent: (
    id: string,
    body: {
      base_revision?: number
      operation_id?: string
      blocks: {
        id?: string
        block_type: string
        content: Record<string, unknown>
        attrs: Record<string, unknown>
      }[]
    },
  ) =>
    request<{ document: Document; blocks: Block[] }>(`/api/documents/${id}/content`, {
      method: 'PUT',
      body: JSON.stringify(body),
    }),

  uploadDocumentAsset: (id: string, file: File) => {
    const body = new FormData()
    body.append('file', file)
    return request<{ asset: DocumentAsset }>(`/api/documents/${id}/assets`, {
      method: 'POST',
      body,
    })
  },

  listMembers: (id: string) =>
    request<{ members: DocumentMember[] }>(`/api/documents/${id}/members`),
  addMember: (id: string, body: { username: string; role: 'editor' | 'viewer' }) =>
    request<{ member: DocumentMember }>(`/api/documents/${id}/members`, {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  updateMember: (id: string, userId: string, body: { role: 'editor' | 'viewer' }) =>
    request<{ member: DocumentMember }>(`/api/documents/${id}/members/${userId}`, {
      method: 'PATCH',
      body: JSON.stringify(body),
    }),
  removeMember: async (id: string, userId: string) => {
    const res = await fetch(`/api/documents/${id}/members/${userId}`, {
      method: 'DELETE',
      credentials: 'include',
    })
    if (!res.ok) {
      let code = 'request_failed'
      let message = res.statusText || 'request failed'
      try {
        const body = await parseJSON<ApiError>(res)
        if (body.error?.code) code = body.error.code
        if (body.error?.message) message = body.error.message
      } catch {
        // ignore
      }
      throw new ApiRequestError(res.status, code, message)
    }
  },

  listSnapshots: (id: string) =>
    request<{ snapshots: SnapshotMeta[] }>(`/api/documents/${id}/snapshots`),
  createSnapshot: (id: string, body: { label?: string } = {}) =>
    request<{ snapshot: SnapshotMeta }>(`/api/documents/${id}/snapshots`, {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  getSnapshot: (id: string, snapshotId: string) =>
    request<{ snapshot: SnapshotDetail }>(`/api/documents/${id}/snapshots/${snapshotId}`),
  compareSnapshot: (id: string, snapshotId: string) =>
    request<{ diff: SnapshotDiff }>(`/api/documents/${id}/snapshots/${snapshotId}/diff`),
  restoreSnapshot: (id: string, snapshotId: string) =>
    request<{ document: Document; blocks: Block[]; snapshot: SnapshotMeta }>(
      `/api/documents/${id}/snapshots/${snapshotId}/restore`,
      { method: 'POST' },
    ),
  cleanupSnapshots: (id: string, keep = 50) =>
    request<{ deleted: number; keep: number }>(
      `/api/documents/${id}/snapshots/cleanup?keep=${keep}`,
      { method: 'POST' },
    ),
}
