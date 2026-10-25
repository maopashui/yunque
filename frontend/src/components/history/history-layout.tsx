import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  api,
  ApiRequestError,
  normalizeSnapshotDiff,
  type Block,
  type SnapshotDiff,
  type SnapshotMeta,
} from '@/lib/api'
import { HistoryDiffStage } from './history-diff-stage'
import { HistoryTimeline } from './history-timeline'
import { RestoreDialog } from './restore-dialog'

type Props = {
  documentId: string
  documentTitle: string
  currentRevision: number
  canEdit: boolean
  onRestored: () => void
}

export function HistoryLayout({
  documentId,
  documentTitle,
  currentRevision,
  canEdit,
  onRestored,
}: Props) {
  const { t } = useTranslation()
  const [list, setList] = useState<SnapshotMeta[]>([])
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [previewBlocks, setPreviewBlocks] = useState<Block[]>([])
  const [currentBlocks, setCurrentBlocks] = useState<Block[]>([])
  const [previewTitle, setPreviewTitle] = useState('')
  const [diff, setDiff] = useState<SnapshotDiff | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [createBusy, setCreateBusy] = useState(false)
  const [restoreTarget, setRestoreTarget] = useState<SnapshotMeta | null>(null)

  const selected = list.find((s) => s.id === selectedId) ?? null

  const reload = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await api.listSnapshots(documentId)
      setList(res.snapshots ?? [])
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : t('history.loadFailed'))
    } finally {
      setLoading(false)
    }
  }, [documentId, t])

  useEffect(() => {
    setSelectedId(null)
    setPreviewBlocks([])
    setDiff(null)
    void reload()
    void (async () => {
      try {
        const res = await api.listBlocks(documentId)
        setCurrentBlocks(res.blocks ?? [])
      } catch {
        setCurrentBlocks([])
      }
    })()
  }, [documentId, reload])

  async function openSnapshot(id: string) {
    setSelectedId(id)
    setError(null)
    try {
      const [detail, diffRes] = await Promise.all([
        api.getSnapshot(documentId, id),
        api.compareSnapshot(documentId, id),
      ])
      setPreviewTitle(detail.snapshot.title)
      setPreviewBlocks(detail.snapshot.blocks ?? [])
      setDiff(normalizeSnapshotDiff(diffRes.diff))
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : t('history.previewFailed'))
    }
  }

  async function createManual() {
    setCreateBusy(true)
    setError(null)
    try {
      await api.createSnapshot(documentId, { label: 'manual' })
      await reload()
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : t('common.createFailed'))
    } finally {
      setCreateBusy(false)
    }
  }

  async function confirmRestore() {
    if (!restoreTarget) return
    setBusy(true)
    setError(null)
    try {
      await api.restoreSnapshot(documentId, restoreTarget.id)
      setRestoreTarget(null)
      onRestored()
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : t('common.restoreFailed'))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="flex h-[calc(100vh-2.75rem)] flex-col overflow-hidden bg-background">
      {error ? (
        <div className="shrink-0 border-b border-border bg-amber-50 px-5 py-2 text-sm text-amber-900 lg:px-6">
          {error}
        </div>
      ) : null}
      <div className="flex min-h-0 flex-1 flex-col lg:flex-row">
        <HistoryDiffStage
          documentTitle={documentTitle}
          currentRevision={currentRevision}
          selected={selected}
          previewTitle={previewTitle}
          previewBlocks={previewBlocks}
          currentBlocks={currentBlocks}
          diff={diff}
          canEdit={canEdit}
          busy={busy}
          onRestore={() => selected && setRestoreTarget(selected)}
        />
        <div className="min-h-0 shrink-0 lg:h-full lg:w-96">
          <HistoryTimeline
            snapshots={list}
            selectedId={selectedId}
            currentRevision={currentRevision}
            canEdit={canEdit}
            loading={loading}
            onSelect={(id) => void openSnapshot(id)}
            onRestore={setRestoreTarget}
            onCreateSnapshot={() => void createManual()}
            createBusy={createBusy}
          />
        </div>
      </div>

      <RestoreDialog
        open={!!restoreTarget}
        revision={restoreTarget?.revision ?? 0}
        label={restoreTarget ? restoreTarget.label : ''}
        busy={busy}
        onConfirm={() => void confirmRestore()}
        onCancel={() => setRestoreTarget(null)}
      />
    </div>
  )
}
