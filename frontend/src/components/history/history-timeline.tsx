import { useTranslation } from 'react-i18next'
import { History, RotateCcw } from 'lucide-react'
import i18n from '@/i18n'
import type { SnapshotMeta } from '@/lib/api'
import { cn, formatDateTime } from '@/lib/utils'

export function snapshotLabelText(label: string) {
  switch (label) {
    case 'auto':
      return i18n.t('history.labelAuto')
    case 'manual':
      return i18n.t('history.labelManual')
    case 'before_restore':
      return i18n.t('history.labelBeforeRestore')
    case 'restored':
      return i18n.t('history.labelRestored')
    default:
      return label || i18n.t('history.labelSnapshot')
  }
}

function authorInitial(name?: string) {
  return (name || '?').slice(0, 1)
}

type Props = {
  snapshots: SnapshotMeta[]
  selectedId: string | null
  currentRevision: number
  canEdit: boolean
  loading?: boolean
  onSelect: (id: string) => void
  onRestore: (snapshot: SnapshotMeta) => void
  onCreateSnapshot?: () => void
  createBusy?: boolean
}

export function HistoryTimeline({
  snapshots,
  selectedId,
  currentRevision,
  canEdit,
  loading,
  onSelect,
  onRestore,
  onCreateSnapshot,
  createBusy,
}: Props) {
  const { t } = useTranslation()

  return (
    <aside className="flex h-full min-h-0 w-full flex-col overflow-hidden border-l border-border bg-white">
      <div className="flex shrink-0 items-center justify-between border-b border-zinc-100 px-4 py-3">
        <div className="flex items-center gap-1.5">
          <History className="h-5 w-5 text-zinc-900" />
          <span className="text-base font-bold text-zinc-900">
            {t('history.timelineTitle')}
          </span>
        </div>
        <span className="rounded border border-zinc-200 bg-zinc-100 px-2 py-0.5 font-mono text-[11px] font-medium text-zinc-700">
          {t('history.snapshotCount', { count: snapshots.length })}
        </span>
      </div>

      <div className="shrink-0 space-y-3 px-4 pt-3">
        <p className="text-sm text-zinc-500">{t('history.timelineHint')}</p>
        {canEdit && onCreateSnapshot ? (
          <button
            type="button"
            disabled={createBusy}
            onClick={onCreateSnapshot}
            className="w-full rounded-lg border border-zinc-200 bg-zinc-50 px-3 py-1.5 text-sm font-medium text-zinc-800 transition-colors hover:bg-zinc-100 disabled:opacity-50"
          >
            {createBusy ? t('history.saving') : t('history.saveSnapshot')}
          </button>
        ) : null}
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto px-4 py-3">
        {loading ? (
          <p className="text-sm text-zinc-500">{t('common.loading')}</p>
        ) : snapshots.length === 0 ? (
          <p className="text-sm text-zinc-500">{t('history.empty')}</p>
        ) : (
          <div className="relative flex flex-col gap-2">
            <div className="absolute bottom-4 left-4 top-4 w-px bg-zinc-200" />
            {snapshots.map((s, i) => {
              const selected = selectedId === s.id
              const isLatest = i === 0
              const cardClass = cn(
                'relative z-10 flex w-full flex-col gap-1 rounded-lg border p-3.5 text-left',
                selected
                  ? 'border-2 border-amber-600 bg-amber-50/50 shadow-sm'
                  : isLatest
                    ? 'border-2 border-[#87d068] bg-white shadow-sm'
                    : 'cursor-pointer border-zinc-200/80 bg-zinc-50 transition-all hover:bg-zinc-100/80',
              )
              const body = (
                <>
                  <div className="flex items-center justify-between gap-2">
                    <div className="flex min-w-0 items-center gap-1.5">
                      <span
                        className={cn(
                          'h-2.5 w-2.5 shrink-0 rounded-full',
                          selected
                            ? 'bg-amber-600 ring-4 ring-amber-100'
                            : isLatest
                              ? 'bg-[#87d068] ring-4 ring-[#87d068]/20'
                              : 'bg-zinc-400',
                        )}
                      />
                      <span className="font-mono text-sm font-bold text-zinc-900">
                        rev {s.revision}
                      </span>
                      {selected ? (
                        <span className="rounded-full bg-amber-600 px-2 py-0.5 font-mono text-[11px] font-medium text-white">
                          {t('history.compareBaseline')}
                        </span>
                      ) : isLatest ? (
                        <span className="rounded-full bg-[#87d068] px-2 py-0.5 font-mono text-[11px] font-medium text-white">
                          {t('history.latestSnapshot')}
                        </span>
                      ) : null}
                    </div>
                    <span
                      className={cn(
                        'shrink-0 font-mono text-[11px]',
                        selected ? 'font-medium text-amber-800' : 'text-zinc-400',
                      )}
                    >
                      {formatDateTime(s.created_at)}
                    </span>
                  </div>
                  <p className="line-clamp-2 text-sm text-zinc-700">
                    {snapshotLabelText(s.label)}
                    {s.title ? ` · ${s.title}` : ''}
                    {s.block_count ? ` · ${s.block_count} blocks` : ''}
                  </p>
                  <div className="flex items-center justify-between gap-2 pt-1">
                    <div className="flex min-w-0 items-center gap-1.5">
                      <div
                        className={cn(
                          'flex h-5 w-5 shrink-0 items-center justify-center rounded-full text-[10px] font-bold text-white shadow-sm',
                          selected
                            ? 'bg-amber-600'
                            : isLatest
                              ? 'bg-[#87d068]'
                              : 'bg-zinc-500',
                        )}
                      >
                        {authorInitial(s.creator_name)}
                      </div>
                      <span className="truncate font-mono text-[11px] font-medium text-zinc-600">
                        {s.creator_name || t('common.system')}
                      </span>
                    </div>
                    {canEdit && !isLatest ? (
                      <span
                        role="button"
                        tabIndex={0}
                        onClick={(e) => {
                          e.stopPropagation()
                          onRestore(s)
                        }}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter' || e.key === ' ') {
                            e.preventDefault()
                            e.stopPropagation()
                            onRestore(s)
                          }
                        }}
                        className={cn(
                          'inline-flex shrink-0 items-center gap-1 rounded px-2.5 py-1 font-mono text-[11px] font-medium transition-colors',
                          selected
                            ? 'bg-zinc-900 text-white hover:bg-black'
                            : 'border border-zinc-300 bg-white text-zinc-800 hover:bg-zinc-100',
                        )}
                      >
                        <RotateCcw className="h-3 w-3" />
                        {t('history.restore')}
                      </span>
                    ) : null}
                  </div>
                  {isLatest && s.revision === currentRevision ? (
                    <span className="mt-0.5 self-start rounded border border-[#87d068] bg-[#87d068]/15 px-2 py-0.5 font-mono text-[11px] font-semibold text-[#87d068]">
                      {t('history.nearCurrent')}
                    </span>
                  ) : null}
                </>
              )
              if (isLatest) {
                return (
                  <div key={s.id} className={cardClass} aria-disabled="true">
                    {body}
                  </div>
                )
              }
              return (
                <button
                  key={s.id}
                  type="button"
                  onClick={() => onSelect(s.id)}
                  className={cardClass}
                >
                  {body}
                </button>
              )
            })}
          </div>
        )}
      </div>
    </aside>
  )
}
