<template>
  <base-content class="q-pa-sm organization-readonly-page">
    <div class="organization-browser-page column no-wrap fit">
      <q-card flat bordered :dark="Dark.isActive" class="organization-page-toolbar">
        <q-card-section class="row items-center justify-between q-gutter-md q-py-sm">
          <div class="organization-page-heading">
            <h1 class="text-h6 text-weight-bold q-my-none">
              {{ t('ui.organizationalStructure') }}
            </h1>
            <p class="text-caption text-grey-7 q-my-none">
              {{ t('ui.browseTwoOrganizationalTreesOfCorporateAndRegulatoryStructures') }}
            </p>
          </div>

          <div class="row items-center justify-end q-gutter-sm">
            <q-select
              v-model="architectureMode"
              :options="architectureModeOptions"
              :dark="Dark.isActive"
              option-value="value"
              option-label="label"
              emit-value
              map-options
              outlined
              dense
              :label="t('ui.structureType')"
              class="architecture-mode-toggle"
              @update:model-value="handleArchitectureModeChange"
            />

            <q-select
              v-if="architectureMode === 'management' && showStructureSwitcher"
              v-model="selectedStructureCode"
              :options="structures"
              :dark="Dark.isActive"
              option-value="code"
              option-label="name"
              emit-value
              map-options
              outlined
              dense
              :loading="structureLoading"
              :label="t('ui.manageView')"
              class="structure-select"
              @update:model-value="handleStructureChange"
            >
              <template #no-option>
                <q-item>
                  <q-item-section class="text-grey-7">{{ t('ui.noManagedView') }}</q-item-section>
                </q-item>
              </template>
            </q-select>

            <q-btn
              icon="refresh"
              :aria-label="t('ui.refreshCurrentView')"
              round
              flat
              dense
              color="primary"
              :loading="treeLoading || structureLoading"
              @click="refreshPage"
            >
              <q-tooltip>{{ t('ui.refreshCurrentView') }}</q-tooltip>
            </q-btn>
          </div>
        </q-card-section>
      </q-card>

      <q-banner v-if="pageError" :dark="Dark.isActive" rounded class="organization-page-error">
        <template #avatar>
          <q-icon name="error_outline" color="negative" />
        </template>
        {{ pageError }}
      </q-banner>

      <main class="row q-col-gutter-sm organization-browser-workspace">
        <div class="col-12 col-md-5 full-height">
          <q-card
            flat
            bordered
            :dark="Dark.isActive"
            class="organization-tree-panel column no-wrap fit"
          >
            <q-card-section class="q-py-sm">
              <div>
                <div class="organization-panel-title text-subtitle2 text-weight-bold">
                  {{ treeTitle }}
                </div>
                <div class="text-caption text-grey-7">{{ treeSummary }}</div>
              </div>
            </q-card-section>

            <q-separator />

            <q-card-section class="q-pa-sm">
              <q-input
                v-model="treeKeyword"
                :dark="Dark.isActive"
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
                    <q-tooltip>{{ t('ui.search') }}</q-tooltip>
                  </q-btn>
                </template>
              </q-input>
            </q-card-section>

            <q-separator />

            <q-card-section class="organization-tree-content col q-pa-none overflow-hidden">
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
        </div>

        <div class="col-12 col-md-7 full-height">
          <q-card
            flat
            bordered
            :dark="Dark.isActive"
            class="organization-detail-panel column no-wrap fit"
          >
            <q-card-section class="row items-center q-pa-md">
              <div v-if="selectionSummary" class="row items-center no-wrap full-width">
                <q-icon :name="selectionSummary.icon" size="28px" class="text-grey-6 q-mr-sm" />
                <div class="organization-detail-heading">
                  <div class="organization-detail-title text-subtitle1 text-weight-bold ellipsis">
                    {{ selectionSummary.name }}
                  </div>
                  <div class="row items-center q-gutter-xs text-caption text-grey-7">
                    <span>{{ selectionSummary.typeLabel }}</span>
                    <span aria-hidden="true">·</span>
                    <span>{{ selectionSummary.code }}</span>
                    <q-chip
                      dense
                      square
                      outline
                      :color="statusColor(selectionSummary.status, selectionSummary.disabled)"
                    >
                      {{ statusLabel(selectionSummary.status) }}
                    </q-chip>
                  </div>
                </div>
              </div>
              <div v-else class="row items-center q-gutter-sm text-grey-6">
                <q-icon name="description" size="28px" />
                <span>{{ detailEmptyText }}</span>
              </div>
            </q-card-section>

            <q-separator />

            <q-card-section class="organization-detail-content col q-px-md q-pb-md q-pt-sm">
              <organization-read-only-detail
                :groups="detailGroups"
                :loading="detailLoading"
                :error="detailError"
                :empty-text="detailEmptyText"
              />
            </q-card-section>
          </q-card>
        </div>
      </main>
    </div>
  </base-content>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'organization_structure' })

