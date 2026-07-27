<template>
  <base-content class="q-pa-sm organization-readonly-page">
    <master-detail-page
      :mode="SysMasterDetailMode.TABLE"
      master-title="管理架构"
      :master-subtitle="treeSummary"
      :detail-title="selectedNode?.name || '组织详情'"
      detail-subtitle="组织主数据镜像"
      master-width="minmax(600px, 48%)"
      min-width="1120px"
      min-height="calc(100vh - 150px)"
    >
      <template #master-actions>
        <q-btn
          v-for="button in refreshButtons"
          :key="button.id || button.code"
          v-bind="menuButtonDisplayProps(button)"
          round
          outline
          :color="button.color || 'primary'"
          :loading="treeLoading || structureLoading"
          @click="refreshPage"
        >
          <q-tooltip>{{ button.name }}</q-tooltip>
        </q-btn>
      </template>

      <template #master-toolbar>
        <div class="organization-tree-toolbar">
          <q-select
            v-model="selectedStructureCode"
            :options="structureOptions"
            option-value="code"
            option-label="label"
            emit-value
            map-options
            use-input
            hide-selected
            fill-input
            outlined
            dense
            :loading="structureLoading"
            label="管理架构"
            class="structure-select"
            @filter="handleStructureFilter"
            @update:model-value="handleStructureChange"
          >
            <template #no-option>
              <q-item>
                <q-item-section class="text-grey-7">暂无管理架构</q-item-section>
              </q-item>
            </template>
          </q-select>
          <q-input
            v-model="treeKeyword"
            outlined
            dense
            clearable
            class="structure-tree-search"
            placeholder="搜索组织编码或名称"
            @keyup.enter="loadTree"
            @clear="loadTree"
          >
            <template #append>
              <q-btn flat dense round icon="search" @click="loadTree">
                <q-tooltip>搜索组织</q-tooltip>
              </q-btn>
            </template>
          </q-input>
        </div>
        <q-banner v-if="treeError || structureError" class="organization-tree-error">
          <template #avatar>
            <q-icon name="error_outline" color="negative" />
          </template>
          {{ treeError || structureError }}
        </q-banner>
      </template>

      <template #master-content>
        <tree-table
          v-if="tree.length || treeLoading"
          class="fit sticky-header-table"
          :data="tree"
          :columns="treeColumns"
          :selected-row-id="selectedNode?.id ?? null"
          :loading="treeLoading"
          :dark="$q.dark.isActive"
          bordered
          flat
          separator="horizontal"
          @node-selected="handleNodeSelected"
        >
          <template #body-cell-name="{ row }">
            <div class="row items-center no-wrap">
              <q-icon name="apartment" color="primary" size="18px" class="q-mr-sm" />
              <div class="ellipsis">{{ row.name }}</div>
            </div>
          </template>
          <template #body-cell-unit_type="{ row }">
            {{ unitTypeLabel(row.unit_type) }}
          </template>
          <template #body-cell-status="{ row }">
            <q-chip dense square :color="statusColor(row.status, row.disabled)" text-color="white">
              {{ statusLabel(row.status) }}
            </q-chip>
          </template>
          <template #body-cell-actions="{ row }">
            <div class="text-center">
              <q-btn
                v-for="button in detailButtons"
                :key="button.id || button.code"
                v-bind="menuButtonDisplayProps(button)"
                flat
                dense
                round
                :color="button.color || 'primary'"
                @click.stop="handleNodeSelected(row)"
              >
                <q-tooltip>{{ button.name }}</q-tooltip>
              </q-btn>
            </div>
          </template>
        </tree-table>

        <div v-else-if="!treeError && !structureError" class="organization-tree-empty">
          <q-icon name="account_tree" size="44px" />
          <div>{{ selectedStructureCode ? '当前架构暂无组织数据' : '暂无可用管理架构' }}</div>
        </div>
      </template>

      <template #detail-context>
        <div v-if="selectedNode" class="organization-detail-context">
          <div class="organization-detail-icon">
            <q-icon name="apartment" />
          </div>
          <div class="organization-detail-heading">
            <div class="organization-detail-title">{{ selectedNode.name }}</div>
            <div class="organization-detail-meta">
              <code>{{ selectedNode.code }}</code>
              <q-chip dense square outline color="primary">
                {{ selectedStructure?.code || '-' }}
              </q-chip>
              <q-chip
                dense
                square
                :color="statusColor(selectedNode.status, selectedNode.disabled)"
                text-color="white"
              >
                {{ statusLabel(selectedNode.status) }}
              </q-chip>
            </div>
          </div>
        </div>
        <div v-else class="organization-detail-context organization-detail-context--empty">
          <q-icon name="description" size="28px" />
          <span>组织详情</span>
        </div>
      </template>

      <template #detail-content>
        <organization-read-only-detail
          :fields="detailFields"
          :loading="detailLoading"
          :error="detailError"
          empty-text="请选择左侧组织单元"
        />
      </template>
    </master-detail-page>
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'organization_structure' })

