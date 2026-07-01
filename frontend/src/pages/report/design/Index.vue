<template>
  <base-content class="q-pa-sm report-designer-page">
    <section class="designer-shell">
      <aside class="designer-left">
        <div class="side-section">
          <div class="section-title">基础信息</div>
          <div class="side-form">
            <q-input v-model="form.report_name" dense outlined label="报表名称" />
            <q-input v-model="form.report_code" dense outlined label="报表编码" />
            <q-select
              v-model="form.report_kind"
              dense
              outlined
              emit-value
              map-options
              label="报表类型"
              :options="kindOptions"
            />
            <q-input v-model="form.category" dense outlined label="分类" />
          </div>
        </div>

        <div class="side-section">
          <div class="section-title">数据集</div>
          <q-select
            v-model="form.data_source_id"
            dense
            outlined
            emit-value
            map-options
            option-label="name"
            option-value="id"
            label="选择数据集"
            :options="dataSources"
            @update:model-value="handleDataSourceChange"
          >
            <template #option="scope">
              <q-item v-bind="scope.itemProps">
                <q-item-section>
                  <q-item-label>{{ scope.opt.name }}</q-item-label>
                  <q-item-label caption>{{ scope.opt.description || scope.opt.code }}</q-item-label>
                </q-item-section>
              </q-item>
            </template>
          </q-select>
          <div class="dataset-meta" v-if="selectedDataSource">
            <q-chip dense square outline color="primary">{{ selectedDataSource.code }}</q-chip>
            <span>{{ selectedDataSource.fields.length }} 个字段</span>
          </div>
        </div>

        <div class="side-section fields-section">
          <div class="section-title row items-center">
            <span>字段</span>
            <q-space />
            <q-btn flat dense color="primary" icon="done_all" label="全选" @click="selectAllFields" />
          </div>
          <q-input v-model="fieldKeyword" dense outlined clearable placeholder="搜索字段">
            <template #prepend>
              <q-icon name="search" />
            </template>
          </q-input>
          <div class="field-list">
            <button
              v-for="field in filteredFields"
              :key="field.code"
              class="field-item"
              :class="{ selected: selectedFieldCodes.includes(field.code) }"
              @click="toggleField(field)"
            >
              <q-icon :name="fieldIcon(field)" />
              <span>
                <strong>{{ field.name }}</strong>
                <em>{{ field.code }}</em>
              </span>
              <q-badge outline color="primary">{{ field.type }}</q-badge>
            </button>
          </div>
        </div>

        <div class="side-section">
          <div class="section-title">组件</div>
          <div class="component-grid">
            <button
              v-for="component in componentOptions"
              :key="component.type"
              class="component-item"
              @click="addWidget(component.type)"
            >
              <q-icon :name="component.icon" />
              <span>{{ component.label }}</span>
            </button>
          </div>
        </div>
      </aside>

      <main class="designer-canvas">
        <div class="canvas-toolbar">
          <div class="toolbar-left">
            <q-btn outline color="primary" icon="arrow_back" label="返回" @click="goBack" />
            <q-btn outline color="primary" icon="preview" label="预览" @click="preview" />
          </div>
          <div class="toolbar-right">
            <q-btn outline color="primary" icon="tune" label="参数" @click="addParameter" />
            <q-btn color="primary" icon="save" label="保存设计" @click="saveReport" />
            <q-btn color="primary" icon="publish" label="发布" @click="publishReport" />
          </div>
        </div>

        <div class="sheet-wrap">
          <section class="report-sheet">
            <div class="sheet-title-row" @click="selectedWidgetId = ''">
              <div>
                <h1>{{ form.report_name || '未命名报表' }}</h1>
                <p>{{ form.description || '配置数据集、参数、组件后保存为可运行报表。' }}</p>
              </div>
              <q-chip dense square outline color="primary">{{ kindLabel(form.report_kind) }}</q-chip>
            </div>

            <div class="parameter-bar" v-if="parameters.length">
              <div
                v-for="param in parameters"
                :key="param.id"
                class="parameter-cell"
                :class="{ active: selectedParameterId === param.id }"
                @click="selectParameter(param.id)"
              >
                <span>{{ param.label }}</span>
                <em>{{ param.field }} · {{ param.operator }}</em>
              </div>
              <q-btn dense outline color="primary" icon="search" label="查询" />
            </div>
            <div v-else class="empty-bar">
              <q-icon name="tune" />
              <span>点击“参数”添加查询条件</span>
            </div>

            <div class="sheet-grid">
              <article
                v-for="widget in widgets"
                :key="widget.id"
                class="sheet-widget"
                :class="{ active: selectedWidgetId === widget.id }"
                @click="selectWidget(widget.id)"
              >
                <div class="widget-head">
                  <div>
                    <strong>{{ widget.title }}</strong>
                    <span>{{ widgetTypeLabel(widget.type) }}</span>
                  </div>
                  <q-btn flat dense round icon="delete" color="negative" @click.stop="removeWidget(widget.id)" />
                </div>

                <div v-if="widget.type === 'metric'" class="metric-preview">
                  <strong>{{ metricPreviewValue(widget) }}</strong>
                  <span>{{ widget.title }}</span>
                </div>
                <div v-else-if="widget.type === 'bar' || widget.type === 'line'" class="chart-preview">
                  <div
                    v-for="index in 7"
                    :key="index"
                    class="chart-bar"
                    :style="{ height: `${30 + index * 8}px` }"
                  />
                </div>
                <q-table
                  v-else
                  flat
                  bordered
                  dense
                  separator="cell"
                  row-key="id"
                  :rows="previewRows"
                  :columns="widgetColumns(widget)"
                  :pagination="{ rowsPerPage: 0 }"
                  hide-pagination
                />
              </article>
            </div>
          </section>
        </div>
      </main>

      <aside class="designer-right">
        <div class="side-section">
          <div class="section-title">属性</div>
          <div class="prop-empty" v-if="!activeWidget && !activeParameter">
            <q-icon name="ads_click" size="32px" />
            <span>点击画布组件或参数进行配置</span>
          </div>

          <div class="side-form" v-if="activeParameter">
            <q-input v-model="activeParameter.label" dense outlined label="参数标题" />
            <q-select
              v-model="activeParameter.field"
              dense
              outlined
              emit-value
              map-options
              label="绑定字段"
              :options="fieldOptions"
            />
            <q-select
              v-model="activeParameter.type"
              dense
              outlined
              emit-value
              map-options
              label="控件类型"
              :options="parameterTypeOptions"
            />
            <q-select
              v-model="activeParameter.operator"
              dense
              outlined
              emit-value
              map-options
              label="匹配方式"
              :options="operatorOptions"
            />
          </div>

          <div class="side-form" v-if="activeWidget">
            <q-input v-model="activeWidget.title" dense outlined label="组件标题" />
            <q-select
              v-model="activeWidget.type"
              dense
              outlined
              emit-value
              map-options
              label="组件类型"
              :options="widgetTypeOptions"
            />
            <q-select
              v-model="activeWidget.fields"
              dense
              outlined
              multiple
              emit-value
              map-options
              use-chips
              label="展示字段"
              :options="fieldOptions"
            />
            <q-select
              v-model="activeWidget.groupBy"
              dense
              outlined
              multiple
              emit-value
              map-options
              use-chips
              label="分组字段"
              :options="dimensionFieldOptions"
            />
          </div>
        </div>

        <div class="side-section">
          <div class="section-title">报表能力</div>
          <div class="capability-list">
            <div><q-icon name="security" /> 继承菜单数据权限</div>
            <div><q-icon name="download" /> 支持导出 XLSX / CSV</div>
            <div><q-icon name="history" /> 记录运行日志</div>
            <div><q-icon name="filter_alt" /> 参数过滤进入后端查询</div>
          </div>
        </div>

        <div class="side-section">
          <div class="section-title">运行预览</div>
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
        </div>
      </aside>
    </section>
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'report_design' })

