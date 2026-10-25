import { useEffect, useState } from 'react'
import { Link2, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

export type PromptDialogProps = {
  open: boolean
  title: string
  description?: string
  label?: string
  initialValue?: string
  confirmLabel?: string
  cancelLabel?: string
  clearLabel?: string
  allowClear?: boolean
  onConfirm: (value: string) => void
  onClear?: () => void
  onCancel: () => void
}

export function PromptDialog({
  open,
  title,
  description,
  label,
  initialValue = '',
  confirmLabel,
  cancelLabel,
  clearLabel,
  allowClear,
  onConfirm,
  onClear,
  onCancel,
}: PromptDialogProps) {
  const { t } = useTranslation()
  const [value, setValue] = useState(initialValue)

  useEffect(() => {
    if (open) setValue(initialValue)
  }, [open, initialValue])

  if (!open) return null

  return (
    <div
      className="fixed inset-0 z-[70] flex items-center justify-center bg-zinc-900/40 px-4 backdrop-blur-[2px]"
      onClick={onCancel}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="prompt-dialog-title"
        className="w-full max-w-md overflow-hidden rounded-lg border border-zinc-200 bg-white shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-start justify-between border-b border-zinc-100 px-5 py-4">
          <div className="flex items-start gap-3">
            <div className="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-zinc-100 text-zinc-700 ring-1 ring-zinc-200">
              <Link2 className="h-4 w-4" />
            </div>
            <div>
              <h2
                id="prompt-dialog-title"
                className="text-base font-semibold text-zinc-900"
              >
                {title}
              </h2>
              {description ? (
                <p className="mt-1 text-sm leading-relaxed text-zinc-600">
                  {description}
                </p>
              ) : null}
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
        <form
          className="space-y-3 px-5 py-4"
          onSubmit={(e) => {
            e.preventDefault()
            onConfirm(value)
          }}
        >
          <div className="space-y-2">
            {label ? <Label htmlFor="prompt-input">{label}</Label> : null}
            <Input
              id="prompt-input"
              autoFocus
              value={value}
              onChange={(e) => setValue(e.target.value)}
              className="h-10 rounded-lg border-zinc-200 bg-zinc-50 focus:bg-white"
            />
          </div>
          <div className="flex items-center justify-end gap-2 pt-1">
            {allowClear ? (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="mr-auto"
                onClick={() => onClear?.()}
              >
                {clearLabel ?? t('common.clear')}
              </Button>
            ) : null}
            <Button type="button" variant="secondary" size="sm" onClick={onCancel}>
              {cancelLabel ?? t('common.cancel')}
            </Button>
            <Button type="submit" size="sm">
              {confirmLabel ?? t('common.confirm')}
            </Button>
          </div>
        </form>
      </div>
    </div>
  )
}
