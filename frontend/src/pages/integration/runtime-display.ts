export const formatRuntimeDateTime = (value?: string | null): string => {
  if (!value || value.startsWith('0001-01-01')) return '-'
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? '-' : parsed.toLocaleString()
}
