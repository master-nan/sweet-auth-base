<template>
  <base-content class="q-pa-sm data-permission-page">
    <master-detail-page
      :mode="SysMasterDetailMode.TABLE"
      master-title="数据维度"
      :master-subtitle="`${dimensions.length} 个维度`"
      detail-title="权限绑定"
      :detail-subtitle="selectedMenu ? `${displayMenuTitle(selectedMenu)} · ${selectedMenu.table_code}` : '选择低代码菜单'"
      master-width="minmax(480px, 42%)"
      min-width="0"
      min-height="calc(100vh - 150px)"
    >
      <template #master-actions>
        <q-btn unelevated color="primary" icon="add" label="新增维度" @click="openDimensionDialog()" />
        <q-btn outline color="primary" round icon="refresh" :loading="loading" @click="loadAll">
          <q-tooltip>刷新</q-tooltip>
        </q-btn>
      </template>

      <template #master-toolbar>
        <div class="data-permission-toolbar">
          <q-input v-model="dimensionKeyword" dense outlined clearable placeholder="搜索维度编码 / 名称">
            <template #prepend>
              <q-icon name="search" />
            </template>
          </q-input>
        </div>
      </template>

      <template #master-content>
        <q-table
          class="fit sticky-header-table"
          :rows="filteredDimensions"
          :columns="dimensionColumns"
          row-key="id"
          flat
          bordered
          separator="cell"
          :loading="loading"
          :pagination="{ rowsPerPage: 0 }"
          hide-pagination
        >
          <template #body-cell-source_code="props">
            <q-td :props="props">
              {{ sourceTableLabel(props.row.source_code) }}
            </q-td>
          </template>
          <template #body-cell-state="props">
            <q-td :props="props">
              <q-badge :color="props.row.state === false ? 'grey-6' : 'positive'" outline>
                {{ props.row.state === false ? '停用' : '启用' }}
              </q-badge>
            </q-td>
          </template>
          <template #body-cell-actions="props">
            <q-td :props="props" class="q-gutter-xs dimension-actions-cell">
              <q-btn
                class="dimension-action-btn"
                flat
                dense
                round
                color="primary"
                icon="edit"
                @click.stop="openDimensionDialog(props.row)"
              >
                <q-tooltip>编辑</q-tooltip>
              </q-btn>
              <q-btn
                class="dimension-action-btn"
                flat
                dense
                round
                color="negative"
                icon="delete"
                @click.stop="confirmDeleteDimension(props.row)"
              >
                <q-tooltip>删除</q-tooltip>
              </q-btn>
            </q-td>
          </template>
        </q-table>
      </template>

      <template #detail-context>
        <div class="data-permission-detail-head">
          <div class="data-permission-icon-tile">
            <q-icon name="rule" />
          </div>
          <div class="data-permission-detail-main">
            <div class="data-permission-detail-title">权限绑定</div>
            <div class="data-permission-detail-meta">
              <q-chip v-if="selectedMenu" dense square color="primary" text-color="white">
                {{ selectedMenu.name }}
              </q-chip>
              <span>{{ selectedMenu?.path || '低代码菜单' }}</span>
            </div>
          </div>
        </div>
      </template>

      <template #detail-toolbar>
        <div class="data-permission-toolbar data-permission-toolbar--detail">
          <q-select
            v-model="selectedMenuId"
            class="data-permission-menu-select"
            dense
            outlined
            emit-value
            map-options
            use-input
            clearable
            input-debounce="150"
            label="低代码菜单"
            :options="filteredMenuOptions"
            @filter="filterMenuOptions"
            @update:model-value="onMenuChange"
          >
            <template #option="scope">
              <q-item v-bind="scope.itemProps">
                <q-item-section avatar>
                  <q-icon :name="scope.opt.icon || 'dynamic_form'" />
                </q-item-section>
                <q-item-section>
                  <q-item-label>{{ scope.opt.label }}</q-item-label>
                  <q-item-label caption>{{ scope.opt.caption }}</q-item-label>
                </q-item-section>
              </q-item>
            </template>
          </q-select>
          <q-space />
          <q-btn
            outline
            dense
            color="primary"
            icon="add"
            label="新增"
            :disable="!selectedMenu || dimensions.length === 0"
            @click="openBindingDialog()"
          />
          <q-btn
            unelevated
            dense
            color="primary"
            icon="save"
            label="保存"
            :loading="savingBindings"
            :disable="!selectedMenu"
            @click="saveBindings"
          />
        </div>
      </template>

      <template #detail-content>
        <div v-if="selectedMenu" class="data-permission-binding-wrap">
          <q-banner v-if="tableFieldOptions.length === 0" rounded class="bg-orange-1 text-warning q-ma-md">
            当前菜单绑定表没有可用字段
          </q-banner>

          <q-expansion-item
            class="data-permission-debug-panel"
            icon="policy"
            label="当前账号权限排查"
            caption="模拟运行时按菜单、绑定表和动作解析出的最终数据范围"
          >
            <div class="data-permission-debug-body">
              <q-select
                v-model="debugAction"
                dense
                outlined
                emit-value
                map-options
                label="动作"
                :options="dataPermissionActionOptions"
              />
              <q-btn
                unelevated
                dense
                color="primary"
                icon="manage_search"
                label="诊断"
                :loading="debugLoading"
                @click="loadDebugResult"
              />
            </div>
            <div v-if="debugResult" class="data-permission-debug-result">
              <q-chip dense square :color="debugScopeColor" text-color="white">
                {{ debugScopeLabel }}
              </q-chip>
              <span>用户 {{ debugResult.user_name }} · 角色 {{ debugResult.role_ids?.join(', ') || '无' }}</span>
              <span>绑定 {{ debugResult.bindings?.length || 0 }} 个</span>
              <span>角色范围 {{ debugResult.role_scopes?.length || 0 }} 条</span>
              <span>用户归属 {{ debugResult.user_dimensions?.length || 0 }} 条</span>
              <span>特殊授权 {{ debugResult.user_overrides?.length || 0 }} 条</span>
              <q-chip v-for="note in debugResult.notes || []" :key="note" dense square color="warning" text-color="white">
                {{ note }}
              </q-chip>
              <pre>{{ debugScopeText }}</pre>
            </div>
          </q-expansion-item>

          <div class="data-permission-binding-table-wrap">
            <q-table
              class="fit sticky-header-table data-permission-binding-table"
              :rows="bindingRows"
              :columns="bindingColumns"
              row-key="local_id"
              flat
              bordered
              separator="cell"
              :pagination="{ rowsPerPage: 0 }"
              hide-pagination
            >
              <template #body-cell-dimension_code="props">
                <q-td :props="props">
                  {{ dimensionLabel(props.row.dimension_code) }}
                </q-td>
              </template>
              <template #body-cell-field_code="props">
                <q-td :props="props">
                  {{ fieldLabel(props.row.field_code) }}
                </q-td>
              </template>
              <template #body-cell-match_type="props">
                <q-td :props="props">
                  {{ matchTypeLabel(props.row.match_type) }}
                </q-td>
              </template>
              <template #body-cell-actions="props">
                <q-td :props="props">
                  <q-chip dense square color="primary" text-color="white">
                    {{ actionsDisplay(props.row.actions) }}
                  </q-chip>
                  <span v-if="actionsTooltip(props.row.actions)" class="data-permission-action-hint">
                    <q-tooltip>
                      {{ actionsTooltip(props.row.actions) }}
                    </q-tooltip>
                  </span>
                </q-td>
              </template>
              <template #body-cell-required="props">
                <q-td :props="props" class="text-center">
                  <q-badge :color="props.row.required === false ? 'grey-6' : 'warning'" outline>
                    {{ props.row.required === false ? '否' : '是' }}
                  </q-badge>
                </q-td>
              </template>
              <template #body-cell-state="props">
                <q-td :props="props" class="text-center">
                  <q-badge :color="props.row.state === false ? 'grey-6' : 'positive'" outline>
                    {{ props.row.state === false ? '停用' : '启用' }}
                  </q-badge>
                </q-td>
              </template>
              <template #body-cell-row_actions="props">
                <q-td :props="props" class="text-center q-gutter-xs">
                  <q-btn flat dense round color="primary" icon="edit" @click="openBindingDialog(props.row)">
                    <q-tooltip>编辑</q-tooltip>
                  </q-btn>
                  <q-btn flat dense round color="negative" icon="delete" @click="removeBindingRow(props.row.local_id)">
                    <q-tooltip>删除</q-tooltip>
                  </q-btn>
                </q-td>
              </template>
              <template #no-data>
                <div class="full-width row flex-center text-grey-7 q-gutter-sm q-pa-xl">
                  <q-icon name="rule_folder" size="32px" />
                  <span>暂无绑定</span>
                </div>
              </template>
            </q-table>
          </div>
        </div>

        <div v-else class="data-permission-empty">
          <q-icon name="ads_click" />
          <span>选择一个低代码菜单</span>
        </div>
      </template>
    </master-detail-page>

    <q-dialog v-model="bindingDialogOpen" persistent>
      <q-card class="binding-dialog-card">
        <q-card-section class="row items-center q-gutter-sm">
          <div>
            <div class="text-h6">{{ editingBindingLocalId ? '编辑绑定' : '新增绑定' }}</div>
            <div class="text-caption text-grey-7">{{ selectedMenu ? displayMenuTitle(selectedMenu) : '低代码菜单' }}</div>
          </div>
          <q-space />
          <q-btn flat round dense icon="close" @click="bindingDialogOpen = false" />
        </q-card-section>
        <q-separator />
        <q-card-section class="binding-form">
          <q-select
            v-model="bindingForm.dimension_code"
            dense
            outlined
            emit-value
            map-options
            options-dense
            label="维度"
            :options="dimensionOptions"
          />
          <q-select
            v-model="bindingForm.field_code"
            dense
            outlined
            emit-value
            map-options
            options-dense
            label="字段"
            :options="tableFieldOptions"
          />
          <q-select
            v-model="bindingForm.match_type"
            dense
            outlined
            emit-value
            map-options
            options-dense
            label="匹配"
            :options="matchTypeOptions"
          />
          <q-select
            v-model="bindingForm.actions"
            class="data-permission-action-select"
            dense
            outlined
            multiple
            emit-value
            map-options
            options-dense
            label="动作"
            :display-value="actionsDisplay(bindingForm.actions)"
            :options="dataPermissionActionOptions"
          >
            <q-tooltip v-if="actionsTooltip(bindingForm.actions)">
              {{ actionsTooltip(bindingForm.actions) }}
            </q-tooltip>
          </q-select>
          <q-toggle v-model="bindingForm.required" color="primary" label="必配授权" />
          <q-toggle v-model="bindingForm.state" color="primary" label="启用" />
        </q-card-section>
        <q-card-actions align="right">
          <q-btn flat color="grey-7" label="取消" @click="bindingDialogOpen = false" />
          <q-btn unelevated color="primary" label="确定" @click="saveBindingDialog" />
        </q-card-actions>
      </q-card>
    </q-dialog>

    <q-dialog v-model="dimensionDialogOpen" persistent>
      <q-card class="dimension-dialog-card">
        <q-card-section class="row items-center q-gutter-sm">
          <div class="text-h6">{{ dimensionForm.id ? '编辑维度' : '新增维度' }}</div>
          <q-space />
          <q-btn flat round dense icon="close" @click="dimensionDialogOpen = false" />
        </q-card-section>
        <q-separator />
        <q-card-section class="dimension-form">
          <q-input v-model="dimensionForm.code" dense outlined label="维度编码" />
          <q-input v-model="dimensionForm.name" dense outlined label="维度名称" />
          <q-select
            v-model="dimensionForm.value_type"
            dense
            outlined
            emit-value
            map-options
            label="值类型"
            :options="valueTypeOptions"
          />
          <q-select
            v-model="dimensionForm.source_type"
            dense
            outlined
            emit-value
            map-options
            label="来源类型"
            :options="sourceTypeOptions"
            @update:model-value="onDimensionSourceTypeChange"
          />
          <q-select
            v-model="dimensionForm.source_code"
            dense
            outlined
            clearable
            emit-value
            map-options
            use-input
            input-debounce="150"
            label="来源表编码"
            :options="filteredSourceTableOptions"
            :disable="dimensionForm.source_type !== 'table'"
            @filter="filterSourceTableOptions"
            @update:model-value="onDimensionSourceTableChange"
          />
          <q-select
            v-model="dimensionForm.label_field"
            dense
            outlined
            clearable
            emit-value
            map-options
            label="展示字段"
            :options="dimensionSourceFieldOptions"
            :disable="dimensionForm.source_type !== 'table'"
          />
          <q-select
            v-model="dimensionForm.value_field"
            dense
            outlined
            clearable
            emit-value
            map-options
            label="值字段"
            :options="dimensionSourceFieldOptions"
            :disable="dimensionForm.source_type !== 'table'"
          />
          <q-select
            v-model="dimensionForm.parent_field"
            dense
            outlined
            clearable
            emit-value
            map-options
            label="父级字段"
            :options="dimensionSourceFieldOptions"
            :disable="dimensionForm.source_type !== 'table'"
          />
          <q-input
            v-model="dimensionForm.memo"
            dense
            outlined
            label="备注"
            type="textarea"
            autogrow
          />
          <q-toggle v-model="dimensionForm.state" color="primary" label="启用" />
        </q-card-section>
        <q-card-actions align="right">
          <q-btn flat color="grey-7" label="取消" @click="dimensionDialogOpen = false" />
          <q-btn
            unelevated
            color="primary"
            label="保存"
            :loading="savingDimension"
            @click="saveDimension"
          />
        </q-card-actions>
      </q-card>
    </q-dialog>
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'system_data_permission' })

