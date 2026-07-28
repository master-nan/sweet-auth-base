<template>
  <base-content class="q-pa-sm organization-readonly-page">
    <div class="organization-browser-page">
      <header class="organization-page-header">
        <div class="organization-page-heading">
          <h1>组织架构</h1>
          <p>统一浏览管理组织与法人主体镜像</p>
        </div>

        <div class="organization-page-actions">
          <q-select
            v-model="architectureMode"
            :options="architectureModeOptions"
            option-value="value"
            option-label="label"
            emit-value
            map-options
            outlined
            dense
            label="架构类型"
            class="architecture-mode-toggle"
            @update:model-value="handleArchitectureModeChange"
          />

          <q-select
            v-if="architectureMode === 'management' && showStructureSwitcher"
            v-model="selectedStructureCode"
            :options="structures"
            option-value="code"
            option-label="name"
            emit-value
            map-options
            outlined
            dense
            :loading="structureLoading"
            label="管理视图"
            class="structure-select"
            @update:model-value="handleStructureChange"
          >
            <template #no-option>
              <q-item>
                <q-item-section class="text-grey-7">暂无管理视图</q-item-section>
              </q-item>
            </template>
          </q-select>

          <q-btn
            v-for="button in refreshButtons"
            :key="button.id || button.code"
            :icon="button.icon || 'refresh'"
            :aria-label="button.name"
            round
            flat
            dense
            :color="button.color || 'primary'"
            :loading="treeLoading || structureLoading"
            class="organization-refresh-button"
            @click="refreshPage"
          >
            <q-tooltip>{{ button.name }}</q-tooltip>
          </q-btn>
        </div>
      </header>

      <q-banner v-if="pageError" class="organization-page-error">
        <template #avatar>
          <q-icon name="error_outline" color="negative" />
        </template>
        {{ pageError }}
      </q-banner>

      <main class="organization-browser-workspace">
        <q-card flat bordered class="organization-tree-panel">
          <q-card-section class="organization-panel-header">
            <div>
              <div class="organization-panel-title">{{ treeTitle }}</div>
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
              :placeholder="searchPlaceholder"
              :disable="architectureMode === 'management' && !selectedStructure"
              @keyup.enter="handleSearch"
              @clear="handleSearch"
            >
              <template #append>
                <q-btn
                  flat
                  dense
                  round
                  icon="search"
                  :disable="architectureMode === 'management' && !selectedStructure"
                  @click="handleSearch"
                >
                  <q-tooltip>搜索</q-tooltip>
                </q-btn>
              </template>
            </q-input>
          </q-card-section>

          <q-separator />

          <q-card-section class="organization-tree-content">
            <organization-read-only-tree
              :nodes="displayTree"
              :selected-id="selectedTreeNodeId"
              :loading="treeLoading"
              :expand-all="Boolean(treeKeyword.trim())"
              :empty-text="treeEmptyText"
              @select="handleNodeSelectedById"
            />
          </q-card-section>
        </q-card>

        <q-card flat bordered class="organization-detail-panel">
          <q-card-section class="organization-detail-summary">
            <div v-if="selectionSummary" class="organization-detail-context">
              <div class="organization-detail-icon">
                <q-icon :name="selectionSummary.icon" />
              </div>
              <div class="organization-detail-heading">
                <div class="organization-detail-title">{{ selectionSummary.name }}</div>
                <div class="organization-detail-meta">
                  <span>{{ selectionSummary.typeLabel }}</span>
                  <span aria-hidden="true">·</span>
                  <span class="organization-detail-code">{{ selectionSummary.code }}</span>
                  <q-chip
                    dense
                    square
                    outline
                    :color="
                      statusColor(selectionSummary.status, selectionSummary.disabled)
                    "
                  >
                    {{ statusLabel(selectionSummary.status) }}
                  </q-chip>
                </div>
              </div>
            </div>
            <div
              v-else
              class="organization-detail-context organization-detail-context--empty"
            >
              <q-icon name="description" size="28px" />
              <span>{{ detailEmptyText }}</span>
            </div>
          </q-card-section>

          <q-separator />

          <q-card-section class="organization-detail-content">
            <organization-read-only-detail
              :groups="detailGroups"
              :loading="detailLoading"
              :error="detailError"
              :empty-text="detailEmptyText"
            />
          </q-card-section>
        </q-card>
      </main>
    </div>
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'organization_structure' })

