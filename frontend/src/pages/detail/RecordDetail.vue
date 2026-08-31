<template>
  <organization-sync-batch-detail
    v-if="isOrganizationSyncBatch && canLoadRecordDetail"
    :record-id="recordId"
    @close="goBackToList"
    @title-change="syncPageChromeTitle"
  />

  <base-content v-else-if="!canLoadRecordDetail" scrollable class="q-pa-sm">
    <q-banner rounded class="bg-red-1 text-negative">{{ t('ui.noDetailsToView') }}</q-banner>
  </base-content>

  <detail-page-shell
    v-else
    :title="pageTitle"
    :icon="sourceIcon"
    :loading="loading"
    :error="loadError"
    retryable
    @retry="loadDetail"
  >
    <template #subtitle>
      {{ sourceLabel }}
      <q-chip v-if="isAudit" dense square color="primary" text-color="white">{{
        tableCode
      }}</q-chip>
      <span v-if="isAudit && recordId">#{{ recordId }}</span>
    </template>
    <template #actions>
      <q-btn
        v-for="button in detailTopButtons"
        :key="button.id || button.code"
        v-bind="menuButtonDisplayProps(button)"
        unelevated
        :color="button.color || 'primary'"
        :loading="executingButtonCode === button.code"
        :disable="isDetailButtonDisabled(button)"
        @click="handleDetailButtonClick(button)"
      />
      <q-btn
        flat
        color="primary"
        icon="arrow_back"
        :label="t('ui.backToList')"
        @click="goBackToList"
      />
      <q-btn
        outline
        color="primary"
        icon="refresh"
        :label="t('ui.refresh')"
        :loading="loading"
        @click="loadDetail"
      />
    </template>

    <dynamic-form-dialog
      v-model="showParamsDialog"
      :edit-data="null"
      :title="paramsDialogTitle"
      :fields="paramsFields"
      :menu-id="resolveDetailMenuId()"
      :table-code="tableCode"
      :submit-btn-text="t('ui.implementation')"
      @submit="handleParamsSubmit"
    />

    <template v-if="record">
      <section class="detail-page-section">
        <div class="detail-page-section__head">
          <div>
            <h3>{{ t('ui.basicInfo') }}</h3>
          </div>
        </div>

        <div class="record-detail-field-grid">
          <article
            v-for="field in compactFields"
            :key="field.key"
            class="record-detail-field"
            :class="{
              'record-detail-field--wide': field.span === 2 || field.wide,
              'record-detail-field--full': (field.span || 1) >= 4,
            }"
          >
            <div class="record-detail-field-label">
              <span>{{ field.label }}</span>
              <q-chip v-if="field.meta" dense square>{{ field.meta }}</q-chip>
            </div>
            <div class="record-detail-field-value">
              <q-chip
                v-if="field.kind === 'boolean'"
                dense
                square
                :color="field.rawValue ? 'positive' : 'grey-5'"
                text-color="white"
              >
                {{ field.rawValue ? t('ui.yes') : t('ui.no') }}
              </q-chip>
              <file-display
                v-else-if="field.kind === 'file'"
                :model-value="field.rawValue"
                :table-code="tableCode"
                :record-id="recordId"
                :menu-id="resolveDetailMenuId()"
                access-action="detail"
              />
              <div
                v-else-if="field.kind === 'rich-text'"
                class="record-detail-rich-text"
                v-html="richTextHtmlMap[field.key] || field.value"
              />
              <code v-else-if="field.kind === 'code'">{{ field.value }}</code>
              <span v-else>{{ field.value }}</span>
            </div>
          </article>
        </div>
      </section>

      <section v-if="longSections.length" class="detail-page-section">
        <div class="detail-page-section__head">
          <div>
            <h3>{{ longPanelTitle }}</h3>
          </div>
        </div>
        <q-list bordered separator class="record-detail-long-list">
          <q-expansion-item
            v-for="section in longSections"
            :key="section.key"
            :label="section.label"
            :caption="section.caption"
            default-opened
          >
            <pre class="record-detail-pre">{{ section.value }}</pre>
          </q-expansion-item>
        </q-list>
      </section>

      <section
        v-if="detailBottomButtons.length"
        class="detail-page-section record-detail-action-panel"
      >
        <div class="detail-page-section__head">
          <div>
            <h3>{{ t('ui.detailActions') }}</h3>
          </div>
        </div>
        <div class="record-detail-action-row">
          <q-btn
            v-for="button in detailBottomButtons"
            :key="button.id || button.code"
            v-bind="menuButtonDisplayProps(button)"
            unelevated
            :color="button.color || 'primary'"
            :loading="executingButtonCode === button.code"
            :disable="isDetailButtonDisabled(button)"
            @click="handleDetailButtonClick(button)"
          />
        </div>
      </section>
    </template>
  </detail-page-shell>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'record_detail_page' })

