<template>
  <base-content class="q-pa-sm">
    <q-table
      class="fit sticky-header-table"
      color="primary"
      :dense="$q.screen.lt.md"
      flat
      bordered
      separator="cell"
      row-key="id"
      v-model:pagination="pagination"
      :rows="rows"
      :columns="columns"
      :loading="loading"
      hide-pagination
    >
      <template #top>
        <div class="row q-gutter-xs full-width">
          <div class="col-grow row q-gutter-xs">
            <q-input
              v-model="query.quick_query!.keyword"
              dense
              outlined
              debounce="300"
              :placeholder="t('report.center.searchPlaceholder')"
              @keyup.enter="handleSearch"
            >
              <template #append>
                <q-icon name="search" />
              </template>
            </q-input>
            <q-btn color="primary" :label="t('report.common.search')" @click="handleSearch" />
          </div>

          <q-space />

          <div class="row q-gutter-xs">
            <q-btn
              outline
              color="primary"
              icon="refresh"
              :label="t('report.common.refresh')"
              @click="fetchData"
            />
            <q-btn
              color="primary"
              icon="add"
              :label="t('report.center.newDesign')"
              @click="openDesigner()"
            />
          </div>
        </div>
      </template>

      <template #body-cell-status="props">
        <q-td :props="props">
          <q-chip dense square :color="statusColor(props.row.status)" text-color="white">
            {{ statusLabel(props.row.status) }}
          </q-chip>
        </q-td>
      </template>

      <template #body-cell-actions="props">
        <q-td :props="props" class="q-gutter-xs">
          <q-btn
            flat
            round
            color="primary"
            icon="visibility"
            size="sm"
            @click="openPreview(props.row)"
          >
            <q-tooltip>{{ t('report.center.preview') }}</q-tooltip>
          </q-btn>
          <q-btn
            flat
            round
            color="primary"
            icon="design_services"
            size="sm"
            @click="openDesigner(props.row)"
          >
            <q-tooltip>{{ t('report.center.goDesign') }}</q-tooltip>
          </q-btn>
        </q-td>
      </template>

      <template #bottom>
        <q-space />
        <table-pagination v-model:page="query.page" v-model:pageSize="query.num" :total="total" />
      </template>
    </q-table>

    <q-dialog v-model="previewVisible">
      <q-card style="width: 900px; max-width: 92vw">
        <q-card-section class="row items-center">
          <div>
            <div class="text-h6">
              {{ previewReport?.report_name || t('report.center.preview') }}
            </div>
            <div class="text-caption text-grey-7">
              {{ previewReport?.data_source_name || '-' }}
            </div>
          </div>
          <q-space />
          <q-btn flat round icon="close" v-close-popup>
            <q-tooltip>{{ t('report.common.close') }}</q-tooltip>
          </q-btn>
        </q-card-section>
        <q-separator />
        <q-card-section>
          <q-table
            flat
            bordered
            dense
            separator="cell"
            row-key="id"
            :rows="previewRows"
            :columns="previewColumns"
            hide-pagination
            :pagination="{ rowsPerPage: 0 }"
          />
        </q-card-section>
      </q-card>
    </q-dialog>
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'report_center' })

import BaseContent from 'components/BaseContent/BaseContent.vue'
import TablePagination from 'components/Table/TablePagination.vue'
import { computed, onMounted, ref, watch } from 'vue'
import { useQuasar, type QTableProps } from 'quasar'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import type { Query } from 'src/types/global'
import {
  useReportApi,
  type Report,
  type ReportField,
  type ReportPreviewRes,
  type ReportStatus,
} from 'src/api/services/report'
import { useLoadingStore } from 'src/stores/loading'
import { storeToRefs } from 'pinia'

const $q = useQuasar()
const router = useRouter()
const { t } = useI18n()
const reportApi = useReportApi()
const loadingStore = useLoadingStore()
const { loading } = storeToRefs(loadingStore)

const query = ref<Query>({
  page: 1,
  num: 20,
  order: {
    field: 'gmt_modify',
    is_asc: false,
  },
  expressions: [],
  quick_query: {
    keyword: '',
  },
  include_deleted: false,
})

const rows = ref<Report[]>([])
const total = ref(0)
const previewVisible = ref(false)
const previewReport = ref<Report | null>(null)
const previewData = ref<ReportPreviewRes>(samplePreview())

