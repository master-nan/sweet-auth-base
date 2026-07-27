<template>
  <base-content class="q-pa-sm organization-readonly-page">
    <div class="organization-browser-page">
      <header class="organization-page-header">
        <div class="organization-page-heading">
          <h1>法人主体</h1>
          <p>法人主数据镜像浏览</p>
        </div>

        <div class="organization-page-actions">
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
        </div>
      </header>

      <q-banner v-if="treeError" class="organization-page-error">
        <template #avatar>
          <q-icon name="error_outline" color="negative" />
        </template>
        {{ treeError }}
      </q-banner>

      <main class="organization-browser-workspace">
        <q-card flat bordered class="organization-tree-panel">
          <q-card-section class="organization-panel-header">
            <div>
              <div class="organization-panel-title">法人树</div>
              <div class="organization-panel-subtitle">{{ treeSummary }}</div>
            </div>
          </q-card-section>

          <q-separator />

          <q-card-section class="organization-tree-toolbar">
            <q-input
              v-model="keyword"
              outlined
              dense
              clearable
              placeholder="搜索法人编码、名称或简称"
            >
              <template #append>
                <q-icon name="search" />
              </template>
            </q-input>
          </q-card-section>

          <q-separator />

          <q-card-section class="organization-tree-content">
            <organization-read-only-tree
              :nodes="displayTree"
              :selected-id="selectedNode?.legal_entity_id ?? null"
              :loading="treeLoading"
              :expand-all="Boolean(keyword.trim())"
              :empty-text="keyword ? '没有匹配的法人主体' : '暂无法人主体数据'"
              @select="handleNodeSelectedById"
            />
          </q-card-section>
        </q-card>

        <section class="organization-detail-panel" aria-label="法人详情">
          <q-card flat bordered class="organization-detail-summary">
            <q-card-section>
              <div v-if="selectedNode" class="organization-detail-context">
                <div class="organization-detail-icon">
                  <q-icon name="account_balance" />
                </div>
                <div class="organization-detail-heading">
                  <div class="organization-detail-title">{{ selectedNode.name }}</div>
                  <div class="organization-detail-meta">
                    <span>{{ entityTypeLabel(selectedNode.entity_type) }}</span>
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
                <span>请选择法人主体</span>
              </div>
            </q-card-section>
          </q-card>

          <organization-read-only-detail
            :groups="detailGroups"
            :loading="detailLoading"
            :error="detailError"
            empty-text="请选择左侧法人主体"
          />
        </section>
      </main>
    </div>
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'organization_legal_entity' })

import { computed, onMounted, ref } from 'vue'
import { date } from 'quasar'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import OrganizationReadOnlyDetail from 'src/pages/organization/components/OrganizationReadOnlyDetail.vue'
import type { OrganizationDetailGroup } from 'src/pages/organization/components/organization-read-only-detail'
import OrganizationReadOnlyTree from 'src/pages/organization/components/OrganizationReadOnlyTree.vue'
import type { OrganizationReadOnlyTreeNode } from 'src/pages/organization/components/organization-read-only-tree'
import {
  getLegalEntityDetail,
  getLegalEntityTree,
  type LegalEntityDetail,
  type LegalEntityTreeNode,
} from 'src/api/services/org'
import { usePageButtons } from 'src/composables/page-buttons'
import { useDictStore } from 'src/stores/dict'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'

const dictStore = useDictStore()
const { all_buttons, top_buttons } = usePageButtons('organization_legal_entity')

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
const canViewDetail = computed(() =>
  all_buttons.value.some((button) => button.event_action === 'detail'),
)
const filteredTree = computed(() => filterLegalEntityTree(tree.value, keyword.value))
const displayTree = computed(() => mapLegalEntityTree(filteredTree.value))
const treeSummary = computed(
  () => `组织主数据镜像 · ${countTreeNodes(tree.value)} 个法人主体`,
)

const detailGroups = computed<OrganizationDetailGroup[]>(() => {
  if (!detail.value) return []
  const entity = detail.value
  const parent = entity.parent_id
    ? findLegalEntityNode(tree.value, entity.parent_id)
    : null

  return [
    {
      key: 'basic',
      title: '基础信息',
      icon: 'account_balance',
      fields: [
        { key: 'name', label: '法人名称', value: displayValue(entity.name) },
        { key: 'short_name', label: '法人简称', value: displayValue(entity.short_name) },
        { key: 'code', label: '法人编码', value: displayValue(entity.code), kind: 'code' },
        {
          key: 'entity_type',
          label: '主体类型',
          value: entityTypeLabel(entity.entity_type),
        },
      ],
    },
    {
      key: 'ownership',
      title: '主体信息',
      icon: 'corporate_fare',
      fields: [
        {
          key: 'parent',
          label: '上级法人',
          value: parent ? `${parent.code} - ${parent.name}` : '-',
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
          value: statusLabel(entity.status),
          kind: 'status',
          color: statusColor(entity.status, false),
        },
        { key: 'valid_from', label: '有效期开始', value: formatDate(entity.valid_from) },
        {
          key: 'valid_to',
          label: '有效期结束',
          value: formatDate(entity.valid_to, '长期有效'),
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
          value: formatDateTime(entity.gmt_modify),
        },
        {
          key: 'local_note',
          label: '平台备注',
          value: displayValue(entity.local_note),
          wide: true,
        },
      ],
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
    treeError.value = errorMessage(error, '法人主体加载失败')
  } finally {
    treeLoading.value = false
  }
}

const handleNodeSelectedById = async (legalEntityId: number) => {
  const node = findLegalEntityNode(filteredTree.value, legalEntityId)
  if (node) await handleNodeSelected(node)
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

function mapLegalEntityTree(nodes: LegalEntityTreeNode[]): OrganizationReadOnlyTreeNode[] {
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
