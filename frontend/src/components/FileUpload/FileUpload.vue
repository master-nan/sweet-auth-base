<template>
  <div class="file-upload">
    <q-field
      :label="label"
      outlined
      dense
      stack-label
      :rules="rules"
      :error="!!errorMessage"
      :error-message="errorMessage"
      :disable="readonly"
    >
      <template v-slot:control>
        <div class="full-width">
          <!-- 已上传文件列表 -->
          <div v-if="fileList.length > 0" class="q-mb-sm">
            <q-chip
              v-for="(file, index) in fileList"
              :key="file.file_uuid || index"
              :removable="!readonly"
              dense
              color="primary"
              text-color="white"
              icon="attach_file"
              @remove="removeFile(index)"
              class="q-mr-xs q-mb-xs"
            >
              <span class="ellipsis" style="max-width: 200px">{{ file.file_name }}</span>
              <q-tooltip>{{ file.file_name }} ({{ formatFileSize(file.file_size) }})</q-tooltip>
              <q-btn
                flat
                dense
                round
                size="xs"
                icon="visibility"
                class="q-ml-xs"
                @click.stop="openUploadedFile(file, 'preview')"
              >
                <q-tooltip>预览</q-tooltip>
              </q-btn>
              <q-btn
                flat
                dense
                round
                size="xs"
                icon="download"
                @click.stop="openUploadedFile(file, 'download')"
              >
                <q-tooltip>下载</q-tooltip>
              </q-btn>
            </q-chip>
          </div>

          <!-- 上传按钮 -->
          <q-btn
            v-if="!readonly && ((!multiple && fileList.length === 0) || multiple)"
            flat
            dense
            no-caps
            color="primary"
            icon="cloud_upload"
            :label="uploading ? uploadStatusText || '上传中...' : '选择文件'"
            :disable="uploading || readonly"
            @click="triggerFileInput"
          />

          <!-- 隐藏的文件输入 -->
          <input
            ref="fileInputRef"
            type="file"
            :accept="accept || '*'"
            :multiple="multiple || false"
            style="display: none"
            @change="handleFileChange"
          />
        </div>
      </template>
    </q-field>

    <file-preview-dialog ref="previewDialogRef" />

    <div v-if="progressVisible" class="file-upload__progress">
      <div class="row items-center no-wrap q-gutter-sm">
        <q-linear-progress
          class="col"
          :value="progressValue"
          :indeterminate="uploading && uploadProgress <= 0"
          color="primary"
          track-color="grey-3"
          rounded
          size="8px"
        />
        <div class="file-upload__percent">{{ uploadProgress }}%</div>
      </div>
      <div class="file-upload__status">
        {{ uploadStatusLabel }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, ref, watch } from 'vue'
import { useFileApi, type FileAccessMode, type FileInfo } from 'src/api/services/file'
import { Notify } from 'quasar'
import SparkMD5 from 'spark-md5'
import { parseFileIds } from 'src/utils/file-value'

type FilePreviewDialogExpose = {
  open: (file: FileInfo) => void | Promise<void>
}

const FilePreviewDialog = defineAsyncComponent(
  () => import('src/components/FileUpload/FilePreviewDialog.vue'),
)

interface FileUploadProps {
  modelValue?: string | string[] | null
  label?: string
  accept?: string
  multiple?: boolean
  maxSize?: number // MB
  chunkThreshold?: number // 超过此大小（MB）启用分片上传，默认 5
  concurrency?: number // 分片并发上传数，默认 10
  rules?: Array<(val: any) => boolean | string>
  tableCode?: string
  menuId?: number
  rowId?: number | string
  fieldCode?: string
  readonly?: boolean
}

const props = withDefaults(defineProps<FileUploadProps>(), {
  modelValue: null,
  label: '文件上传',
  accept: '*',
  multiple: false,
  maxSize: 50000,
  chunkThreshold: 5,
  concurrency: 10,
  rules: () => [],
  tableCode: '',
  menuId: 0,
  rowId: 0,
  fieldCode: '',
  readonly: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: string | string[] | null]
}>()

