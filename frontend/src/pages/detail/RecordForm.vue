<template>
  <base-content class="q-pa-sm record-form-page">
    <dynamic-form-dialog
      v-model="formVisible"
      embedded
      :edit-data="editData"
      :title="pageTitle"
      :fields="tableFields"
      :menu-id="resolvedMenuId"
      :table-code="tableCode"
      :submit-btn-text="submitText"
      @submit="handleSubmit"
    />

    <q-inner-loading :showing="loading">
      <q-spinner color="primary" size="42px" />
    </q-inner-loading>

    <q-banner v-if="loadError" rounded class="record-form-error">
      <template #avatar>
        <q-icon name="error_outline" color="negative" />
      </template>
      {{ loadError }}
    </q-banner>
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'record_form_page' })

import { computed, onMounted, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import cloneDeep from 'lodash/cloneDeep'
import { useRoute, useRouter } from 'vue-router'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import DynamicFormDialog from 'src/components/FormDialog/DynamicFormDialog.vue'
import { useGeneralizationApi } from 'src/api/services/generalization'
import {
  useTableApi,
  type RuntimeTableMetadata,
  type TableField,
} from 'src/api/services/sys-table'
import type { RouteData } from 'src/types'
import { useUserStore } from 'src/stores/user'
import { useTagViewStore } from 'src/stores/tagView'
import { useBreadcrumbsStore } from 'src/stores/breadcrumbs'
import { useLoadingStore } from 'src/stores/loading'
import {
  findMenuByTableCode,
  findMenuPathByTableCode,
  findMenuTrailById,
} from 'src/utils/menu-context'

type FormMode = 'create' | 'edit' | 'copy'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const tagViewStore = useTagViewStore()
const breadcrumbsStore = useBreadcrumbsStore()
const loadingStore = useLoadingStore()
const { loading } = storeToRefs(loadingStore)
const tableApi = useTableApi()
const generalizationApi = useGeneralizationApi()

const loadError = ref('')
const formVisible = ref(true)
const table = ref<RuntimeTableMetadata | null>(null)
const tableFields = ref<TableField[]>([])
const editData = ref<Record<string, any> | null>(null)

const mode = computed<FormMode>(() => {
  const value = String(route.params.mode || 'create')
  return value === 'edit' || value === 'copy' ? value : 'create'
})
const tableCode = computed(() => String(route.params.table_code || ''))
const recordId = computed(() => Number(route.params.id || 0))
const resolvedMenuId = computed(() => {
  return findMenuByTableCode(userStore.menus, tableCode.value)?.id || 0
})
const currentMenu = computed(() => findMenuByTableCode(userStore.menus, tableCode.value))

const pageTitle = computed(() => {
  const tableName = table.value?.table_name || tableCode.value || '记录'
  if (mode.value === 'edit') return `编辑${tableName}`
  if (mode.value === 'copy') return `复制${tableName}`
  return `新增${tableName}`
})

const submitText = computed(() => (mode.value === 'edit' ? '保存' : '创建'))
const sourceListPath = computed(() => {
  return (
    findMenuPathByTableCode(userStore.menus, tableCode.value) ||
    `/admin/develop/generalization/${tableCode.value}`
  )
})

watch(formVisible, (value) => {
  if (!value) {
    goBackToList()
  }
})

onMounted(() => {
  void loadForm()
})

watch(
  () => [mode.value, tableCode.value, recordId.value],
  () => {
    void loadForm()
  },
)

watch(
  pageTitle,
  (title) => {
    syncPageChromeTitle(title)
  },
  { immediate: true },
)

function syncPageChromeTitle(title = pageTitle.value) {
  tagViewStore.updateTagViewTitle(route.fullPath, title)
  breadcrumbsStore.setBreadcrumbItems(buildFormBreadcrumbs(title))
}

function buildFormBreadcrumbs(title: string): RouteData[] {
  const formCrumb: RouteData = {
    title,
    fullPath: route.fullPath,
    name: route.name,
    icon: 'edit_note',
    keepAlive: false,
  }
  const menuTrail = findMenuTrailById(userStore.menus, currentMenu.value?.id || 0)
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
  if (menuCrumbs.length) return [...menuCrumbs, formCrumb]
  return [
    {
      title: table.value?.table_name || tableCode.value || '记录',
      fullPath: sourceListPath.value,
      name: currentMenu.value?.name || sourceListPath.value,
      icon: currentMenu.value?.icon || 'dynamic_form',
    },
    formCrumb,
  ]
}

async function loadForm() {
  if (!tableCode.value) {
    loadError.value = '缺少表编码，无法打开表单'
    return
  }

  loadError.value = ''
  editData.value = null
  formVisible.value = true

  try {
    const tableRes = await tableApi.queryRuntimeTableByCode(tableCode.value)
    if (!tableRes.success || !tableRes.data) {
      throw new Error(tableRes.message || '表元数据不存在')
    }
    table.value = tableRes.data
    tableFields.value = tableRes.data.table_fields || []
    syncPageChromeTitle()

    if (mode.value === 'edit' || mode.value === 'copy') {
      if (!recordId.value) {
        throw new Error('缺少记录ID，无法加载表单数据')
      }
      const row = await loadRecord(recordId.value)
      editData.value = mode.value === 'copy' ? stripRecordIdentity(row) : row
    }
  } catch (error) {
    const message = error instanceof Error ? error.message : '表单加载失败'
    loadError.value = message
  }
}

async function loadRecord(id: number) {
  const res = await generalizationApi.getGeneralizationDetailByCode(
    tableCode.value,
    id,
    resolvedMenuId.value,
  )
  if (!res.success || !res.data) {
    throw new Error(res.message || '记录不存在')
  }
  return res.data
}

function stripRecordIdentity(row: Record<string, any>) {
  const copied = cloneDeep(row)
  ;[
    'id',
    'gmt_create',
    'gmt_modify',
    'gmt_delete',
    'create_user',
    'modify_user',
    'delete_user',
    'createUser',
    'modifyUser',
    'deleteUser',
    'createName',
    'modifyName',
    'deleteName',
  ].forEach((key) => delete copied[key])
  return copied
}

async function handleSubmit(formPayload: { data: Record<string, any>; isEdit: boolean; id?: number }) {
  if (!tableCode.value) return
  const result =
    mode.value === 'edit' && recordId.value
      ? await generalizationApi.updateGeneralization({
          id: recordId.value,
          table_code: tableCode.value,
          data: formPayload.data,
          menu_id: resolvedMenuId.value,
        })
      : await generalizationApi.createGeneralization({
          table_code: tableCode.value,
          data: formPayload.data,
          menu_id: resolvedMenuId.value,
        })

  if (result.success) {
    goBackToList()
  }
}

function goBackToList() {
  const from =
    typeof window !== 'undefined' && typeof window.history.state?.recordFormFrom === 'string'
      ? window.history.state.recordFormFrom
      : ''
  if (from) {
    void router.push(String(from))
    return
  }
  void router.push(sourceListPath.value)
}
</script>

<style scoped lang="scss">
.record-form-page {
  display: flex;
  flex-direction: column;
  min-height: 0 !important;
  overflow: hidden;
  background: #f6f7fb;
}

.record-form-page :deep(.form-dialog-shell--embedded) {
  flex: 1 1 auto;
  width: 100%;
  min-height: 0;
  max-height: none;
}

.record-form-error {
  margin-top: 10px;
  color: $negative;
  background: rgba($negative, 0.08);
}
</style>