import { computed, onMounted, ref } from 'vue'
import { date } from 'quasar'
import { useRoute, useRouter } from 'vue-router'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import OrganizationReadOnlyDetail from 'src/pages/organization/components/OrganizationReadOnlyDetail.vue'
import type { OrganizationDetailGroup } from 'src/pages/organization/components/organization-read-only-detail'
import OrganizationReadOnlyTree from 'src/pages/organization/components/OrganizationReadOnlyTree.vue'
import type { OrganizationReadOnlyTreeNode } from 'src/pages/organization/components/organization-read-only-tree'
import {
  getLegalEntityDetail,
  getLegalEntityTree,
  getOrgUnitDetail,
  getStructureOrgTree,
  queryStructures,
  type LegalEntityDetail,
  type LegalEntityTreeNode,
  type OrganizationStructure,
  type OrgUnitDetail,
  type StructureOrgTreeNode,
} from 'src/api/services/org'
import { usePageButtons } from 'src/composables/page-buttons'
import { useDictStore } from 'src/stores/dict'

type ArchitectureMode = 'management' | 'legal'

interface SelectionSummary {
  name: string
  code: string
  typeLabel: string
  status: string
  disabled: boolean
  icon: string
}

const dictStore = useDictStore()
const { top_buttons } = usePageButtons('organization_structure')
const route = useRoute()
const router = useRouter()

const architectureMode = ref<ArchitectureMode>(
  route.query.architecture === 'legal' ? 'legal' : 'management',
)
const structures = ref<OrganizationStructure[]>([])
const selectedStructureCode = ref<string | null>(null)
const managementTree = ref<StructureOrgTreeNode[]>([])
const legalTree = ref<LegalEntityTreeNode[]>([])
const selectedManagementNode = ref<StructureOrgTreeNode | null>(null)
const selectedLegalNode = ref<LegalEntityTreeNode | null>(null)
const managementDetail = ref<OrgUnitDetail | null>(null)
const legalDetail = ref<LegalEntityDetail | null>(null)
const treeKeyword = ref('')
const structureLoading = ref(false)
const treeLoading = ref(false)
const detailLoading = ref(false)
const structureError = ref('')
const treeError = ref('')
const detailError = ref('')
const legalTreeLoaded = ref(false)
let detailRequestSequence = 0

const architectureModeOptions = [
  { label: '管理架构', value: 'management' },
  { label: '法人架构', value: 'legal' },
]