import BaseContent from 'components/BaseContent/BaseContent.vue'
import { computed, onMounted, reactive, ref } from 'vue'
import type { QTableProps } from 'quasar'
import { useQuasar } from 'quasar'
import { useRoute, useRouter } from 'vue-router'
import {
  useReportApi,
  type ReportDataSource,
  type ReportField,
  type ReportKind,
  type ReportParameter,
  type ReportPreviewRes,
  type ReportSaveReq,
  type ReportWidget,
  type ReportWidgetType,
} from 'src/api/services/report'

const $q = useQuasar()
const route = useRoute()
const router = useRouter()
const reportApi = useReportApi()

const form = reactive<ReportSaveReq>({
  report_name: '',
  report_code: '',
  report_kind: 'detail',
  category: '',
  description: '',
  data_source_id: undefined,
  permission_menu_id: 0,
  permission_table_code: '',
  fields: [],
  parameters: [],
  widgets: [],
})

const dataSources = ref<ReportDataSource[]>([])
const selectedFields = ref<ReportField[]>([])
const parameters = ref<ReportParameter[]>([])
const widgets = ref<ReportWidget[]>([])
const previewData = ref<ReportPreviewRes>({ columns: [], rows: [] })
const fieldKeyword = ref('')
const selectedWidgetId = ref('')
const selectedParameterId = ref('')
const reportId = computed(() => Number(route.query.id || 0))

