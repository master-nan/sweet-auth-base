<template>
  <base-content class="q-pa-sm organization-readonly-page">
    <master-detail-page
      :mode="SysMasterDetailMode.TABLE"
      master-title="法人架构"
      :master-subtitle="treeSummary"
      :detail-title="selectedNode?.name || '法人详情'"
      detail-subtitle="组织主数据镜像"
      master-width="minmax(560px, 46%)"
      min-width="1080px"
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
          :loading="treeLoading"
          @click="loadTree"
        >
          <q-tooltip>{{ button.name }}</q-tooltip>
        </q-btn>
      </template>

      <template #master-toolbar>
        <div class="organization-tree-toolbar">
          <q-input
            v-model="keyword"
            outlined
            dense
            clearable
            class="full-width"
            placeholder="搜索法人编码、名称或简称"
          >
            <template #append>
              <q-icon name="search" />
            </template>
          </q-input>
        </div>
        <q-banner v-if="treeError" class="organization-tree-error">
          <template #avatar>
            <q-icon name="error_outline" color="negative" />
          </template>
          {{ treeError }}
        </q-banner>
      </template>

      <template #master-content>
        <tree-table
          v-if="filteredTree.length || treeLoading"
          class="fit sticky-header-table"
          :data="filteredTree"
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
              <q-icon name="account_balance" color="primary" size="18px" class="q-mr-sm" />
              <div class="ellipsis">{{ row.name }}</div>
            </div>
          </template>
          <template #body-cell-entity_type="{ row }">
            {{ entityTypeLabel(row.entity_type) }}
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

        <div v-else-if="!treeError" class="organization-tree-empty">
          <q-icon name="account_tree" size="44px" />
          <div>{{ keyword ? '没有匹配的法人主体' : '暂无法人主体数据' }}</div>
        </div>
      </template>

      <template #detail-context>
        <div v-if="selectedNode" class="organization-detail-context">
          <div class="organization-detail-icon">
            <q-icon name="account_balance" />
          </div>
          <div class="organization-detail-heading">
            <div class="organization-detail-title">{{ selectedNode.name }}</div>
            <div class="organization-detail-meta">
              <code>{{ selectedNode.code }}</code>
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
          <span>法人详情</span>
        </div>
      </template>

      <template #detail-content>
        <organization-read-only-detail
          :fields="detailFields"
          :loading="detailLoading"
          :error="detailError"
          empty-text="请选择左侧法人主体"
        />
      </template>
    </master-detail-page>
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'organization_legal_entity' })

