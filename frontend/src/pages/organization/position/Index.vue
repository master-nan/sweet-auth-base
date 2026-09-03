<template>
  <base-content :scrollable="showDetailDialog && detailMode === 'page'" class="q-pa-sm">
    <q-table
      v-if="!showDetailDialog || detailMode === 'dialog'"
      class="fit sticky-header-table"
      flat
      bordered
      separator="cell"
      row-key="id"
      :rows="rows"
      :columns="columns"
      :visible-columns="visibleColumns"
      :loading="loading"
      :no-data-label="emptyMessage"
      v-model:pagination="pagination"
      hide-pagination
    >
      <template #top>
        <standard-table-toolbar :refreshing="loading" @refresh="fetchData">
          <template #query-controls>
            <query-scheme-controls
              :controller="schemePage"
              :query-state="queryState"
              :fields="advancedFields"
              :advanced-title="t('ui.postAdvancedQuery')"
              :show-filter-count="false"
            >
              <template #quick-search>
                <q-input
                  v-model="keyword"
                  dense
                  outlined
                  debounce="300"
                  :placeholder="quickSearchPlaceholder"
                  @keyup.enter="search"
                >
                  <template #append><q-icon name="search" /></template>
                </q-input>
                <q-btn color="primary" :label="t('ui.search')" :disable="loading" @click="search" />
              </template>
            </query-scheme-controls>
          </template>
          <template #column-selector>
            <table-column-selector v-model="visibleColumns" :columns="columns" />
          </template>
        </standard-table-toolbar>
      </template>

      <template #body-cell-position_type="props">
        <q-td :props="props">
          {{ dictLabel('org_position_type', props.row.position_type) }}
        </q-td>
      </template>
      <template #body-cell-status="props">
        <q-td :props="props">
          <status-chip
            :color="organizationStatusColor(props.row.status)"
            :label="dictLabel('org_object_status', props.row.status)"
          />
        </q-td>
      </template>
      <template #body-cell-validity="props">
        <q-td :props="props">
          {{ formatOrganizationDate(props.row.valid_from) }}
          <span class="text-grey-6 q-mx-xs">{{ t('ui.to') }}</span>
          {{ formatOrganizationDate(props.row.valid_to, t('ui.longTerm')) }}
        </q-td>
      </template>
      <template #body-cell-is_manager_position="props">
        <q-td :props="props">{{ props.row.is_manager_position ? t('ui.yes') : t('ui.no') }}</q-td>
      </template>
      <template #body-cell-actions="props">
        <q-td :props="props" class="q-gutter-xs">
          <q-btn
            v-for="button in rowButtons"
            :key="button.id || button.code"
            flat
            v-bind="menuButtonDisplayProps(button)"
            :color="button.color || 'primary'"
            size="sm"
            @click="handleRowAction(button, props.row)"
          >
            <q-tooltip>{{ button.name }}</q-tooltip>
          </q-btn>
        </q-td>
      </template>
      <template #bottom>
        <q-space />
        <table-pagination v-model:page="query.page" v-model:pageSize="query.num" :total="total" />
      </template>
    </q-table>

    <organization-record-detail-dialog
      v-model="showDetailDialog"
      :title="positionDetail?.name || t('ui.jobDetails')"
      :subtitle="positionDetail?.code || ''"
      :sections="detailSections"
      icon="work"
      :status-label="positionDetail ? dictLabel('org_object_status', positionDetail.status) : ''"
      :status-color="positionDetail ? organizationStatusColor(positionDetail.status) : 'positive'"
      :loading="detailLoading"
      :error="detailError"
      :mode="detailMode"
      :top-buttons="record_detail_top_buttons"
      :bottom-buttons="record_detail_bottom_buttons"
      :record-context="positionDetail"
      @button-click="handleDetailAction"
    />
  </base-content>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'organization_position' })

