export const formatRuntimeDateTime = (value?: string | null): string => {
  if (!value || value.startsWith('0001-01-01')) return '-'
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? '-' : parsed.toLocaleString()
}

const retryReasonLabels: Record<string, string> = {
  retry_allowed: '符合自动重试条件',
  retry_attempts_exhausted: '已达到最大尝试次数',
  retry_window_expired: '重试窗口已结束',
  retry_error_not_allowed: '当前错误不允许自动重试',
  retry_http_status_not_allowed: '当前 HTTP 状态不允许自动重试',
  retry_unknown_not_idempotent: '结果未知且远端操作不具备幂等保障',
  retry_execution_cancelled: '执行已取消',
  retry_policy_invalid: '冻结的重试策略无效',
  retry_after_window_exceeded: '远端建议时间超过重试窗口',
  retry_method_not_supported: '当前请求方法不支持自动重试',
  retry_remote_idempotency_missing: '缺少远端幂等键保障',
  retry_after_invalid: '远端重试时间无效，已使用本地退避',
  retry_schedule_invalid: '无法生成安全重试计划',
  retry_execution_not_runnable: '执行已不满足重试条件',
}

export const formatRetryReason = (value?: string | null): string => {
  if (!value) return '-'
  return retryReasonLabels[value] || '其他受控原因'
}