import { computed, onMounted, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useRoute, useRouter } from 'vue-router'
import { useQuasar } from 'quasar'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import DetailPageShell from 'src/components/Detail/DetailPageShell.vue'
import DynamicFormDialog from 'src/components/FormDialog/DynamicFormDialog.vue'
import FileDisplay from 'src/components/FileUpload/FileDisplay.vue'
import OrganizationSyncBatchDetail from 'src/pages/organization/sync-batch/Detail.vue'
import { useAccessLogApi, type AccessLog } from 'src/api/services/access-log'
import { useFileApi } from 'src/api/services/file'
import { useGeneralizationApi } from 'src/api/services/generalization'
import type { MenuButton } from 'src/api/services/sys-menu'
import { type RuntimeTableMetadata, type TableField } from 'src/api/services/sys-table'
import { useConfirmDialog } from 'src/composables/confirm-dialog'
import { useRuntimeTableMetadata } from 'src/composables/runtime-table-metadata'
import { useDictStore } from 'src/stores/dict'
import { useBreadcrumbsStore } from 'src/stores/breadcrumbs'
import { useTagViewStore } from 'src/stores/tagView'
import { useUserStore } from 'src/stores/user'
import { useLoadingStore } from 'src/stores/loading'
import {
  SysMenuButtonPosition,
  SysTableFieldInputType,
  SysTableFieldType,
  SysTableFieldTypeMap,
} from 'src/types/enum'
import type { RouteData } from 'src/types'
import {
  findMenuByName,
  findMenuByTableCode,
  findMenuPathByTableCode,
  findMenuTrailById,
} from 'src/utils/menu-context'
import { hasButtonActionCapability, isAvailablePageButton } from 'src/utils/menu-button'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'
import {
  evaluateButtonDisabled,
  executeButtonAction,
  runAfterHooks,
  runBeforeHooks,
  type ButtonActionContext,
} from 'src/utils/button-handlers'
import {
  buildColumnFormat,
  buildRelationLookups,
  hydrateRelationLookups,
  type LookupMap,
} from 'src/utils/column-format'
import { hydrateRichTextFileUrls } from 'src/utils/rich-text-files'
import {
  getFieldDetailSpan,
  isDetailFieldVisible,
  normalizeFieldLabel,
} from 'src/utils/field-layout'
import { parseParamsSchema } from 'src/utils/params-schema'

const { t } = useI18n({ useScope: 'global' })

interface DetailField {
  key: string
  label: string
  value: string
  rawValue: unknown
  kind: 'text' | 'boolean' | 'code' | 'file' | 'rich-text'
  meta?: string
  wide?: boolean
  span?: number
}

interface LongSection {
  key: string
  label: string
  caption: string
  value: string
}