import { computed, onMounted, ref } from 'vue'
import { type QTableProps, useQuasar } from 'quasar'
import BaseContent from 'components/BaseContent/BaseContent.vue'
import MasterDetailPage from 'src/components/MasterDetail/MasterDetailPage.vue'
import { useMenuApi, type Menu } from 'src/api/services/sys-menu'
import { useTableApi, type Table, type TableField } from 'src/api/services/sys-table'
import {
  dataPermissionActionOptions,
  type DataPermissionBinding,
  type DataPermissionBindingSaveItem,
  type DataPermissionDebugResult,
  type DataPermissionDimension,
  type DataPermissionDimensionSaveReq,
  useDataPermissionApi,
} from 'src/api/services/data-permission'
import type { Query } from 'src/types/global'
import { SysMasterDetailMode } from 'src/types/enum'
import { useLoadingStore } from 'src/stores/loading'
import { storeToRefs } from 'pinia'
import { useConfirmDialog } from 'src/composables/confirm-dialog'
import { useI18n } from 'vue-i18n'
import { compactSelectionDisplay, compactSelectionTooltip } from 'src/utils/select-display'

type BindingRow = DataPermissionBindingSaveItem & {
  local_id: string
  table_code?: string
}

type MenuOption = {
  label: string
  value: number
  caption: string
  icon?: string
  menu: Menu
}

