<template>
  <base-content class="q-pa-sm">
    <q-table
      class="fit sticky-header-table"
      flat bordered separator="cell" row-key="id"
      :rows="rows" :columns="columns" :loading="loading"
      hide-pagination
    >
      <template #top>
        <div class="full-width">
          <q-tabs v-model="activeType" dense align="left" active-color="primary" indicator-color="primary" @update:model-value="reloadFirstPage">
            <q-tab :name="QuerySchemeType.PERSONAL" label="我的方案" />
            <q-tab :name="QuerySchemeType.PUBLIC" label="公共方案" />
            <q-tab :name="QuerySchemeType.ROLE" label="角色方案" />
            <q-tab :name="QuerySchemeType.PAGE_DEFAULT" label="页面默认" />
          </q-tabs>
          <q-separator class="q-mb-sm" />
          <standard-table-toolbar :refreshing="loading" @refresh="fetchData">
            <template #quick-search>
              <q-input v-model="nameFilter" dense outlined clearable debounce="300" label="方案名称" @keyup.enter="reloadFirstPage" />
              <q-select v-model="scopeFilter" dense outlined clearable emit-value map-options :options="scopeOptions" label="所属页面" style="min-width: 190px" @update:model-value="reloadFirstPage" />
              <q-btn color="primary" label="查询" @click="reloadFirstPage" />
            </template>
            <template #right-actions>
              <q-btn v-if="canManageShared && activeType !== QuerySchemeType.PERSONAL" color="primary" icon="add" label="新建方案" @click="openCreate" />
            </template>
          </standard-table-toolbar>
        </div>
      </template>

      <template #body-cell-name="props"><q-td :props="props"><div class="text-weight-medium">{{ props.row.name }}</div><div class="text-caption text-grey-7">{{ scopeLabel(props.row.scope_label) }}</div></q-td></template>
      <template #body-cell-type="props"><q-td :props="props"><status-chip color="primary" :label="typeLabel(props.row)" /></q-td></template>
      <template #body-cell-is_default="props"><q-td :props="props"><q-icon v-if="props.row.is_default" name="star" color="amber-7" size="sm"><q-tooltip>默认方案</q-tooltip></q-icon><span v-else>-</span></q-td></template>
      <template #body-cell-status="props"><q-td :props="props"><status-chip :label="statusLabel(props.row.status)" :color="statusColor(props.row.status)" /></q-td></template>
      <template #body-cell-enabled="props"><q-td :props="props"><status-chip :label="props.row.enabled ? '已启用' : '已停用'" :color="props.row.enabled ? 'positive' : 'grey'" /></q-td></template>
      <template #body-cell-actions="props">
        <q-td :props="props" class="q-gutter-xs no-wrap">
          <q-btn flat round dense icon="visibility" aria-label="查看方案详情" @click="openDetail(props.row)"><q-tooltip>详情</q-tooltip></q-btn>
          <q-btn flat round dense icon="play_arrow" color="primary" aria-label="使用方案" @click="useScheme(props.row)"><q-tooltip>使用</q-tooltip></q-btn>
          <q-btn v-if="props.row.type !== QuerySchemeType.PERSONAL" flat round dense icon="content_copy" aria-label="复制为我的方案" @click="copyScheme(props.row)"><q-tooltip>复制为我的方案</q-tooltip></q-btn>
          <q-btn v-if="canEdit(props.row)" flat round dense icon="edit" color="primary" aria-label="编辑方案" @click="openEdit(props.row)"><q-tooltip>编辑</q-tooltip></q-btn>
          <q-btn v-if="props.row.type === QuerySchemeType.PERSONAL" flat round dense icon="star" color="amber-8" aria-label="设置默认方案" @click="setPersonalDefault(props.row)"><q-tooltip>{{ props.row.is_default ? '取消默认' : '设为默认' }}</q-tooltip></q-btn>
          <q-btn v-if="canManageShared && props.row.type !== QuerySchemeType.PERSONAL" flat round dense :icon="props.row.enabled ? 'toggle_off' : 'toggle_on'" :color="props.row.enabled ? 'warning' : 'positive'" :aria-label="props.row.enabled ? '停用方案' : '启用方案'" @click="toggleEnabled(props.row)"><q-tooltip>{{ props.row.enabled ? '停用' : '启用' }}</q-tooltip></q-btn>
          <q-btn v-if="canEdit(props.row)" flat round dense icon="delete" color="negative" aria-label="删除方案" @click="deleteScheme(props.row)"><q-tooltip>删除</q-tooltip></q-btn>
        </q-td>
      </template>
      <template #no-data><div class="full-width row flex-center q-pa-xl text-grey-7">{{ error || '当前分类暂无查询方案' }}</div></template>
      <template #bottom><q-space /><table-pagination v-model:page="page" v-model:page-size="pageSize" :total="total" /></template>
    </q-table>

    <query-scheme-detail-drawer v-model="showDetail" :scheme-id="selectedId" :editable="!!selected && canEdit(selected)" @edit="editDetail" @copy="copyScheme" />
    <query-scheme-edit-dialog v-model="showEdit" :scheme-type="activeType" :detail="editValue" :scope-options="scopeOptions" @saved="fetchData" />
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'query_scheme_manager' })
import { computed, onMounted, ref, watch } from 'vue'
import { useQuasar } from 'quasar'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import StandardTableToolbar from 'src/components/Table/StandardTableToolbar.vue'
import TablePagination from 'src/components/Table/TablePagination.vue'
import StatusChip from 'src/components/Display/StatusChip.vue'
import QuerySchemeDetailDrawer from './QuerySchemeDetailDrawer.vue'
import QuerySchemeEditDialog from './QuerySchemeEditDialog.vue'
import { useQuerySchemeApi } from 'src/api/services/query-scheme'
import { collectQueryScopes } from 'src/composables/query-scope'
import { useConfirmDialog } from 'src/composables/confirm-dialog'
import { useUserStore } from 'src/stores/user'
import { hasGrantedActionCapability } from 'src/utils/menu-button'
import type { TableColumn } from 'src/types/global'
import {
  QUERY_SCHEME_TYPE_LABELS,
  QuerySchemeType,
  QuerySchemeValidationStatus,
  type QuerySchemeDetail,
  type QuerySchemeListItem,
  type QuerySchemeType as SchemeType,
  type QuerySchemeValidationStatus as ValidationStatus,
} from 'src/modules/query-scheme/types'
import { notifyQuerySchemeDeleted } from 'src/modules/query-scheme/events'

