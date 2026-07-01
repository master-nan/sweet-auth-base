<template>
  <base-content class="q-pa-sm" scrollable>
    <div class="report-designer q-gutter-sm">
      <q-card flat bordered>
        <q-card-section class="row items-center q-gutter-sm">
          <div>
            <div class="text-h6">{{ t('report.design.title') }}</div>
            <div class="text-caption text-grey-7">{{ t('report.design.subtitle') }}</div>
          </div>
          <q-space />
          <q-btn
            outline
            color="primary"
            icon="arrow_back"
            :label="t('report.common.back')"
            @click="goBack"
          />
          <q-btn
            outline
            color="primary"
            icon="preview"
            :label="t('report.design.preview')"
            @click="preview"
          />
          <q-btn color="primary" icon="save" :label="t('report.common.save')" @click="saveReport" />
        </q-card-section>
      </q-card>

      <div class="row q-col-gutter-sm">
        <div class="col-12 col-md-4">
          <q-card flat bordered class="fit">
            <q-card-section>
              <div class="text-subtitle1">{{ t('report.design.basicInfo') }}</div>
            </q-card-section>
            <q-separator />
            <q-card-section class="q-gutter-md">
              <q-input
                v-model="form.report_name"
                dense
                outlined
                :label="t('report.fields.reportName')"
              />
              <q-input
                v-model="form.report_code"
                dense
                outlined
                :label="t('report.fields.reportCode')"
              />
              <q-input
                v-model="form.category"
                dense
                outlined
                :label="t('report.fields.category')"
              />
              <q-input
                v-model="form.description"
                dense
                outlined
                autogrow
                :label="t('report.fields.description')"
              />
            </q-card-section>
          </q-card>
        </div>

        <div class="col-12 col-md-8">
          <q-card flat bordered class="fit">
            <q-card-section>
              <div class="text-subtitle1">{{ t('report.design.dataSource') }}</div>
            </q-card-section>
            <q-separator />
            <q-card-section>
              <q-select
                v-model="form.data_source_id"
                dense
                outlined
                emit-value
                map-options
                option-label="name"
                option-value="id"
                :options="dataSources"
                :label="t('report.fields.dataSource')"
                @update:model-value="handleDataSourceChange"
              >
                <template #option="scope">
                  <q-item v-bind="scope.itemProps">
                    <q-item-section>
                      <q-item-label>{{ scope.opt.name }}</q-item-label>
                      <q-item-label caption>{{
                        scope.opt.description || scope.opt.code
                      }}</q-item-label>
                    </q-item-section>
                  </q-item>
                </template>
              </q-select>

              <div class="row q-col-gutter-sm q-mt-sm">
                <div class="col-12 col-md-6">
                  <q-banner dense rounded class="bg-grey-2 text-grey-8">
                    <template #avatar>
                      <q-icon name="dataset" color="primary" />
                    </template>
                    {{ selectedDataSource?.description || t('report.design.selectSourceTip') }}
                  </q-banner>
                </div>
                <div class="col-12 col-md-6">
                  <q-banner dense rounded class="bg-blue-1 text-primary">
                    <template #avatar>
                      <q-icon name="view_column" color="primary" />
                    </template>
                    {{ t('report.design.selectedFieldCount', { count: selectedFieldRows.length }) }}
                  </q-banner>
                </div>
              </div>
            </q-card-section>
          </q-card>
        </div>
      </div>

      <div class="row q-col-gutter-sm">
        <div class="col-12 col-md-5">
          <q-card flat bordered>
            <q-card-section class="row items-center">
              <div class="text-subtitle1">{{ t('report.design.fields') }}</div>
              <q-space />
              <q-btn
                flat
                dense
                color="primary"
                icon="done_all"
                :label="t('report.design.selectAll')"
                @click="selectAllFields"
              />
            </q-card-section>
            <q-separator />
            <q-card-section>
              <q-table
                flat
                bordered
                dense
                separator="cell"
                selection="multiple"
                row-key="code"
                v-model:selected="selectedFieldRows"
                :rows="availableFields"
                :columns="fieldColumns"
                :pagination="{ rowsPerPage: 0 }"
                hide-pagination
              />
            </q-card-section>
          </q-card>
        </div>

        <div class="col-12 col-md-7">
          <q-card flat bordered>
            <q-card-section class="row items-center">
              <div>
                <div class="text-subtitle1">{{ t('report.design.previewTable') }}</div>
                <div class="text-caption text-grey-7">{{ t('report.design.previewHint') }}</div>
              </div>
              <q-space />
              <q-btn
                outline
                color="primary"
                icon="refresh"
                :label="t('report.design.refreshPreview')"
                @click="preview"
              />
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
                :pagination="{ rowsPerPage: 0 }"
                hide-pagination
              />
            </q-card-section>
          </q-card>
        </div>
      </div>
    </div>
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'report_design' })

import BaseContent from 'components/BaseContent/BaseContent.vue'
import { computed, onMounted, reactive, ref } from 'vue'
import type { QTableProps } from 'quasar'
import { useQuasar } from 'quasar'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import {
  useReportApi,
  type ReportDataSource,
  type ReportField,
  type ReportPreviewRes,
  type ReportSaveReq,
} from 'src/api/services/report'

const $q = useQuasar()
const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const reportApi = useReportApi()

const form = reactive<ReportSaveReq>({
  report_name: '',
  report_code: '',
  category: '',
  description: '',
  data_source_id: undefined,
  fields: [],
})

const dataSources = ref<ReportDataSource[]>([])
const selectedFieldRows = ref<ReportField[]>([])
const previewData = ref<ReportPreviewRes>({ columns: [], rows: [] })