type TableOption = {
  label: string
  value: string
  table: Table
}

const $q = useQuasar()
const { t } = useI18n()
const { confirmDanger } = useConfirmDialog($q)
const loadingStore = useLoadingStore()
const { loading } = storeToRefs(loadingStore)
const menuApi = useMenuApi()
const tableApi = useTableApi()
const dataPermissionApi = useDataPermissionApi()

const dimensionKeyword = ref('')
const dimensions = ref<DataPermissionDimension[]>([])
const menuTree = ref<Menu[]>([])
const menuOptions = ref<MenuOption[]>([])
const filteredMenuOptions = ref<MenuOption[]>([])
const sourceTableOptions = ref<TableOption[]>([])
const filteredSourceTableOptions = ref<TableOption[]>([])
const dimensionSourceFieldOptions = ref<Array<{ label: string; value: string }>>([])
const selectedMenuId = ref<number | null>(null)
const selectedMenu = ref<Menu | null>(null)
const tableFieldOptions = ref<Array<{ label: string; value: string }>>([])
const bindingRows = ref<BindingRow[]>([])
const savingBindings = ref(false)
const bindingDialogOpen = ref(false)
const editingBindingLocalId = ref('')
const bindingForm = ref<BindingRow>(emptyBindingForm())
const debugAction = ref('query')
const debugTableCode = ref('')
const debugLoading = ref(false)
const debugResult = ref<DataPermissionDebugResult | null>(null)
const dimensionDialogOpen = ref(false)
const savingDimension = ref(false)
const dimensionForm = ref<DataPermissionDimensionSaveReq>(emptyDimensionForm())

