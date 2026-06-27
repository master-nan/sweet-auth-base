<template>
  <q-dialog v-model="visible" maximized transition-show="fade" transition-hide="fade">
    <q-card class="file-preview-dialog">
      <q-card-section class="row items-center q-px-lg q-py-md">
        <q-icon name="visibility" color="primary" size="28px" class="q-mr-sm" />
        <div class="col">
          <div class="text-h6 text-weight-bold ellipsis">{{ currentFile?.file_name || '文件预览' }}</div>
          <div class="text-caption text-grey-7">
            {{ currentFile ? formatFileSize(currentFile.file_size) : '' }}
          </div>
        </div>
        <q-btn
          v-if="downloadUrl"
          flat
          color="primary"
          icon="download"
          label="下载"
          @click="openDownload"
        />
        <q-btn flat round dense icon="close" v-close-popup />
      </q-card-section>

      <q-separator />

      <q-card-section class="file-preview-dialog__body">
        <q-inner-loading :showing="loading">
          <q-spinner color="primary" size="42px" />
        </q-inner-loading>

        <open-file-viewer
          v-if="previewUrl && currentFile"
          :key="previewKey"
          :file="previewUrl"
          :file-name="currentFile.file_name"
          :mime-type="currentFile.file_type"
          width="100%"
          height="100%"
          fit="contain"
          fallback="download"
          :theme="$q.dark.isActive ? 'dark' : 'light'"
          :toolbar="toolbarOptions"
          :plugins="viewerPlugins"
          @error="handlePreviewError"
          @unsupported="handlePreviewUnsupported"
        />

        <div v-else-if="!loading" class="file-preview-dialog__empty">
          <q-icon name="insert_drive_file" size="56px" color="grey-5" />
          <div>暂无可预览文件</div>
        </div>
      </q-card-section>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { markRaw, ref } from 'vue'
import { useQuasar } from 'quasar'
import { OpenFileViewer } from '@open-file-viewer/vue'
import {
  audioPlugin,
  fallbackPlugin,
  imagePlugin,
  officePlugin,
  pdfPlugin,
  type PreviewToolbarOptions,
  textPlugin,
  videoPlugin,
} from '@open-file-viewer/core'
import '@open-file-viewer/core/style.css'
import pdfWorkerSrc from 'pdfjs-dist/build/pdf.worker.mjs?url'
import { useFileApi, type FileInfo } from 'src/api/services/file'

const $q = useQuasar()
const fileApi = useFileApi()

const visible = ref(false)
const loading = ref(false)
const currentFile = ref<FileInfo | null>(null)
const previewUrl = ref('')
const downloadUrl = ref('')
const previewKey = ref(0)

const viewerPlugins = markRaw([
  imagePlugin(),
  videoPlugin(),
  audioPlugin(),
  textPlugin(),
  pdfPlugin({ workerSrc: pdfWorkerSrc, useFetchData: true }),
  officePlugin(),
  fallbackPlugin(),
])

const toolbarOptions: PreviewToolbarOptions = {
  zoom: true,
  rotate: true,
  download: true,
  fullscreen: true,
  print: true,
  search: true,
  labels: {
    previous: '上一个',
    next: '下一个',
    queue: '文件',
    'zoom-out': '缩小',
    'zoom-in': '放大',
    'zoom-reset': '重置',
    'rotate-right': '旋转',
    download: '下载',
    fullscreen: '全屏',
    print: '打印',
    search: '搜索',
  },
  titles: {
    download: '下载文件',
    fullscreen: '全屏预览',
    print: '打印',
    search: '搜索内容',
  },
}

const open = async (file: FileInfo) => {
  if (!file.file_uuid) {
    $q.notify({ type: 'warning', position: 'top-right', message: '文件缺少访问标识' })
    return
  }

  currentFile.value = file
  previewUrl.value = ''
  downloadUrl.value = ''
  visible.value = true
  loading.value = true

  try {
    const [previewRes, downloadRes] = await Promise.all([
      fileApi.getFilePreviewAccessUrl(file.file_uuid),
      fileApi.getFileDownloadAccessUrl(file.file_uuid),
    ])
    if (!previewRes.success || !previewRes.data?.url) {
      throw new Error(previewRes.message || '获取文件预览地址失败')
    }
    previewUrl.value = previewRes.data.url
    downloadUrl.value = downloadRes.success && downloadRes.data?.url ? downloadRes.data.url : ''
    previewKey.value += 1
  } catch (error: any) {
    $q.notify({
      type: 'negative',
      position: 'top-right',
      message: error?.message || '获取文件预览地址失败',
    })
  } finally {
    loading.value = false
  }
}

const openDownload = () => {
  if (!downloadUrl.value) return
  window.open(downloadUrl.value, '_blank', 'noopener,noreferrer')
}

const handlePreviewUnsupported = () => {
  $q.notify({
    type: 'warning',
    position: 'top-right',
    message: '当前文件暂不支持在线预览，可点击下载查看',
  })
}

const handlePreviewError = (error: Error) => {
  $q.notify({
    type: 'negative',
    position: 'top-right',
    message: error.message || '文件预览失败',
  })
}

const formatFileSize = (size: number) => {
  if (!size) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let value = size
  let unitIndex = 0
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024
    unitIndex += 1
  }
  return `${value.toFixed(value >= 10 || unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`
}

defineExpose({ open })
</script>

<style scoped lang="scss">
.file-preview-dialog {
  display: flex;
  flex-direction: column;
  height: 100vh;
}

.file-preview-dialog__body {
  position: relative;
  flex: 1;
  min-height: 0;
  padding: 0;
}

.file-preview-dialog__empty {
  display: flex;
  height: 100%;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: $grey-6;
  gap: 12px;
}
</style>