const reportId = computed(() => Number(route.query.id || 0))
const selectedDataSource = computed(() => {
  return dataSources.value.find((item) => item.id === form.data_source_id)
})
const availableFields = computed(() => selectedDataSource.value?.fields || [])

const fieldColumns = computed<QTableProps['columns']>(() => [
  { name: 'name', field: 'name', label: t('report.fields.fieldName'), align: 'left' },
  { name: 'code', field: 'code', label: t('report.fields.fieldCode'), align: 'left' },
  { name: 'type', field: 'type', label: t('report.fields.fieldType'), align: 'left' },
])

const previewColumns = computed<QTableProps['columns']>(() => {
  return previewData.value.columns.map((field) => ({
    name: field.code,
    field: field.code,
    label: field.name,
    align: 'left',
  }))
})

const previewRows = computed(() => previewData.value.rows)

onMounted(async () => {
  await loadDataSources()
  await loadReport()
  if (!form.data_source_id && dataSources.value.length) {
    const firstDataSource = dataSources.value[0]
    if (firstDataSource) {
      form.data_source_id = firstDataSource.id
      handleDataSourceChange()
    }
  } else if (form.data_source_id && selectedFieldRows.value.length === 0) {
    handleDataSourceChange()
  }
  buildLocalPreview()
})

async function loadDataSources() {
  try {
    const res = await reportApi.queryDataSources()
    dataSources.value = res.data || []
  } catch {
    dataSources.value = sampleDataSources()
  }
}

async function loadReport() {
  if (!reportId.value) {
    form.report_name = t('report.samples.salesName')
    form.report_code = 'sales_overview'
    form.category = t('report.samples.categoryBusiness')
    form.description = t('report.design.defaultDescription')
    return
  }

  try {
    const res = await reportApi.queryReportById(reportId.value)
    Object.assign(form, {
      id: res.data.id,
      report_name: res.data.report_name,
      report_code: res.data.report_code,
      category: res.data.category || '',
      description: res.data.description || '',
      data_source_id: res.data.data_source_id,
    })
    const savedFields = reportApi.getSelectedFields(res.data)
    if (savedFields.length) {
      selectedFieldRows.value = savedFields
    }
  } catch {
    form.id = reportId.value
    form.report_name = t('report.samples.salesName')
    form.report_code = 'sales_overview'
    form.category = t('report.samples.categoryBusiness')
    form.description = t('report.design.defaultDescription')
  }
}

function handleDataSourceChange() {
  selectedFieldRows.value = availableFields.value.slice(0, 4)
  buildLocalPreview()
}

function selectAllFields() {
  selectedFieldRows.value = [...availableFields.value]
  buildLocalPreview()
}

async function preview() {
  form.fields = selectedFieldRows.value
  const req = {
    fields: form.fields,
  } as {
    report_id?: number
    data_source_id?: number | string
    fields: ReportField[]
  }
  if (form.id) req.report_id = form.id
  if (form.data_source_id) req.data_source_id = form.data_source_id

  try {
    const res = await reportApi.previewReport(req)
    previewData.value = res.data
  } catch {
    buildLocalPreview()
  }
}

async function saveReport() {
  form.fields = selectedFieldRows.value
  try {
    if (form.id) {
      await reportApi.updateReport(form)
    } else {
      const res = await reportApi.createReport(form)
      form.id = res.data
    }
    $q.notify({ type: 'positive', message: t('report.design.saveSuccess') })
  } catch {
    $q.notify({ type: 'warning', message: t('report.design.localSaveTip') })
  }
}

function goBack() {
  void router.push({ name: 'report_center' })
}

function buildLocalPreview() {
  const columns = selectedFieldRows.value.length
    ? selectedFieldRows.value
    : availableFields.value.slice(0, 4)
  previewData.value = {
    columns,
    rows: [1, 2, 3].map((id) => {
      const row: Record<string, unknown> = { id }
      columns.forEach((field, index) => {
        row[field.code] = sampleCellValue(field, id, index)
      })
      return row
    }),
  }
}

function sampleCellValue(field: ReportField, rowIndex: number, fieldIndex: number) {
  if (field.type === 'number') return rowIndex * 1000 + fieldIndex * 120
  if (field.type === 'date') return `2026-0${rowIndex + 3}-01`
  if (field.type === 'percent') return `${10 + rowIndex + fieldIndex}%`
  return `${field.name} ${rowIndex}`
}

function sampleDataSources(): ReportDataSource[] {
  return [
    {
      id: 'orders',
      name: t('report.samples.orderDataSource'),
      code: 'orders',
      description: t('report.samples.orderDataSourceDesc'),
      fields: [
        { name: t('report.samples.month'), code: 'month', type: 'date' },
        { name: t('report.samples.amount'), code: 'amount', type: 'number' },
        { name: t('report.samples.orders'), code: 'orders', type: 'number' },
        { name: t('report.samples.conversion'), code: 'conversion', type: 'percent' },
        { name: t('report.samples.region'), code: 'region', type: 'string' },
      ],
    },
    {
      id: 'users',
      name: t('report.samples.userDataSource'),
      code: 'users',
      description: t('report.samples.userDataSourceDesc'),
      fields: [
        { name: t('report.samples.date'), code: 'date', type: 'date' },
        { name: t('report.samples.newUsers'), code: 'new_users', type: 'number' },
        { name: t('report.samples.activeUsers'), code: 'active_users', type: 'number' },
        { name: t('report.samples.channel'), code: 'channel', type: 'string' },
      ],
    },
  ]
}
</script>

<style lang="scss" scoped>
.report-designer {
  min-height: 100%;
}
</style>
