const fileIdKeys = ['id', 'file_id', 'fileId']

const normalizeFileId = (value: unknown): number | null => {
  if (typeof value === 'object' && value !== null) {
    const record = value as Record<string, unknown>
    for (const key of fileIdKeys) {
      const id = normalizeFileId(record[key])
      if (id) return id
    }
    return null
  }

  if (typeof value === 'number') {
    return Number.isFinite(value) && value > 0 ? value : null
  }

  if (typeof value !== 'string') return null

  const text = value.trim()
  if (!/^\d+$/.test(text)) return null
  const id = Number(text)
  return Number.isFinite(id) && id > 0 ? id : null
}

export const parseFileIds = (value: unknown): number[] => {
  if (value === null || value === undefined || value === '') return []

  if (Array.isArray(value)) {
    return Array.from(
      new Set(value.map((item) => normalizeFileId(item)).filter((id): id is number => !!id)),
    )
  }

  const singleId = normalizeFileId(value)
  if (singleId) return [singleId]

  if (typeof value !== 'string') return []

  const text = value.trim()
  if (!text) return []

  try {
    const parsed = JSON.parse(text)
    return parseFileIds(parsed)
  } catch {
    return Array.from(
      new Set(
        text
          .split(/[,;，；\n\r]+/)
          .map((item) => normalizeFileId(item))
          .filter((id): id is number => !!id),
      ),
    )
  }
}