import { computed, onMounted, ref } from 'vue'
import { date, useQuasar } from 'quasar'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import MasterDetailPage from 'src/components/MasterDetail/MasterDetailPage.vue'
import TreeTable from 'src/components/TreeTable/TreeTable.vue'
import OrganizationReadOnlyDetail from 'src/pages/organization/components/OrganizationReadOnlyDetail.vue'
import {
  getLegalEntityDetail,
  getLegalEntityTree,
  type LegalEntityDetail,
  type LegalEntityTreeNode,
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

const dictStore = useDictStore()
const $q = useQuasar()
const { all_buttons, top_buttons, line_buttons } = usePageButtons('organization_legal_entity')

const tree = ref<LegalEntityTreeNode[]>([])
const selectedNode = ref<LegalEntityTreeNode | null>(null)
const detail = ref<LegalEntityDetail | null>(null)
const keyword = ref('')
const treeLoading = ref(false)
const detailLoading = ref(false)
const treeError = ref('')
const detailError = ref('')
let detailRequestSequence = 0

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
    { name: 'name', label: '法人名称', field: 'name', align: 'left' },
    { name: 'code', label: '法人编码', field: 'code', align: 'left' },
    { name: 'entity_type', label: '主体类型', field: 'entity_type', align: 'left' },
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

const filteredTree = computed(() => filterLegalEntityTree(tree.value, keyword.value))
const treeSummary = computed(() => `${countTreeNodes(tree.value)} 个法人主体`)

const detailFields = computed<DetailField[]>(() => {
  if (!detail.value) return []
  const entity = detail.value
  return [
    { key: 'code', label: '法人编码', value: displayValue(entity.code), kind: 'code' },
    { key: 'name', label: '法人名称', value: displayValue(entity.name) },
    { key: 'short_name', label: '法人简称', value: displayValue(entity.short_name) },
    {
      key: 'entity_type',
      label: '主体类型',
      value: entityTypeLabel(entity.entity_type),
    },
    {
      key: 'status',
      label: '状态',
      value: statusLabel(entity.status),
      kind: 'status',
      color: statusColor(entity.status, false),
    },
    {
      key: 'parent_id',
      label: '上级法人ID',
      value: entity.parent_id ? String(entity.parent_id) : '-',
    },
    {
      key: 'unified_social_credit_code',
      label: '统一社会信用代码',
      value: displayValue(entity.unified_social_credit_code),
      kind: 'code',
    },
    {
      key: 'accounting_code',
      label: '核算编码',
      value: displayValue(entity.accounting_code),
      kind: 'code',
    },
    { key: 'valid_from', label: '有效期开始', value: formatDate(entity.valid_from) },
    { key: 'valid_to', label: '有效期结束', value: formatDate(entity.valid_to, '长期有效') },
    {
      key: 'gmt_modify',
      label: '镜像更新时间',
      value: formatDateTime(entity.gmt_modify),
    },
    {
      key: 'local_note',
      label: '平台备注',
      value: displayValue(entity.local_note),
      wide: true,
    },
  ]
})

const loadTree = async () => {
  treeLoading.value = true
  treeError.value = ''
  try {
    tree.value = await getLegalEntityTree({ only_effective: true })
    const current = selectedNode.value
      ? findLegalEntityNode(tree.value, selectedNode.value.legal_entity_id)
      : null
    const next = current || firstLegalEntityNode(tree.value)
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
    treeError.value = errorMessage(error, '法人架构加载失败')
  } finally {
    treeLoading.value = false
  }
}

const handleNodeSelected = async (node: LegalEntityTreeNode) => {
  selectedNode.value = node
  detail.value = null
  detailError.value = ''
  if (!canViewDetail.value) {
    detailError.value = '当前账号没有法人详情权限'
    return
  }

  const sequence = ++detailRequestSequence
  detailLoading.value = true
  try {
    const result = await getLegalEntityDetail(node.legal_entity_id, {
      only_effective: true,
    })
    if (sequence === detailRequestSequence) detail.value = result
  } catch (error) {
    if (sequence === detailRequestSequence) {
      detailError.value = errorMessage(error, '法人详情加载失败')
    }
  } finally {
    if (sequence === detailRequestSequence) detailLoading.value = false
  }
}

const entityTypeLabel = (value: string) =>
  dictStore.getDictLabel('org_legal_entity_type', value) || displayValue(value)

const statusLabel = (value: string) =>
  dictStore.getDictLabel('org_object_status', value) || displayValue(value)

onMounted(async () => {
  await dictStore.loadDicts(['org_legal_entity_type', 'org_object_status'])
  await loadTree()
})

function filterLegalEntityTree(
  nodes: LegalEntityTreeNode[],
  rawKeyword: string,
): LegalEntityTreeNode[] {
  const normalized = rawKeyword.trim().toLocaleLowerCase()
  if (!normalized) return nodes

  return nodes.flatMap((node) => {
    const children = filterLegalEntityTree(node.children || [], normalized)
    const matched = [node.code, node.name, node.short_name].some((value) =>
      String(value || '')
        .toLocaleLowerCase()
        .includes(normalized),
    )
    return matched || children.length ? [{ ...node, children }] : []
  })
}

function firstLegalEntityNode(nodes: LegalEntityTreeNode[]): LegalEntityTreeNode | null {
  for (const node of nodes) {
    if (!node.disabled) return node
    const child = firstLegalEntityNode(node.children || [])
    if (child) return child
  }
  return nodes[0] || null
}

function findLegalEntityNode(
  nodes: LegalEntityTreeNode[],
  legalEntityId: number,
): LegalEntityTreeNode | null {
  for (const node of nodes) {
    if (node.legal_entity_id === legalEntityId) return node
    const child = findLegalEntityNode(node.children || [], legalEntityId)
    if (child) return child
  }
  return null
}

function countTreeNodes(nodes: LegalEntityTreeNode[]): number {
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
  padding: 10px 12px;
  border-bottom: 1px solid #e3e8f2;
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
</style>