const {
  uploadFile,
  initChunkUpload,
  uploadChunk,
  mergeChunks,
  getUploadProgress,
  getFileById,
  getFilePreviewAccessUrl,
  getFileDownloadAccessUrl,
} = useFileApi()
const fileInputRef = ref<HTMLInputElement | null>(null)
const previewDialogRef = ref<FilePreviewDialogExpose | null>(null)
const uploading = ref(false)
const errorMessage = ref('')
const fileList = ref<FileInfo[]>([])
const isInternalUpdate = ref(false)
const uploadProgress = ref(0)
const uploadStatusText = ref('')
const currentUploadName = ref('')
let clearStatusTimer: ReturnType<typeof setTimeout> | null = null

const progressVisible = computed(() => uploading.value || !!uploadStatusText.value)
const progressValue = computed(
  () => Math.min(Math.max(uploadProgress.value, uploading.value ? 1 : 0), 100) / 100,
)
const uploadStatusLabel = computed(() => {
  const name = currentUploadName.value ? ` · ${currentUploadName.value}` : ''
  return `${uploadStatusText.value || '准备上传'}${name}`
})

const clearUploadStatusLater = (delay = 1600) => {
  if (clearStatusTimer) {
    clearTimeout(clearStatusTimer)
  }
  clearStatusTimer = setTimeout(() => {
    if (uploading.value) return
    uploadProgress.value = 0
    uploadStatusText.value = ''
    currentUploadName.value = ''
  }, delay)
}

// ─── 计算文件 MD5（流式分片计算，避免大文件内存溢出） ───
const calculateFileMD5 = (
  file: File,
  onProgress?: (currentChunk: number, totalChunks: number) => void,
): Promise<string> => {
  return new Promise((resolve, reject) => {
    const chunkSize = 2 * 1024 * 1024 // 2MB per chunk for MD5 calculation
    const chunks = Math.ceil(file.size / chunkSize)
    const spark = new SparkMD5.ArrayBuffer()
    const reader = new FileReader()
    let currentChunk = 0

    reader.onload = (e) => {
      spark.append(e.target!.result as ArrayBuffer)
      currentChunk++
      onProgress?.(currentChunk, chunks)
      if (currentChunk < chunks) {
        loadNext()
      } else {
        resolve(spark.end())
      }
    }
    reader.onerror = () => reject(new Error('文件读取失败'))

    const loadNext = () => {
      const start = currentChunk * chunkSize
      const end = Math.min(start + chunkSize, file.size)
      reader.readAsArrayBuffer(file.slice(start, end))
    }
    loadNext()
  })
}