const kindOptions = [
  { label: '明细表', value: 'detail' },
  { label: '汇总表', value: 'summary' },
  { label: '图表', value: 'chart' },
  { label: '交叉表', value: 'pivot' },
]
const componentOptions: Array<{ type: ReportWidgetType; label: string; icon: string }> = [
  { type: 'table', label: '明细表', icon: 'table_rows' },
  { type: 'pivot', label: '交叉表', icon: 'pivot_table_chart' },
  { type: 'bar', label: '柱状图', icon: 'bar_chart' },
  { type: 'line', label: '折线图', icon: 'show_chart' },
  { type: 'metric', label: '指标卡', icon: 'monitoring' },
]
const widgetTypeOptions = componentOptions.map((item) => ({ label: item.label, value: item.type }))
const parameterTypeOptions = [
  { label: '文本', value: 'text' },
  { label: '下拉', value: 'select' },
  { label: '日期', value: 'date' },
  { label: '日期范围', value: 'date_range' },
  { label: '数字', value: 'number' },
]
const operatorOptions = [
  { label: '等于', value: 'eq' },
  { label: '包含', value: 'like' },
  { label: '区间', value: 'between' },
  { label: '大于等于', value: 'gte' },
  { label: '小于等于', value: 'lte' },
]

const selectedDataSource = computed(() =>
  dataSources.value.find((item) => item.id === form.data_source_id),
)
const availableFields = computed(() => selectedDataSource.value?.fields || [])
const selectedFieldCodes = computed(() => selectedFields.value.map((item) => item.code))
const filteredFields = computed(() => {
  const keyword = fieldKeyword.value.trim().toLowerCase()
  if (!keyword) return availableFields.value
  return availableFields.value.filter((field) =>
    [field.name, field.code].some((value) => value.toLowerCase().includes(keyword)),
  )
})
const fieldOptions = computed(() =>
  availableFields.value.map((field) => ({
    label: `${field.name} (${field.code})`,
    value: field.code,
  })),
)
const dimensionFieldOptions = computed(() =>
  availableFields.value
    .filter((field) => field.role !== 'metric')
    .map((field) => ({ label: `${field.name} (${field.code})`, value: field.code })),
)
const activeWidget = computed(() => widgets.value.find((item) => item.id === selectedWidgetId.value))
const activeParameter = computed(() =>
  parameters.value.find((item) => item.id === selectedParameterId.value),
)
const previewColumns = computed<QTableProps['columns']>(() =>
  previewData.value.columns.map((field) => ({
    name: field.code,
    field: field.code,
    label: field.name,
    align: 'left',
  })),
)
const previewRows = computed(() => previewData.value.rows)

onMounted(async () => {
  await loadDataSources()
  await loadReport()
  buildLocalPreview()
})

async function loadDataSources() {
  try {
    const res = await reportApi.queryDataSources()
    dataSources.value = res.data || []
  } catch {
    dataSources.value = []
    $q.notify({ type: 'negative', message: '数据集加载失败，请检查后端服务或数据源权限' })
  }
}

async function loadReport() {
  if (!reportId.value) {
    initNewReport()
    return
  }
  try {
    const res = await reportApi.queryReportById(reportId.value)
    Object.assign(form, {
      id: res.data.id,
      report_name: res.data.report_name,
      report_code: res.data.report_code,
      report_kind: res.data.report_kind,
      category: res.data.category || '',
      description: res.data.description || '',
      data_source_id: res.data.data_source_id,
      permission_menu_id: res.data.permission_menu_id || 0,
      permission_table_code: res.data.permission_table_code || String(res.data.data_source_id || ''),
    })
    selectedFields.value = reportApi.getSelectedFields(res.data)
    parameters.value = res.data.layout_config?.parameters || res.data.query_config?.parameters || []
    widgets.value = res.data.layout_config?.widgets?.length
      ? res.data.layout_config.widgets
      : createDefaultWidgets()
    if (selectedFields.value.length === 0) selectDefaultFields()
  } catch {
    $q.notify({ type: 'negative', message: '报表详情加载失败' })
    goBack()
  }
}