const selectedStructure = computed(
  () => structures.value.find((item) => item.code === selectedStructureCode.value) || null,
)
const showStructureSwitcher = computed(() => structures.value.length > 1)
const refreshButtons = computed(() =>
  top_buttons.value.filter((button) => button.event_action === 'refresh'),
)
const pageError = computed(() => structureError.value || treeError.value)
const filteredLegalTree = computed(() =>
  filterLegalEntityTree(legalTree.value, treeKeyword.value),
)
const displayTree = computed<OrganizationReadOnlyTreeNode[]>(() =>
  architectureMode.value === 'management'
    ? mapStructureTree(managementTree.value)
    : mapLegalEntityTree(filteredLegalTree.value),
)
const selectedTreeNodeId = computed(() =>
  architectureMode.value === 'management'
    ? selectedManagementNode.value?.structure_node_id ?? null
    : selectedLegalNode.value?.legal_entity_id ?? null,
)
const treeTitle = computed(() =>
  architectureMode.value === 'management' ? '管理组织树' : '法人树',
)
const treeSummary = computed(() => {
  if (architectureMode.value === 'legal') {
    return `法人主体镜像 · ${countLegalEntityNodes(legalTree.value)} 个法人主体`
  }
  if (!selectedStructure.value) return '管理组织镜像 · 暂无管理视图'
  return `管理组织镜像 · ${selectedStructure.value.name} · ${countStructureNodes(
    managementTree.value,
  )} 个组织`
})
const searchPlaceholder = computed(() =>
  architectureMode.value === 'management'
    ? '搜索组织编码或名称'
    : '搜索法人编码、名称或简称',
)
const treeEmptyText = computed(() => {
  if (architectureMode.value === 'legal') {
    return treeKeyword.value ? '没有匹配的法人主体' : '暂无法人主体数据'
  }
  if (!selectedStructure.value) return '暂无可用管理视图'
  return treeKeyword.value ? '当前视图没有匹配的组织' : '当前视图暂无组织数据'
})
const detailEmptyText = computed(() =>
  architectureMode.value === 'management'
    ? '请选择左侧管理组织'
    : '请选择左侧法人主体',
)
const selectionSummary = computed<SelectionSummary | null>(() => {
  if (architectureMode.value === 'legal') {
    const node = selectedLegalNode.value
    if (!node) return null
    return {
      name: node.name,
      code: node.code,
      typeLabel: legalEntityTypeLabel(node.entity_type),
      status: node.status,
      disabled: node.disabled,
      icon: 'account_balance',
    }
  }

  const node = selectedManagementNode.value
  if (!node) return null
  return {
    name: node.name,
    code: node.code,
    typeLabel: unitTypeLabel(node.unit_type),
    status: node.status,
    disabled: node.disabled,
    icon: 'corporate_fare',
  }
})
const detailGroups = computed<OrganizationDetailGroup[]>(() =>
  architectureMode.value === 'management'
    ? managementDetailGroups(managementDetail.value)
    : legalEntityDetailGroups(legalDetail.value),
)

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

    if (previousCode && structures.value.some((item) => item.code === previousCode)) {
      selectedStructureCode.value = previousCode
      return
    }

    const defaultStructure = structures.value.find((item) => item.is_default)
    selectedStructureCode.value =
      defaultStructure?.code || structures.value[0]?.code || null
  } catch (error) {
    structures.value = []
    selectedStructureCode.value = null
    structureError.value = errorMessage(error, '管理视图加载失败')
  } finally {
    structureLoading.value = false
  }
}

const loadManagementTree = async () => {
  const structure = selectedStructure.value
  treeError.value = ''
  if (!structure) {
    managementTree.value = []
    selectedManagementNode.value = null
    managementDetail.value = null
    return
  }

  treeLoading.value = true
  try {
    const normalizedKeyword = treeKeyword.value.trim()
    managementTree.value = await getStructureOrgTree({
      structure_id: structure.id,
      only_effective: true,
      ...(normalizedKeyword ? { keyword: normalizedKeyword } : {}),
    })
    const current = selectedManagementNode.value
      ? findStructureNode(
          managementTree.value,
          selectedManagementNode.value.structure_node_id,
        )
      : null
    const next = current || firstStructureNode(managementTree.value)
    if (next) {
      await selectManagementNode(next)
    } else {
      selectedManagementNode.value = null
      managementDetail.value = null
      detailError.value = ''
    }
  } catch (error) {
    managementTree.value = []
    selectedManagementNode.value = null
    managementDetail.value = null
    treeError.value = errorMessage(error, '管理组织树加载失败')
  } finally {
    treeLoading.value = false
  }
}

