<template>
  <base-content class="q-pa-sm data-permission-page">
    <div class="data-permission-shell">
      <div class="data-permission-summary">
        <div class="data-permission-summary-card">
          <q-icon name="badge" />
          <div>
            <strong>{{ dimensions.length }}</strong>
            <span>数据维度</span>
          </div>
        </div>
        <div class="data-permission-summary-card">
          <q-icon name="rule" />
          <div>
            <strong>{{ bindingRows.length }}</strong>
            <span>当前绑定</span>
          </div>
        </div>
        <div class="data-permission-summary-card">
          <q-icon name="dynamic_form" />
          <div>
            <strong>{{ menuOptions.length }}</strong>
            <span>可配置菜单</span>
          </div>
        </div>
        <div class="data-permission-summary-card">
          <q-icon name="policy" />
          <div>
            <strong>{{ debugScopeLabel }}</strong>
            <span>诊断结果</span>
          </div>
        </div>
      </div>

      <div class="data-permission-workspace">
        <q-tabs
          v-model="activeWorkspaceTab"
          class="data-permission-tabs"
          active-color="primary"
          indicator-color="primary"
          align="left"
          no-caps
        >
          <q-tab name="dimensions" icon="badge" label="数据维度" />
          <q-tab name="bindings" icon="rule" label="权限绑定" />
          <q-tab name="debug" icon="policy" label="权限诊断" />
          <q-tab name="checks" icon="fact_check" label="配置检查" />
        </q-tabs>

        <q-separator />

        <q-tab-panels
          v-model="activeWorkspaceTab"
          animated
          keep-alive
          class="data-permission-panels"
        >
          <q-tab-panel name="dimensions" class="data-permission-tab-panel">
            <div class="data-permission-panel-head">
              <div>
                <div class="data-permission-panel-title">数据维度</div>
                <div class="data-permission-panel-caption">
                  定义公司、部门、区域等可复用业务维度，角色和用户授权都会引用这里的维度。
                </div>
              </div>
              <div class="data-permission-panel-actions">
                <q-btn
                  unelevated
                  color="primary"
                  icon="add"
                  label="新增维度"
                  @click="openDimensionDialog()"
                />
                <q-btn
                  outline
                  color="primary"
                  round
                  icon="refresh"
                  :loading="loading"
                  @click="loadAll"
                >
                  <q-tooltip>刷新</q-tooltip>
                </q-btn>
              </div>
            </div>

            <div class="data-permission-panel-toolbar">
              <q-input
                v-model="dimensionKeyword"
                dense
                outlined
                clearable
                placeholder="搜索维度编码 / 名称"
              >
                <template #prepend>
                  <q-icon name="search" />
                </template>
              </q-input>
            </div>

            <q-table
              class="sticky-header-table data-permission-table"
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
          </q-tab-panel>

          <q-tab-panel name="bindings" class="data-permission-tab-panel">
            <div class="data-permission-panel-head">
              <div>
                <div class="data-permission-panel-title">权限绑定</div>
                <div class="data-permission-panel-caption">
                  给低代码菜单绑定数据维度和表字段，运行时会按角色范围与用户归属合并出最终可见数据。
                </div>
              </div>
              <div class="data-permission-panel-actions">
                <q-btn
                  unelevated
                  color="primary"
                  icon="add"
                  label="新增"
                  :disable="!selectedMenu || dimensions.length === 0"
                  @click="openBindingDialog()"
                />
              </div>
            </div>

            <div class="data-permission-panel-toolbar">
              <sweet-select
                v-model="selectedMenuId"
                class="data-permission-menu-select"
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
              </sweet-select>
            </div>

            <div v-if="selectedMenu" class="data-permission-menu-context">
              <q-icon :name="selectedMenu.icon || 'dynamic_form'" />
              <div>
                <div class="data-permission-context-title">{{ selectedMenuDisplayTitle }}</div>
                <div class="data-permission-context-meta">
                  <q-chip dense square color="primary" text-color="white">{{
                    selectedMenu.name
                  }}</q-chip>
                  <q-chip dense square outline color="primary"
                    >绑定表 {{ selectedMenu.table_code }}</q-chip
                  >
                </div>
              </div>
            </div>

            <q-banner
              v-if="selectedMenu && tableFieldOptions.length === 0"
              rounded
              class="bg-orange-1 text-warning q-mb-md"
            >
              当前菜单绑定表没有可用字段
            </q-banner>

            <q-table
              v-if="selectedMenu"
              class="sticky-header-table data-permission-binding-table"
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
                  <span
                    v-if="actionsTooltip(props.row.actions)"
                    class="data-permission-action-hint"
                  >
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
                  <q-btn
                    flat
                    dense
                    round
                    color="primary"
                    icon="edit"
                    @click="openBindingDialog(props.row)"
                  >
                    <q-tooltip>编辑</q-tooltip>
                  </q-btn>
                  <q-btn
                    flat
                    dense
                    round
                    color="negative"
                    icon="delete"
                    @click="removeBindingRow(props.row.local_id)"
                  >
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

            <div v-else class="data-permission-empty">
              <q-icon name="ads_click" />
              <span>选择一个低代码菜单</span>
            </div>
          </q-tab-panel>

          <q-tab-panel name="debug" class="data-permission-tab-panel">
            <div class="data-permission-panel-head">
              <div>
                <div class="data-permission-panel-title">权限诊断</div>
                <div class="data-permission-panel-caption">
                  用当前账号模拟菜单和动作，查看最终解析出的数据范围。
                </div>
              </div>
            </div>

            <div class="data-permission-debug-card">
              <sweet-select
                v-model="selectedMenuId"
                class="data-permission-menu-select"
                emit-value
                map-options
                use-input
                clearable
                input-debounce="150"
                label="低代码菜单"
                :options="filteredMenuOptions"
                @filter="filterMenuOptions"
                @update:model-value="onMenuChange"
              />
              <sweet-select
                v-model="debugAction"
                emit-value
                map-options
                label="动作"
                :options="dataPermissionActionOptions"
              />
              <q-btn
                unelevated
                color="primary"
                icon="manage_search"
                label="诊断"
                :disable="!selectedMenu"
                :loading="debugLoading"
                @click="loadDebugResult"
              />
            </div>

            <div v-if="debugResult" class="data-permission-debug-result">
              <q-chip dense square :color="debugScopeColor" text-color="white">
                {{ debugScopeLabel }}
              </q-chip>
              <span
                >用户 {{ debugResult.user_name }} · 角色
                {{ debugResult.role_ids?.join(', ') || '无' }}</span
              >
              <span>绑定 {{ debugResult.bindings?.length || 0 }} 个</span>
              <span>角色范围 {{ debugResult.role_scopes?.length || 0 }} 条</span>
              <span>用户归属 {{ debugResult.user_dimensions?.length || 0 }} 条</span>
              <span>特殊授权 {{ debugResult.user_overrides?.length || 0 }} 条</span>
              <q-chip
                v-for="note in debugResult.notes || []"
                :key="note"
                dense
                square
                color="warning"
                text-color="white"
              >
                {{ note }}
              </q-chip>
              <pre>{{ debugScopeText }}</pre>
            </div>
            <div v-else class="data-permission-empty data-permission-empty--panel">
              <q-icon name="policy" />
              <span>选择菜单和动作后点击诊断</span>
            </div>
          </q-tab-panel>

          <q-tab-panel name="checks" class="data-permission-tab-panel">
            <div class="data-permission-panel-head">
              <div>
                <div class="data-permission-panel-title">配置检查</div>
                <div class="data-permission-panel-caption">
                  快速发现维度来源、菜单字段和当前绑定里容易影响授权结果的配置。
                </div>
              </div>
              <div class="data-permission-panel-actions">
                <q-btn
                  outline
                  color="primary"
                  round
                  icon="refresh"
                  :loading="loading"
                  @click="loadAll"
                >
                  <q-tooltip>刷新</q-tooltip>
                </q-btn>
              </div>
            </div>

            <div class="data-permission-check-grid">
              <div
                v-for="item in configCheckItems"
                :key="item.label"
                class="data-permission-check-card"
              >
                <q-icon :name="item.icon" :color="item.color" />
                <div>
                  <strong>{{ item.value }}</strong>
                  <span>{{ item.label }}</span>
                  <em>{{ item.caption }}</em>
                </div>
              </div>
            </div>
          </q-tab-panel>
        </q-tab-panels>
      </div>
    </div>

    <q-dialog v-model="bindingDialogOpen" persistent>
      <q-card class="binding-dialog-card">
        <q-card-section class="row items-center q-gutter-sm">
          <div>
            <div class="text-h6">{{ editingBindingLocalId ? '编辑绑定' : '新增绑定' }}</div>
            <div class="text-caption text-grey-7">
              {{ selectedMenuDisplayTitle || '低代码菜单' }}
            </div>
          </div>
          <q-space />
          <q-btn flat round dense icon="close" @click="bindingDialogOpen = false" />
        </q-card-section>
        <q-separator />
        <q-card-section class="binding-form">
          <sweet-select
            v-model="bindingForm.dimension_code"
            emit-value
            map-options
            options-dense
            label="维度"
            :options="dimensionOptions"
          />
          <sweet-select
            v-model="bindingForm.field_code"
            emit-value
            map-options
            options-dense
            label="字段"
            :options="tableFieldOptions"
          />
          <sweet-select
            v-model="bindingForm.match_type"
            emit-value
            map-options
            options-dense
            label="匹配"
            :options="matchTypeOptions"
          />
          <sweet-select
            v-model="bindingForm.actions"
            class="data-permission-action-select"
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
          </sweet-select>
          <q-toggle v-model="bindingForm.required" color="primary" label="必配授权" />
          <q-toggle v-model="bindingForm.state" color="primary" label="启用" />
        </q-card-section>
        <q-card-actions align="right">
          <q-btn flat color="grey-7" label="取消" @click="bindingDialogOpen = false" />
          <q-btn
            unelevated
            color="primary"
            label="保存"
            :loading="savingBindings"
            @click="saveBindingDialog"
          />
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
          <sweet-select
            v-model="dimensionForm.value_type"
            emit-value
            map-options
            label="值类型"
            :options="valueTypeOptions"
          />
          <sweet-select
            v-model="dimensionForm.source_type"
            emit-value
            map-options
            label="来源类型"
            :options="sourceTypeOptions"
            @update:model-value="onDimensionSourceTypeChange"
          />
          <sweet-select
            v-model="dimensionForm.source_code"
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
          <sweet-select
            v-model="dimensionForm.label_field"
            clearable
            emit-value
            map-options
            label="展示字段"
            :options="dimensionSourceFieldOptions"
            :disable="dimensionForm.source_type !== 'table'"
          />
          <sweet-select
            v-model="dimensionForm.value_field"
            clearable
            emit-value
            map-options
            label="值字段"
            :options="dimensionSourceFieldOptions"
            :disable="dimensionForm.source_type !== 'table'"
          />
          <sweet-select
            v-model="dimensionForm.parent_field"
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
import SweetSelect from 'src/components/Select/SweetSelect.vue'
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
const activeWorkspaceTab = ref<'dimensions' | 'bindings' | 'debug' | 'checks'>('dimensions')

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
  {
    name: 'dimension_code',
    label: '维度',
    field: 'dimension_code',
    align: 'left',
    style: 'min-width: 220px; width: 260px;',
  },
  {
    name: 'field_code',
    label: '字段',
    field: 'field_code',
    align: 'left',
    style: 'min-width: 190px; width: 220px;',
  },
  {
    name: 'match_type',
    label: '匹配',
    field: 'match_type',
    align: 'left',
    style: 'min-width: 140px; width: 150px;',
  },
  {
    name: 'actions',
    label: '动作',
    field: 'actions',
    align: 'left',
    style: 'min-width: 260px; width: 320px;',
  },
  { name: 'required', label: '必配', field: 'required', align: 'center', style: 'width: 92px;' },
  { name: 'state', label: '启用', field: 'state', align: 'center', style: 'width: 92px;' },
  {
    name: 'row_actions',
    label: '操作',
    field: 'row_actions',
    align: 'center',
    style: 'width: 92px;',
  },
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