function initNewReport() {
  const first = dataSources.value[0]
  form.report_name = 'TMS 运单汇总'
  form.report_code = `report_${Date.now().toString().slice(-6)}`
  form.report_kind = 'detail'
  form.category = '经营分析'
  form.description = '用于按数据权限查看业务数据的报表'
  form.data_source_id = first?.id
  form.permission_table_code = first?.code || ''
  selectDefaultFields()
  parameters.value = []
  widgets.value = createDefaultWidgets()
}

function handleDataSourceChange() {
  const source = selectedDataSource.value
  form.permission_table_code = source?.code || ''
  selectDefaultFields()
  parameters.value = []
  widgets.value = createDefaultWidgets()
  buildLocalPreview()
}

function selectDefaultFields() {
  selectedFields.value = availableFields.value.slice(0, 6)
}

function selectAllFields() {
  selectedFields.value = [...availableFields.value]
  ensureWidgetFields()
  buildLocalPreview()
}

function toggleField(field: ReportField) {
  const exists = selectedFieldCodes.value.includes(field.code)
  selectedFields.value = exists
    ? selectedFields.value.filter((item) => item.code !== field.code)
    : [...selectedFields.value, field]
  ensureWidgetFields()
  buildLocalPreview()
}

function createDefaultWidgets(): ReportWidget[] {
  const fields = selectedFields.value.map((field) => field.code).slice(0, 5)
  return [
    {
      id: `widget_${Date.now()}`,
      type: 'table',
      title: '明细表',
      fields,
      groupBy: [],
      metrics: [],
    },
  ]
}

function ensureWidgetFields() {
  widgets.value.forEach((widget) => {
    widget.fields = widget.fields.filter((code) => selectedFieldCodes.value.includes(code))
    if (widget.fields.length === 0) {
      widget.fields = selectedFieldCodes.value.slice(0, 5)
    }
  })
}

function addWidget(type: ReportWidgetType) {
  const title = widgetTypeLabel(type)
  const widget: ReportWidget = {
    id: `widget_${Date.now()}`,
    type,
    title,
    fields: selectedFieldCodes.value.slice(0, type === 'metric' ? 1 : 5),
    groupBy: [],
    metrics: [],
  }
  widgets.value = [...widgets.value, widget]
  selectWidget(widget.id)
}

function removeWidget(id: string) {
  widgets.value = widgets.value.filter((item) => item.id !== id)
  if (selectedWidgetId.value === id) selectedWidgetId.value = widgets.value[0]?.id || ''
}

function selectWidget(id: string) {
  selectedWidgetId.value = id
  selectedParameterId.value = ''
}

function selectParameter(id: string) {
  selectedParameterId.value = id
  selectedWidgetId.value = ''
}

function addParameter() {
  const field = selectedFields.value[0] || availableFields.value[0]
  if (!field) return
  const parameter: ReportParameter = {
    id: `param_${Date.now()}`,
    label: field.name,
    field: field.code,
    type: field.role === 'time' ? 'date_range' : 'text',
    operator: field.role === 'time' ? 'between' : 'like',
    placeholder: `请输入${field.name}`,
  }
  parameters.value = [...parameters.value, parameter]
  selectParameter(parameter.id)
}

async function preview() {
  if (!form.id) {
    buildLocalPreview()
    $q.notify({ type: 'info', message: '当前为未保存设计的本地预览，保存后可运行真实数据' })
    return
  }
  try {
    const res = await reportApi.previewReport({
      report_id: form.id,
      data_source_id: form.data_source_id,
    })
    previewData.value = res.data
  } catch {
    previewData.value = { columns: [], rows: [] }
    $q.notify({ type: 'negative', message: '真实数据预览失败，请检查报表配置或数据权限' })
  }
}