// ─── 分片上传核心逻辑 ───
const doChunkUpload = async (file: File): Promise<FileInfo | null> => {
  uploadStatusText.value = '计算文件指纹...'
  uploadProgress.value = 1

  // 1. 计算文件 MD5
  const fileMd5 = await calculateFileMD5(file, (current, total) => {
    uploadStatusText.value = `计算文件指纹 ${current}/${total}`
    uploadProgress.value = Math.max(1, Math.round((current / total) * 8))
  })

  // 2. 初始化分片上传
  uploadStatusText.value = '初始化上传...'
  uploadProgress.value = Math.max(uploadProgress.value, 9)
  const initRes = await initChunkUpload({
    file_name: file.name,
    file_size: file.size,
    file_md5: fileMd5,
    file_type: file.type || 'application/octet-stream',
  })

  if (!initRes.success || !initRes.data) {
    throw new Error(initRes.message || '初始化分片上传失败')
  }

  // 秒传
  if (initRes.data.fast_upload && initRes.data.file_id) {
    uploadProgress.value = 100
    uploadStatusText.value = '秒传完成'
    const fileRes = await getFileById(initRes.data.file_id)
    if (fileRes.success && fileRes.data) {
      return fileRes.data
    }
    return null
  }

  const { upload_id, chunk_size, chunk_count } = initRes.data

  // 3. 查询已上传的分片（断点续传）
  let uploadedSet = new Set<number>()
  try {
    const progressRes = await getUploadProgress(upload_id)
    if (progressRes.success && progressRes.data?.uploaded_indexes) {
      uploadedSet = new Set(progressRes.data.uploaded_indexes)
    }
  } catch {
    // 忽略，从头开始上传
  }

  // 4. 并发上传分片
  const totalChunks = chunk_count
  let completedChunks = uploadedSet.size

  uploadProgress.value = Math.max(10, 10 + Math.round((completedChunks / totalChunks) * 80))
  if (completedChunks > 0) {
    uploadStatusText.value = `断点续传，已跳过 ${completedChunks}/${totalChunks} 个分片`
  } else {
    uploadStatusText.value = `准备上传 ${totalChunks} 个分片`
  }

  // 构建待上传分片列表
  const pendingIndexes: number[] = []
  for (let i = 0; i < totalChunks; i++) {
    if (!uploadedSet.has(i)) {
      pendingIndexes.push(i)
    }
  }

  // 并发控制器
  const uploadWithConcurrency = async (indexes: number[]) => {
    const queue = [...indexes]
    const workers: Promise<void>[] = []

    for (let i = 0; i < Math.min(props.concurrency, queue.length); i++) {
      workers.push(
        (async () => {
          while (queue.length > 0) {
            const idx = queue.shift()!
            const start = idx * chunk_size
            const end = Math.min(start + chunk_size, file.size)
            const blob = file.slice(start, end)

            uploadStatusText.value = `分片上传 ${completedChunks + 1}/${totalChunks}`
            await uploadChunk(upload_id, idx, blob, (loaded, total) => {
              const currentChunkRatio = total > 0 ? loaded / total : 0
              const nextProgress =
                10 + Math.round(((completedChunks + currentChunkRatio) / totalChunks) * 80)
              uploadProgress.value = Math.max(uploadProgress.value, nextProgress)
            })

            completedChunks++
            uploadProgress.value = Math.max(
              uploadProgress.value,
              10 + Math.round((completedChunks / totalChunks) * 80),
            )
          }
        })(),
      )
    }
    await Promise.all(workers)
  }

  await uploadWithConcurrency(pendingIndexes)

  // 5. 合并分片
  uploadStatusText.value = '合并文件...'
  uploadProgress.value = 95
  const mergeRes = await mergeChunks(upload_id)
  if (!mergeRes.success || !mergeRes.data) {
    throw new Error(mergeRes.message || '合并分片失败')
  }

  uploadProgress.value = 100
  uploadStatusText.value = '上传完成'
  return mergeRes.data
}

// ─── 从 modelValue 恢复文件信息 ───
const loadFileInfo = async (value: string | string[] | null) => {
  if (!value) {
    fileList.value = []
    return
  }

  const ids = parseFileIds(value)
  if (ids.length === 0) {
    fileList.value = []
    return
  }

  const files: FileInfo[] = []
  for (const id of ids) {
    if (isNaN(id) || id <= 0) continue
    try {
      const res = await getFileById(id)
      if (res.success && res.data) {
        files.push(res.data)
      }
    } catch (e) {
      console.warn('Failed to load file info for id:', id)
      console.error(e)
    }
  }
  fileList.value = files
}

// 监听 modelValue 变化
watch(
  () => props.modelValue,
  (newVal) => {
    if (isInternalUpdate.value) {
      isInternalUpdate.value = false
      return
    }
    loadFileInfo(newVal)
  },
  { immediate: true },
)

const triggerFileInput = () => {
  if (props.readonly) return
  fileInputRef.value?.click()
}

