import type { QVueGlobals } from 'quasar'

type ConfirmDialogOptions = {
  title?: string
  message: string
  okLabel?: string
  cancelLabel?: string
  color?: string
  loading?: boolean
  disable?: boolean
  className?: string
}

export const useConfirmDialog = ($q: QVueGlobals) => {
  const confirmAction = (options: ConfirmDialogOptions) => {
    return $q.dialog({
      title: options.title || '确认操作',
      message: options.message,
      persistent: true,
      class: `app-confirm-dialog app-confirm-dialog--action ${options.className || ''}`,
      ok: {
        label: options.okLabel || '确认',
        color: options.color || 'primary',
        unelevated: true,
        loading: options.loading,
        disable: options.disable,
      },
      cancel: {
        label: options.cancelLabel || '取消',
        color: 'grey-7',
        flat: true,
        disable: options.disable,
      },
    })
  }

  const confirmDanger = (options: ConfirmDialogOptions) => {
    return $q.dialog({
      title: options.title || '确认删除',
      message: options.message,
      persistent: true,
      class: `app-confirm-dialog app-confirm-dialog--danger ${options.className || ''}`,
      ok: {
        label: options.okLabel || '删除',
        color: options.color || 'negative',
        unelevated: true,
        loading: options.loading,
        disable: options.disable,
      },
      cancel: {
        label: options.cancelLabel || '取消',
        color: 'grey-7',
        flat: true,
        disable: options.disable,
      },
    })
  }

  return {
    confirmAction,
    confirmDanger,
  }
}