async function saveReport() {
  syncForm()
  if (!validateReport()) return
  try {
    if (form.id) {
      await reportApi.updateReport(form)
    } else {
      const res = await reportApi.createReport(form)
      form.id = res.data
    }
    $q.notify({ type: 'positive', message: '报表设计已保存' })
    await preview()
  } catch {
    $q.notify({ type: 'negative', message: '报表保存失败' })
  }
}

async function publishReport() {
  await saveReport()
  if (form.id) {
    $q.notify({ type: 'positive', message: '报表已发布，可在报表中心运行' })
  }
}

function syncForm() {
  form.fields = selectedFields.value
  form.parameters = parameters.value
  form.widgets = widgets.value
  form.permission_table_code = form.permission_table_code || String(form.data_source_id || '')
}

function validateReport() {
  if (!form.report_name.trim()) {
    $q.notify({ type: 'warning', message: '请填写报表名称' })
    return false
  }
  if (!form.report_code.trim()) {
    $q.notify({ type: 'warning', message: '请填写报表编码' })
    return false
  }
  if (!form.data_source_id) {
    $q.notify({ type: 'warning', message: '请选择数据集' })
    return false
  }
  if (selectedFields.value.length === 0) {
    $q.notify({ type: 'warning', message: '请至少选择一个字段' })
    return false
  }
  if (widgets.value.length === 0) {
    $q.notify({ type: 'warning', message: '请至少添加一个报表组件' })
    return false
  }
  return true
}

function goBack() {
  void router.push({ name: 'report_center' })
}

function buildLocalPreview() {
  const columns = selectedFields.value.length ? selectedFields.value : availableFields.value.slice(0, 5)
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

function widgetColumns(widget: ReportWidget): QTableProps['columns'] {
  const codes = widget.fields.length ? widget.fields : selectedFieldCodes.value.slice(0, 5)
  return codes
    .map((code) => selectedFields.value.find((field) => field.code === code))
    .filter(Boolean)
    .map((field) => ({
      name: field!.code,
      field: field!.code,
      label: field!.name,
      align: 'left',
    }))
}

function metricPreviewValue(widget: ReportWidget) {
  const field = selectedFields.value.find((item) => item.code === widget.fields[0])
  if (!field) return '0'
  return field.role === 'metric' ? '128,600' : '42'
}

function sampleCellValue(field: ReportField, rowIndex: number, fieldIndex: number) {
  if (field.role === 'metric') return rowIndex * 1000 + fieldIndex * 120
  if (field.role === 'time') return `2026-07-${String(rowIndex + fieldIndex).padStart(2, '0')}`
  if (field.code.includes('status')) return ['待发车', '已创建', '已发车'][rowIndex - 1] || '已创建'
  return `${field.name}${rowIndex}`
}

function fieldIcon(field: ReportField) {
  if (field.role === 'metric') return 'pin'
  if (field.role === 'time') return 'event'
  if (field.role === 'dimension') return 'category'
  return 'text_fields'
}

function widgetTypeLabel(type: ReportWidgetType) {
  const map: Record<ReportWidgetType, string> = {
    filter: '参数区',
    table: '明细表',
    pivot: '交叉表',
    bar: '柱状图',
    line: '折线图',
    metric: '指标卡',
  }
  return map[type] || '组件'
}

function kindLabel(kind: ReportKind) {
  const map: Record<ReportKind, string> = {
    detail: '明细表',
    summary: '汇总表',
    chart: '图表',
    pivot: '交叉表',
  }
  return map[kind] || '明细表'
}

</script>

<style scoped lang="scss">
.report-designer-page {
  min-height: 0;
}

.designer-shell {
  min-height: calc(100vh - 150px);
  display: grid;
  grid-template-columns: 300px minmax(0, 1fr) 330px;
  border: 1px solid #dfe5f2;
  border-radius: 8px;
  overflow: hidden;
  background: #fff;
}

.designer-left,
.designer-right {
  min-width: 0;
  overflow: auto;
  background: #fbfcff;
}

.designer-left {
  border-right: 1px solid #dfe5f2;
}

.designer-right {
  border-left: 1px solid #dfe5f2;
}

.side-section {
  padding: 14px;
  border-bottom: 1px solid #dfe5f2;
}

.section-title {
  margin-bottom: 10px;
  font-size: 16px;
  font-weight: 800;
  color: #172033;
}

.side-form {
  display: grid;
  gap: 10px;
}

.dataset-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 10px;
  color: #71809a;
}