const route = useRoute()
const router = useRouter()
const $q = useQuasar()
const { confirmAction } = useConfirmDialog($q)
const accessLogApi = useAccessLogApi()
const fileApi = useFileApi()
const generalizationApi = useGeneralizationApi()
const dictStore = useDictStore()
const breadcrumbsStore = useBreadcrumbsStore()
const tagViewStore = useTagViewStore()
const userStore = useUserStore()
const loadingStore = useLoadingStore()
const { loading } = storeToRefs(loadingStore)

const loadError = ref('')
const record = ref<Record<string, any> | null>(null)
const table = ref<RuntimeTableMetadata | null>(null)
const tableFields = ref<TableField[]>([])
const relationLookups = ref<Record<string, LookupMap>>({})
const richTextHtmlMap = ref<Record<string, string>>({})
const executingButtonCode = ref('')
const showParamsDialog = ref(false)
const paramsFields = ref<TableField[]>([])
const pendingDetailButton = ref<MenuButton | null>(null)

const source = computed(() => String(route.params.source || 'generalization'))
const tableCode = computed(() => String(route.params.table_code || ''))
const recordId = computed(() => Number(route.params.id || 0))
const {
  metadata: runtimeMetadata,
  fields: runtimeFields,
  metadataError,
  loadMetadata,
} = useRuntimeTableMetadata(tableCode)

const isAudit = computed(() => source.value === 'audit' || tableCode.value === 'access_log')
const isOrganizationSyncBatch = computed(
  () => source.value === 'organization' && tableCode.value === 'org_sync_batch',
)
const sourceIcon = computed(() =>
  isAudit.value ? 'manage_search' : isOrganizationSyncBatch.value ? 'sync' : 'article',
)

const currentMenu = computed(() => {
  if (isAudit.value) return findMenuByName(userStore.menus, 'system_audit')
  return findMenuByTableCode(userStore.menus, tableCode.value)
})

const tableName = computed(() => {
  return (
    table.value?.table_name ||
    currentMenu.value?.title ||
    currentMenu.value?.name ||
    tableCode.value
  )
})

const sourceLabel = computed(() => tableName.value || tableCode.value)

const detailMenuButtons = computed(() =>
  (currentMenu.value?.menu_buttons || [])
    .filter(isAvailablePageButton)
    .slice()
    .sort((a, b) => (a.sequence || 0) - (b.sequence || 0)),
)

const canLoadRecordDetail = computed(() =>
  hasButtonActionCapability(detailMenuButtons.value, 'detail'),
)

const detailTopButtons = computed(() =>
  detailMenuButtons.value.filter((button) => button.position === SysMenuButtonPosition.DETAIL_TOP),
)

const detailBottomButtons = computed(() =>
  detailMenuButtons.value.filter(
    (button) => button.position === SysMenuButtonPosition.DETAIL_BOTTOM,
  ),
)

const paramsDialogTitle = computed(() => pendingDetailButton.value?.name || t('ui.detailActions'))

const pageTitle = computed(() => {
  const label = recordLabel.value
  return label && label !== '-'
    ? t('ui.detailsOf', { value1: tableName.value, label: label })
    : t('ui.namedDetailsTitle', { value1: tableName.value })
})

const displayFields = computed(() =>
  buildDisplayFields(record.value, tableFields.value, isAudit.value),
)

const compactFields = computed(() => displayFields.value.filter((field) => !isLongField(field)))

const longPanelTitle = computed(() =>
  isAudit.value ? t('ui.requestAndResponse') : t('ui.extension'),
)

const longSections = computed<LongSection[]>(() =>
  displayFields.value
    .filter((field) => isLongField(field))
    .map((field) => ({
      key: field.key,
      label: field.label,
      caption: isAudit.value ? field.meta || field.key : '',
      value: stringifyValue(field.rawValue),
    })),
)

const titleField = computed(() => {
  return (
    displayFields.value.find((field) =>
      /user_name|name|title|用户|名称|标题/i.test(field.key + field.label),
    ) || displayFields.value.find((field) => field.key !== 'id')
  )
})

const recordLabel = computed(() => {
  const value = titleField.value?.value
  return value && value !== '-' ? value : String(recordId.value || '')
})

