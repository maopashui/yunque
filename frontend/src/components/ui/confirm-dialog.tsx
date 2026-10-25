import { AlertTriangle, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

export type ConfirmDialogProps = {
  open: boolean
  title: string
  description: string
  confirmLabel?: string
  cancelLabel?: string
  variant?: 'default' | 'danger'
  busy?: boolean
  onConfirm: () => void
  onCancel: () => void
}

export function ConfirmDialog({
  open,
  title,
  description,
  confirmLabel,
  cancelLabel,
  variant = 'default',
  busy,
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  const { t } = useTranslation()
  if (!open) return null

  const danger = variant === 'danger'

  return (
    <div
      className="fixed inset-0 z-[70] flex items-center justify-center bg-zinc-900/40 px-4 backdrop-blur-[2px]"
      onClick={busy ? undefined : onCancel}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="confirm-dialog-title"
        className="w-full max-w-md overflow-hidden rounded-lg border border-zinc-200 bg-white shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-start justify-between border-b border-zinc-100 px-5 py-4">
          <div className="flex items-start gap-3">
            <div
              className={cn(
                'mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-lg ring-1',
                danger
                  ? 'bg-red-50 text-red-700 ring-red-200'
                  : 'bg-amber-50 text-amber-700 ring-amber-200',
              )}
            >
              <AlertTriangle className="h-4 w-4" />
            </div>
            <div>
              <h2
                id="confirm-dialog-title"
                className="text-base font-semibold text-zinc-900"
              >
                {title}
              </h2>
              <p className="mt-1 text-sm leading-relaxed text-zinc-600">
                {description}
              </p>
            </div>
          </div>
          <button
            type="button"
            aria-label={t('common.close')}
            disabled={busy}
            className="rounded-lg p-1 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-900 disabled:opacity-50"
            onClick={onCancel}
          >
            <X className="h-4 w-4" />
          </button>
        </div>
        <div className="flex items-center justify-end gap-2 bg-zinc-50 px-5 py-3">
          <Button
            variant="secondary"
            size="sm"
            disabled={busy}
            onClick={onCancel}
          >
            {cancelLabel ?? t('common.cancel')}
          </Button>
          <Button
            size="sm"
            disabled={busy}
            onClick={onConfirm}
            className={
              danger
                ? 'bg-red-600 text-white hover:bg-red-700 focus-visible:ring-red-600'
                : undefined
            }
          >
            {confirmLabel ?? t('common.confirm')}
          </Button>
        </div>
      </div>
    </div>
  )
}