import { computed, onMounted, ref } from 'vue'
import { date, useQuasar } from 'quasar'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import MasterDetailPage from 'src/components/MasterDetail/MasterDetailPage.vue'
import TreeTable from 'src/components/TreeTable/TreeTable.vue'
import OrganizationReadOnlyDetail from 'src/pages/organization/components/OrganizationReadOnlyDetail.vue'
import {
  getOrgUnitDetail,
  getStructureOrgTree,
  queryStructureOptions,
  type OrganizationSelectorOption,
  type OrgUnitDetail,
  type StructureOrgTreeNode,
} from 'src/api/services/org'
import { usePageButtons } from 'src/composables/page-buttons'
import { useDictStore } from 'src/stores/dict'
import { SysMasterDetailMode } from 'src/types/enum'
import type { TableColumn } from 'src/types/global'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'

interface DetailField {
  key: string
  label: string
  value: string
  kind?: 'text' | 'code' | 'status'
  color?: string
  wide?: boolean
}

type SelectFilterUpdate = (callback: () => void) => void

const dictStore = useDictStore()
const $q = useQuasar()
const { all_buttons, top_buttons, line_buttons } = usePageButtons('organization_structure')

const structureOptions = ref<OrganizationSelectorOption[]>([])
const selectedStructureCode = ref<string | null>(null)
const tree = ref<StructureOrgTreeNode[]>([])
const selectedNode = ref<StructureOrgTreeNode | null>(null)
const detail = ref<OrgUnitDetail | null>(null)
const treeKeyword = ref('')
const structureLoading = ref(false)
const treeLoading = ref(false)
const detailLoading = ref(false)
const structureError = ref('')
const treeError = ref('')
const detailError = ref('')
let detailRequestSequence = 0

const selectedStructure = computed(
  () =>
    structureOptions.value.find((option) => option.code === selectedStructureCode.value) || null,
)
const refreshButtons = computed(() =>
  top_buttons.value.filter((button) => button.event_action === 'refresh'),
)
const detailButtons = computed(() =>
  line_buttons.value.filter((button) => button.event_action === 'detail'),
)
const canViewDetail = computed(() =>
  all_buttons.value.some((button) => button.event_action === 'detail'),
)

const treeColumns = computed<TableColumn[]>(() => {
  const columns: TableColumn[] = [
    { name: 'name', label: '组织名称', field: 'name', align: 'left' },
    { name: 'code', label: '组织编码', field: 'code', align: 'left' },
    { name: 'unit_type', label: '组织类型', field: 'unit_type', align: 'left' },
    { name: 'status', label: '状态', field: 'status', align: 'center' },
  ]
  if (detailButtons.value.length) {
    columns.push({
      name: 'actions',
      label: '操作',
      field: 'id',
      align: 'center',
      style: 'width: 64px',
    })
  }
  return columns
})

const treeSummary = computed(() => {
  const label = selectedStructure.value?.label || '请选择管理架构'
  return `${label} · ${countTreeNodes(tree.value)} 个组织节点`
})