const sourceListPath = computed(() => {
  if (isAudit.value) return '/admin/system/audit'
  return (
    findMenuPathByTableCode(userStore.menus, tableCode.value) ||
    `/admin/develop/generalization/${tableCode.value}`
  )
})

function syncPageChromeTitle(title = pageTitle.value) {
  if (!title) return
  tagViewStore.updateTagViewTitle(route.fullPath, title)
  breadcrumbsStore.setBreadcrumbItems(buildDetailBreadcrumbs(title))
}

function buildDetailBreadcrumbs(title: string): RouteData[] {
  const detailCrumb: RouteData = {
    title,
    fullPath: route.fullPath,
    name: route.name,
    icon: sourceIcon.value,
    keepAlive: false,
  }

  const menuId = currentMenu.value?.id || 0
  const menuTrail = findMenuTrailById(userStore.menus, menuId)
  const menuCrumbs = menuTrail.map<RouteData>(({ menu, fullPath }) => {
    const crumb: RouteData = {
      title: menu.title || menu.name || fullPath,
      fullPath,
      name: menu.name || fullPath,
    }
    if (menu.icon) {
      crumb.icon = menu.icon
    }
    return crumb
  })

  if (menuCrumbs.length) return [...menuCrumbs, detailCrumb]
  return [
    {
      title: sourceLabel.value,
      fullPath: sourceListPath.value,
      name: currentMenu.value?.name || sourceListPath.value,
      icon: currentMenu.value?.icon || sourceIcon.value,
    },
    detailCrumb,
  ]
}

function fieldTypeLabel(field?: TableField) {
  if (!field) return ''
  return SysTableFieldTypeMap[field.field_type] || ''
}

function formatByField(value: unknown, row: Record<string, any>, field?: TableField) {
  if (!field) return defaultFormat(value)
  const fmt = buildColumnFormat(field, {
    getDictLabel: dictStore.getDictLabel,
    relationLookups: relationLookups.value,
  })
  const nextValue = fmt ? fmt(value, row) : value
  return defaultFormat(nextValue)
}

function defaultFormat(value: unknown) {
  if (value === null || value === undefined || value === '') return '-'
  if (typeof value === 'boolean') return value ? t('ui.yes') : t('ui.no')
  if (typeof value === 'object') return stringifyValue(value)
  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'bigint') {
    return String(value)
  }
  return '-'
}

function stringifyValue(value: unknown) {
  if (value === null || value === undefined || value === '') return '-'
  if (typeof value === 'string') {
    const trimmed = value.trim()
    if (
      (trimmed.startsWith('{') && trimmed.endsWith('}')) ||
      (trimmed.startsWith('[') && trimmed.endsWith(']'))
    ) {
      try {
        return JSON.stringify(JSON.parse(trimmed), null, 2)
      } catch {
        return value
      }
    }
    return value
  }
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return '-'
  }
}

function isCodeLike(key: string, value: unknown) {
  if (typeof value !== 'string') return false
  return /code|path|url|method|ip|编码|路径/i.test(key)
}

function toDetailField(key: string, row: Record<string, any>, field?: TableField): DetailField {
  const rawValue = row[key]
  const label = normalizeFieldLabel(field?.field_name || auditFieldLabels[key] || key)
  const value = formatByField(rawValue, row, field)
  const isFile = field?.input_type === SysTableFieldInputType.FILE_PICKER
  const isRichText = field?.input_type === SysTableFieldInputType.RICH_TEXT
  return {
    key,
    label,
    value,
    rawValue,
    kind: isFile
      ? 'file'
      : isRichText
        ? 'rich-text'
        : typeof rawValue === 'boolean' || field?.field_type === SysTableFieldType.BOOLEAN
          ? 'boolean'
          : isCodeLike(key, rawValue)
            ? 'code'
            : 'text',
    meta: isAudit.value ? fieldTypeLabel(field) : '',
    wide: !field && !isFile && value.length > 42,
    span: field ? getFieldDetailSpan(field) : isRichText ? 4 : !isFile && value.length > 42 ? 2 : 1,
  }
}

