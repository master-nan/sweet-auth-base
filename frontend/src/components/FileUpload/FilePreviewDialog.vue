<template>
  <q-dialog v-model="visible" maximized transition-show="fade" transition-hide="fade">
    <q-card class="file-preview-dialog">
      <q-card-section class="row items-center q-px-lg q-py-md">
        <q-icon name="visibility" color="primary" size="28px" class="q-mr-sm" />
        <div class="col">
          <div class="text-h6 text-weight-bold ellipsis">
            {{ currentFile?.file_name || t('ui.filePreview') }}
          </div>
          <div class="text-caption text-grey-7">
            {{ currentFile ? formatFileSize(currentFile.file_size) : '' }}
          </div>
        </div>
        <q-btn
          v-if="downloadUrl"
          flat
          color="primary"
          icon="download"
          :label="t('ui.download')"
          @click="openDownload"
        />
        <q-btn flat round dense icon="close" v-close-popup />
      </q-card-section>

      <q-separator />

      <q-card-section class="file-preview-dialog__body">
        <q-inner-loading :showing="loading">
          <q-spinner color="primary" size="42px" />
        </q-inner-loading>

        <div
          v-if="previewUrl && currentFile && (designPreviewDisabled || viewerLoadError)"
          :key="previewKey"
          class="file-preview-dialog__unsupported"
        >
          <q-icon name="insert_drive_file" size="56px" color="grey-5" />
          <div class="text-subtitle2">{{ unsupportedPreviewMessage }}</div>
          <div class="text-caption text-grey-7">
            {{ t('ui.youCanViewItInLocalApplicationsByClickingOn') }}
          </div>
        </div>

        <component
          :is="OpenFileViewerComponent"
          v-else-if="previewUrl && currentFile && OpenFileViewerComponent"
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

        <div v-else-if="previewUrl && currentFile && !loading" class="file-preview-dialog__empty">
          <template v-if="viewerLoading">
            <q-spinner color="primary" size="42px" />
            <div>{{ t('ui.loadingFilePreviewComponent') }}</div>
          </template>
          <template v-else>
            <q-icon name="insert_drive_file" size="56px" color="grey-5" />
            <div>{{ t('ui.noPreviewableFiles') }}</div>
          </template>
        </div>

        <div v-else-if="!loading" class="file-preview-dialog__empty">
          <q-icon name="insert_drive_file" size="56px" color="grey-5" />
          <div>{{ t('ui.noPreviewableFiles') }}</div>
        </div>
      </q-card-section>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { computed, markRaw, ref, shallowRef, type Component } from 'vue'
import { useQuasar } from 'quasar'
import type { PreviewPlugin, PreviewToolbarOptions } from '@open-file-viewer/core'
import '@open-file-viewer/core/style.css'
import { useFileApi, type FileBusinessContext, type FileInfo } from '@/api/services/file'

const { t } = useI18n({ useScope: 'global' })

const $q = useQuasar()
const fileApi = useFileApi()

const visible = ref(false)
const loading = ref(false)
const currentFile = ref<FileInfo | null>(null)
const previewUrl = ref('')
const downloadUrl = ref('')
const previewKey = ref(0)
const viewerLoading = ref(false)
const viewerLoadError = ref(false)
const OpenFileViewerComponent = shallowRef<Component | null>(null)
const viewerPlugins = shallowRef<PreviewPlugin[]>([])

const toolbarOptions: PreviewToolbarOptions = {
  zoom: true,
  rotate: true,
  download: true,
  fullscreen: true,
  print: true,
  search: true,
  labels: {
    get previous() {
      return t('ui.previous')
    },
    get next() {
      return t('ui.next')
    },
    get queue() {
      return t('ui.documentation')
    },
    get 'zoom-out'() {
      return t('ui.zoomOut')
    },
    get 'zoom-in'() {
      return t('ui.zoomIn')
    },
    get 'zoom-reset'() {
      return t('ui.reset')
    },
    get 'rotate-right'() {
      return t('ui.rotate')
    },
    get download() {
      return t('ui.download')
    },
    get fullscreen() {
      return t('ui.fullscreen')
    },
    get print() {
      return t('ui.print')
    },
    get search() {
      return t('ui.search')
    },
  },
  titles: {
    get download() {
      return t('ui.downloadFile')
    },
    get fullscreen() {
      return t('ui.fullScreenPreview')
    },
    get print() {
      return t('ui.print')
    },
    get search() {
      return t('ui.searchForContents')
    },
  },
}