.fields-section {
  max-height: 360px;
  display: flex;
  flex-direction: column;
}

.field-list {
  min-height: 0;
  display: grid;
  gap: 8px;
  margin-top: 10px;
  overflow: auto;
}

.field-item,
.component-item {
  width: 100%;
  border: 1px solid #dfe5f2;
  border-radius: 8px;
  background: #fff;
  color: #172033;
  cursor: pointer;
}

.field-item {
  min-height: 58px;
  display: grid;
  grid-template-columns: 24px 1fr auto;
  gap: 8px;
  align-items: center;
  padding: 8px 10px;
  text-align: left;
}

.field-item.selected {
  border-color: var(--q-primary);
  background: #f7f5ff;
}

.field-item strong,
.field-item em {
  display: block;
}

.field-item em {
  margin-top: 2px;
  color: #71809a;
  font-style: normal;
  font-size: 12px;
}

.component-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.component-item {
  min-height: 66px;
  display: grid;
  place-items: center;
  gap: 4px;
  color: var(--q-primary);
  font-weight: 800;
}

.designer-canvas {
  min-width: 0;
  display: grid;
  grid-template-rows: 58px 1fr;
}

.canvas-toolbar {
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 10px 12px;
  border-bottom: 1px solid #dfe5f2;
  background: #fff;
}

.toolbar-left,
.toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.sheet-wrap {
  min-width: 0;
  overflow: auto;
  padding: 18px;
  background:
    linear-gradient(#eef2fb 1px, transparent 1px),
    linear-gradient(90deg, #eef2fb 1px, transparent 1px);
  background-size: 32px 32px;
}

.report-sheet {
  min-width: 900px;
  min-height: 620px;
  padding: 18px;
  border: 1px solid #cfd6e6;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 10px 24px rgba(24, 32, 51, 0.08);
}

.sheet-title-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding-bottom: 14px;
  border-bottom: 1px solid #dfe5f2;
  cursor: pointer;
}

.sheet-title-row h1 {
  margin: 0;
  font-size: 24px;
}

.sheet-title-row p {
  margin: 6px 0 0;
  color: #71809a;
}

.parameter-bar,
.empty-bar {
  min-height: 58px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 0;
  border-bottom: 1px solid #dfe5f2;
}

.parameter-cell {
  min-width: 150px;
  display: grid;
  gap: 2px;
  padding: 8px 10px;
  border: 1px solid #dfe5f2;
  border-radius: 8px;
  cursor: pointer;
}

.parameter-cell.active {
  border-color: var(--q-primary);
  background: #f7f5ff;
}

.parameter-cell em,
.empty-bar {
  color: #71809a;
  font-style: normal;
}

.sheet-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  padding-top: 14px;
}

.sheet-widget {
  min-width: 0;
  min-height: 220px;
  padding: 12px;
  border: 1px solid #dfe5f2;
  border-radius: 8px;
  background: #fff;
  cursor: pointer;
}

.sheet-widget.active {
  border-color: var(--q-primary);
  box-shadow: 0 0 0 2px rgba(115, 103, 240, 0.12);
}

.widget-head {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 10px;
}

.widget-head strong,
.widget-head span {
  display: block;
}

.widget-head span {
  color: #71809a;
  margin-top: 2px;
}

.metric-preview {
  min-height: 150px;
  display: grid;
  place-items: center;
  align-content: center;
  color: var(--q-primary);
}

.metric-preview strong {
  font-size: 36px;
}

.chart-preview {
  min-height: 160px;
  display: flex;
  align-items: end;
  gap: 12px;
  padding: 18px;
  border: 1px dashed #dfe5f2;
  border-radius: 8px;
}

.chart-bar {
  width: 34px;
  border-radius: 6px 6px 0 0;
  background: var(--q-primary);
}

.prop-empty {
  min-height: 160px;
  display: grid;
  place-items: center;
  gap: 8px;
  color: #71809a;
}

.capability-list {
  display: grid;
  gap: 10px;
  color: #5f6f88;
}

.capability-list div {
  display: flex;
  align-items: center;
  gap: 8px;
}

@media (max-width: 1280px) {
  .designer-shell {
    grid-template-columns: 1fr;
  }

  .designer-left,
  .designer-right {
    border: 0;
    border-bottom: 1px solid #dfe5f2;
  }

  .sheet-grid {
    grid-template-columns: 1fr;
  }
}
</style>
