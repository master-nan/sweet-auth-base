export const parseContentDispositionFilename = (contentDisposition?: string): string => {
  if (!contentDisposition) return ''

  const encodedFilenameMatch = /filename\*\s*=\s*(?:UTF-8''|utf-8'')?([^;]+)/i.exec(
    contentDisposition,
  )
  if (encodedFilenameMatch?.[1]) {
    const value = encodedFilenameMatch[1].trim().replace(/^"|"$/g, '')
    try {
      return decodeURIComponent(value)
    } catch {
      return value
    }
  }

  const filenameMatch = /filename\s*=\s*([^;]+)/i.exec(contentDisposition)
  if (!filenameMatch?.[1]) return ''

  const value = filenameMatch[1].trim().replace(/^"|"$/g, '')
  try {
    return decodeURIComponent(value)
  } catch {
    return value
  }
}

export const parseBlobJsonError = async (blob: Blob): Promise<string | undefined> => {
  if (!blob.size) return undefined

  try {
    const text = await blob.text()
    if (!text.trim()) return undefined

    const parsed = JSON.parse(text) as {
      message?: string
      msg?: string
      error?: string
      error_message?: string
      data?: { message?: string; msg?: string; error?: string; error_message?: string } | string
    }

    if (typeof parsed.data === 'string') return parsed.data

    return (
      parsed.error_message ||
      parsed.message ||
      parsed.msg ||
      parsed.error ||
      parsed.data?.error_message ||
      parsed.data?.message ||
      parsed.data?.msg ||
      parsed.data?.error
    )
  } catch {
    return undefined
  }
}

export const downloadBlob = (blob: Blob, filename: string): void => {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename || 'download'
  link.style.display = 'none'
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}
