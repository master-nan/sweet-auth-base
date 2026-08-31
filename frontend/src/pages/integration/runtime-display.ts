import { translate as t } from 'src/boot/i18n'
export const formatRuntimeDateTime = (value?: string | null): string => {
  if (!value || value.startsWith('0001-01-01')) return '-'
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? '-' : parsed.toLocaleString()
}

const retryReasonLabels: Record<string, string> = {
  get retry_allowed() {
    return t('ui.matchTheAutomaticRetryCondition')
  },
  get retry_attempts_exhausted() {
    return t('ui.maximumNumberOfAttemptsReached')
  },
  get retry_window_expired() {
    return t('ui.retryWindowClosed')
  },
  get retry_error_not_allowed() {
    return t('ui.currentErrorDoesNotAllowAutomaticRetry')
  },
  get retry_http_status_not_allowed() {
    return t('ui.currentHttpStateDoesNotAllowAutomaticRetrying')
  },
  get retry_unknown_not_idempotent() {
    return t('ui.theResultsAreUnknownAndRemoteOperationsAreNotGuaranteedForExample')
  },
  get retry_execution_cancelled() {
    return t('ui.executionCanceled')
  },
  get retry_policy_invalid() {
    return t('ui.theFrozenRetryStrategyIsInvalid')
  },
  get retry_after_window_exceeded() {
    return t('ui.remotelyRecommendTimeBeyondRetryWindow')
  },
  get retry_method_not_supported() {
    return t('ui.currentRequestMethodDoesNotSupportAutomaticRetrying')
  },
  get retry_remote_idempotency_missing() {
    return t('ui.lackOfRemoteKeySecurity')
  },
  get retry_after_invalid() {
    return t('ui.remoteRetryTimeInvalidLocalRetreatUsed')
  },
  get retry_schedule_invalid() {
    return t('ui.couldNotGenerateSafeRetestPlan')
  },
  get retry_execution_not_runnable() {
    return t('ui.implementationDoesNotMeetRetryConditions')
  },
}

export const formatRetryReason = (value?: string | null): string => {
  if (!value) return '-'
  return retryReasonLabels[value] || t('ui.otherControlledReasons')
}
