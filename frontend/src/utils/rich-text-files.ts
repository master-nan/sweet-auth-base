import type { FileAccessMode } from 'src/api/services/file'

type FileAccessResolver = (uuid: string, mode: FileAccessMode) => Promise<string | undefined>

const ACCESS_FILE_PATH_RE = /\/files\/access\/(?:preview|download)\/([^/?#]+)/i
const PUBLIC_FILE_PATH_RE = /\/files\/(?!access(?:\/|$))([^/?#]+)/i
const RESERVED_FILE_PATH_SEGMENTS = new Set(['access', 'preview', 'download', 'files'])

export const getDefaultFileBaseUrl = () => {
  const apiBase = String(import.meta.env.VITE_API_URL || '/sweet_admin').replace(/\/+$/, '')
  return `${apiBase}/files`
}

export const canonicalFileUrl = (uuid: string, baseUrl = getDefaultFileBaseUrl()) => {
  return `${baseUrl.replace(/\/+$/, '')}/${encodeURIComponent(uuid)}`
}

export const extractFileUuidFromUrl = (rawUrl?: string | null) => {
  if (!rawUrl) {
    return ''
  }

  const hintedUuid = normalizeFileUuid(extractUuidHint(rawUrl))
  if (hintedUuid) {
    return hintedUuid
  }

  const pathname = toPathname(rawUrl)
  const accessMatch = pathname.match(ACCESS_FILE_PATH_RE)
  const publicMatch = pathname.match(PUBLIC_FILE_PATH_RE)
  const encodedUuid = accessMatch?.[1] || publicMatch?.[1] || ''
  if (!encodedUuid) {
    return ''
  }

  try {
    return normalizeFileUuid(decodeURIComponent(encodedUuid))
  } catch {
    return normalizeFileUuid(encodedUuid)
  }
}

export const serializeRichTextFileUrls = (html: string) => {
  if (!html) {
    return ''
  }

  const doc = parseHtml(html)
  doc.body.querySelectorAll<HTMLImageElement>('img[src], img[data-file-uuid]').forEach((img) => {
    const uuid = normalizeFileUuid(img.dataset.fileUuid) || extractFileUuidFromUrl(img.getAttribute('src'))
    if (!uuid) {
      return
    }
    img.dataset.fileUuid = uuid
    img.dataset.fileMode = 'preview'
    img.setAttribute('src', canonicalFileUrl(uuid))
  })

  doc.body.querySelectorAll<HTMLAnchorElement>('a[href], a[data-file-uuid]').forEach((link) => {
    const uuid = normalizeFileUuid(link.dataset.fileUuid) || extractFileUuidFromUrl(link.getAttribute('href'))
    if (!uuid) {
      return
    }
    link.dataset.fileUuid = uuid
    link.dataset.fileMode = link.dataset.fileMode === 'preview' ? 'preview' : 'download'
    link.setAttribute('href', canonicalFileUrl(uuid))
  })

  return doc.body.innerHTML
}

export const hydrateRichTextFileUrls = async (html: string, resolveAccessUrl: FileAccessResolver) => {
  if (!html) {
    return ''
  }

  const doc = parseHtml(html)
  const accessUrlCache = new Map<string, Promise<string | undefined>>()
  const pending: Array<Promise<void>> = []

  const resolveOnce = (uuid: string, mode: FileAccessMode) => {
    const cacheKey = `${mode}:${uuid}`
    const cached = accessUrlCache.get(cacheKey)
    if (cached) {
      return cached
    }
    const next = resolveAccessUrl(uuid, mode).catch(() => undefined)
    accessUrlCache.set(cacheKey, next)
    return next
  }

  doc.body.querySelectorAll<HTMLImageElement>('img[src], img[data-file-uuid]').forEach((img) => {
    const uuid = normalizeFileUuid(img.dataset.fileUuid) || extractFileUuidFromUrl(img.getAttribute('src'))
    if (!uuid) {
      return
    }
    img.dataset.fileUuid = uuid
    img.dataset.fileMode = 'preview'
    pending.push(
      resolveOnce(uuid, 'preview').then((url) => {
        if (url) {
          img.setAttribute('src', url)
        }
      }),
    )
  })

  doc.body.querySelectorAll<HTMLAnchorElement>('a[href], a[data-file-uuid]').forEach((link) => {
    const uuid = normalizeFileUuid(link.dataset.fileUuid) || extractFileUuidFromUrl(link.getAttribute('href'))
    if (!uuid) {
      return
    }
    const mode: FileAccessMode = link.dataset.fileMode === 'preview' ? 'preview' : 'download'
    link.dataset.fileUuid = uuid
    link.dataset.fileMode = mode
    pending.push(
      resolveOnce(uuid, mode).then((url) => {
        if (url) {
          link.setAttribute('href', url)
        }
      }),
    )
  })

  await Promise.all(pending)
  return doc.body.innerHTML
}

const parseHtml = (html: string) => new DOMParser().parseFromString(html, 'text/html')

const normalizeFileUuid = (uuid?: string | null) => {
  const text = String(uuid || '').trim()
  if (!text || RESERVED_FILE_PATH_SEGMENTS.has(text.toLowerCase())) {
    return ''
  }
  return text
}

const extractUuidHint = (rawUrl: string) => {
  try {
    const url = new URL(rawUrl, 'http://sweet-admin.local')
    return url.searchParams.get('file_uuid') || new URLSearchParams(url.hash.slice(1)).get('file_uuid') || ''
  } catch {
    return ''
  }
}

const toPathname = (rawUrl: string) => {
  try {
    return new URL(rawUrl, 'http://sweet-admin.local').pathname
  } catch {
    return rawUrl
  }
}