function buildDisplayFields(
  row: Record<string, any> | null,
  fields: TableField[],
  includeTechnicalFields: boolean,
) {
  if (!row) return []
  const sortedFields = fields.slice().sort((a, b) => (a.sequence || 0) - (b.sequence || 0))
  const keys = new Set<string>()
  const result: DetailField[] = []

  sortedFields.forEach((field) => {
    const key = field.field_code
    if (!key || !(key in row) || (!includeTechnicalFields && !isDetailFieldVisible(field))) return
    keys.add(key)
    result.push(toDetailField(key, row, field))
  })

  if (!includeTechnicalFields) return result

  preferredSystemKeys.forEach((key) => {
    if (!(key in row) || keys.has(key)) return
    keys.add(key)
    result.push(toDetailField(key, row))
  })

  Object.keys(row).forEach((key) => {
    if (keys.has(key)) return
    result.push(toDetailField(key, row))
  })

  return result
}

function isLongField(field: DetailField) {
  if (field.kind === 'file' || field.kind === 'rich-text') return false
  const value = stringifyValue(field.rawValue)
  const key = field.key.toLowerCase()
  return (
    key === 'body' ||
    key === 'query' ||
    key === 'response' ||
    key.includes('json') ||
    field.meta === 'JSON' ||
    value.length > 180 ||
    value.includes('\n')
  )
}

async function hydrateDetailRichText() {
  if (!record.value) {
    richTextHtmlMap.value = {}
    return
  }

  const next: Record<string, string> = {}
  const richTextFields = tableFields.value.filter(
    (field) => field.input_type === SysTableFieldInputType.RICH_TEXT,
  )

  await Promise.all(
    richTextFields.map(async (field) => {
      const key = field.field_code
      const rawValue = record.value?.[key]
      if (typeof rawValue !== 'string' || rawValue.trim() === '') return

      next[key] = await hydrateRichTextFileUrls(rawValue, async (fileUuid, mode) => {
        const response = await fileApi.getFileAccessUrl(fileUuid, mode, 900, {
          table_code: tableCode.value,
          record_id: recordId.value,
          menu_id: resolveDetailMenuId(),
          action: 'detail',
        })
        return response.success ? response.data?.url : undefined
      })
    }),
  )

  richTextHtmlMap.value = next
}

async function loadAuditDetail() {
  const response = await accessLogApi.getAccessLogById(recordId.value)
  if (!response.success) {
    throw new Error(response.message || t('ui.failedToLoadAuditDetails'))
  }
  record.value = response.data as AccessLog
  table.value = {
    id: 0,
    table_name: t('ui.auditLogs'),
    table_code: 'access_log',
    table_type: 1 as any,
    master_detail_mode: 'auto' as any,
    form_open_mode: 'auto' as any,
    detail_open_mode: 'auto' as any,
    table_fields: [],
    table_relations: [],
  }
  tableFields.value = buildAuditFields()
}

async function loadGeneralizationDetail() {
  const loadedMetadata = await loadMetadata()
  if (!loadedMetadata || !runtimeMetadata.value) {
    throw new Error(metadataError.value || t('ui.failedToLoadTableMetadata'))
  }

  table.value = runtimeMetadata.value
  tableFields.value = runtimeFields.value
  const dictCodes = tableFields.value
    .map((field) => field.dict_code)
    .filter((code): code is string => !!code)
  const [, initialLookups] = await Promise.all([
    dictStore.loadDicts(dictCodes),
    buildRelationLookups(tableFields.value),
  ])
  relationLookups.value = initialLookups

  const response = await generalizationApi.getGeneralizationDetailByCode(
    tableCode.value,
    recordId.value,
    resolveDetailMenuId(),
  )
  if (!response.success) {
    throw new Error(response.message || t('ui.failedToLoadRecordDetails'))
  }
  const row = response.data || null
  if (!row) {
    throw new Error(t('ui.recordsDoNotExistOrAreNotAccessible'))
  }
  record.value = row
  relationLookups.value = await hydrateRelationLookups(
    tableFields.value,
    [row],
    relationLookups.value,
    resolveDetailMenuId(),
  )
}