const valueTypeOptions = [
  { label: '字符串', value: 'string' },
  { label: '数字', value: 'number' },
]

const sourceTypeOptions = [
  { label: '无来源', value: 'none' },
  { label: '数据表', value: 'table' },
]

const matchTypeOptions = [
  { label: '包含', value: 'in' },
  { label: '等于', value: 'eq' },
]

const defaultBindingActions = dataPermissionActionOptions.map((option) => option.value)

const dimensionColumns: QTableProps['columns'] = [
  { name: 'code', label: '编码', field: 'code', align: 'left', sortable: true },
  { name: 'name', label: '名称', field: 'name', align: 'left', sortable: true },
  { name: 'value_type', label: '值类型', field: 'value_type', align: 'left' },
  { name: 'source_type', label: '来源', field: 'source_type', align: 'left' },
  { name: 'source_code', label: '来源表', field: 'source_code', align: 'left' },
  { name: 'label_field', label: '展示字段', field: 'label_field', align: 'left' },
  { name: 'value_field', label: '值字段', field: 'value_field', align: 'left' },
  { name: 'parent_field', label: '父级字段', field: 'parent_field', align: 'left' },
  { name: 'memo', label: '备注', field: 'memo', align: 'left' },
  { name: 'state', label: '状态', field: 'state', align: 'center' },
  { name: 'actions', label: '操作', field: 'actions', align: 'center' },
]

const bindingColumns: QTableProps['columns'] = [
  { name: 'dimension_code', label: '维度', field: 'dimension_code', align: 'left', style: 'min-width: 220px; width: 260px;' },
  { name: 'field_code', label: '字段', field: 'field_code', align: 'left', style: 'min-width: 190px; width: 220px;' },
  { name: 'match_type', label: '匹配', field: 'match_type', align: 'left', style: 'min-width: 140px; width: 150px;' },
  { name: 'actions', label: '动作', field: 'actions', align: 'left', style: 'min-width: 260px; width: 320px;' },
  { name: 'required', label: '必配', field: 'required', align: 'center', style: 'width: 92px;' },
  { name: 'state', label: '启用', field: 'state', align: 'center', style: 'width: 92px;' },
  { name: 'row_actions', label: '操作', field: 'row_actions', align: 'center', style: 'width: 92px;' },
]

