import { useCallback, useState } from 'react'
import {
  ConfirmDialog,
  type ConfirmDialogProps,
} from '@/components/ui/confirm-dialog'

type ConfirmOptions = {
  title: string
  description: string
  confirmLabel?: string
  cancelLabel?: string
  variant?: ConfirmDialogProps['variant']
}

type Pending = ConfirmOptions & {
  resolve: (value: boolean) => void
}

export function useConfirmDialog() {
  const [pending, setPending] = useState<Pending | null>(null)

  const confirm = useCallback((options: ConfirmOptions) => {
    return new Promise<boolean>((resolve) => {
      setPending({ ...options, resolve })
    })
  }, [])

  const close = useCallback((value: boolean) => {
    setPending((cur) => {
      cur?.resolve(value)
      return null
    })
  }, [])

  const dialog = (
    <ConfirmDialog
      open={pending != null}
      title={pending?.title ?? ''}
      description={pending?.description ?? ''}
      confirmLabel={pending?.confirmLabel}
      cancelLabel={pending?.cancelLabel}
      variant={pending?.variant}
      onConfirm={() => close(true)}
      onCancel={() => close(false)}
    />
  )

  return { confirm, dialog }
}
