import { useTranslation } from 'react-i18next'
import { Diff, History, RotateCcw } from 'lucide-react'
import type { Block, SnapshotDiff, SnapshotMeta } from '@/lib/api'
import {
  buildUnifiedRows,
  summarizeRows,
  type DiffRow,
} from '@/lib/block-diff'
import { cn } from '@/lib/utils'
import { snapshotLabelText } from './history-timeline'

function DiffLine({ row }: { row: DiffRow }) {
  if (row.kind === 'hunk') {
    return (
      <div className="flex border-y border-sky-100 bg-sky-50/80 font-mono text-[12px] text-sky-800">
        <div className="w-12 shrink-0 select-none border-r border-sky-100 px-1 py-1.5 text-center text-sky-500">
          ···
        </div>
        <div className="w-12 shrink-0 select-none border-r border-sky-100 px-1 py-1.5 text-center text-sky-500">
          ···
        </div>
        <div className="w-6 shrink-0 select-none py-1.5 text-center text-sky-600">@</div>
        <div className="min-w-0 flex-1 px-2 py-1.5">
          <span className="font-medium">{row.meta ?? row.text}</span>
        </div>
      </div>
    )
  }

  const sign = row.kind === 'add' ? '+' : row.kind === 'del' ? '-' : ' '
  return (
    <div
      className={cn(
        'group flex border-b border-zinc-100/80 font-mono text-[13px] leading-6 last:border-b-0',
        row.kind === 'add' && 'bg-emerald-50/90 text-emerald-950',
        row.kind === 'del' && 'bg-rose-50/90 text-rose-950',
        row.kind === 'ctx' && 'bg-white text-zinc-700',
      )}
    >
      <div
        className={cn(
          'w-12 shrink-0 select-none border-r px-1 py-1.5 text-right tabular-nums',
          row.kind === 'add' && 'border-emerald-100 text-emerald-700/50',
          row.kind === 'del' && 'border-rose-100 text-rose-700/50',
          row.kind === 'ctx' && 'border-zinc-100 text-zinc-300',
        )}
      >
        {row.oldNo ?? ''}
      </div>
      <div
        className={cn(
          'w-12 shrink-0 select-none border-r px-1 py-1.5 text-right tabular-nums',
          row.kind === 'add' && 'border-emerald-100 text-emerald-700/50',
          row.kind === 'del' && 'border-rose-100 text-rose-700/50',
          row.kind === 'ctx' && 'border-zinc-100 text-zinc-300',
        )}
      >
        {row.newNo ?? ''}
      </div>
      <div
        className={cn(
          'w-6 shrink-0 select-none py-1.5 text-center font-bold',
          row.kind === 'add' && 'text-emerald-700',
          row.kind === 'del' && 'text-rose-700',
          row.kind === 'ctx' && 'text-zinc-300',
        )}
      >
        {sign}
      </div>
      <div
        className={cn(
          'min-w-0 flex-1 whitespace-pre-wrap break-words px-2 py-1.5',
          row.kind === 'del' && 'opacity-90',
        )}
      >
        {row.text.trim() ? (
          row.text
        ) : (
          <span className="select-none text-zinc-300">&nbsp;</span>
        )}
      </div>
    </div>
  )
}

type Props = {
  documentTitle: string
  currentRevision: number
  selected: SnapshotMeta | null
  previewTitle: string
  previewBlocks: Block[]
  currentBlocks: Block[]
  diff: SnapshotDiff | null
  canEdit: boolean
  busy?: boolean
  onRestore: () => void
}