const dimensionOptions = computed(() =>
  dimensions.value
    .filter((dimension) => dimension.state !== false)
    .map((dimension) => ({
      label: `${dimension.name} (${dimension.code})`,
      value: dimension.code,
    })),
)

const filteredDimensions = computed(() => {
  const keyword = dimensionKeyword.value.trim().toLowerCase()
  if (!keyword) return dimensions.value
  return dimensions.value.filter((dimension) =>
    [dimension.code, dimension.name, dimension.memo].join(' ').toLowerCase().includes(keyword),
  )
})

const debugScope = computed(() => debugResult.value?.scope)

const debugScopeLabel = computed(() => {
  const scope = debugScope.value
  if (!scope) return '未诊断'
  if (scope.DenyAll || scope.deny_all) return '拒绝'
  if (scope.AllowAll || scope.allow_all) return '全部'
  return `条件 ${((scope.Conditions || scope.conditions) ?? []).length}`
})

const debugScopeColor = computed(() => {
  const scope = debugScope.value
  if (scope?.DenyAll || scope?.deny_all) return 'negative'
  if (scope?.AllowAll || scope?.allow_all) return 'positive'
  return 'primary'
})

const debugScopeText = computed(() => {
  if (!debugResult.value) return ''
  return JSON.stringify(debugResult.value.scope || {}, null, 2)
})

function emptyDimensionForm(): DataPermissionDimensionSaveReq {
  return {
    code: '',
    name: '',
    value_type: 'string',
    source_type: 'none',
    source_code: '',
    label_field: '',
    value_field: '',
    parent_field: '',
    memo: '',
    state: true,
  }
}

function emptyBindingForm(): BindingRow {
  return {
    local_id: '',
    dimension_code: '',
    field_code: '',
    match_type: 'in',
    required: true,
    actions: [],
    state: true,
    table_code: '',
  }
}

const emptyQuery = (): Query => ({
  page: 1,
  num: 1000,
  expressions: [{ rules: [{ field: '', value: null }], nested: [] }],
  quick_query: { keyword: '' },
  include_deleted: false,
})

const displayMenuTitle = (menu: Menu) => {
  const title = menu.title || menu.name
  return title.startsWith('router.') ? t(title) : title
}

const flattenMenus = (menus: Menu[]): Menu[] =>
  menus.flatMap((menu) => [menu, ...flattenMenus(menu.children || [])])

const buildMenuOptions = (menus: Menu[]) =>
  flattenMenus(menus)
    .filter((menu) => !!menu.table_code && !menu.is_hidden && menu.state !== false)
    .map<MenuOption>((menu) => ({
      label: displayMenuTitle(menu),
      value: menu.id,
      caption: `${menu.name} · ${menu.table_code}`,
      ...(menu.icon ? { icon: menu.icon } : {}),
      menu,
    }))

const loadDimensions = async () => {
  const result = await dataPermissionApi.queryDimensions(emptyQuery())
  dimensions.value = result.success ? result.data || [] : []
}

const loadTables = async () => {
  const result = await tableApi.queryTable(emptyQuery())
  const tables = result.success && Array.isArray(result.data) ? result.data : []
  sourceTableOptions.value = tables.map((table) => ({
    label: `${table.table_name || table.table_code} (${table.table_code})`,
    value: table.table_code,
    table,
  }))
  filteredSourceTableOptions.value = sourceTableOptions.value
}

const loadMenus = async () => {
  const result = await menuApi.queryMenu({
    ...emptyQuery(),
    order: { field: 'sequence', is_asc: true },
  })
  menuTree.value = result.success ? result.data || [] : []
  menuOptions.value = buildMenuOptions(menuTree.value)
  filteredMenuOptions.value = menuOptions.value
  const firstOption = menuOptions.value[0]
  if (!selectedMenuId.value && firstOption) {
    selectedMenuId.value = firstOption.value
    await onMenuChange(selectedMenuId.value)
  }
}

const loadAll = async () => {
  await Promise.all([loadDimensions(), loadMenus(), loadTables()])
  if (selectedMenuId.value) await onMenuChange(selectedMenuId.value)
}

const filterMenuOptions = (value: string, update: (callback: () => void) => void) => {
  update(() => {
    const keyword = value.trim().toLowerCase()
    filteredMenuOptions.value = keyword
      ? menuOptions.value.filter((option) =>
          [option.label, option.caption].join(' ').toLowerCase().includes(keyword),
        )
      : menuOptions.value
  })
}

