import { translate as t } from 'src/i18n/runtime/instance'
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

type ReasonDialogOptions = ConfirmDialogOptions & {
  reasonLabel?: string
  defaultReason?: string
  maxLength?: number
}

export const useConfirmDialog = ($q: QVueGlobals) => {
  const confirmAction = (options: ConfirmDialogOptions) => {
    return $q.dialog({
      title: options.title || t('ui.confirmOperation'),
      message: options.message,
      persistent: true,
      class: `app-confirm-dialog app-confirm-dialog--action ${options.className || ''}`,
      ok: {
        label: options.okLabel || t('ui.confirm'),
        color: options.color || 'primary',
        unelevated: true,
        loading: options.loading,
        disable: options.disable,
      },
      cancel: {
        label: options.cancelLabel || t('ui.cancel'),
        color: 'grey-7',
        flat: true,
        disable: options.disable,
      },
    })
  }

  const confirmDanger = (options: ConfirmDialogOptions) => {
    return $q.dialog({
      title: options.title || t('ui.confirmDelete'),
      message: options.message,
      persistent: true,
      class: `app-confirm-dialog app-confirm-dialog--danger ${options.className || ''}`,
      ok: {
        label: options.okLabel || t('ui.delete'),
        color: options.color || 'negative',
        unelevated: true,
        loading: options.loading,
        disable: options.disable,
      },
      cancel: {
        label: options.cancelLabel || t('ui.cancel'),
        color: 'grey-7',
        flat: true,
        disable: options.disable,
      },
    })
  }

  const confirmWithReason = (options: ReasonDialogOptions) => {
    const maxLength = options.maxLength || 160
    return $q.dialog({
      title: options.title || t('ui.confirmOperation'),
      message: options.message,
      persistent: true,
      class: `app-confirm-dialog app-confirm-dialog--action ${options.className || ''}`,
      prompt: {
        model: options.defaultReason || '',
        type: 'textarea',
        outlined: true,
        label: options.reasonLabel || t('ui.reasonForOperation'),
        maxlength: maxLength,
        counter: true,
        autogrow: true,
        rules: [(value: string) => Boolean(value.trim()) || t('ui.pleaseFillOutTheReason')],
      },
      ok: {
        label: options.okLabel || t('ui.confirm'),
        color: options.color || 'primary',
        unelevated: true,
      },
      cancel: {
        label: options.cancelLabel || t('ui.cancel'),
        color: 'grey-7',
        flat: true,
      },
    })
  }

  return {
    confirmAction,
    confirmDanger,
    confirmWithReason,
  }
}