const detailFields = computed<DetailField[]>(() => {
  if (!detail.value || !selectedNode.value) return []
  const unit = detail.value
  const primaryLegalEntity = unit.primary_legal_entity
    ? `${unit.primary_legal_entity.code} - ${unit.primary_legal_entity.name}`
    : unit.primary_legal_entity_id
      ? String(unit.primary_legal_entity_id)
      : '-'

  return [
    { key: 'code', label: '组织编码', value: displayValue(unit.code), kind: 'code' },
    { key: 'name', label: '组织名称', value: displayValue(unit.name) },
    { key: 'unit_type', label: '组织类型', value: unitTypeLabel(unit.unit_type) },
    {
      key: 'status',
      label: '状态',
      value: statusLabel(unit.status),
      kind: 'status',
      color: statusColor(unit.status, false),
    },
    {
      key: 'structure',
      label: '管理架构',
      value: selectedStructure.value?.label || '-',
    },
    {
      key: 'level',
      label: '架构层级',
      value: String(selectedNode.value.level),
    },
    {
      key: 'primary_legal_entity',
      label: '主要法人',
      value: primaryLegalEntity,
    },
    {
      key: 'node_status',
      label: '节点状态',
      value: statusLabel(selectedNode.value.node_status),
      kind: 'status',
      color: statusColor(selectedNode.value.node_status, selectedNode.value.disabled),
    },
    { key: 'valid_from', label: '有效期开始', value: formatDate(unit.valid_from) },
    { key: 'valid_to', label: '有效期结束', value: formatDate(unit.valid_to, '长期有效') },
    {
      key: 'gmt_modify',
      label: '镜像更新时间',
      value: formatDateTime(unit.gmt_modify),
    },
    {
      key: 'local_note',
      label: '平台备注',
      value: displayValue(unit.local_note),
      wide: true,
    },
  ]
})

const loadStructureOptions = async (keyword = '') => {
  structureLoading.value = true
  structureError.value = ''
  try {
    const previous = selectedStructure.value
    const normalizedKeyword = keyword.trim()
    const result = await queryStructureOptions({
      page: 1,
      num: 100,
      only_effective: true,
      ...(normalizedKeyword ? { keyword: normalizedKeyword } : {}),
    })
    const options = result.items
    if (previous && !options.some((option) => option.code === previous.code)) {
      options.unshift(previous)
    }
    structureOptions.value = options
    if (
      !selectedStructureCode.value ||
      !structureOptions.value.some((option) => option.code === selectedStructureCode.value)
    ) {
      selectedStructureCode.value = structureOptions.value[0]?.code || null
    }
  } catch (error) {
    structureOptions.value = []
    selectedStructureCode.value = null
    structureError.value = errorMessage(error, '管理架构加载失败')
  } finally {
    structureLoading.value = false
  }
}

const loadTree = async () => {
  const structure = selectedStructure.value
  treeError.value = ''
  if (!structure) {
    tree.value = []
    selectedNode.value = null
    detail.value = null
    return
  }

  treeLoading.value = true
  try {
    const normalizedKeyword = treeKeyword.value.trim()
    tree.value = await getStructureOrgTree({
      structure_id: structure.value,
      only_effective: true,
      ...(normalizedKeyword ? { keyword: normalizedKeyword } : {}),
    })
    const current = selectedNode.value
      ? findStructureNode(tree.value, selectedNode.value.structure_node_id)
      : null
    const next = current || firstStructureNode(tree.value)
    if (next) {
      await handleNodeSelected(next)
    } else {
      selectedNode.value = null
      detail.value = null
      detailError.value = ''
    }
  } catch (error) {
    tree.value = []
    selectedNode.value = null
    detail.value = null
    treeError.value = errorMessage(error, '管理组织树加载失败')
  } finally {
    treeLoading.value = false
  }
}

const refreshPage = async () => {
  await loadStructureOptions()
  await loadTree()
}

const handleStructureChange = async () => {
  treeKeyword.value = ''
  selectedNode.value = null
  detail.value = null
  detailError.value = ''
  await loadTree()
}

const handleStructureFilter = (value: string, update: SelectFilterUpdate, abort: () => void) => {
  void loadStructureOptions(value)
    .then(() => update(() => undefined))
    .catch(() => abort())
}