const filterSourceTableOptions = (value: string, update: (callback: () => void) => void) => {
  update(() => {
    const keyword = value.trim().toLowerCase()
    filteredSourceTableOptions.value = keyword
      ? sourceTableOptions.value.filter((option) =>
          [option.label, option.value].join(' ').toLowerCase().includes(keyword),
        )
      : sourceTableOptions.value
  })
}

const sourceTableLabel = (tableCode: string) =>
  sourceTableOptions.value.find((option) => option.value === tableCode)?.label || tableCode || '-'

const dimensionLabel = (dimensionCode: string) =>
  dimensionOptions.value.find((option) => option.value === dimensionCode)?.label || dimensionCode || '-'

const fieldLabel = (fieldCode: string) =>
  tableFieldOptions.value.find((option) => option.value === fieldCode)?.label || fieldCode || '-'

const matchTypeLabel = (matchType: string) =>
  matchTypeOptions.find((option) => option.value === matchType)?.label || matchType || '-'

const sourceFieldOptionsFromTable = (table?: Table | null) =>
  (table?.table_fields || []).map((field) => ({
    label: `${field.field_name || field.field_code} (${field.field_code})`,
    value: field.field_code,
  }))

const loadDimensionSourceFields = async (tableCode: string) => {
  if (!tableCode) {
    dimensionSourceFieldOptions.value = []
    return
  }
  const selectedOption = sourceTableOptions.value.find((option) => option.value === tableCode)
  if (selectedOption?.table.table_fields?.length) {
    dimensionSourceFieldOptions.value = sourceFieldOptionsFromTable(selectedOption.table)
    return
  }
  const result = await tableApi.queryTableByCode(tableCode)
  dimensionSourceFieldOptions.value = result.success ? sourceFieldOptionsFromTable(result.data) : []
}

const onDimensionSourceTypeChange = async (value: string) => {
  if (value !== 'table') {
    dimensionForm.value.source_code = ''
    dimensionForm.value.label_field = ''
    dimensionForm.value.value_field = ''
    dimensionForm.value.parent_field = ''
    dimensionSourceFieldOptions.value = []
    return
  }
  await loadDimensionSourceFields(dimensionForm.value.source_code)
}

const onDimensionSourceTableChange = async (value: string | null) => {
  const tableCode = value || ''
  dimensionForm.value.source_code = tableCode
  dimensionForm.value.label_field = ''
  dimensionForm.value.value_field = ''
  dimensionForm.value.parent_field = ''
  await loadDimensionSourceFields(tableCode)
}

const onMenuChange = async (menuId: number | null) => {
  selectedMenu.value = menuOptions.value.find((option) => option.value === menuId)?.menu || null
  tableFieldOptions.value = []
  bindingRows.value = []
  debugResult.value = null
  debugTableCode.value = selectedMenu.value?.table_code || ''
  if (!selectedMenu.value?.table_code) return
  const [tableResult, bindingResult] = await Promise.all([
    tableApi.queryTableByCode(selectedMenu.value.table_code),
    dataPermissionApi.getMenuBindings(selectedMenu.value.id),
  ])
  tableFieldOptions.value = tableResult.success
    ? (tableResult.data?.table_fields || []).map((field: TableField) => ({
        label: `${field.field_name || field.field_code} (${field.field_code})`,
        value: field.field_code,
      }))
    : []
  bindingRows.value = (bindingResult.success ? bindingResult.data || [] : []).map((binding) =>
    bindingToRow(binding),
  )
}

const loadDebugResult = async () => {
  if (!selectedMenu.value || !debugTableCode.value) return
  debugLoading.value = true
  try {
    const result = await dataPermissionApi.debugDataScope({
      menu_id: selectedMenu.value.id,
      table_code: debugTableCode.value,
      action: debugAction.value,
    })
    debugResult.value = result.success ? result.data || null : null
  } finally {
    debugLoading.value = false
  }
}

const bindingToRow = (binding: DataPermissionBinding): BindingRow => ({
  local_id: String(binding.id || `${binding.dimension_code}-${binding.field_code}`),
  id: binding.id,
  dimension_code: binding.dimension_code,
  field_code: binding.field_code,
  match_type: binding.match_type || 'in',
  required: binding.required !== false,
  actions: binding.actions?.length ? binding.actions : defaultBindingActions,
  state: binding.state !== false,
  table_code: binding.table_code,
})

const nextBindingLocalId = () => `new-${Date.now()}-${Math.random().toString(16).slice(2)}`

const openBindingDialog = (row?: BindingRow) => {
  editingBindingLocalId.value = row?.local_id || ''
  bindingForm.value = row
    ? {
        ...row,
        actions: [...(row.actions?.length ? row.actions : defaultBindingActions)],
      }
    : {
        ...emptyBindingForm(),
        local_id: nextBindingLocalId(),
        dimension_code: dimensionOptions.value[0]?.value || '',
        field_code: tableFieldOptions.value[0]?.value || '',
        actions: [...defaultBindingActions],
        table_code: selectedMenu.value?.table_code || '',
      }
  bindingDialogOpen.value = true
}