const loadLegalTree = async () => {
  treeLoading.value = true
  treeError.value = ''
  try {
    legalTree.value = await getLegalEntityTree({ only_effective: true })
    legalTreeLoaded.value = true
    const current = selectedLegalNode.value
      ? findLegalEntityNode(legalTree.value, selectedLegalNode.value.legal_entity_id)
      : null
    const next = current || firstLegalEntityNode(legalTree.value)
    if (next) {
      await selectLegalNode(next)
    } else {
      selectedLegalNode.value = null
      legalDetail.value = null
      detailError.value = ''
    }
  } catch (error) {
    legalTree.value = []
    selectedLegalNode.value = null
    legalDetail.value = null
    legalTreeLoaded.value = false
    treeError.value = errorMessage(error, '法人架构加载失败')
  } finally {
    treeLoading.value = false
  }
}

const refreshPage = async () => {
  if (architectureMode.value === 'legal') {
    await loadLegalTree()
    return
  }
  await loadStructures()
  await loadManagementTree()
}

const handleArchitectureModeChange = async () => {
  detailRequestSequence++
  treeKeyword.value = ''
  treeError.value = ''
  detailError.value = ''
  await router.replace({
    query: {
      ...route.query,
      architecture: architectureMode.value,
    },
  })

  if (architectureMode.value === 'legal') {
    if (!legalTreeLoaded.value) {
      await loadLegalTree()
    } else if (selectedLegalNode.value && !legalDetail.value) {
      await selectLegalNode(selectedLegalNode.value)
    }
    return
  }

  if (!structures.value.length) await loadStructures()
  if (!managementTree.value.length) {
    await loadManagementTree()
  } else if (selectedManagementNode.value && !managementDetail.value) {
    await selectManagementNode(selectedManagementNode.value)
  }
}

const handleStructureChange = async () => {
  treeKeyword.value = ''
  selectedManagementNode.value = null
  managementDetail.value = null
  detailError.value = ''
  await loadManagementTree()
}

const handleSearch = async () => {
  if (architectureMode.value === 'management') await loadManagementTree()
}

const handleNodeSelectedById = async (nodeId: number) => {
  if (architectureMode.value === 'legal') {
    const node = findLegalEntityNode(filteredLegalTree.value, nodeId)
    if (node) await selectLegalNode(node)
    return
  }

  const node = findStructureNode(managementTree.value, nodeId)
  if (node) await selectManagementNode(node)
}

const selectManagementNode = async (node: StructureOrgTreeNode) => {
  selectedManagementNode.value = node
  managementDetail.value = null
  detailError.value = ''

  const sequence = ++detailRequestSequence
  detailLoading.value = true
  try {
    const result = await getOrgUnitDetail(node.org_unit_id, {
      only_effective: true,
    })
    if (sequence === detailRequestSequence) managementDetail.value = result
  } catch (error) {
    if (sequence === detailRequestSequence) {
      detailError.value = errorMessage(error, '管理组织详情加载失败')
    }
  } finally {
    if (sequence === detailRequestSequence) detailLoading.value = false
  }
}

const selectLegalNode = async (node: LegalEntityTreeNode) => {
  selectedLegalNode.value = node
  legalDetail.value = null
  detailError.value = ''

  const sequence = ++detailRequestSequence
  detailLoading.value = true
  try {
    const result = await getLegalEntityDetail(node.legal_entity_id, {
      only_effective: true,
    })
    if (sequence === detailRequestSequence) legalDetail.value = result
  } catch (error) {
    if (sequence === detailRequestSequence) {
      detailError.value = errorMessage(error, '法人详情加载失败')
    }
  } finally {
    if (sequence === detailRequestSequence) detailLoading.value = false
  }
}

const unitTypeLabel = (value: string) =>
  dictStore.getDictLabel('org_unit_type', value) || displayValue(value)

const legalEntityTypeLabel = (value: string) =>
  dictStore.getDictLabel('org_legal_entity_type', value) || displayValue(value)

const statusLabel = (value: string) =>
  dictStore.getDictLabel('org_object_status', value) || displayValue(value)

onMounted(async () => {
  await dictStore.loadDicts([
    'org_unit_type',
    'org_legal_entity_type',
    'org_object_status',
  ])
  if (architectureMode.value === 'legal') {
    await loadLegalTree()
    return
  }
  await loadStructures()
  await loadManagementTree()
})