function resolveDetailMenuId() {
  return currentMenu.value?.id || 0
}

function readRecordValue(path: string) {
  if (!record.value || !path) return undefined
  return path
    .replace(/^row\./, '')
    .split('.')
    .reduce<unknown>((current, key) => {
      if (current && typeof current === 'object') return (current as Record<string, unknown>)[key]
      return undefined
    }, record.value)
}

function isDetailButtonDisabled(button: MenuButton) {
  if (loading.value || executingButtonCode.value || button.is_disabled || !record.value) return true
  return evaluateButtonDisabled(button, {
    row: record.value,
    selection: record.value ? [record.value] : [],
    selectionCount: record.value ? 1 : 0,
    query: {},
    params: {},
  })
}

function interpolateDetailActionPath(path: string) {
  return path.replace(/:([A-Za-z_][A-Za-z0-9_]*)/g, (_match, key: string) => {
    const value =
      key === 'id'
        ? recordId.value
        : key === 'table_code' || key === 'tableCode' || key === 'code'
          ? tableCode.value
          : readRecordValue(key)
    const pathValue =
      typeof value === 'string' || typeof value === 'number' || typeof value === 'bigint'
        ? String(value)
        : ''
    return encodeURIComponent(pathValue)
  })
}

async function executeDetailApi(button: MenuButton, params?: Record<string, any>) {
  const method = (button.http_method || 'POST').toUpperCase()
  const payload = {
    table_code: tableCode.value,
    menu_id: resolveDetailMenuId(),
    id: recordId.value,
    row: record.value,
    params: params || {},
  }
  await generalizationApi.executeRuntimeAction({
    path: interpolateDetailActionPath(button.api_path),
    method,
    payload,
  })
}

async function executeDetailButton(button: MenuButton, params?: Record<string, any>) {
  if (!record.value) return
  executingButtonCode.value = button.code
  const actionName = button.event_action || 'custom'
  const ctx: ButtonActionContext = {
    table_code: tableCode.value,
    row: record.value,
    onRefresh: loadDetail,
    onNavigate: async (path) => {
      const target = interpolateDetailActionPath(button.api_path || path)
      if (target) await router.push(target)
    },
    onCustom: async () => {
      if (!button.api_path) {
        $q.notify({
          type: 'warning',
          position: 'top-right',
          get message() {
            return t('ui.theDetailsButtonDoesNotConfigureTheExecutionInterfaceOr')
          },
        })
        return
      }
      await executeDetailApi(button, params)
      await loadDetail()
    },
  }
  if (params) {
    ctx.params = params
  }

  try {
    const shouldProceed = await runBeforeHooks(button.before_hooks, ctx)
    if (!shouldProceed) {
      $q.notify({
        type: 'warning',
        position: 'top-right',
        get message() {
          return t('ui.buttonPrefixNotPassed')
        },
      })
      return
    }
    if (button.api_path && actionName !== 'navigate') {
      await executeDetailApi(button, params)
      await loadDetail()
    } else {
      await executeButtonAction(actionName, ctx)
    }
    await runAfterHooks(button.after_hooks, ctx)
  } finally {
    executingButtonCode.value = ''
  }
}

function runDetailButtonWithConfirm(button: MenuButton, params?: Record<string, any>) {
  const run = () => void executeDetailButton(button, params)
  if (button.confirm_text) {
    confirmAction({
      get title() {
        return t('ui.confirmOperation')
      },
      message: button.confirm_text,
      get okLabel() {
        return t('ui.confirm')
      },
    }).onOk(run)
    return
  }
  run()
}