const $q = useQuasar()
const api = useQuerySchemeApi()
const router = useRouter()
const { t } = useI18n()
const userStore = useUserStore()
const { confirmDanger, confirmAction } = useConfirmDialog($q)
const canManageShared = computed(() =>
  hasGrantedActionCapability(userStore.menus, 'query_scheme_shared_manage'),
)
const activeType = ref<SchemeType>(QuerySchemeType.PERSONAL)
const nameFilter = ref('')
const scopeFilter = ref<string | null>(null)
const page = ref(1)
const pageSize = ref(15)
const total = ref(0)
const rows = ref<QuerySchemeListItem[]>([])
const loading = ref(false)
const error = ref('')
const showDetail = ref(false)
const showEdit = ref(false)
const selected = ref<QuerySchemeListItem | null>(null)
const selectedId = computed(() => selected.value?.id || 0)
const editValue = ref<QuerySchemeDetail | null>(null)
const scopeLabel = (label: string) => t(label)
const scopeOptions = computed(() => collectQueryScopes(userStore.menus).map((scope) => ({ label: scopeLabel(scope.scope_label), value: scope.scope_code })))
const scopeMenus = computed(() => collectQueryScopes(userStore.menus))
const columns: TableColumn<QuerySchemeListItem>[] = [
  { name: 'name', field: 'name', label: '方案名称', align: 'left' },
  { name: 'type', field: 'type', label: '类型', align: 'center' },
  { name: 'is_default', field: 'is_default', label: '默认', align: 'center' },
  { name: 'status', field: 'status', label: '校验状态', align: 'center' },
  { name: 'enabled', field: 'enabled', label: '启用状态', align: 'center' },
  { name: 'creator', field: 'creator_display_name', label: '创建人', align: 'left' },
  { name: 'updated_at', field: 'updated_at', label: '更新时间', align: 'left' },
  { name: 'actions', field: 'actions', label: '操作', align: 'center' },
]