const handleNodeSelected = async (node: StructureOrgTreeNode) => {
  selectedNode.value = node
  detail.value = null
  detailError.value = ''
  if (!canViewDetail.value) {
    detailError.value = '当前账号没有组织详情权限'
    return
  }

  const sequence = ++detailRequestSequence
  detailLoading.value = true
  try {
    const result = await getOrgUnitDetail(node.org_unit_id, {
      only_effective: true,
    })
    if (sequence === detailRequestSequence) detail.value = result
  } catch (error) {
    if (sequence === detailRequestSequence) {
      detailError.value = errorMessage(error, '组织详情加载失败')
    }
  } finally {
    if (sequence === detailRequestSequence) detailLoading.value = false
  }
}

const unitTypeLabel = (value: string) =>
  dictStore.getDictLabel('org_unit_type', value) || displayValue(value)

const statusLabel = (value: string) =>
  dictStore.getDictLabel('org_object_status', value) || displayValue(value)

onMounted(async () => {
  await dictStore.loadDicts(['org_unit_type', 'org_object_status', 'org_structure_type'])
  await loadStructureOptions()
  await loadTree()
})

function firstStructureNode(nodes: StructureOrgTreeNode[]): StructureOrgTreeNode | null {
  for (const node of nodes) {
    if (!node.disabled) return node
    const child = firstStructureNode(node.children || [])
    if (child) return child
  }
  return nodes[0] || null
}

function findStructureNode(
  nodes: StructureOrgTreeNode[],
  structureNodeId: number,
): StructureOrgTreeNode | null {
  for (const node of nodes) {
    if (node.structure_node_id === structureNodeId) return node
    const child = findStructureNode(node.children || [], structureNodeId)
    if (child) return child
  }
  return null
}

function countTreeNodes(nodes: StructureOrgTreeNode[]): number {
  return nodes.reduce((total, node) => total + 1 + countTreeNodes(node.children || []), 0)
}

function displayValue(value: unknown): string {
  if (value === null || value === undefined || value === '') return '-'
  return String(value)
}

function formatDate(value?: string | null, empty = '-'): string {
  return value ? date.formatDate(value, 'YYYY-MM-DD') : empty
}

function formatDateTime(value?: string | null): string {
  return value ? date.formatDate(value, 'YYYY-MM-DD HH:mm:ss') : '-'
}

function statusColor(status: string, disabled: boolean): string {
  if (disabled || status === 'disabled') return 'grey-6'
  return status === 'enabled' ? 'positive' : 'warning'
}

function errorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message) return error.message
  return fallback
}
</script>

<style scoped lang="scss">
.organization-readonly-page {
  overflow: auto;
}

.organization-tree-toolbar {
  display: grid;
  grid-template-columns: minmax(220px, 0.9fr) minmax(240px, 1.1fr);
  gap: 10px;
  padding: 10px 12px;
  border-bottom: 1px solid #e3e8f2;
}

.structure-select,
.structure-tree-search {
  min-width: 0;
}

.organization-tree-error {
  border-bottom: 1px solid #ffcdd2;
  background: #fff5f5;
  color: #b71c1c;
}

.organization-tree-empty {
  height: 100%;
  min-height: 240px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: #8792a6;
}

.organization-detail-context {
  display: flex;
  align-items: center;
  gap: 12px;
}

.organization-detail-context--empty {
  color: #8792a6;
}

.organization-detail-icon {
  width: 38px;
  height: 38px;
  display: grid;
  place-items: center;
  flex: 0 0 auto;
  border: 1px solid #d9e2f1;
  border-radius: 8px;
  color: var(--q-primary);
  font-size: 21px;
}

.organization-detail-heading {
  min-width: 0;
}

.organization-detail-title {
  overflow: hidden;
  color: #172033;
  font-size: 16px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.organization-detail-meta {
  min-height: 24px;
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
  color: #657189;
  font-size: 12px;
}

@media (max-width: 1280px) {
  .organization-tree-toolbar {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