export function HistoryDiffStage({
  documentTitle,
  currentRevision,
  selected,
  previewTitle,
  previewBlocks,
  currentBlocks,
  diff,
  canEdit,
  busy,
  onRestore,
}: Props) {
  const { t } = useTranslation()
  const rows = selected
    ? buildUnifiedRows(previewBlocks ?? [], currentBlocks ?? [], 3)
    : []
  const { additions, deletions } = summarizeRows(rows)
  const hasDiff =
    additions > 0 || deletions > 0 || Boolean(diff?.title_changed)

  return (
    <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
      <div className="z-30 flex shrink-0 flex-wrap items-center justify-between gap-3 border-b border-border bg-white px-5 py-2.5 lg:px-6">
        <div className="flex min-w-0 flex-wrap items-center gap-3">
          <div className="flex items-center gap-1.5 rounded-md border border-zinc-200 bg-zinc-100 px-2.5 py-1 font-mono text-[11px] text-zinc-800">
            <History className="h-4 w-4 text-zinc-900" />
            <span className="font-medium">{t('history.stageTitle')}</span>
          </div>
          <div className="hidden h-4 w-px bg-zinc-200 sm:block" />
          <div className="flex min-w-0 flex-wrap items-center gap-2 text-sm">
            <span className="whitespace-nowrap text-zinc-500">{t('history.comparing')}</span>
            <div className="flex items-center gap-1.5 rounded-md bg-zinc-900 px-2.5 py-1 font-medium text-white shadow-sm">
              <span className="h-1.5 w-1.5 rounded-full bg-white" />
              <span className="font-mono text-[12px]">
                rev {currentRevision} · {t('history.current')}
              </span>
            </div>
            <span className="font-mono text-[11px] font-medium text-zinc-400">vs</span>
            {selected ? (
              <div className="flex items-center gap-1.5 rounded-md border border-zinc-300 bg-zinc-100 px-2.5 py-1 font-medium text-zinc-800">
                <span className="h-1.5 w-1.5 rounded-full bg-zinc-600" />
                <span className="truncate font-mono text-[12px]">
                  rev {selected.revision}
                  {selected.creator_name ? ` · ${selected.creator_name}` : ''}
                </span>
              </div>
            ) : (
              <span className="text-sm text-zinc-400">{t('history.pickVersion')}</span>
            )}
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {selected ? (
            <div className="flex flex-wrap items-center gap-1.5">
              <span className="inline-flex items-center gap-1 rounded-md border border-emerald-200 bg-emerald-50 px-2 py-0.5 font-mono text-[11px] font-semibold text-emerald-800">
                <span className="font-bold text-emerald-700">+</span>
                {additions}
              </span>
              <span className="inline-flex items-center gap-1 rounded-md border border-rose-200 bg-rose-50 px-2 py-0.5 font-mono text-[11px] font-semibold text-rose-800">
                <span className="font-bold text-rose-700">-</span>
                {deletions}
              </span>
            </div>
          ) : null}
          {selected && canEdit ? (
            <button
              type="button"
              disabled={busy}
              onClick={onRestore}
              className="inline-flex items-center gap-1.5 rounded-lg bg-zinc-900 px-3.5 py-1.5 text-sm font-medium text-white shadow-sm transition-colors hover:bg-black disabled:opacity-50"
            >
              <RotateCcw className="h-4 w-4" />
              {t('history.rollbackTo', { revision: selected.revision })}
            </button>
          ) : null}
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto px-5 py-5 lg:px-6">
        <div className="flex w-full flex-col gap-5">
          <div className="flex flex-col gap-2">
            <div className="flex flex-wrap items-center gap-2">
              <span className="rounded-full bg-zinc-900 px-2.5 py-0.5 font-mono text-[11px] font-medium text-white">
                {documentTitle || t('common.untitledDoc')}
              </span>
              <span className="inline-flex items-center gap-1 rounded border border-sky-200 bg-sky-50 px-2 py-0.5 font-mono text-[11px] font-medium text-sky-800">
                <Diff className="h-3.5 w-3.5 text-sky-600" />
                Unified Diff
              </span>
            </div>
            <h1 className="text-2xl font-bold tracking-tight text-zinc-900">
              {previewTitle || documentTitle || t('history.untitledParen')}
            </h1>
            {selected ? (
              <p className="text-sm leading-relaxed text-zinc-600">
                {t('history.baselineToCurrent', {
                  label: snapshotLabelText(selected.label),
                  from: selected.revision,
                  to: currentRevision,
                })}
                {diff?.title_changed
                  ? t('history.titleChanged', {
                      from: diff.from_title,
                      to: diff.to_title,
                    })
                  : ''}
              </p>
            ) : (
              <p className="text-sm text-zinc-500">{t('history.pickSnapshotHint')}</p>
            )}
          </div>

          {selected ? (
            <div className="overflow-hidden rounded-lg border border-zinc-200 bg-white shadow-sm">
              <div className="flex flex-wrap items-center justify-between gap-2 border-b border-zinc-200 bg-zinc-50 px-3 py-2">
                <div className="flex items-center gap-2 font-mono text-[12px] text-zinc-600">
                  <span className="font-semibold text-zinc-900">
                    rev {selected.revision} → rev {currentRevision}
                  </span>
                  <span className="text-zinc-300">|</span>
                  <span>
                    {(previewBlocks ?? []).length} → {(currentBlocks ?? []).length} blocks
                  </span>
                </div>
                <div className="flex items-center gap-3 font-mono text-[11px] text-zinc-500">
                  <span className="inline-flex items-center gap-1">
                    <span className="font-bold text-emerald-700">+</span>
                    {t('history.legendAdd')}
                  </span>
                  <span className="inline-flex items-center gap-1">
                    <span className="font-bold text-rose-700">-</span>
                    {t('history.legendDel')}
                  </span>
                </div>
              </div>

              {!hasDiff ? (
                <div className="px-4 py-8 text-center text-sm text-zinc-500">
                  {t('history.noDiff')}
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <div className="flex border-b border-zinc-100 bg-zinc-50/80 font-mono text-[11px] text-zinc-400">
                    <div className="w-12 shrink-0 border-r border-zinc-100 px-1 py-1 text-center">
                      {t('history.colOld')}
                    </div>
                    <div className="w-12 shrink-0 border-r border-zinc-100 px-1 py-1 text-center">
                      {t('history.colNew')}
                    </div>
                    <div className="w-6 shrink-0" />
                    <div className="flex-1 px-2 py-1">{t('history.colContent')}</div>
                  </div>
                  {rows.map((row) => (
                    <DiffLine key={row.key} row={row} />
                  ))}
                </div>
              )}
            </div>
          ) : null}
        </div>
      </div>
    </div>
  )
}