const statusLabel = (status: ValidationStatus) => status === QuerySchemeValidationStatus.VALID ? '可用' : status === QuerySchemeValidationStatus.DEGRADED ? '需要修复' : '不可用'
const statusColor = (status: ValidationStatus) => status === QuerySchemeValidationStatus.VALID ? 'positive' : status === QuerySchemeValidationStatus.DEGRADED ? 'warning' : 'negative'
const canEdit = (row: QuerySchemeListItem) => row.type === QuerySchemeType.PERSONAL || canManageShared.value
const typeLabel = (row: QuerySchemeListItem) => QUERY_SCHEME_TYPE_LABELS[row.type]

const fetchData = async () => {
  loading.value = true
  error.value = ''
  try {
    const response = await api.list({
      page: page.value,
      num: pageSize.value,
      scheme_type: activeType.value,
      ...(nameFilter.value.trim() ? { name: nameFilter.value.trim() } : {}),
      ...(scopeFilter.value ? { scope_code: scopeFilter.value } : {}),
    })
    rows.value = response.data || []
    total.value = response.total || 0
  } catch (cause) {
    rows.value = []
    total.value = 0
    error.value = cause instanceof Error ? cause.message : '查询方案加载失败'
  } finally {
    loading.value = false
  }
}
const reloadFirstPage = () => { if (page.value !== 1) page.value = 1; else void fetchData() }
const openDetail = (row: QuerySchemeListItem) => { selected.value = row; showDetail.value = true }
const useScheme = (row: QuerySchemeListItem) => {
  const target = scopeMenus.value.find((scope) => scope.scope_code === row.scope_code)
  if (!target) {
    $q.notify({ type: 'warning', message: '当前账号无法进入该方案所属页面' })
    return
  }
  void router.push({ name: target.route_name, query: { query_scheme_id: String(row.id) } })
}
const openCreate = () => { editValue.value = null; showEdit.value = true }
const openEdit = async (row: QuerySchemeListItem) => { editValue.value = (await api.detail(row.id)).data || null; if (editValue.value) showEdit.value = true }
const editDetail = (detail: QuerySchemeDetail) => { editValue.value = detail; showDetail.value = false; showEdit.value = true }
const copyScheme = (row: QuerySchemeListItem | QuerySchemeDetail) => {
  $q.dialog({ title: '复制为我的方案', message: '请输入新方案名称', prompt: { model: `${row.name} 副本`, type: 'text', maxlength: 64 }, cancel: true, persistent: true }).onOk((name: string) => {
    void api.copyToPersonal(row.id, row.scope_code, name.trim()).then(fetchData)
  })
}
const setPersonalDefault = (row: QuerySchemeListItem) => confirmAction({ title: row.is_default ? '取消默认方案' : '设置默认方案', message: `确认${row.is_default ? '取消' : '设置'}“${row.name}”为个人默认方案？` }).onOk(() => { void api.setPersonalDefault(row.id, !row.is_default, row.revision).then(fetchData) })
const toggleEnabled = (row: QuerySchemeListItem) => confirmAction({ title: row.enabled ? '停用方案' : '启用方案', message: `确认${row.enabled ? '停用' : '启用'}“${row.name}”？` }).onOk(() => { void api.setSharedEnabled(row.id, !row.enabled, row.revision).then(fetchData) })
const deleteScheme = (row: QuerySchemeListItem) => confirmDanger({ title: '删除查询方案', message: `删除“${row.name}”后无法恢复，确认继续？` }).onOk(() => { const request = row.type === QuerySchemeType.PERSONAL ? api.deletePersonal(row.id, row.revision) : api.deleteShared(row.id, row.revision); void request.then(() => { notifyQuerySchemeDeleted(row.id); return fetchData() }) })

watch(() => [page.value, pageSize.value], () => void fetchData())
onMounted(fetchData)
</script>