const configCheckItems = computed(() => {
  const disabledDimensions = dimensions.value.filter(
    (dimension) => dimension.state === false,
  ).length
  const noSourceDimensions = dimensions.value.filter(
    (dimension) => dimension.source_type !== 'table',
  ).length
  const selectedMenuMissingFields =
    selectedMenu.value && tableFieldOptions.value.length === 0 ? 1 : 0
  const disabledBindings = bindingRows.value.filter((binding) => binding.state === false).length

  return [
    {
      label: '停用维度',
      value: disabledDimensions,
      caption: '停用后不会作为有效授权维度使用',
      icon: 'pause_circle',
      color: disabledDimensions > 0 ? 'warning' : 'positive',
    },
    {
      label: '无来源维度',
      value: noSourceDimensions,
      caption: '无来源维度需要手工录入范围值',
      icon: 'edit_note',
      color: noSourceDimensions > 0 ? 'warning' : 'positive',
    },
    {
      label: '当前菜单无字段',
      value: selectedMenuMissingFields,
      caption: '绑定表没有字段时无法新增规则',
      icon: 'view_column',
      color: selectedMenuMissingFields > 0 ? 'negative' : 'positive',
    },
    {
      label: '停用绑定',
      value: disabledBindings,
      caption: '当前菜单中不会参与解析的绑定',
      icon: 'rule_folder',
      color: disabledBindings > 0 ? 'warning' : 'positive',
    },
  ]
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

const selectedMenuDisplayTitle = computed(() =>
  selectedMenu.value ? displayMenuTitle(selectedMenu.value) : '',
)

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
  dimensionOptions.value.find((option) => option.value === dimensionCode)?.label ||
  dimensionCode ||
  '-'

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

const saveBindingDialog = async () => {
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
  const nextRows = editingBindingLocalId.value
    ? bindingRows.value.map((item) => (item.local_id === editingBindingLocalId.value ? row : item))
    : [...bindingRows.value, row]
  const saved = await persistBindings(nextRows)
  if (saved) bindingDialogOpen.value = false
}

const removeBindingRow = (localId: string) => {
  void persistBindings(
    bindingRows.value.filter((row) => row.local_id !== localId),
    '绑定已删除',
  )
}

const actionsDisplay = (actions: string[]) => {
  return compactSelectionDisplay(actions, dataPermissionActionOptions, 2, '全部动作')
}

const actionsTooltip = (actions: string[]) => {
  return compactSelectionTooltip(actions, dataPermissionActionOptions)
}

const validateBindingRows = (rows: BindingRow[]) => {
  const uniqueKeys = new Set<string>()
  for (const row of rows) {
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

const persistBindings = async (rows: BindingRow[], successMessage = '绑定已保存') => {
  if (!selectedMenu.value || !validateBindingRows(rows)) return false
  savingBindings.value = true
  try {
    const payload = rows.map<DataPermissionBindingSaveItem>((row) => ({
      dimension_code: row.dimension_code,
      field_code: row.field_code,
      match_type: row.match_type || 'in',
      required: row.required !== false,
      actions: row.actions?.length ? row.actions : defaultBindingActions,
      state: row.state !== false,
    }))
    const result = await dataPermissionApi.saveMenuBindings(selectedMenu.value.id, payload)
    if (result.success) {
      $q.notify({ type: 'positive', position: 'top-right', message: successMessage })
      await onMenuChange(selectedMenu.value.id)
      return true
    }
    return false
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
  if (
    form.source_type === 'table' &&
    (!form.source_code.trim() || !form.label_field.trim() || !form.value_field.trim())
  ) {
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
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: #f6f7fb;
}

.data-permission-shell {
  flex: 1 1 auto;
  display: flex;
  min-height: 0;
  height: 100%;
  flex-direction: column;
  gap: 12px;
}

.data-permission-summary {
  display: grid;
  flex-shrink: 0;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.data-permission-summary-card {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 76px;
  padding: 14px 16px;
  border: 1px solid #e3e8f2;
  border-radius: 8px;
  background: #fff;
}

.data-permission-summary-card > .q-icon {
  width: 42px;
  height: 42px;
  display: grid;
  place-items: center;
  border-radius: 8px;
  color: var(--q-primary);
  background: rgba(105, 93, 238, 0.1);
  font-size: 24px;
}

.data-permission-summary-card strong {
  display: block;
  color: #172033;
  font-size: 22px;
  line-height: 1.1;
}

.data-permission-summary-card span {
  color: #7b879d;
  font-size: 13px;
}

.data-permission-workspace {
  flex: 1 1 auto;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
  border: 1px solid #e3e8f2;
  border-radius: 8px;
  background: #fff;
}

.data-permission-tabs {
  flex-shrink: 0;
  min-height: 58px;
  padding: 0 14px;
}

.data-permission-tabs :deep(.q-tab) {
  min-height: 58px;
  padding: 0 18px;
}

.data-permission-panels {
  flex: 1 1 auto;
  min-height: 0;
  overflow: hidden;
  background: #f8fafc;
}

.data-permission-tab-panel {
  height: 100%;
  display: flex;
  min-height: 0;
  flex-direction: column;
  padding: 14px;
  overflow: hidden;
}

.data-permission-panel-head {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 12px;
}

.data-permission-panel-title {
  color: #172033;
  font-size: 20px;
  font-weight: 800;
}

.data-permission-panel-caption {
  margin-top: 3px;
  color: #748198;
  font-size: 13px;
}

.data-permission-panel-actions {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  gap: 10px;
}

.data-permission-panel-toolbar {
  display: flex;
  flex-shrink: 0;
  gap: 10px;
  align-items: center;
  margin-bottom: 12px;
}

.data-permission-panel-toolbar .q-field {
  width: 320px;
  max-width: 100%;
}

.data-permission-table,
.data-permission-binding-table {
  flex: 1 1 auto;
  min-height: 0;
  height: auto;
  overflow: hidden;
}

.data-permission-table :deep(.q-table__middle),
.data-permission-binding-table :deep(.q-table__middle) {
  flex: 1 1 auto;
  min-height: 0;
}

.data-permission-menu-context {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
  margin-bottom: 12px;
  padding: 12px 14px;
  border: 1px solid #dfe6f3;
  border-radius: 8px;
  background: #fff;
}

.data-permission-menu-context > .q-icon {
  width: 42px;
  height: 42px;
  display: grid;
  place-items: center;
  border-radius: 8px;
  color: var(--q-primary);
  background: rgba(105, 93, 238, 0.1);
  font-size: 24px;
}

.data-permission-context-title {
  color: #172033;
  font-size: 17px;
  font-weight: 800;
}

.data-permission-context-meta {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-top: 5px;
}

.data-permission-debug-card {
  display: grid;
  grid-template-columns: minmax(260px, 420px) 180px max-content;
  gap: 10px;
  align-items: center;
  flex-shrink: 0;
  margin-bottom: 12px;
  padding: 14px;
  border: 1px solid #dfe6f3;
  border-radius: 8px;
  background: #fff;
}

.data-permission-check-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.data-permission-check-card {
  display: flex;
  gap: 12px;
  min-height: 120px;
  padding: 16px;
  border: 1px solid #e3e8f2;
  border-radius: 8px;
  background: #fff;
}

.data-permission-check-card > .q-icon {
  margin-top: 2px;
  font-size: 28px;
}

.data-permission-check-card strong {
  display: block;
  color: #172033;
  font-size: 26px;
  line-height: 1;
}

.data-permission-check-card span {
  display: block;
  margin-top: 8px;
  color: #172033;
  font-weight: 700;
}

.data-permission-check-card em {
  display: block;
  margin-top: 6px;
  color: #7b879d;
  font-size: 12px;
  font-style: normal;
  line-height: 1.5;
}

.data-permission-menu-select {
  flex: 1;
  min-width: 220px;
  max-width: 360px;
}

.dimension-action-btn {
  width: 30px;
  height: 30px;
}

.dimension-action-btn :deep(.q-icon) {
  font-size: 18px;
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
  min-height: 260px;
  flex: 1;
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

.data-permission-empty--panel {
  border: 1px dashed #d8e0ec;
  border-radius: 8px;
  background: #fff;
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
}
</style>