const handleFileChange = async (event: Event) => {
  const input = event.target as HTMLInputElement
  const files = input.files
  if (!files || files.length === 0) return

  uploading.value = true
  errorMessage.value = ''
  uploadProgress.value = 0
  uploadStatusText.value = ''
  currentUploadName.value = ''

  if (!props.multiple) {
    fileList.value = []
  }

  try {
    for (const file of Array.from(files)) {
      currentUploadName.value = file.name
      // 校验文件大小
      if (file.size > props.maxSize * 1024 * 1024) {
        Notify.create({
          type: 'negative',
          message: `文件 ${file.name} 大小超过 ${props.maxSize}MB 限制`,
        })
        continue
      }

      let fileInfo: FileInfo | null = null
      const chunkThresholdBytes = props.chunkThreshold * 1024 * 1024

      if (file.size > chunkThresholdBytes) {
        // 大文件 → 分片上传
        fileInfo = await doChunkUpload(file)
      } else {
        // 小文件 → 直传
        uploadStatusText.value = '上传中...'
        uploadProgress.value = 5
        const res = await uploadFile(file, (loaded, total) => {
          uploadProgress.value = Math.max(5, Math.round((loaded / total) * 95))
        })
        if (res.success && res.data) {
          fileInfo = res.data
        }
        uploadProgress.value = 100
        uploadStatusText.value = '上传完成'
      }

      if (fileInfo) {
        fileList.value.push(fileInfo)
      } else {
        Notify.create({
          type: 'negative',
          message: `文件 ${file.name} 上传失败`,
        })
      }
    }

    // 更新 modelValue
    emitValue()
  } catch (e: any) {
    errorMessage.value = e?.message || '上传失败'
    uploadStatusText.value = '上传失败'
    Notify.create({
      type: 'negative',
      message: errorMessage.value,
    })
  } finally {
    uploading.value = false
    if (!errorMessage.value) {
      uploadProgress.value = 100
      uploadStatusText.value = '上传完成'
      clearUploadStatusLater()
    }
    // 重置 input，允许再次选择同一文件
    if (fileInputRef.value) {
      fileInputRef.value.value = ''
    }
  }
}

const removeFile = (index: number) => {
  if (props.readonly) return
  fileList.value.splice(index, 1)
  emitValue()
}

const openUploadedFile = async (file: FileInfo, mode: FileAccessMode) => {
  if (!file.file_uuid) {
    Notify.create({ type: 'warning', message: '文件缺少访问标识' })
    return
  }

  if (mode === 'preview') {
    previewDialogRef.value?.open(file)
    return
  }

  const response =
    mode === 'download'
      ? await getFileDownloadAccessUrl(file.file_uuid)
      : await getFilePreviewAccessUrl(file.file_uuid)
  if (!response.success || !response.data?.url) {
    Notify.create({ type: 'negative', message: response.message || '获取文件访问地址失败' })
    return
  }
  window.open(response.data.url, '_blank', 'noopener,noreferrer')
}

const emitValue = () => {
  isInternalUpdate.value = true
  if (fileList.value.length === 0) {
    emit('update:modelValue', null)
  } else {
    emit('update:modelValue', JSON.stringify(fileList.value.map((f) => f.id)))
  }
}

const formatFileSize = (bytes: number): string => {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}
</script>

<style scoped lang="scss">
.file-upload {
  width: 100%;
}

.file-upload__progress {
  margin-top: 6px;
  padding: 8px 12px;
  border-radius: 8px;
  color: $primary;
  background: rgba($primary, 0.08);
}

.file-upload__status {
  margin-top: 4px;
  overflow: hidden;
  color: #6f7890;
  font-size: 12px;
  line-height: 1.4;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-upload__percent {
  min-width: 42px;
  color: $primary;
  font-size: 12px;
  font-weight: 700;
  text-align: right;
}
</style>
