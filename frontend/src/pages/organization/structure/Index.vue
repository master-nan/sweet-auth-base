<template>
  <base-content class="q-pa-sm organization-readonly-page">
    <div class="organization-browser-page">
      <header class="organization-page-header">
        <div class="organization-page-heading">
          <h1>组织架构</h1>
          <p>组织主数据镜像浏览</p>
        </div>

        <div class="organization-page-actions">
          <q-select
            v-if="showStructureSwitcher"
            v-model="selectedStructureCode"
            :options="structures"
            option-value="code"
            option-label="name"
            emit-value
            map-options
            outlined
            dense
            :loading="structureLoading"
            label="组织视图"
            class="structure-select"
            @update:model-value="handleStructureChange"
          >
            <template #no-option>
              <q-item>
                <q-item-section class="text-grey-7">暂无组织视图</q-item-section>
              </q-item>
            </template>
          </q-select>

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
        </div>
      </header>

      <q-banner v-if="treeError || structureError" class="organization-page-error">
        <template #avatar>
          <q-icon name="error_outline" color="negative" />
        </template>
        {{ treeError || structureError }}
      </q-banner>

      <main class="organization-browser-workspace">
        <q-card flat bordered class="organization-tree-panel">
          <q-card-section class="organization-panel-header">
            <div>
              <div class="organization-panel-title">组织树</div>
              <div class="organization-panel-subtitle">{{ treeSummary }}</div>
            </div>
          </q-card-section>

          <q-separator />

          <q-card-section class="organization-tree-toolbar">
            <q-input
              v-model="treeKeyword"
              outlined
              dense
              clearable
              placeholder="搜索组织编码或名称"
              :disable="!selectedStructure"
              @keyup.enter="loadTree"
              @clear="loadTree"
            >
              <template #append>
                <q-btn
                  flat
                  dense
                  round
                  icon="search"
                  :disable="!selectedStructure"
                  @click="loadTree"
                >
                  <q-tooltip>搜索组织</q-tooltip>
                </q-btn>
              </template>
            </q-input>
          </q-card-section>

          <q-separator />

          <q-card-section class="organization-tree-content">
            <organization-read-only-tree
              :nodes="displayTree"
              :selected-id="selectedNode?.structure_node_id ?? null"
              :loading="treeLoading"
              :expand-all="Boolean(treeKeyword.trim())"
              :empty-text="treeEmptyText"
              @select="handleNodeSelectedById"
            />
          </q-card-section>
        </q-card>

        <section class="organization-detail-panel" aria-label="组织详情">
          <q-card flat bordered class="organization-detail-summary">
            <q-card-section>
              <div v-if="selectedNode" class="organization-detail-context">
                <div class="organization-detail-icon">
                  <q-icon name="apartment" />
                </div>
                <div class="organization-detail-heading">
                  <div class="organization-detail-title">{{ selectedNode.name }}</div>
                  <div class="organization-detail-meta">
                    <span>{{ unitTypeLabel(selectedNode.unit_type) }}</span>
                    <span aria-hidden="true">·</span>
                    <span class="organization-detail-code">{{ selectedNode.code }}</span>
                    <q-chip
                      dense
                      square
                      outline
                      :color="statusColor(selectedNode.status, selectedNode.disabled)"
                    >
                      {{ statusLabel(selectedNode.status) }}
                    </q-chip>
                  </div>
                </div>
              </div>
              <div
                v-else
                class="organization-detail-context organization-detail-context--empty"
              >
                <q-icon name="description" size="28px" />
                <span>请选择组织节点</span>
              </div>
            </q-card-section>
          </q-card>

          <organization-read-only-detail
            :groups="detailGroups"
            :loading="detailLoading"
            :error="detailError"
            empty-text="请选择左侧组织单元"
          />
        </section>
      </main>
    </div>
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'organization_structure' })