const saveBindingDialog = () => {
  const row = {
    ...bindingForm.value,
    actions: bindingForm.value.actions?.length ? [...bindingForm.value.actions] : [],
  }
  if (!row.dimension_code || !row.field_code) {
    $q.notify({ type: 'warning', position: 'top-right', message: '请选择维度和字段' })
    return
  }
  if (!row.actions.length) {
    $q.notify({ type: 'warning', position: 'top-right', message: '请选择动作' })
    return
  }
  const duplicate = bindingRows.value.some(
    (item) =>
      item.local_id !== editingBindingLocalId.value &&
      item.dimension_code === row.dimension_code &&
      item.field_code === row.field_code,
  )
  if (duplicate) {
    $q.notify({ type: 'warning', position: 'top-right', message: '同一维度和字段不能重复绑定' })
    return
  }
  if (editingBindingLocalId.value) {
    bindingRows.value = bindingRows.value.map((item) =>
      item.local_id === editingBindingLocalId.value ? row : item,
    )
  } else {
    bindingRows.value = [...bindingRows.value, row]
  }
  bindingDialogOpen.value = false
}

const removeBindingRow = (localId: string) => {
  bindingRows.value = bindingRows.value.filter((row) => row.local_id !== localId)
}

const actionsDisplay = (actions: string[]) => {
  return compactSelectionDisplay(actions, dataPermissionActionOptions, 2, '全部动作')
}

const actionsTooltip = (actions: string[]) => {
  return compactSelectionTooltip(actions, dataPermissionActionOptions)
}

const validateBindings = () => {
  const uniqueKeys = new Set<string>()
  for (const row of bindingRows.value) {
    if (!row.dimension_code || !row.field_code) {
      $q.notify({ type: 'warning', position: 'top-right', message: '请选择维度和字段' })
      return false
    }
    if (!row.actions?.length) {
      $q.notify({ type: 'warning', position: 'top-right', message: '请选择动作' })
      return false
    }
    const uniqueKey = `${row.dimension_code}:${row.field_code}`
    if (uniqueKeys.has(uniqueKey)) {
      $q.notify({ type: 'warning', position: 'top-right', message: '同一维度和字段不能重复绑定' })
      return false
    }
    uniqueKeys.add(uniqueKey)
  }
  return true
}

const saveBindings = async () => {
  if (!selectedMenu.value || !validateBindings()) return
  savingBindings.value = true
  try {
    const payload = bindingRows.value.map<DataPermissionBindingSaveItem>((row) => ({
      dimension_code: row.dimension_code,
      field_code: row.field_code,
      match_type: row.match_type || 'in',
      required: row.required !== false,
      actions: row.actions?.length ? row.actions : defaultBindingActions,
      state: row.state !== false,
    }))
    const result = await dataPermissionApi.saveMenuBindings(selectedMenu.value.id, payload)
    if (result.success) {
      $q.notify({ type: 'positive', position: 'top-right', message: '绑定已保存' })
      await onMenuChange(selectedMenu.value.id)
    }
  } finally {
    savingBindings.value = false
  }
}

const openDimensionDialog = async (dimension?: DataPermissionDimension) => {
  dimensionForm.value = dimension
    ? {
        id: dimension.id,
        code: dimension.code,
        name: dimension.name,
        value_type: dimension.value_type || 'string',
        source_type: dimension.source_type || 'none',
        source_code: dimension.source_code || '',
        label_field: dimension.label_field || '',
        value_field: dimension.value_field || '',
        parent_field: dimension.parent_field || '',
        memo: dimension.memo || '',
        state: dimension.state !== false,
      }
    : emptyDimensionForm()
  await loadDimensionSourceFields(dimensionForm.value.source_code)
  dimensionDialogOpen.value = true
}

const validateDimension = () => {
  const form = dimensionForm.value
  if (!form.code.trim() || !form.name.trim()) {
    $q.notify({ type: 'warning', position: 'top-right', message: '请填写维度编码和名称' })
    return false
  }
  if (form.source_type === 'table' && (!form.source_code.trim() || !form.label_field.trim() || !form.value_field.trim())) {
    $q.notify({ type: 'warning', position: 'top-right', message: '请填写来源表、展示字段和值字段' })
    return false
  }
  return true
}