function handleDetailButtonClick(button: MenuButton) {
  if (isDetailButtonDisabled(button)) return
  if (button.params_schema) {
    const fields = parseParamsSchema(button.params_schema)
    if (fields.length > 0) {
      pendingDetailButton.value = button
      paramsFields.value = fields
      showParamsDialog.value = true
      return
    }
  }
  runDetailButtonWithConfirm(button)
}

function handleParamsSubmit(formPayload: { data: Record<string, any> }) {
  const button = pendingDetailButton.value
  if (!button) return
  showParamsDialog.value = false
  runDetailButtonWithConfirm(button, formPayload.data)
  pendingDetailButton.value = null
  paramsFields.value = []
}

async function loadDetail() {
  if (!canLoadRecordDetail.value) return
  if (isOrganizationSyncBatch.value) return
  if (!recordId.value || !tableCode.value) {
    loadError.value = t('ui.detailsRouteMissingLogIdOrTableCode')
    return
  }
  loadError.value = ''
  richTextHtmlMap.value = {}
  try {
    if (isAudit.value) {
      await loadAuditDetail()
    } else {
      await loadGeneralizationDetail()
    }
    await hydrateDetailRichText()
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : t('ui.loadingDetailsFailed')
    record.value = null
  }
}

async function goBackToList() {
  if (isAudit.value) {
    await router.push('/admin/system/audit')
    return
  }
  await router.push(sourceListPath.value)
}

const auditFieldLabels: Record<string, string> = {
  id: 'ID',
  get gmt_create() {
    return t('ui.time')
  },
  get user_name() {
    return t('ui.user')
  },
  action: t('ui.action'),
  get resource_type() {
    return t('ui.resourceType')
  },
  resource_code: t('ui.resourceCode'),
  get resource_id() {
    return t('ui.resourceId')
  },
  method: t('ui.method'),
  url: t('ui.path'),
  get status_code() {
    return t('ui.statusCode')
  },
  get success() {
    return t('ui.result')
  },
  get duration_ms() {
    return t('ui.duration')
  },
  ip: 'IP',
  get locality() {
    return t('ui.ownership')
  },
  get request_id() {
    return t('ui.requestId')
  },
  get trace_id() {
    return t('ui.traceId')
  },
  get result() {
    return t('ui.auditResult')
  },
  get error_code() {
    return t('ui.errorCode')
  },
  get error_message() {
    return t('ui.safeErrorMessage')
  },
}

const preferredSystemKeys = [
  'id',
  'state',
  'gmt_create',
  'create_user',
  'create_name',
  'gmt_modify',
  'modify_user',
  'modify_name',
  'gmt_delete',
  'delete_user',
  'delete_name',
]

function auditField(
  name: string,
  code: string,
  type: SysTableFieldType,
  inputType = SysTableFieldInputType.INPUT,
): TableField {
  return {
    id: 0,
    table_id: 0,
    field_name: name,
    field_code: code,
    field_type: type,
    field_length: 0,
    field_decimal_length: 0,
    input_type: inputType,
    default_value: '',
    dict_code: '',
    is_primary_key: code === 'id',
    is_index: false,
    is_quick_search: false,
    is_advanced_search: false,
    is_sort: false,
    is_null: true,
    is_list_show: true,
    is_insert_show: false,
    is_update_show: false,
    sequence: 0,
    original_field_id: 0,
    binding: '',
  }
}