import { computed, onMounted, ref } from 'vue'
import { date } from 'quasar'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import OrganizationReadOnlyDetail from 'src/pages/organization/components/OrganizationReadOnlyDetail.vue'
import type { OrganizationDetailGroup } from 'src/pages/organization/components/organization-read-only-detail'
import OrganizationReadOnlyTree from 'src/pages/organization/components/OrganizationReadOnlyTree.vue'
import type { OrganizationReadOnlyTreeNode } from 'src/pages/organization/components/organization-read-only-tree'
import {
  getOrgUnitDetail,
  getStructureOrgTree,
  queryStructures,
  type OrganizationStructure,
  type OrgUnitDetail,
  type StructureOrgTreeNode,
} from 'src/api/services/org'
import { usePageButtons } from 'src/composables/page-buttons'
import { useDictStore } from 'src/stores/dict'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'

const dictStore = useDictStore()
const { all_buttons, top_buttons } = usePageButtons('organization_structure')

const structures = ref<OrganizationStructure[]>([])
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
  () => structures.value.find((item) => item.code === selectedStructureCode.value) || null,
)
const showStructureSwitcher = computed(() => structures.value.length > 1)
const refreshButtons = computed(() =>
  top_buttons.value.filter((button) => button.event_action === 'refresh'),
)
const canViewDetail = computed(() =>
  all_buttons.value.some((button) => button.event_action === 'detail'),
)
const displayTree = computed(() => mapStructureTree(tree.value))
const treeSummary = computed(() => {
  if (!selectedStructure.value) {
    return structures.value.length > 1
      ? '组织主数据镜像 · 请选择组织视图'
      : '组织主数据镜像 · 暂无组织视图'
  }
  return `组织主数据镜像 · ${selectedStructure.value.name} · ${countTreeNodes(tree.value)} 个组织`
})
const treeEmptyText = computed(() => {
  if (!selectedStructure.value) {
    return structures.value.length > 1 ? '请选择组织视图' : '暂无可用组织视图'
  }
  return treeKeyword.value ? '当前视图没有匹配的组织' : '当前视图暂无组织数据'
})

const detailGroups = computed<OrganizationDetailGroup[]>(() => {
  if (!detail.value || !selectedNode.value) return []
  const unit = detail.value
  const primaryLegalEntity = unit.primary_legal_entity
    ? `${unit.primary_legal_entity.code} - ${unit.primary_legal_entity.name}`
    : unit.primary_legal_entity_id
      ? String(unit.primary_legal_entity_id)
      : '-'

  return [
    {
      key: 'basic',
      title: '基础信息',
      icon: 'domain',
      fields: [
        { key: 'name', label: '组织名称', value: displayValue(unit.name) },
        { key: 'code', label: '组织编码', value: displayValue(unit.code), kind: 'code' },
        { key: 'unit_type', label: '组织类型', value: unitTypeLabel(unit.unit_type) },
      ],
    },
    {
      key: 'ownership',
      title: '归属信息',
      icon: 'account_tree',
      fields: [
        {
          key: 'structure',
          label: '组织视图',
          value: selectedStructure.value?.name || '-',
        },
        {
          key: 'primary_legal_entity',
          label: '主要法人',
          value: primaryLegalEntity,
        },
      ],
    },
    {
      key: 'status',
      title: '状态信息',
      icon: 'event_available',
      fields: [
        {
          key: 'status',
          label: '状态',
          value: statusLabel(unit.status),
          kind: 'status',
          color: statusColor(unit.status, false),
        },
        { key: 'valid_from', label: '有效期开始', value: formatDate(unit.valid_from) },
        {
          key: 'valid_to',
          label: '有效期结束',
          value: formatDate(unit.valid_to, '长期有效'),
        },
      ],
    },
    {
      key: 'mirror',
      title: '镜像信息',
      icon: 'sync',
      fields: [
        {
          key: 'gmt_modify',
          label: '更新时间',
          value: formatDateTime(unit.gmt_modify),
        },
        {
          key: 'local_note',
          label: '平台备注',
          value: displayValue(unit.local_note),
          wide: true,
        },
      ],
    },
  ]
})