import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import BaseContent from '@/components/BaseContent/BaseContent.vue'
import TablePagination from '@/components/Table/TablePagination.vue'
import StandardTableToolbar from '@/components/Table/StandardTableToolbar.vue'
import TableColumnSelector from '@/components/Table/TableColumnSelector.vue'
import StatusChip from '@/components/Display/StatusChip.vue'
import QuerySchemeControls from '@/components/QueryScheme/QuerySchemeControls.vue'
import {
  getPositionDetail,
  queryPositions,
  type PositionDetail,
  type PositionListItem,
  type PositionQueryRequest,
} from '@/api/services/org'
import type { MenuButton } from '@/api/services/sys-menu'
import { usePageButtons } from '@/composables/page-buttons'
import { useRuntimeTableMetadata } from '@/composables/runtime-table-metadata'
import { useTableQueryState } from '@/composables/table-query-state'
import { useQuerySchemePage } from '@/composables/query-scheme-page'
import OrganizationRecordDetailDialog from '@/pages/organization/components/OrganizationRecordDetailDialog.vue'
import type { OrganizationDetailSection } from '@/pages/organization/components/organization-record-detail'
import { useOrganizationDetailMode } from '@/pages/organization/use-organization-detail-mode'
import {
  createOrganizationQuery,
  formatOrganizationDate,
  formatOrganizationValue,
  organizationStatusColor,
  referenceLabel,
} from '@/pages/organization/organization-list-page'
import { useDictStore } from '@/stores/dict'
import { menuButtonDisplayProps } from '@/utils/menu-button-display'
import { resolveRuntimeColumns } from '@/utils/column-format'
import { resolveTableEmptyMessage } from '@/utils/table-state'
import type { TableColumn } from '@/types/global'

const { t } = useI18n({ useScope: 'global' })

const router = useRouter()
const dictStore = useDictStore()
const {
  line_buttons,
  has_line_buttons,
  record_detail_top_buttons,
  record_detail_bottom_buttons,
  hasGrantedCapability,
} = usePageButtons('organization_position')
const detailMode = useOrganizationDetailMode('organization_position', 'dialog')

const rows = ref<PositionListItem[]>([])
const total = ref(0)
const loading = ref(false)
const loadError = ref('')
const canQueryPositions = computed(() => hasGrantedCapability('organization_position_query'))
const queryState = useTableQueryState<PositionQueryRequest>({
  createInitialQuery: () => ({
    ...createOrganizationQuery('org_position'),
    only_effective: true,
  }),
})
const { query, keyword } = queryState
const showDetailDialog = ref(false)
const detailLoading = ref(false)
const detailError = ref('')
const positionDetail = ref<PositionDetail | null>(null)
const resetAndFetch = () => {
  if (query.value.page !== 1) query.value.page = 1
  else void fetchData()
}
const schemePage = useQuerySchemePage('organization_position', queryState, resetAndFetch)
const initialized = ref(false)

const rowButtons = computed(() =>
  line_buttons.value.filter((button) => button.event_action === 'detail'),
)
const columns = ref<TableColumn<PositionListItem>[]>([])
const visibleColumns = ref<string[]>([])
const sortableFields = ref<ReadonlySet<string>>(new Set())
const pagination = ref({ page: 1, rowsPerPage: 0, sortBy: '', descending: false })
const {
  fields: metadataFields,
  quickSearchPlaceholder,
  advancedSearchFields: advancedFields,
  loadMetadata,
} = useRuntimeTableMetadata('org_position')
const emptyMessage = computed(() =>
  resolveTableEmptyMessage({
    canRead: canQueryPositions.value,
    error: loadError.value,
    hasQuery: !!keyword.value,
  }),
)

const dictLabel = (code: string, value: unknown) =>
  dictStore.getDictLabel(code, value) || formatOrganizationValue(value)