const columns = computed<QTableProps['columns']>(() => [
  {
    name: 'report_name',
    field: 'report_name',
    label: t('report.fields.reportName'),
    align: 'left',
    sortable: true,
  },
  {
    name: 'report_code',
    field: 'report_code',
    label: t('report.fields.reportCode'),
    align: 'left',
    sortable: true,
  },
  {
    name: 'category',
    field: 'category',
    label: t('report.fields.category'),
    align: 'left',
  },
  {
    name: 'data_source_name',
    field: 'data_source_name',
    label: t('report.fields.dataSource'),
    align: 'left',
  },
  {
    name: 'status',
    field: 'status',
    label: t('report.fields.status'),
    align: 'center',
    sortable: true,
  },
  {
    name: 'owner',
    field: 'owner',
    label: t('report.fields.owner'),
    align: 'left',
  },
  {
    name: 'updated_at',
    field: (row) => row.updated_at || row.gmt_modify || '-',
    label: t('report.fields.updatedAt'),
    align: 'left',
    sortable: true,
  },
  {
    name: 'actions',
    field: 'actions',
    label: t('report.common.actions'),
    align: 'center',
  },
])

const pagination = ref({
  page: query.value.page,
  rowsPerPage: 0,
  sortBy: 'updated_at',
  descending: true,
})

const previewColumns = computed<QTableProps['columns']>(() =>
  previewData.value.columns.map((field) => ({
    name: field.code,
    field: field.code,
    label: field.name,
    align: 'left',
  })),
)

const previewRows = computed(() => previewData.value.rows)

watch(
  () => [query.value.page, query.value.num],
  () => {
    void fetchData()
  },
)

onMounted(() => {
  void fetchData()
})

function handleSearch() {
  if (query.value.page !== 1) {
    query.value.page = 1
    return
  }
  void fetchData()
}

async function fetchData() {
  try {
    const res = await reportApi.queryReports(query.value)
    rows.value = res.data || []
    total.value = res.total ?? rows.value.length
  } catch {
    const keyword = query.value.quick_query?.keyword?.trim().toLowerCase() || ''
    const filtered = sampleReports().filter((item) => {
      return [item.report_name, item.report_code, item.category, item.data_source_name]
        .filter(Boolean)
        .some((value) => String(value).toLowerCase().includes(keyword))
    })
    rows.value = keyword ? filtered : sampleReports()
    total.value = rows.value.length
  }
}

async function openPreview(row: Report) {
  previewReport.value = row
  previewData.value = samplePreview()
  previewVisible.value = true
  try {
    const res = await reportApi.previewReport({
      report_id: row.id,
      fields: [],
    })
    previewData.value = res.data
  } catch {
    previewData.value = samplePreview()
  }
}

function openDesigner(row?: Report) {
  void router.push({
    name: 'report_design',
    query: row?.id ? { id: row.id } : {},
  })
}

function statusLabel(status: ReportStatus) {
  return t(`report.status.${status}`)
}

function statusColor(status: ReportStatus) {
  const map: Record<ReportStatus, string> = {
    draft: 'grey-7',
    published: 'positive',
    archived: 'warning',
  }
  return map[status] || 'grey-7'
}

function sampleReports(): Report[] {
  return [
    {
      id: 1,
      report_name: t('report.samples.salesName'),
      report_code: 'sales_overview',
      category: t('report.samples.categoryBusiness'),
      data_source_name: t('report.samples.orderDataSource'),
      status: 'published',
      owner: 'admin',
      updated_at: '2026-06-30 10:20:00',
    },
    {
      id: 2,
      report_name: t('report.samples.userName'),
      report_code: 'user_growth',
      category: t('report.samples.categoryOperation'),
      data_source_name: t('report.samples.userDataSource'),
      status: 'draft',
      owner: 'admin',
      updated_at: '2026-06-28 16:45:00',
    },
  ]
}

function samplePreview(): ReportPreviewRes {
  const columns: ReportField[] = [
    { name: t('report.samples.month'), code: 'month', type: 'string' },
    { name: t('report.samples.amount'), code: 'amount', type: 'number' },
    { name: t('report.samples.orders'), code: 'orders', type: 'number' },
    { name: t('report.samples.conversion'), code: 'conversion', type: 'string' },
  ]
  return {
    columns,
    rows: [
      { id: 1, month: '2026-04', amount: 128600, orders: 342, conversion: '12.8%' },
      { id: 2, month: '2026-05', amount: 152300, orders: 389, conversion: '13.5%' },
      { id: 3, month: '2026-06', amount: 176900, orders: 421, conversion: '14.1%' },
    ],
  }
}
</script>
