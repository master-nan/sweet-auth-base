<template>
  <div class="rich-text-editor">
    <div v-if="label" class="rich-text-label q-mb-xs">{{ label }}</div>
    <div class="rich-text-wrapper" :class="{ 'rich-text-error': hasError }">
      <Toolbar
        :editor="editorRef"
        :defaultConfig="toolbarConfig"
        mode="default"
        class="rich-text-toolbar"
      />
      <Editor
        :defaultConfig="editorConfig"
        mode="default"
        :modelValue="editorHtml"
        @onCreated="handleCreated"
        @onChange="handleChange"
        class="rich-text-content"
        :style="{ minHeight: minHeight + 'px' }"
      />
    </div>
    <!-- 验证错误提示 -->
    <div v-if="errorMessage" class="rich-text-error-message text-negative q-mt-xs">
      {{ errorMessage }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { ref, shallowRef, onBeforeUnmount, watch, type PropType } from 'vue'
import { Editor, Toolbar } from '@wangeditor/editor-for-vue'
import type { IEditorConfig, IToolbarConfig, IDomEditor } from '@wangeditor/editor'
import { useFileApi, type FileAccessMode, type FileInfo } from '@/api/services/file'
import { hydrateRichTextFileUrls, serializeRichTextFileUrls } from '@/utils/rich-text-files'

const { t } = useI18n({ useScope: 'global' })

const props = defineProps({
  modelValue: {
    type: String as PropType<string | null>,
    default: '',
  },
  label: {
    type: String,
    default: '',
  },
  placeholder: {
    type: String,
    default: '',
  },
  minHeight: {
    type: Number,
    default: 300,
  },
  /** Quasar 风格的验证规则 */
  rules: {
    type: Array as PropType<Array<(val: any) => boolean | string>>,
    default: () => [],
  },
  disabled: {
    type: Boolean,
    default: false,
  },
  tableCode: {
    type: String,
    default: '',
  },
  menuId: {
    type: Number,
    default: 0,
  },
  rowId: {
    type: [Number, String] as PropType<number | string>,
    default: 0,
  },
  fieldCode: {
    type: String,
    default: '',
  },
})

const emit = defineEmits<{
  (e: 'update:modelValue', val: string): void
}>()

const fileApi = useFileApi()
const editorRef = shallowRef<IDomEditor>()
const editorHtml = ref('')
const hasError = ref(false)
const errorMessage = ref('')
const applyingExternalHtml = ref(false)
const hydrateVersion = ref(0)
const lastEmittedValue = ref<string | null>(null)
const fileAccessCache = new Map<string, { url: string; expiresAt: number }>()

// ─── 工具栏配置 ──────────────────────────────────
const toolbarConfig: Partial<IToolbarConfig> = {
  excludeKeys: [
    'group-video', // 暂不支持视频
  ],
}

// ─── 自定义图片上传 ──────────────────────────────
type InsertImageFn = (url: string, alt?: string, href?: string) => void

const customUploadImage = async (file: File, insertFn: InsertImageFn) => {
  try {
    const res = await fileApi.uploadFile(file)
    if (res.data?.file_uuid || res.data?.file_url) {
      const url = await resolveUploadedFileUrl(res.data, 'preview')
      if (url) {
        if (res.data.file_uuid && insertFileHtml('image', res.data, url)) {
          return
        }
        insertFn(url, res.data.file_name, '')
      }
    }
  } catch (err) {
    console.error('图片上传失败', err)
  }
}

// ─── 自定义附件上传 ──────────────────────────────
type InsertAttachmentFn = (fileName: string, url: string) => void

const customUploadAttachment = async (file: File, insertFn: InsertAttachmentFn) => {
  try {
    const res = await fileApi.uploadFile(file)
    if (res.data?.file_uuid || res.data?.file_url) {
      const url = await resolveUploadedFileUrl(res.data, 'download')
      if (url) {
        if (res.data.file_uuid && insertFileHtml('attachment', res.data, url)) {
          return
        }
        insertFn(res.data.file_name, url)
      }
    }
  } catch (err) {
    console.error('附件上传失败', err)
  }
}

// ─── 编辑器配置 ──────────────────────────────────
const editorConfig: Partial<IEditorConfig> = {
  placeholder: props.placeholder || t('ui.enterContent'),
  readOnly: props.disabled,
  MENU_CONF: {
    uploadImage: {
      customUpload: customUploadImage,
    },
    uploadAttachment: {
      customUpload: customUploadAttachment,
    },
  },
}

// ─── 事件处理 ──────────────────────────────────
const handleCreated = (editor: IDomEditor) => {
  editorRef.value = editor
  if (editorHtml.value && editor.getHtml() !== editorHtml.value) {
    editor.setHtml(editorHtml.value)
  }
}

const handleChange = (editor: IDomEditor) => {
  if (applyingExternalHtml.value) {
    return
  }
  const html = editor.getHtml()
  // wangEditor 空内容时返回 '<p><br></p>'
  const isEmpty = html === '<p><br></p>' || html === ''
  const val = isEmpty ? '' : serializeRichTextFileUrls(html)
  lastEmittedValue.value = val
  emit('update:modelValue', val)
  validate(val)
}

const hydrateEditorValue = async (value: string) => {
  const version = ++hydrateVersion.value
  const hydrated = await hydrateRichTextFileUrls(value, getFileAccessUrl)
  if (version !== hydrateVersion.value) {
    return
  }

  applyingExternalHtml.value = true
  editorHtml.value = hydrated
  const editor = editorRef.value
  if (editor && editor.getHtml() !== hydrated) {
    editor.setHtml(hydrated)
  }
  setTimeout(() => {
    if (version === hydrateVersion.value) {
      applyingExternalHtml.value = false
    }
  })
}

const getFileAccessUrl = async (uuid: string, mode: FileAccessMode) => {
  const cacheKey = `${mode}:${uuid}`
  const now = Math.floor(Date.now() / 1000)
  const cached = fileAccessCache.get(cacheKey)
  if (cached && cached.expiresAt - 60 > now) {
    return cached.url
  }

  const res = await fileApi.getFileAccessUrl(uuid, mode, 900)
  if (!res.data?.url) {
    return undefined
  }
  fileAccessCache.set(cacheKey, {
    url: res.data.url,
    expiresAt: res.data.expires_at,
  })
  return res.data.url
}

const resolveUploadedFileUrl = async (fileInfo: FileInfo, mode: FileAccessMode) => {
  if (!fileInfo.file_uuid) {
    return fileInfo.file_url
  }
  const url = (await getFileAccessUrl(fileInfo.file_uuid, mode)) || fileInfo.file_url
  return appendFileUuidHint(url, fileInfo.file_uuid)
}

const insertFileHtml = (type: 'image' | 'attachment', fileInfo: FileInfo, url: string) => {
  const editor = editorRef.value
  if (!editor) {
    return false
  }

  const uuid = escapeHtmlAttr(fileInfo.file_uuid)
  const name = escapeHtmlText(fileInfo.file_name || fileInfo.file_uuid)
  const safeUrl = escapeHtmlAttr(url)
  if (type === 'image') {
    editor.dangerouslyInsertHtml(
      `<p><img src="${safeUrl}" alt="${name}" data-file-uuid="${uuid}" data-file-mode="preview"/></p>`,
    )
  } else {
    editor.dangerouslyInsertHtml(
      `<p><a href="${safeUrl}" target="_blank" data-file-uuid="${uuid}" data-file-mode="download">${name}</a></p>`,
    )
  }
  return true
}

const escapeHtmlAttr = (value: string) =>
  String(value || '')
    .replace(/&/g, '&amp;')
    .replace(/"/g, '&quot;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')

const escapeHtmlText = (value: string) =>
  String(value || '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')

const appendFileUuidHint = (rawUrl: string, uuid: string) => {
  try {
    const url = new URL(rawUrl, window.location.origin)
    const hashParams = new URLSearchParams(url.hash.slice(1))
    hashParams.set('file_uuid', uuid)
    url.hash = hashParams.toString()
    if (url.origin === window.location.origin) {
      return url.pathname + url.search + url.hash
    }
    return url.toString()
  } catch {
    const connector = rawUrl.includes('#') ? '&' : '#'
    return `${rawUrl}${connector}file_uuid=${encodeURIComponent(uuid)}`
  }
}

// ─── 验证 ──────────────────────────────────
const validate = (val?: string) => {
  const value = val ?? props.modelValue ?? ''
  for (const rule of props.rules) {
    const result = rule(value)
    if (typeof result === 'string') {
      hasError.value = true
      errorMessage.value = result
      return false
    }
    if (result === false) {
      hasError.value = true
      errorMessage.value = t('ui.theFieldDoesNotMeetTheRequirements')
      return false
    }
  }
  hasError.value = false
  errorMessage.value = ''
  return true
}

// 暴露 validate 方法给父组件
defineExpose({ validate })

// ─── 禁用状态变化 ──────────────────────────────
watch(
  () => props.disabled,
  (val) => {
    if (editorRef.value) {
      val ? editorRef.value.disable() : editorRef.value.enable()
    }
  },
)

watch(
  () => props.modelValue ?? '',
  (val) => {
    if (val === lastEmittedValue.value) {
      return
    }
    void hydrateEditorValue(val)
  },
  { immediate: true },
)

// ─── 销毁编辑器 ──────────────────────────────
onBeforeUnmount(() => {
  const editor = editorRef.value
  if (editor) {
    editor.destroy()
  }
})
</script>

<style src="@wangeditor/editor/dist/css/style.css"></style>

<style scoped lang="scss">
.rich-text-editor {
  width: 100%;
}

.rich-text-label {
  font-size: 0.85rem;
  color: rgba(0, 0, 0, 0.6);
}

.rich-text-wrapper {
  border: 1px solid rgba(0, 0, 0, 0.24);
  border-radius: 4px;
  overflow: hidden;
  transition: border-color 0.2s;

  &:hover {
    border-color: rgba(0, 0, 0, 0.87);
  }

  &:focus-within {
    border-color: var(--q-primary);
    border-width: 2px;
  }

  &.rich-text-error {
    border-color: var(--q-negative);
  }
}

.rich-text-toolbar {
  border-bottom: 1px solid #e8e8e8;
}

.rich-text-content {
  overflow-y: auto;
  max-height: 600px;
}

.rich-text-error-message {
  font-size: 0.75rem;
  line-height: 1;
  padding: 0 12px;
}
</style>