const detailSections = computed<OrganizationDetailSection[]>(() => {
  const detail = positionDetail.value
  if (!detail) return []
  return [
    {
      key: 'basic',
      get label() {
        return t('ui.basicInformation')
      },
      get caption() {
        return t('ui.positionDefinitionAndValidity')
      },
      icon: 'work',
      items: [
        {
          get label() {
            return t('ui.jobEncoding')
          },
          value: detail.code,
        },
        {
          get label() {
            return t('ui.nameOfPost')
          },
          value: detail.name,
        },
        {
          get label() {
            return t('ui.typeOfPost')
          },
          value: dictLabel('org_position_type', detail.position_type),
        },
        {
          get label() {
            return t('ui.level')
          },
          value: detail.job_level,
        },
        {
          get label() {
            return t('ui.managementPositions')
          },
          value: detail.is_manager_position,
        },
        {
          get label() {
            return t('ui.status')
          },
          value: dictLabel('org_object_status', detail.status),
          chip: true,
          color: organizationStatusColor(detail.status),
        },
        {
          get label() {
            return t('ui.validFrom')
          },
          value: formatOrganizationDate(detail.valid_from),
        },
        {
          get label() {
            return t('ui.validUntil')
          },
          value: formatOrganizationDate(detail.valid_to, t('ui.longTerm')),
        },
      ],
    },
    {
      key: 'ownership',
      get label() {
        return t('ui.attributionInformation')
      },
      get caption() {
        return t('ui.legalPersonsAndOrganizationalAttribution')
      },
      icon: 'account_tree',
      items: [
        {
          get label() {
            return t('ui.organization')
          },
          value: referenceLabel(detail.org_unit),
        },
        {
          get label() {
            return t('ui.owningLegalEntity')
          },
          value: referenceLabel(detail.legal_entity),
        },
      ],
    },
    {
      key: 'mirror',
      get label() {
        return t('ui.imageInformation')
      },
      get caption() {
        return t('ui.platformExtensionInformation')
      },
      icon: 'sync',
      items: [
        {
          get label() {
            return t('ui.platformNotes')
          },
          value: detail.local_note,
          fullWidth: true,
        },
      ],
    },
  ]
})

const search = () => {
  schemePage.runQueryChange(queryState.submitQuickSearch)
}

const fetchData = async () => {
  if (!canQueryPositions.value) return
  loading.value = true
  loadError.value = ''
  try {
    const result = await queryPositions(query.value)
    rows.value = result.items
    total.value = result.total
  } catch {
    rows.value = []
    total.value = 0
    loadError.value = t('ui.jobDataLoadedFailed')
  } finally {
    loading.value = false
  }
}

const openDetail = async (row: PositionListItem) => {
  positionDetail.value = null
  detailError.value = ''
  detailLoading.value = true
  showDetailDialog.value = true
  try {
    positionDetail.value = await getPositionDetail(row.id)
  } catch {
    detailError.value = t('ui.loadingOfJobDetailsFailed')
  } finally {
    detailLoading.value = false
  }
}

const handleRowAction = (button: MenuButton, row: PositionListItem) => {
  if (button.event_action === 'detail') void openDetail(row)
  if (button.event_action === 'view_sync') {
    void router.push({
      name: 'organization_sync_error',
      query: { object_type: 'position', local_id: String(row.id) },
    })
  }
}

const handleDetailAction = (button: MenuButton) => {
  if (!positionDetail.value || button.event_action !== 'view_sync') return
  void router.push({
    name: 'organization_sync_error',
    query: { object_type: 'position', local_id: String(positionDetail.value.id) },
  })
}

watch(
  () => [query.value.page, query.value.num],
  () => {
    if (initialized.value) void fetchData()
  },
)

watch(
  () => [pagination.value.sortBy, pagination.value.descending] as const,
  ([sortBy, descending], previous) => {
    if (!initialized.value) return
    if (sortBy === previous[0] && descending === previous[1]) return
    if (!queryState.applySorting(sortBy || '', descending, sortableFields.value)) return
    void fetchData()
  },
)

onMounted(async () => {
  await Promise.all([
    dictStore.loadDicts(['org_position_type', 'org_object_status']),
    loadMetadata(),
  ])
  const resolution = resolveRuntimeColumns<PositionListItem>(metadataFields.value, {
    context: { getDictLabel: dictStore.getDictLabel },
    overrides: [
      { fieldCode: 'org_unit_id', visible: false },
      { fieldCode: 'position_type', align: 'center' },
      { fieldCode: 'is_manager_position', align: 'center' },
      { fieldCode: 'status', align: 'center' },
    ],
    virtualColumns: [
      {
        name: 'validity',
        field: 'validity',
        get label() {
          return t('ui.validityPeriod')
        },
        order: 90,
      },
      {
        name: 'actions',
        field: 'actions',
        get label() {
          return t('ui.actions')
        },
        align: 'center',
        order: 100,
        defaultVisible: has_line_buttons.value,
      },
    ],
  })
  columns.value = resolution.columns
  visibleColumns.value = resolution.visibleColumns
  sortableFields.value = resolution.sortableFields
  await schemePage.initialize()
  await fetchData()
  initialized.value = true
})
</script>
