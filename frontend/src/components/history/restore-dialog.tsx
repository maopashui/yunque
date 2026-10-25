import { useTranslation } from 'react-i18next'
import { AlertTriangle, RotateCcw, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { snapshotLabelText } from './history-timeline'

type Props = {
  open: boolean
  revision: number
  label: string
  busy?: boolean
  onConfirm: () => void
  onCancel: () => void
}

export function RestoreDialog({
  open,
  revision,
  label,
  busy,
  onConfirm,
  onCancel,
}: Props) {
  const { t } = useTranslation()
  if (!open) return null

  const labelText = label ? snapshotLabelText(label) : ''
  const labelPart = labelText
    ? t('history.restoreLabelPart', { label: labelText })
    : ''

  return (
    <div
      className="fixed inset-0 z-[60] flex items-center justify-center bg-zinc-900/40 px-4 backdrop-blur-[2px]"
      onClick={onCancel}
    >
      <div
        role="dialog"
        aria-labelledby="restore-dialog-title"
        className="w-full max-w-md overflow-hidden rounded-lg border border-zinc-200 bg-white shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-start justify-between border-b border-zinc-100 px-5 py-4">
          <div className="flex items-start gap-3">
            <div className="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-amber-50 text-amber-700 ring-1 ring-amber-200">
              <AlertTriangle className="h-4 w-4" />
            </div>
            <div>
              <h2 id="restore-dialog-title" className="text-base font-semibold text-zinc-900">
                {t('history.restoreTitle')}
              </h2>
              <p className="mt-1 text-sm leading-relaxed text-zinc-600">
                {t('history.restoreBody', { revision, labelPart })}
              </p>
            </div>
          </div>
          <button
            type="button"
            aria-label={t('common.close')}
            className="rounded-lg p-1 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-900"
            onClick={onCancel}
          >
            <X className="h-4 w-4" />
          </button>
        </div>
        <div className="flex items-center justify-end gap-2 bg-zinc-50 px-5 py-3">
          <Button variant="secondary" size="sm" disabled={busy} onClick={onCancel}>
            {t('common.cancel')}
          </Button>
          <Button size="sm" disabled={busy} onClick={onConfirm}>
            <RotateCcw className="h-4 w-4" />
            {busy ? t('history.restoring') : t('history.confirmRollback')}
          </Button>
        </div>
      </div>
    </div>
  )
}