const extension = computed(() => {
  const file = currentFile.value
  const raw = file?.file_ext || file?.file_name?.split('.').pop() || ''
  return raw.replace(/^\./, '').toLowerCase()
})

const mimeType = computed(() => String(currentFile.value?.file_type || '').toLowerCase())

const designPreviewDisabled = computed(() =>
  isDesignPreviewDisabled(extension.value, mimeType.value),
)

const unsupportedPreviewMessage = computed(() => {
  const ext = extension.value
  if (viewerLoadError.value) return t('ui.filePreviewComponentLoadedFailed')
  if (designExtensions.has(ext))
    return t('ui.filesAreNotSupportedForOnlinePreview', {
      value1: ext.toUpperCase() || t('ui.design'),
    })
  return t('ui.theOnlinePreviewIsNotSupportedForTheCurrentFile')
})

const designExtensions = new Set(['psd', 'psb', 'abr', 'ai', 'eps'])

const open = async (file: FileInfo, context?: FileBusinessContext) => {
  if (!file.file_uuid) {
    $q.notify({
      type: 'warning',
      position: 'top-right',
      get message() {
        return t('ui.fileAccessIdentifierMissing')
      },
    })
    return
  }

  currentFile.value = file
  previewUrl.value = ''
  downloadUrl.value = ''
  visible.value = true
  loading.value = true
  viewerLoadError.value = false

  try {
    const [previewRes, downloadRes] = await Promise.all([
      fileApi.getFilePreviewAccessUrl(file.file_uuid, 900, context),
      fileApi.getFileDownloadAccessUrl(file.file_uuid, 900, context),
    ])
    if (!previewRes.success || !previewRes.data?.url) {
      throw new Error(previewRes.message || t('ui.failedToGetFilePreviewUrl'))
    }
    previewUrl.value = previewRes.data.url
    downloadUrl.value = downloadRes.success && downloadRes.data?.url ? downloadRes.data.url : ''
    if (!isDesignPreviewDisabled(extension.value, mimeType.value)) {
      await ensureOpenFileViewer()
    }
    previewKey.value += 1
  } catch (error: any) {
    $q.notify({
      type: 'negative',
      position: 'top-right',
      message: error?.message || t('ui.failedToGetFilePreviewUrl'),
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
    get message() {
      return t('ui.theCurrentFileDoesNotSupportOnlinePreviewsAndCan')
    },
  })
}

const handlePreviewError = (error: Error) => {
  $q.notify({
    type: 'negative',
    position: 'top-right',
    message: error?.message || t('ui.filePreviewFailed'),
  })
}

const ensureOpenFileViewer = async () => {
  if (OpenFileViewerComponent.value && viewerPlugins.value.length > 0) return
  viewerLoading.value = true
  try {
    const [viewerModule, coreModule, pdfWorkerModule] = await Promise.all([
      import('@open-file-viewer/vue'),
      import('@open-file-viewer/core'),
      import('pdfjs-dist/build/pdf.worker.mjs?url'),
    ])
    const pdfWorkerSrc = String(pdfWorkerModule.default)
    OpenFileViewerComponent.value = markRaw(viewerModule.OpenFileViewer)
    viewerPlugins.value = markRaw([
      coreModule.imagePlugin(),
      coreModule.videoPlugin(),
      coreModule.audioPlugin(),
      coreModule.textPlugin(),
      coreModule.pdfPlugin({ workerSrc: pdfWorkerSrc, useFetchData: true }),
      coreModule.officePlugin({ pdf: { workerSrc: pdfWorkerSrc, useFetchData: true } }),
      coreModule.fallbackPlugin(),
    ])
  } catch (error) {
    viewerLoadError.value = true
    throw error
  } finally {
    viewerLoading.value = false
  }
}

function isDesignPreviewDisabled(ext: string, mime: string) {
  return designExtensions.has(ext) || mime.includes('photoshop')
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

.file-preview-dialog__unsupported {
  display: flex;
  height: 100%;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: $grey-7;
  gap: 8px;
  text-align: center;
}
</style>