const loadStructures = async () => {
  structureLoading.value = true
  structureError.value = ''
  try {
    const previousCode = selectedStructureCode.value
    const result = await queryStructures({
      page: 1,
      num: 100,
      only_effective: true,
    })
    structures.value = result.items

    if (structures.value.length === 1) {
      selectedStructureCode.value = structures.value[0]!.code
      return
    }
    if (previousCode && structures.value.some((item) => item.code === previousCode)) {
      selectedStructureCode.value = previousCode
      return
    }

    const defaults = structures.value.filter((item) => item.is_default)
    selectedStructureCode.value = defaults.length === 1 ? defaults[0]!.code : null
  } catch (error) {
    structures.value = []
    selectedStructureCode.value = null
    structureError.value = errorMessage(error, '组织视图加载失败')
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
      structure_id: structure.id,
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
    treeError.value = errorMessage(error, '组织树加载失败')
  } finally {
    treeLoading.value = false
  }
}

const refreshPage = async () => {
  await loadStructures()
  await loadTree()
}

const handleStructureChange = async () => {
  treeKeyword.value = ''
  selectedNode.value = null
  detail.value = null
  detailError.value = ''
  await loadTree()
}

const handleNodeSelectedById = async (structureNodeId: number) => {
  const node = findStructureNode(tree.value, structureNodeId)
  if (node) await handleNodeSelected(node)
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
  await dictStore.loadDicts(['org_unit_type', 'org_object_status'])
  await loadStructures()
  await loadTree()
})

function mapStructureTree(nodes: StructureOrgTreeNode[]): OrganizationReadOnlyTreeNode[] {
  return nodes.map((node) => ({
    id: node.structure_node_id,
    code: node.code,
    name: node.name,
    icon: 'apartment',
    typeLabel: unitTypeLabel(node.unit_type),
    statusLabel: statusLabel(node.status),
    statusColor: statusColor(node.status, node.disabled),
    muted: node.disabled,
    children: mapStructureTree(node.children || []),
  }))
}

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
  if (
    typeof value === 'string' ||
    typeof value === 'number' ||
    typeof value === 'boolean'
  ) {
    return String(value)
  }
  return '-'
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

.organization-browser-page {
  min-width: 980px;
}

.organization-page-header {
  min-height: 72px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 12px 16px;
  margin-bottom: 12px;
  border-bottom: 1px solid #dfe5ee;
  background: #fff;
}

.organization-page-heading {
  min-width: 0;
}

.organization-page-heading h1 {
  margin: 0;
  color: #172033;
  font-size: 20px;
  font-weight: 750;
  line-height: 28px;
}

.organization-page-heading p {
  margin: 2px 0 0;
  color: #718096;
  font-size: 13px;
  line-height: 20px;
}

.organization-page-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
}

.structure-select {
  width: 240px;
}

.organization-page-error {
  margin-bottom: 12px;
  border: 1px solid #ffcdd2;
  background: #fff5f5;
  color: #b71c1c;
}

.organization-browser-workspace {
  height: calc(100vh - 184px);
  min-height: 560px;
  display: grid;
  grid-template-columns: minmax(370px, 40%) minmax(0, 1fr);
  align-items: stretch;
  gap: 14px;
}

.organization-tree-panel {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-color: #dfe5ee;
  border-radius: 8px;
}

.organization-panel-header {
  min-height: 62px;
  display: flex;
  align-items: center;
  padding: 10px 14px;
}

.organization-panel-title {
  color: #24324a;
  font-size: 15px;
  font-weight: 700;
  line-height: 22px;
}

.organization-panel-subtitle {
  margin-top: 2px;
  color: #7b8798;
  font-size: 12px;
  line-height: 18px;
}

.organization-tree-toolbar {
  padding: 10px 12px;
}

.organization-tree-content {
  flex: 1;
  min-height: 0;
  padding: 0;
  overflow: hidden;
}

.organization-detail-panel {
  min-width: 0;
  min-height: 0;
  overflow: auto;
  padding-right: 2px;
}

.organization-detail-summary {
  margin-bottom: 12px;
  border-color: #dfe5ee;
  border-radius: 8px;
  background: #fff;
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

.organization-detail-code {
  color: #7a879b;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
}

@media (max-width: 1180px) {
  .organization-browser-workspace {
    grid-template-columns: minmax(360px, 42%) minmax(0, 1fr);
  }
}
</style>