function managementDetailGroups(
  detail: OrgUnitDetail | null,
): OrganizationDetailGroup[] {
  if (!detail) return []
  const primaryLegalEntity = detail.primary_legal_entity
    ? `${detail.primary_legal_entity.code} - ${detail.primary_legal_entity.name}`
    : detail.primary_legal_entity_id
      ? String(detail.primary_legal_entity_id)
      : '-'

  return [
    {
      key: 'management-detail',
      title: '管理组织详情',
      fields: [
        { key: 'name', label: '组织名称', value: displayValue(detail.name) },
        {
          key: 'code',
          label: '组织编码',
          value: displayValue(detail.code),
          kind: 'code',
        },
        {
          key: 'unit_type',
          label: '组织类型',
          value: unitTypeLabel(detail.unit_type),
        },
        {
          key: 'structure',
          label: '管理视图',
          value: selectedStructure.value?.name || '-',
        },
        {
          key: 'primary_legal_entity',
          label: '主要法人',
          value: primaryLegalEntity,
        },
        {
          key: 'status',
          label: '状态',
          value: statusLabel(detail.status),
          kind: 'status',
          color: statusColor(detail.status, false),
        },
        {
          key: 'valid_from',
          label: '有效期开始',
          value: formatDate(detail.valid_from),
        },
        {
          key: 'valid_to',
          label: '有效期结束',
          value: formatDate(detail.valid_to, '长期有效'),
        },
        {
          key: 'gmt_modify',
          label: '更新时间',
          value: formatDateTime(detail.gmt_modify),
        },
        {
          key: 'local_note',
          label: '平台备注',
          value: displayValue(detail.local_note),
          wide: true,
        },
      ],
    },
  ]
}

function legalEntityDetailGroups(
  detail: LegalEntityDetail | null,
): OrganizationDetailGroup[] {
  if (!detail) return []
  const parent = detail.parent_id
    ? findLegalEntityNode(legalTree.value, detail.parent_id)
    : null

  return [
    {
      key: 'legal-detail',
      title: '法人详情',
      fields: [
        { key: 'name', label: '法人名称', value: displayValue(detail.name) },
        {
          key: 'short_name',
          label: '法人简称',
          value: displayValue(detail.short_name),
        },
        {
          key: 'code',
          label: '法人编码',
          value: displayValue(detail.code),
          kind: 'code',
        },
        {
          key: 'entity_type',
          label: '主体类型',
          value: legalEntityTypeLabel(detail.entity_type),
        },
        {
          key: 'parent',
          label: '上级法人',
          value: parent ? `${parent.code} - ${parent.name}` : '-',
        },
        {
          key: 'unified_social_credit_code',
          label: '统一社会信用代码',
          value: displayValue(detail.unified_social_credit_code),
          kind: 'code',
        },
        {
          key: 'accounting_code',
          label: '核算编码',
          value: displayValue(detail.accounting_code),
          kind: 'code',
        },
        {
          key: 'status',
          label: '状态',
          value: statusLabel(detail.status),
          kind: 'status',
          color: statusColor(detail.status, false),
        },
        {
          key: 'valid_from',
          label: '有效期开始',
          value: formatDate(detail.valid_from),
        },
        {
          key: 'valid_to',
          label: '有效期结束',
          value: formatDate(detail.valid_to, '长期有效'),
        },
        {
          key: 'gmt_modify',
          label: '更新时间',
          value: formatDateTime(detail.gmt_modify),
        },
        {
          key: 'local_note',
          label: '平台备注',
          value: displayValue(detail.local_note),
          wide: true,
        },
      ],
    },
  ]
}

function mapStructureTree(
  nodes: StructureOrgTreeNode[],
): OrganizationReadOnlyTreeNode[] {
  return nodes.map((node) => ({
    id: node.structure_node_id,
    code: node.code,
    name: node.name,
    icon: 'corporate_fare',
    typeLabel: unitTypeLabel(node.unit_type),
    statusLabel: statusLabel(node.status),
    statusColor: statusColor(node.status, node.disabled),
    muted: node.disabled,
    children: mapStructureTree(node.children || []),
  }))
}

