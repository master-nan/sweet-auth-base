<template>
  <div class="file-display" :class="{ 'file-display--dense': dense }">
    <q-skeleton v-if="loading" type="QChip" width="120px" />

    <template v-else-if="files.length">
      <q-chip
        v-for="file in visibleFiles"
        :key="file.id"
        dense
        square
        outline
        color="primary"
        class="file-display__chip"
      >
        <q-icon name="attach_file" size="16px" class="q-mr-xs" />
        <span class="ellipsis file-display__name">{{ file.file_name || `文件 #${file.id}` }}</span>
        <q-tooltip
          >{{ file.file_name || `文件 #${file.id}`
          }}{{ file.file_size ? ` · ${formatFileSize(file.file_size)}` : '' }}</q-tooltip
        >

        <q-btn
          flat
          dense
          round
          size="xs"
          icon="visibility"
          class="q-ml-xs"
          @click.stop="openFile(file, 'preview')"
        >
          <q-tooltip>预览</q-tooltip>
        </q-btn>
        <q-btn flat dense round size="xs" icon="download" @click.stop="openFile(file, 'download')">
          <q-tooltip>下载</q-tooltip>
        </q-btn>
      </q-chip>

      <q-chip v-if="hiddenCount > 0" dense square color="grey-2" text-color="grey-8">
        +{{ hiddenCount }}
      </q-chip>
    </template>

    <span v-else class="text-grey-6">{{ emptyText }}</span>

    <file-preview-dialog ref="previewDialogRef" />
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, ref, watch } from 'vue'
import { useQuasar } from 'quasar'
import {
  useFileApi,
  type FileAccessMode,
  type FileBusinessContext,
  type FileInfo,
} from 'src/api/services/file'
import { parseFileIds } from 'src/utils/file-value'

type FilePreviewDialogExpose = {
  open: (file: FileInfo, context?: FileBusinessContext) => void | Promise<void>
}

const FilePreviewDialog = defineAsyncComponent(
  () => import('src/components/FileUpload/FilePreviewDialog.vue'),
)

interface FileDisplayProps {
  modelValue?: unknown
  dense?: boolean
  maxVisible?: number
  emptyText?: string
  tableCode?: string
  recordId?: number | string
  menuId?: number
  accessAction?: FileBusinessContext['action']
}

const props = withDefaults(defineProps<FileDisplayProps>(), {
  dense: false,
  maxVisible: 3,
  emptyText: '-',
  tableCode: '',
  recordId: 0,
  menuId: 0,
  accessAction: 'detail',
})

const $q = useQuasar()
const fileApi = useFileApi()
const loading = ref(false)
const files = ref<FileInfo[]>([])
const fileInfoCache = new Map<number, FileInfo>()
const previewDialogRef = ref<FilePreviewDialogExpose | null>(null)

const visibleFiles = computed(() => files.value.slice(0, props.maxVisible))
const hiddenCount = computed(() => Math.max(files.value.length - visibleFiles.value.length, 0))

async function loadFiles() {
  const ids = parseFileIds(props.modelValue)
  if (!ids.length) {
    files.value = []
    return
  }

  loading.value = true
  try {
    const result = await Promise.all(
      ids.map(async (id) => {
        const cached = fileInfoCache.get(id)
        if (cached) return cached
        const response = await fileApi.getFileById(id)
        if (response.success && response.data) {
          fileInfoCache.set(id, response.data)
        }
        return response.success ? response.data : null
      }),
    )
    files.value = result.filter((file): file is FileInfo => !!file)
  } finally {
    loading.value = false
  }
}

async function openFile(file: FileInfo, mode: FileAccessMode) {
  if (!file.file_uuid) {
    $q.notify({ type: 'warning', position: 'top-right', message: '文件缺少访问标识' })
    return
  }

  if (mode === 'preview') {
    previewDialogRef.value?.open(file, fileBusinessContext.value)
    return
  }

  const response =
    mode === 'download'
      ? await fileApi.getFileDownloadAccessUrl(file.file_uuid, 900, fileBusinessContext.value)
      : await fileApi.getFilePreviewAccessUrl(file.file_uuid, 900, fileBusinessContext.value)
  if (!response.success || !response.data?.url) {
    $q.notify({
      type: 'negative',
      position: 'top-right',
      message: response.message || '获取文件访问地址失败',
    })
    return
  }
  window.open(response.data.url, '_blank', 'noopener,noreferrer')
}

const fileBusinessContext = computed<FileBusinessContext | undefined>(() => {
  if (!props.tableCode || !props.recordId) return undefined
  return {
    table_code: props.tableCode,
    record_id: props.recordId,
    menu_id: props.menuId || 0,
    action: props.accessAction || 'detail',
  }
})

function formatFileSize(size: number) {
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

watch(() => props.modelValue, loadFiles, { immediate: true })
</script>

<style scoped lang="scss">
.file-display {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  align-items: center;
}

.file-display__chip {
  margin: 0;
  max-width: 220px;
}

.file-display__name {
  max-width: 120px;
}

.file-display--dense {
  justify-content: center;

  .file-display__chip {
    max-width: 180px;
  }
}
</style>
