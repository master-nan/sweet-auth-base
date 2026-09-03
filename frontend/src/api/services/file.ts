import type { Basic, ResponseData } from '@/types/global'
import { instance } from '@/boot/axios'

export interface FileInfo extends Basic {
  file_name: string
  file_type: string
  file_url: string
  file_size: number
  file_ext: string
  file_uuid: string
}

export type FileAccessMode = 'preview' | 'download'

export interface FileBusinessContext {
  table_code?: string
  record_id?: number | string
  menu_id?: number
  action?: 'query' | 'detail' | 'update' | 'delete'
}

export interface FileAccessUrl {
  url: string
  expires_at: number
}

const uploadRequestConfig = () => ({
  headers: {
    'Content-Type': 'multipart/form-data',
    'X-Skip-Global-Loading': 'true',
  },
})

const fileRequestConfig = () => ({
  headers: {
    'X-Skip-Global-Loading': 'true',
  },
})

// 分片上传初始化请求
export interface ChunkUploadInitReq {
  file_name: string
  file_size: number
  file_md5: string
  file_type: string
}

// 分片上传初始化响应
export interface ChunkUploadInitRes {
  upload_id: string
  file_id?: number
  chunk_size: number
  chunk_count: number
  fast_upload: boolean
}

// 分片上传进度响应
export interface ChunkUploadProgressRes {
  upload_id: string
  chunk_count: number
  uploaded_count: number
  uploaded_indexes: number[]
}

export const useFileApi = () => {
  // ─── 普通上传（小文件） ───
  const uploadFile = async (file: File, onProgress?: (loaded: number, total: number) => void) => {
    const formData = new FormData()
    formData.append('file', file)
    return instance
      .post<ResponseData<FileInfo>>('/admin/file/upload', formData, {
        ...uploadRequestConfig(),
        onUploadProgress: (e) => {
          if (onProgress && e.total) {
            onProgress(e.loaded, e.total)
          }
        },
      })
      .then((res) => res.data)
  }

  // ─── 分片上传 ───

  /** 初始化分片上传 */
  const initChunkUpload = async (req: ChunkUploadInitReq) => {
    return instance
      .post<ResponseData<ChunkUploadInitRes>>('/admin/file/upload/init', req, fileRequestConfig())
      .then((res) => res.data)
  }

  /** 上传单个分片 */
  const uploadChunk = async (
    uploadId: string,
    chunkIndex: number,
    chunk: Blob,
    onProgress?: (loaded: number, total: number) => void,
  ) => {
    const formData = new FormData()
    formData.append('upload_id', uploadId)
    formData.append('chunk_index', String(chunkIndex))
    formData.append('file', chunk)
    return instance
      .post<ResponseData<void>>('/admin/file/upload/chunk', formData, {
        ...uploadRequestConfig(),
        onUploadProgress: (e) => {
          if (onProgress && e.total) {
            onProgress(e.loaded, e.total)
          }
        },
      })
      .then((res) => res.data)
  }

  /** 合并分片 */
  const mergeChunks = async (uploadId: string) => {
    return instance
      .post<ResponseData<FileInfo>>(`/admin/file/upload/merge/${uploadId}`, null, fileRequestConfig())
      .then((res) => res.data)
  }

  /** 获取上传进度（断点续传） */
  const getUploadProgress = async (uploadId: string) => {
    return instance
      .get<ResponseData<ChunkUploadProgressRes>>(
        `/admin/file/upload/progress/${uploadId}`,
        fileRequestConfig(),
      )
      .then((res) => res.data)
  }

  // ─── 基础操作 ───

  const getFileById = (id: number, context?: FileBusinessContext) => {
    return instance
      .get<ResponseData<FileInfo>>(`/admin/file/${id}`, { params: context || {} })
      .then((res) => res.data)
  }

  const deleteFile = (id: number) => {
    return instance.delete<ResponseData<void>>(`/admin/file/${id}`).then((res) => res.data)
  }

  const getFileDownloadUrl = (uuid: string) => {
    return `/admin/file/download/${uuid}`
  }

  const getFilePreviewAccessUrl = (uuid: string, ttl = 900, context?: FileBusinessContext) => {
    return instance
      .get<ResponseData<FileAccessUrl>>(`/admin/file/preview-url/${uuid}`, {
        params: { ttl, ...(context || {}) },
      })
      .then((res) => res.data)
  }

  const getFileDownloadAccessUrl = (uuid: string, ttl = 900, context?: FileBusinessContext) => {
    return instance
      .get<ResponseData<FileAccessUrl>>(`/admin/file/download-url/${uuid}`, {
        params: { ttl, ...(context || {}) },
      })
      .then((res) => res.data)
  }

  const getFileAccessUrl = (
    uuid: string,
    mode: FileAccessMode = 'preview',
    ttl = 900,
    context?: FileBusinessContext,
  ) => {
    if (mode === 'download') {
      return getFileDownloadAccessUrl(uuid, ttl, context)
    }
    return getFilePreviewAccessUrl(uuid, ttl, context)
  }

  return {
    uploadFile,
    initChunkUpload,
    uploadChunk,
    mergeChunks,
    getUploadProgress,
    getFileById,
    deleteFile,
    getFileDownloadUrl,
    getFilePreviewAccessUrl,
    getFileDownloadAccessUrl,
    getFileAccessUrl,
  }
}