function mapLegalEntityTree(
  nodes: LegalEntityTreeNode[],
): OrganizationReadOnlyTreeNode[] {
  return nodes.map((node) => ({
    id: node.legal_entity_id,
    code: node.code,
    name: node.name,
    icon: 'account_balance',
    statusLabel: statusLabel(node.status),
    statusColor: statusColor(node.status, node.disabled),
    muted: node.disabled,
    children: mapLegalEntityTree(node.children || []),
  }))
}

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

function firstStructureNode(
  nodes: StructureOrgTreeNode[],
): StructureOrgTreeNode | null {
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

function firstLegalEntityNode(
  nodes: LegalEntityTreeNode[],
): LegalEntityTreeNode | null {
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

function countStructureNodes(nodes: StructureOrgTreeNode[]): number {
  return nodes.reduce(
    (total, node) => total + 1 + countStructureNodes(node.children || []),
    0,
  )
}

function countLegalEntityNodes(nodes: LegalEntityTreeNode[]): number {
  return nodes.reduce(
    (total, node) => total + 1 + countLegalEntityNodes(node.children || []),
    0,
  )
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
  min-width: 0;
}

.organization-page-header {
  min-height: 76px;
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
  flex-wrap: wrap;
  gap: 10px;
}

.organization-refresh-button {
  width: 36px;
  height: 36px;
  min-width: 36px;
  min-height: 36px;
  flex: 0 0 36px;
}

.architecture-mode-toggle {
  width: 168px;
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
  height: calc(100vh - 188px);
  min-height: 560px;
  display: grid;
  grid-template-columns: minmax(390px, 40%) minmax(0, 1fr);
  align-items: stretch;
  gap: 14px;
}

.organization-tree-panel,
.organization-detail-panel {
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  border-color: #dfe5ee;
  border-radius: 8px;
  background: #fff;
}

.organization-tree-panel {
  display: flex;
  flex-direction: column;
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
  overflow: auto;
}

.organization-detail-summary {
  min-height: 86px;
  display: flex;
  align-items: center;
  padding: 16px 18px;
}

.organization-detail-content {
  padding: 8px 18px 20px;
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
  width: 36px;
  height: 36px;
  display: grid;
  place-items: center;
  flex: 0 0 auto;
  color: #667085;
  font-size: 24px;
}

.organization-detail-heading {
  min-width: 0;
}

.organization-detail-title {
  overflow: hidden;
  color: #172033;
  font-size: 17px;
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

:global(.body--dark .organization-page-header) {
  border-bottom-color: var(--app-dark-border);
  background: var(--app-dark-surface);
}

:global(.body--dark .organization-page-heading h1),
:global(.body--dark .organization-panel-title),
:global(.body--dark .organization-detail-title) {
  color: var(--app-dark-heading);
}

:global(.body--dark .organization-page-heading p),
:global(.body--dark .organization-panel-subtitle),
:global(.body--dark .organization-detail-meta),
:global(.body--dark .organization-detail-code),
:global(.body--dark .organization-detail-context--empty) {
  color: var(--app-dark-muted);
}

:global(.body--dark .organization-tree-panel),
:global(.body--dark .organization-detail-panel) {
  border-color: var(--app-dark-border);
  background: var(--app-dark-surface);
}

:global(.body--dark .organization-detail-icon) {
  color: var(--app-dark-muted);
}

:global(.body--dark .organization-page-error) {
  border-color: rgba(239, 83, 80, 0.45);
  background: rgba(239, 83, 80, 0.12);
  color: #ffb4b4;
}

@media (max-width: 1180px) {
  .organization-browser-workspace {
    grid-template-columns: minmax(380px, 43%) minmax(0, 1fr);
  }
}
</style>