const saveDimension = async () => {
  if (!validateDimension()) return
  savingDimension.value = true
  try {
    const payload = {
      ...dimensionForm.value,
      code: dimensionForm.value.code.trim(),
      name: dimensionForm.value.name.trim(),
      source_code: dimensionForm.value.source_code.trim(),
      label_field: dimensionForm.value.label_field.trim(),
      value_field: dimensionForm.value.value_field.trim(),
      parent_field: dimensionForm.value.parent_field.trim(),
      memo: dimensionForm.value.memo.trim(),
    }
    const result = payload.id
      ? await dataPermissionApi.updateDimension(payload)
      : await dataPermissionApi.createDimension(payload)
    if (result.success) {
      dimensionDialogOpen.value = false
      await loadDimensions()
    }
  } finally {
    savingDimension.value = false
  }
}

const confirmDeleteDimension = (dimension: DataPermissionDimension) => {
  confirmDanger({
    message: `确定要删除维度 "${dimension.name}" 吗？`,
    loading: loading.value,
    disable: loading.value,
  }).onOk(() => {
    void (async () => {
      const result = await dataPermissionApi.deleteDimension(dimension.id)
      if (result.success) {
        await loadDimensions()
      }
    })()
  })
}

onMounted(() => {
  void loadAll()
})
</script>

<style scoped lang="scss">
.data-permission-page {
  background: #f6f7fb;
}

.data-permission-toolbar {
  display: flex;
  gap: 10px;
  align-items: center;
  padding: 12px 14px;
  border-bottom: 1px solid #e3e8f2;
  background: #fff;
}

.data-permission-toolbar--detail {
  min-height: 64px;
  gap: 8px;
}

.data-permission-toolbar--detail .q-btn {
  min-height: 36px;
}

.data-permission-toolbar--detail :deep(.q-btn__content) {
  white-space: nowrap;
}

.data-permission-menu-select {
  flex: 1;
  min-width: 220px;
  max-width: 360px;
}

.data-permission-detail-head {
  display: flex;
  gap: 12px;
  align-items: center;
}

.data-permission-icon-tile {
  width: 42px;
  height: 42px;
  display: grid;
  place-items: center;
  border-radius: 8px;
  color: var(--q-primary);
  background: rgba(25, 118, 210, 0.1);
}

.data-permission-icon-tile .q-icon {
  font-size: 24px;
}

.dimension-action-btn {
  width: 30px;
  height: 30px;
}

.dimension-action-btn :deep(.q-icon) {
  font-size: 18px;
}

.data-permission-detail-main {
  min-width: 0;
}

.data-permission-detail-title {
  color: #172033;
  font-size: 17px;
  font-weight: 800;
}

.data-permission-detail-meta {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-top: 4px;
  color: #657189;
  font-size: 12px;
}

.data-permission-binding-wrap {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 12px;
  overflow: hidden;
  background: #f8fafc;
}

.data-permission-debug-panel {
  flex-shrink: 0;
  margin-bottom: 12px;
  border: 1px solid #dbe3ef;
  border-radius: 8px;
  background: #fff;
}

.data-permission-debug-body {
  display: grid;
  grid-template-columns: 180px max-content;
  gap: 10px;
  align-items: center;
  padding: 0 16px 12px;
}

.data-permission-debug-result {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  padding: 0 16px 16px;
  color: #4b5870;
  font-size: 13px;
}

.data-permission-debug-result pre {
  width: 100%;
  max-height: 180px;
  margin: 4px 0 0;
  overflow: auto;
  padding: 10px;
  border-radius: 8px;
  background: #f4f6fa;
  color: #22304a;
  font-size: 12px;
  line-height: 1.5;
}

.data-permission-binding-table-wrap {
  flex: 1;
  min-height: 0;
  overflow: auto;
}

.data-permission-binding-table {
  height: 100%;
}

.data-permission-action-hint {
  display: inline-block;
}

.data-permission-action-select :deep(.q-field__native) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.data-permission-empty {
  height: 100%;
  display: flex;
  gap: 10px;
  align-items: center;
  justify-content: center;
  color: #7a869f;
  font-size: 14px;
}

.data-permission-empty .q-icon {
  font-size: 36px;
}

.dimension-dialog-card {
  width: 760px;
  max-width: 92vw;
  border-radius: 8px;
}

.binding-dialog-card {
  width: 720px;
  max-width: 92vw;
  border-radius: 8px;
}

.binding-form {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.binding-form .data-permission-action-select {
  grid-column: 1 / -1;
}

.dimension-form {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.dimension-form .q-textarea,
.dimension-form .q-toggle {
  grid-column: 1 / -1;
}

@media (max-width: 900px) {
  .dimension-form {
    grid-template-columns: 1fr;
  }

  .binding-form {
    grid-template-columns: 1fr;
  }

  .data-permission-menu-select {
    min-width: 0;
    width: 100%;
  }

  .data-permission-debug-body {
    grid-template-columns: 1fr;
  }
}
</style>