import { computed, onMounted, ref } from 'vue'
import { Dark, date } from 'quasar'
import BaseContent from '@/components/BaseContent/BaseContent.vue'
import OrganizationReadOnlyDetail from '@/pages/organization/components/OrganizationReadOnlyDetail.vue'
import type { OrganizationDetailGroup } from '@/pages/organization/components/organization-read-only-detail'
import OrganizationReadOnlyTree from '@/pages/organization/components/OrganizationReadOnlyTree.vue'
import type { OrganizationReadOnlyTreeNode } from '@/pages/organization/components/organization-read-only-tree'
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
} from '@/api/services/org'
import { useDictStore } from '@/stores/dict'

const { t } = useI18n({ useScope: 'global' })

type ArchitectureMode = 'management' | 'legal'
type LegalArchitectureNodeKind = 'legal_entity' | 'legal_unit'

interface LegalArchitectureNode {
  id: number
  kind: LegalArchitectureNodeKind
  code: string
  name: string
  shortName?: string
  status: string
  disabled: boolean
  legalEntityId?: number
  orgUnitId?: number
  entityType?: string
  unitType?: string
  children: LegalArchitectureNode[]
}

interface SelectionSummary {
  name: string
  code: string
  typeLabel: string
  status: string
  disabled: boolean
  icon: string
}

const dictStore = useDictStore()

const architectureMode = ref<ArchitectureMode>('management')
const structures = ref<OrganizationStructure[]>([])
const legalStructure = ref<OrganizationStructure | null>(null)
const selectedStructureCode = ref<string | null>(null)
const managementTree = ref<StructureOrgTreeNode[]>([])
const legalTree = ref<LegalArchitectureNode[]>([])
const selectedManagementNode = ref<StructureOrgTreeNode | null>(null)
const selectedLegalNode = ref<LegalArchitectureNode | null>(null)
const managementDetail = ref<OrgUnitDetail | null>(null)
const legalDetail = ref<LegalEntityDetail | null>(null)
const legalUnitDetail = ref<OrgUnitDetail | null>(null)
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
  {
    get label() {
      return t('ui.managementStructure')
    },
    value: 'management',
  },
  {
    get label() {
      return t('ui.legalEntityStructure')
    },
    value: 'legal',
  },
]