function buildAuditFields() {
  return [
    auditField('ID', 'id', SysTableFieldType.BIGINT),
    auditField(
      t('ui.time'),
      'gmt_create',
      SysTableFieldType.DATETIME,
      SysTableFieldInputType.DATETIME_PICKER,
    ),
    auditField(t('ui.user'), 'user_name', SysTableFieldType.VARCHAR),
    auditField(t('ui.action'), 'action', SysTableFieldType.VARCHAR),
    auditField(t('ui.resourceType'), 'resource_type', SysTableFieldType.VARCHAR),
    auditField(t('ui.resourceCode'), 'resource_code', SysTableFieldType.VARCHAR),
    auditField(t('ui.resourceId'), 'resource_id', SysTableFieldType.VARCHAR),
    auditField(t('ui.method'), 'method', SysTableFieldType.VARCHAR),
    auditField(t('ui.path'), 'url', SysTableFieldType.VARCHAR),
    auditField(t('ui.statusCode'), 'status_code', SysTableFieldType.INT),
    auditField(
      t('ui.result'),
      'success',
      SysTableFieldType.BOOLEAN,
      SysTableFieldInputType.BOOLEAN,
    ),
    auditField(t('ui.duration'), 'duration_ms', SysTableFieldType.BIGINT),
    auditField('IP', 'ip', SysTableFieldType.VARCHAR),
    auditField(t('ui.ownership'), 'locality', SysTableFieldType.VARCHAR),
    auditField(t('ui.requestId'), 'request_id', SysTableFieldType.VARCHAR),
    auditField(t('ui.traceId'), 'trace_id', SysTableFieldType.VARCHAR),
    auditField(t('ui.auditResult'), 'result', SysTableFieldType.VARCHAR),
    auditField(t('ui.errorCode'), 'error_code', SysTableFieldType.VARCHAR),
    auditField(
      t('ui.safeErrorMessage'),
      'error_message',
      SysTableFieldType.TEXT,
      SysTableFieldInputType.TEXTAREA,
    ),
  ]
}

onMounted(loadDetail)

watch(
  () => route.fullPath,
  () => {
    void loadDetail()
  },
)

watch(
  pageTitle,
  (title) => {
    syncPageChromeTitle(title)
  },
  { immediate: true },
)
</script>

<style scoped lang="scss">
.record-detail-field-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(290px, 1fr));
  gap: 12px;
}

.record-detail-field {
  min-width: 0;
  padding: 12px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-surface-muted);
}

.record-detail-field--wide {
  grid-column: span 2;
}

.record-detail-field--full {
  grid-column: 1 / -1;
}

.record-detail-field-label {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  color: var(--app-text-muted);
  font-size: 13px;
}

.record-detail-field-value {
  margin-top: 8px;
  color: var(--app-text-strong);
  font-size: 15px;
  font-weight: 600;
  overflow-wrap: anywhere;
}

.record-detail-field-value code {
  padding: 2px 6px;
  border-radius: 5px;
  color: var(--q-primary);
  background: var(--app-primary-soft);
  white-space: pre-wrap;
}

.record-detail-rich-text {
  min-height: 28px;
  color: var(--app-text-strong);
  font-weight: 400;
  line-height: 1.65;
  overflow-wrap: anywhere;
}

.record-detail-rich-text p {
  margin: 0 0 8px;
}

.record-detail-rich-text img {
  max-width: 100%;
  height: auto;
  border-radius: 6px;
  vertical-align: middle;
}

.record-detail-long-list {
  overflow: hidden;
  border-color: var(--app-border);
  border-radius: 8px;
  background: var(--app-surface-muted);
}

.record-detail-action-panel {
  padding-top: 16px;
}

.record-detail-action-row {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.record-detail-pre {
  min-height: 88px;
  max-height: 420px;
  margin: 0;
  padding: 14px;
  overflow: auto;
  border-top: 1px solid var(--app-border);
  color: var(--app-text-strong);
  background: var(--app-surface-muted);
  white-space: pre-wrap;
  word-break: break-word;
}

@media (max-width: 1180px) {
  .record-detail-field-grid {
    grid-template-columns: repeat(2, minmax(160px, 1fr));
  }
}

@media (max-width: 720px) {
  .record-detail-field-grid {
    grid-template-columns: 1fr;
  }

  .record-detail-field--wide {
    grid-column: auto;
  }
}
</style>
