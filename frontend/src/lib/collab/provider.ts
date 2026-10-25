import * as Y from 'yjs'
import {
  Awareness,
  applyAwarenessUpdate,
  encodeAwarenessUpdate,
  removeAwarenessStates,
} from 'y-protocols/awareness'
import i18n from '@/i18n'
import { resolveAvatarColor } from '@/lib/avatar-color'

export type CollabStatus = 'connecting' | 'synced' | 'offline' | 'error'

export type CollabUser = {
  id: string
  name: string
  color: string
}

export type PresencePeer = {
  clientId: number
  user: CollabUser
}

type HelloMsg = {
  type: 'hello'
  doc_id: string
  empty: boolean
  can_edit: boolean
  user: { id: string; username: string; display_name: string }
}

type SyncMsg = {
  type: 'sync'
  snapshot?: string
  updates?: { seq: number; operation_id: string; update: string }[]
  update_seq?: number
}

type WireMsg = {
  type: string
  update?: string
  operation_id?: string
  payload?: string
  snapshot?: string
  update_seq?: number
  code?: string
  message?: string
  empty?: boolean
  can_edit?: boolean
  user?: HelloMsg['user']
  updates?: SyncMsg['updates']
}

function toBase64(bytes: Uint8Array): string {
  let s = ''
  for (let i = 0; i < bytes.length; i++) s += String.fromCharCode(bytes[i]!)
  return btoa(s)
}

function fromBase64(b64: string): Uint8Array {
  const bin = atob(b64)
  const out = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i)
  return out
}

function wsURL(documentId: string): string {
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${proto}//${window.location.host}/api/documents/${documentId}/collab`
}

export type YunQueCollabProviderOptions = {
  documentId: string
  user: CollabUser
  onStatus?: (status: CollabStatus, message?: string) => void
  onPeers?: (peers: PresencePeer[]) => void
  onHello?: (info: { empty: boolean; canEdit: boolean }) => void
}

export class YunQueCollabProvider {
  readonly doc: Y.Doc
  readonly awareness: Awareness
  readonly documentId: string

  private ws: WebSocket | null = null
  private destroyed = false
  private synced = false
  private reconnectTimer: number | null = null
  private opts: YunQueCollabProviderOptions
  private localUser: CollabUser

  constructor(opts: YunQueCollabProviderOptions) {
    this.opts = opts
    this.documentId = opts.documentId
    this.localUser = opts.user
    this.doc = new Y.Doc()
    this.awareness = new Awareness(this.doc)
    this.awareness.setLocalStateField('user', this.localUser)

    this.doc.on('update', this.onDocUpdate)
    this.awareness.on('update', this.onAwarenessUpdate)
    this.awareness.on('change', this.emitPeers)
    this.connect()
  }

  get isSynced() {
    return this.synced
  }

  private setStatus(status: CollabStatus, message?: string) {
    this.opts.onStatus?.(status, message)
  }

  private emitPeers = () => {
    const peers: PresencePeer[] = []
    this.awareness.getStates().forEach((state, clientId) => {
      const user = state.user as CollabUser | undefined
      if (!user?.id) return
      peers.push({ clientId, user })
    })
    this.opts.onPeers?.(peers)
  }

  private onDocUpdate = (update: Uint8Array, origin: unknown) => {
    if (origin === 'remote' || !this.synced || !this.ws || this.ws.readyState !== WebSocket.OPEN) {
      return
    }
    this.send({
      type: 'update',
      operation_id: crypto.randomUUID(),
      update: toBase64(update),
    })
  }

  private onAwarenessUpdate = (
    { added, updated, removed }: { added: number[]; updated: number[]; removed: number[] },
    origin: unknown,
  ) => {
    if (origin === 'remote' || !this.ws || this.ws.readyState !== WebSocket.OPEN) return
    const changed = added.concat(updated, removed)
    if (!changed.length) return
    const encoded = encodeAwarenessUpdate(this.awareness, changed)
    this.send({ type: 'awareness', payload: toBase64(encoded) })
  }

  private send(msg: WireMsg) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return
    this.ws.send(JSON.stringify(msg))
  }

  private connect() {
    if (this.destroyed) return
    this.setStatus('connecting')
    const ws = new WebSocket(wsURL(this.documentId))
    this.ws = ws

    ws.onopen = () => {
      // wait for hello + sync from server
    }

    ws.onmessage = (ev) => {
      let msg: WireMsg
      try {
        msg = JSON.parse(String(ev.data)) as WireMsg
      } catch {
        return
      }
      this.handleMessage(msg)
    }

    ws.onclose = () => {
      this.synced = false
      this.setStatus('offline')
      if (!this.destroyed) this.scheduleReconnect()
    }

    ws.onerror = () => {
      this.setStatus('error', i18n.t('collab.wsError'))
    }
  }

  private scheduleReconnect() {
    if (this.reconnectTimer != null) return
    this.reconnectTimer = window.setTimeout(() => {
      this.reconnectTimer = null
      this.connect()
    }, 1500)
  }

  private handleMessage(msg: WireMsg) {
    switch (msg.type) {
      case 'hello': {
        const empty = Boolean(msg.empty)
        const canEdit = Boolean(msg.can_edit)
        this.opts.onHello?.({ empty, canEdit })
        break
      }
      case 'sync': {
        if (msg.snapshot) {
          Y.applyUpdate(this.doc, fromBase64(msg.snapshot), 'remote')
        }
        for (const u of msg.updates ?? []) {
          Y.applyUpdate(this.doc, fromBase64(u.update), 'remote')
        }
        this.synced = true
        this.setStatus('synced')
        this.send({ type: 'sync_done' })
        // announce awareness after sync
        const encoded = encodeAwarenessUpdate(this.awareness, [this.doc.clientID])
        this.send({ type: 'awareness', payload: toBase64(encoded) })
        this.emitPeers()
        break
      }
      case 'update': {
        if (!msg.update) return
        Y.applyUpdate(this.doc, fromBase64(msg.update), 'remote')
        break
      }
      case 'awareness': {
        if (!msg.payload) return
        applyAwarenessUpdate(this.awareness, fromBase64(msg.payload), 'remote')
        break
      }
      case 'error': {
        this.setStatus('error', msg.message || msg.code || i18n.t('collab.collabError'))
        break
      }
      default:
        break
    }
  }

  /** Persist a merged snapshot to reduce future sync payload. */
  pushSnapshot() {
    if (!this.synced) return
    const snap = Y.encodeStateAsUpdate(this.doc)
    this.send({
      type: 'snapshot',
      snapshot: toBase64(snap),
    })
  }

  destroy() {
    this.destroyed = true
    if (this.reconnectTimer != null) {
      window.clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    this.doc.off('update', this.onDocUpdate)
    this.awareness.off('update', this.onAwarenessUpdate)
    this.awareness.off('change', this.emitPeers)
    removeAwarenessStates(this.awareness, [this.doc.clientID], 'local')
    this.ws?.close()
    this.ws = null
    this.doc.destroy()
  }
}

export function makeCollabUser(
  id: string,
  name: string,
  avatarColor?: string | null,
): CollabUser {
  return {
    id,
    name: name || i18n.t('common.user'),
    color: resolveAvatarColor(id, avatarColor),
  }
}