const selectedStructure = computed(
  () => structures.value.find((item) => item.code === selectedStructureCode.value) || null,
)
const showStructureSwitcher = computed(() => structures.value.length > 1)
const pageError = computed(() => structureError.value || treeError.value)
const filteredLegalTree = computed(() =>
  filterLegalArchitectureTree(legalTree.value, treeKeyword.value),
)
const displayTree = computed<OrganizationReadOnlyTreeNode[]>(() =>
  architectureMode.value === 'management'
    ? mapStructureTree(managementTree.value)
    : mapLegalArchitectureTree(filteredLegalTree.value),
)
const selectedTreeNodeId = computed(() =>
  architectureMode.value === 'management'
    ? (selectedManagementNode.value?.structure_node_id ?? null)
    : (selectedLegalNode.value?.id ?? null),
)
const treeTitle = computed(() =>
  architectureMode.value === 'management'
    ? t('ui.managementStructure')
    : t('ui.legalEntityStructure'),
)
const treeSummary = computed(() => {
  if (architectureMode.value === 'legal') {
    const counts = countLegalArchitectureNodes(legalTree.value)
    return t('ui.legalEntityDepartment', {
      value1: counts.legalEntities,
      value2: counts.legalUnits,
    })
  }
  if (!selectedStructure.value) return t('ui.noManagementStructureDataAvailable')
  return t('ui.organizationCountSummary', {
    value1: selectedStructure.value.name,
    value2: countStructureNodes(managementTree.value),
  })
})
const searchPlaceholder = computed(() =>
  architectureMode.value === 'management'
    ? t('ui.searchForOrganizationalCodeOrName')
    : t('ui.searchForLegalPersonOrSectorCodeName'),
)
const treeEmptyText = computed(() => {
  if (architectureMode.value === 'legal') {
    return treeKeyword.value
      ? t('ui.noMatchingCorporateEntityOrDepartment')
      : t('ui.organizationArchitectureUnavailable')
  }
  if (!selectedStructure.value) return t('ui.noManagementViewAvailableForNow')
  return treeKeyword.value
    ? t('ui.noMatchingOrganizationForTheCurrentView')
    : t('ui.noOrganizationDataInCurrentView')
})
const detailEmptyText = computed(() =>
  architectureMode.value === 'management'
    ? t('ui.selectTheLeftManipulationOrganization')
    : t('ui.selectTheLegalEntityOrDepartmentOnTheLeftSide'),
)
const selectionSummary = computed<SelectionSummary | null>(() => {
  if (architectureMode.value === 'legal') {
    const node = selectedLegalNode.value
    if (!node) return null
    return {
      name: node.name,
      code: node.code,
      typeLabel:
        node.kind === 'legal_entity'
          ? legalEntityTypeLabel(node.entityType || '')
          : unitTypeLabel(node.unitType || ''),
      status: node.status,
      disabled: node.disabled,
      icon: node.kind === 'legal_entity' ? 'account_balance' : 'corporate_fare',
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
    : selectedLegalNode.value?.kind === 'legal_unit'
      ? legalUnitDetailGroups(legalUnitDetail.value)
      : legalEntityDetailGroups(legalDetail.value),
)

const loadStructures = async () => {
  structureLoading.value = true
  structureError.value = ''
  try {
    const previousCode = selectedStructureCode.value
    const [managementResult, legalResult] = await Promise.all([
      queryStructures({
        page: 1,
        num: 100,
        only_effective: true,
        structure_type: 'management',
      }),
      queryStructures({
        page: 1,
        num: 100,
        only_effective: true,
        structure_type: 'legal',
      }),
    ])
    structures.value = managementResult.items
    legalStructure.value = legalResult.items.find((item) => item.structure_type === 'legal') || null

    if (previousCode && structures.value.some((item) => item.code === previousCode)) {
      selectedStructureCode.value = previousCode
      return
    }

    const defaultStructure = structures.value.find((item) => item.is_default)
    selectedStructureCode.value = defaultStructure?.code || structures.value[0]?.code || null
  } catch (error) {
    structures.value = []
    legalStructure.value = null
    selectedStructureCode.value = null
    structureError.value = errorMessage(error, t('ui.managingViewLoadingFailed'))
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
      ? findStructureNode(managementTree.value, selectedManagementNode.value.structure_node_id)
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
    treeError.value = errorMessage(error, t('ui.managingOrganisationalTreeLoadingFailed'))
  } finally {
    treeLoading.value = false
  }
}

const loadLegalTree = async () => {
  treeLoading.value = true
  treeError.value = ''
  try {
    if (!legalStructure.value) await loadStructures()
    const [legalEntities, legalUnits] = await Promise.all([
      getLegalEntityTree({ only_effective: true }),
      legalStructure.value
        ? getStructureOrgTree({
            structure_id: legalStructure.value.id,
            only_effective: true,
          })
        : Promise.resolve([]),
    ])
    legalTree.value = buildLegalArchitectureTree(legalEntities, legalUnits)
    legalTreeLoaded.value = true
    const current = selectedLegalNode.value
      ? findLegalArchitectureNode(legalTree.value, selectedLegalNode.value.id)
      : null
    const next = current || firstLegalArchitectureNode(legalTree.value)
    if (next) {
      await selectLegalNode(next)
    } else {
      selectedLegalNode.value = null
      legalDetail.value = null
      legalUnitDetail.value = null
      detailError.value = ''
    }
  } catch (error) {
    legalTree.value = []
    selectedLegalNode.value = null
    legalDetail.value = null
    legalUnitDetail.value = null
    legalTreeLoaded.value = false
    treeError.value = errorMessage(error, t('ui.failedToLoadCorporateArchitecture'))
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
    const node = findLegalArchitectureNode(filteredLegalTree.value, nodeId)
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
      detailError.value = errorMessage(error, t('ui.failedToManageOrganizationDetailsLoaded'))
    }
  } finally {
    if (sequence === detailRequestSequence) detailLoading.value = false
  }
}

const selectLegalNode = async (node: LegalArchitectureNode) => {
  selectedLegalNode.value = node
  legalDetail.value = null
  legalUnitDetail.value = null
  detailError.value = ''

  const sequence = ++detailRequestSequence
  detailLoading.value = true
  try {
    if (node.kind === 'legal_entity' && node.legalEntityId) {
      const result = await getLegalEntityDetail(node.legalEntityId, {
        only_effective: true,
      })
      if (sequence === detailRequestSequence) legalDetail.value = result
    } else if (node.kind === 'legal_unit' && node.orgUnitId) {
      const result = await getOrgUnitDetail(node.orgUnitId, {
        only_effective: true,
      })
      if (sequence === detailRequestSequence) legalUnitDetail.value = result
    }
  } catch (error) {
    if (sequence === detailRequestSequence) {
      detailError.value = errorMessage(error, t('ui.loadingDetailsOfCorporateArchitectureFailed'))
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
  await dictStore.loadDicts(['org_unit_type', 'org_legal_entity_type', 'org_object_status'])
  if (architectureMode.value === 'legal') {
    await loadLegalTree()
    return
  }
  await loadStructures()
  await loadManagementTree()
})

function managementDetailGroups(detail: OrgUnitDetail | null): OrganizationDetailGroup[] {
  if (!detail) return []
  const primaryLegalEntity = detail.primary_legal_entity
    ? `${detail.primary_legal_entity.code} - ${detail.primary_legal_entity.name}`
    : detail.primary_legal_entity_id
      ? String(detail.primary_legal_entity_id)
      : '-'

  return [
    {
      key: 'management-detail',
      get title() {
        return t('ui.manageOrganizationalDetails')
      },
      fields: [
        {
          key: 'name',
          get label() {
            return t('ui.nameOfOrganization')
          },
          value: displayValue(detail.name),
        },
        {
          key: 'code',
          get label() {
            return t('ui.organizationCodeLabel')
          },
          value: displayValue(detail.code),
          kind: 'code',
        },
        {
          key: 'unit_type',
          get label() {
            return t('ui.organizationType')
          },
          value: unitTypeLabel(detail.unit_type),
        },
        {
          key: 'structure',
          get label() {
            return t('ui.manageView')
          },
          value: selectedStructure.value?.name || '-',
        },
        {
          key: 'primary_legal_entity',
          get label() {
            return t('ui.primaryLegalEntityLabel')
          },
          value: primaryLegalEntity,
        },
        {
          key: 'status',
          get label() {
            return t('ui.status')
          },
          value: statusLabel(detail.status),
          kind: 'status',
          color: statusColor(detail.status, false),
        },
        {
          key: 'valid_from',
          get label() {
            return t('ui.validFrom')
          },
          value: formatDate(detail.valid_from),
        },
        {
          key: 'valid_to',
          get label() {
            return t('ui.validUntil')
          },
          value: formatDate(detail.valid_to, t('ui.noExpiration')),
        },
        {
          key: 'gmt_modify',
          get label() {
            return t('ui.updatedAt')
          },
          value: formatDateTime(detail.gmt_modify),
        },
        {
          key: 'local_note',
          get label() {
            return t('ui.platformNotes')
          },
          value: displayValue(detail.local_note),
          wide: true,
        },
      ],
    },
  ]
}

function legalEntityDetailGroups(detail: LegalEntityDetail | null): OrganizationDetailGroup[] {
  if (!detail) return []
  const parent = detail.parent_id
    ? findLegalArchitectureNode(legalTree.value, detail.parent_id)
    : null

  return [
    {
      key: 'legal-detail',
      get title() {
        return t('ui.detailsOfLegalPersons')
      },
      fields: [
        {
          key: 'name',
          get label() {
            return t('ui.nameOfLegalPerson')
          },
          value: displayValue(detail.name),
        },
        {
          key: 'short_name',
          get label() {
            return t('ui.abbreviationsForLegalPersons')
          },
          value: displayValue(detail.short_name),
        },
        {
          key: 'code',
          get label() {
            return t('ui.legalPersonCode')
          },
          value: displayValue(detail.code),
          kind: 'code',
        },
        {
          key: 'entity_type',
          get label() {
            return t('ui.subjectType')
          },
          value: legalEntityTypeLabel(detail.entity_type),
        },
        {
          key: 'parent',
          get label() {
            return t('ui.parentLegalEntity')
          },
          value: parent ? `${parent.code} - ${parent.name}` : '-',
        },
        {
          key: 'unified_social_credit_code',
          get label() {
            return t('ui.unifiedSocialCreditCode')
          },
          value: displayValue(detail.unified_social_credit_code),
          kind: 'code',
        },
        {
          key: 'accounting_code',
          get label() {
            return t('ui.accountingCode')
          },
          value: displayValue(detail.accounting_code),
          kind: 'code',
        },
        {
          key: 'status',
          get label() {
            return t('ui.status')
          },
          value: statusLabel(detail.status),
          kind: 'status',
          color: statusColor(detail.status, false),
        },
        {
          key: 'valid_from',
          get label() {
            return t('ui.validFrom')
          },
          value: formatDate(detail.valid_from),
        },
        {
          key: 'valid_to',
          get label() {
            return t('ui.validUntil')
          },
          value: formatDate(detail.valid_to, t('ui.noExpiration')),
        },
        {
          key: 'gmt_modify',
          get label() {
            return t('ui.updatedAt')
          },
          value: formatDateTime(detail.gmt_modify),
        },
        {
          key: 'local_note',
          get label() {
            return t('ui.platformNotes')
          },
          value: displayValue(detail.local_note),
          wide: true,
        },
      ],
    },
  ]
}

function legalUnitDetailGroups(detail: OrgUnitDetail | null): OrganizationDetailGroup[] {
  if (!detail) return []
  const legalEntity = detail.primary_legal_entity
    ? `${detail.primary_legal_entity.code} - ${detail.primary_legal_entity.name}`
    : '-'

  return [
    {
      key: 'legal-unit-detail',
      get title() {
        return t('ui.detailsOfTheLegalEntity')
      },
      fields: [
        {
          key: 'name',
          get label() {
            return t('ui.departmentName')
          },
          value: displayValue(detail.name),
        },
        {
          key: 'code',
          get label() {
            return t('ui.sectorCode')
          },
          value: displayValue(detail.code),
          kind: 'code',
        },
        {
          key: 'unit_type',
          get label() {
            return t('ui.organizationType')
          },
          value: unitTypeLabel(detail.unit_type),
        },
        {
          key: 'legal_entity',
          get label() {
            return t('ui.owningLegalEntity')
          },
          value: legalEntity,
        },
        {
          key: 'status',
          get label() {
            return t('ui.status')
          },
          value: statusLabel(detail.status),
          kind: 'status',
          color: statusColor(detail.status, false),
        },
        {
          key: 'valid_from',
          get label() {
            return t('ui.validFrom')
          },
          value: formatDate(detail.valid_from),
        },
        {
          key: 'valid_to',
          get label() {
            return t('ui.validUntil')
          },
          value: formatDate(detail.valid_to, t('ui.noExpiration')),
        },
        {
          key: 'gmt_modify',
          get label() {
            return t('ui.updatedAt')
          },
          value: formatDateTime(detail.gmt_modify),
        },
        {
          key: 'local_note',
          get label() {
            return t('ui.platformNotes')
          },
          value: displayValue(detail.local_note),
          wide: true,
        },
      ],
    },
  ]
}

function mapStructureTree(nodes: StructureOrgTreeNode[]): OrganizationReadOnlyTreeNode[] {
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

function mapLegalArchitectureTree(nodes: LegalArchitectureNode[]): OrganizationReadOnlyTreeNode[] {
  return nodes.map((node) => ({
    id: node.id,
    code: node.code,
    name: node.name,
    icon: node.kind === 'legal_entity' ? 'account_balance' : 'corporate_fare',
    typeLabel:
      node.kind === 'legal_entity'
        ? legalEntityTypeLabel(node.entityType || '')
        : unitTypeLabel(node.unitType || ''),
    statusLabel: statusLabel(node.status),
    statusColor: statusColor(node.status, node.disabled),
    muted: node.disabled,
    children: mapLegalArchitectureTree(node.children || []),
  }))
}

function buildLegalArchitectureTree(
  legalEntities: LegalEntityTreeNode[],
  legalUnits: StructureOrgTreeNode[],
): LegalArchitectureNode[] {
  const entitiesById = new Map<number, LegalArchitectureNode>()
  const mapLegalEntity = (source: LegalEntityTreeNode): LegalArchitectureNode => {
    const node: LegalArchitectureNode = {
      id: source.legal_entity_id,
      kind: 'legal_entity',
      legalEntityId: source.legal_entity_id,
      code: source.code,
      name: source.name,
      shortName: source.short_name,
      entityType: source.entity_type,
      status: source.status,
      disabled: source.disabled,
      children: [],
    }
    entitiesById.set(source.legal_entity_id, node)
    node.children = (source.children || []).map(mapLegalEntity)
    return node
  }
  const roots = legalEntities.map(mapLegalEntity)

  const mapLegalUnit = (source: StructureOrgTreeNode): LegalArchitectureNode => ({
    id: source.structure_node_id,
    kind: 'legal_unit',
    orgUnitId: source.org_unit_id,
    code: source.code,
    name: source.name,
    unitType: source.unit_type,
    status: source.status,
    disabled: source.disabled,
    children: (source.children || []).map(mapLegalUnit),
  })

  const detachedUnits: LegalArchitectureNode[] = []
  for (const unitRoot of legalUnits) {
    const unitNode = mapLegalUnit(unitRoot)
    const legalEntity = unitRoot.primary_legal_entity_id
      ? entitiesById.get(unitRoot.primary_legal_entity_id)
      : null
    if (legalEntity) legalEntity.children.push(unitNode)
    else detachedUnits.push(unitNode)
  }
  return [...roots, ...detachedUnits]
}

function filterLegalArchitectureTree(
  nodes: LegalArchitectureNode[],
  rawKeyword: string,
): LegalArchitectureNode[] {
  const normalized = rawKeyword.trim().toLocaleLowerCase()
  if (!normalized) return nodes

  return nodes.flatMap((node) => {
    const children = filterLegalArchitectureTree(node.children || [], normalized)
    const matched = [node.code, node.name, node.shortName].some((value) =>
      String(value || '')
        .toLocaleLowerCase()
        .includes(normalized),
    )
    return matched || children.length ? [{ ...node, children }] : []
  })
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

function firstLegalArchitectureNode(nodes: LegalArchitectureNode[]): LegalArchitectureNode | null {
  for (const node of nodes) {
    if (!node.disabled) return node
    const child = firstLegalArchitectureNode(node.children || [])
    if (child) return child
  }
  return nodes[0] || null
}

function findLegalArchitectureNode(
  nodes: LegalArchitectureNode[],
  id: number,
): LegalArchitectureNode | null {
  for (const node of nodes) {
    if (node.id === id) return node
    const child = findLegalArchitectureNode(node.children || [], id)
    if (child) return child
  }
  return null
}

function countStructureNodes(nodes: StructureOrgTreeNode[]): number {
  return nodes.reduce((total, node) => total + 1 + countStructureNodes(node.children || []), 0)
}

function countLegalArchitectureNodes(nodes: LegalArchitectureNode[]): {
  legalEntities: number
  legalUnits: number
} {
  return nodes.reduce(
    (total, node) => {
      const children = countLegalArchitectureNodes(node.children || [])
      return {
        legalEntities:
          total.legalEntities + children.legalEntities + (node.kind === 'legal_entity' ? 1 : 0),
        legalUnits: total.legalUnits + children.legalUnits + (node.kind === 'legal_unit' ? 1 : 0),
      }
    },
    { legalEntities: 0, legalUnits: 0 },
  )
}

function displayValue(value: unknown): string {
  if (value === null || value === undefined || value === '') return '-'
  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
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
  min-height: 0 !important;
  overflow: hidden;
}

.organization-browser-page {
  min-height: 0;
  gap: 10px;
}

.organization-page-toolbar,
.organization-page-error {
  flex: 0 0 auto;
}

.organization-page-heading {
  min-width: 0;
}

.architecture-mode-toggle {
  width: 168px;
}

.structure-select {
  width: 240px;
}

.organization-browser-workspace {
  flex: 1 1 auto;
  min-height: 0;
}

.organization-tree-panel,
.organization-detail-panel {
  min-height: 0;
}

.organization-detail-panel {
  overflow: hidden;
}

.organization-detail-content {
  min-height: 0;
  overflow: auto;
}

.organization-detail-heading {
  min-width: 0;
}

@media (max-width: 1023px) {
  .organization-readonly-page {
    overflow: auto;
  }

  .organization-browser-page {
    height: auto !important;
  }

  .organization-browser-workspace {
    flex: 0 0 auto;
    height: auto;
    min-height: 0;
  }

  .organization-tree-panel,
  .organization-detail-panel {
    min-height: 520px;
  }
}
</style>
