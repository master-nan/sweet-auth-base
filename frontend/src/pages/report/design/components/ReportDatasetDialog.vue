<template>
  <q-dialog
    :model-value="modelValue"
    persistent
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <q-card class="dataset-dialog">
      <q-card-section class="dialog-head">
        <div>
          <div class="dialog-title">
            {{ editingDataset ? t('ui.editDataSet') : t('ui.addDataSet') }}
          </div>
          <div class="dialog-caption">
            {{ t('ui.tableDataSetsCanBeDirectlySelectedForTheSystem') }}
          </div>
        </div>
        <q-btn flat round dense icon="close" @click="$emit('update:modelValue', false)" />
      </q-card-section>

      <q-card-section class="dialog-body">
        <div class="dataset-form">
          <q-select
            :model-value="draft.type"
            dense
            outlined
            emit-value
            map-options
            :label="t('ui.dataSetType')"
            :disable="!!editingDataset"
            :options="datasetTypeOptions"
            @update:model-value="$emit('update:type', $event as ReportDatasetType)"
          />
          <q-input
            :model-value="draft.name"
            dense
            outlined
            :label="t('ui.datasetName')"
            @update:model-value="$emit('update:name', String($event || ''))"
          />
          <q-select
            v-if="draft.type === 'table'"
            :model-value="draft.source_code"
            dense
            outlined
            emit-value
            map-options
            option-label="name"
            option-value="code"
            :label="t('ui.sourceTable')"
            use-input
            input-debounce="80"
            :options="tableSourceOptions"
            @filter="filterDataSources"
            @update:model-value="$emit('update:sourceCode', String($event || ''))"
          />
          <q-input
            v-else
            :model-value="draft.sql"
            outlined
            type="textarea"
            :label="t('ui.sqlStatement')"
            placeholder="select company_name, amount from tms_waybill where ..."
            @update:model-value="$emit('update:sql', String($event || ''))"
          />
          <div v-if="draft.type === 'sql'" class="sql-tools">
            <q-btn
              outline
              color="primary"
              icon="schema"
              :label="t('ui.parsingFields')"
              :loading="sqlFieldsLoading"
              @click="$emit('inferSqlFields')"
            />
            <span>{{
              t('ui.theSqlResultsColumnAutomaticallyGeneratesFieldsAvoidingManualMaintenance')
            }}</span>
          </div>
          <q-banner v-if="draft.type === 'sql'" dense rounded class="sql-note">
            <template #avatar>
              <q-icon name="verified" color="primary" />
            </template>
            {{ t('ui.sqlDataSetsMustBeGeneratedBeforeSavingTickThe') }}
          </q-banner>
          <q-expansion-item
            v-if="draft.type === 'sql'"
            dense
            class="manual-fields"
            icon="edit_note"
            :label="t('ui.manuallyFillFields')"
            :caption="t('ui.onlySqlsThatCannotBeParsedNeedToBeFilled')"
          >
            <q-input
              :model-value="draft.fieldsText"
              dense
              outlined
              :label="t('ui.field')"
              :hint="t('ui.commaSeparatedEGCompanyNameAmount')"
              @update:model-value="$emit('update:fieldsText', String($event || ''))"
            />
          </q-expansion-item>
        </div>

        <div class="field-preview">
          <div class="preview-head">
            <strong>{{ t('ui.fieldPreview') }}</strong>
            <span>{{ previewFields.length }} {{ t('ui.fieldCountSuffix') }}</span>
          </div>
          <q-table
            flat
            bordered
            dense
            class="field-preview-table"
            row-key="code"
            separator="cell"
            :rows="previewFields"
            :columns="fieldColumns"
            :pagination="{ rowsPerPage: 0 }"
            hide-pagination
          />
        </div>
      </q-card-section>

      <q-card-actions align="right">
        <q-btn flat :label="t('ui.cancel')" @click="$emit('update:modelValue', false)" />
        <q-btn
          color="primary"
          unelevated
          icon="save"
          :label="editingDataset ? t('ui.save') : t('ui.add')"
          @click="$emit('confirm')"
        />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { computed, ref, watch } from 'vue'
import type { QTableProps } from 'quasar'
import type {
  ReportDataSource,
  ReportDataset,
  ReportDatasetType,
  ReportField,
} from 'src/api/services/report'

const { t } = useI18n({ useScope: 'global' })

const props = defineProps<{
  modelValue: boolean
  editingDataset?: ReportDataset | undefined
  draft: {
    type: ReportDatasetType
    name: string
    source_code: string
    sql: string
    fieldsText: string
  }
  datasetTypeOptions: Array<{ label: string; value: ReportDatasetType }>
  dataSources: ReportDataSource[]
  previewFields: ReportField[]
  sqlFieldsLoading?: boolean
}>()

defineEmits<{
  'update:modelValue': [value: boolean]
  'update:type': [value: ReportDatasetType]
  'update:name': [value: string]
  'update:sourceCode': [value: string]
  'update:sql': [value: string]
  'update:fieldsText': [value: string]
  inferSqlFields: []
  confirm: []
}>()

const fieldColumns = computed<QTableProps['columns']>(() => [
  {
    name: 'name',
    field: 'name',
    get label() {
      return t('ui.fieldName')
    },
    align: 'left',
  },
  {
    name: 'code',
    field: 'code',
    get label() {
      return t('ui.fieldCode')
    },
    align: 'left',
  },
  {
    name: 'type',
    field: (row: ReportField) => fieldTypeLabel(row.type),
    get label() {
      return t('ui.type')
    },
    align: 'left',
  },
  {
    name: 'role',
    field: (row: ReportField) => row.role || '-',
    get label() {
      return t('ui.role')
    },
    align: 'left',
  },
])

const tableSourceOptions = ref<ReportDataSource[]>([])

watch(
  () => props.dataSources,
  (items) => {
    tableSourceOptions.value = [...items]
  },
  { immediate: true },
)

function filterDataSources(value: string, update: (callback: () => void) => void) {
  const keyword = value.trim().toLowerCase()
  update(() => {
    if (!keyword) {
      tableSourceOptions.value = [...props.dataSources]
      return
    }
    tableSourceOptions.value = props.dataSources.filter((item) => {
      return item.name.toLowerCase().includes(keyword) || item.code.toLowerCase().includes(keyword)
    })
  })
}

function fieldTypeLabel(type: string | number | undefined) {
  const value = String(type || '').toLowerCase()
  const labels: Record<string, string> = {
    get '1'() {
      return t('ui.number')
    },
    get '2'() {
      return t('ui.largeNumber')
    },
    get '3'() {
      return t('ui.string')
    },
    get '4'() {
      return t('ui.text')
    },
    get '5'() {
      return t('ui.boolean')
    },
    get '6'() {
      return t('ui.date')
    },
    get '7'() {
      return t('ui.time')
    },
    get number() {
      return t('ui.number')
    },
    get bigint() {
      return t('ui.largeNumber')
    },
    get integer() {
      return t('ui.number')
    },
    get string() {
      return t('ui.string')
    },
    get text() {
      return t('ui.text')
    },
    get boolean() {
      return t('ui.boolean')
    },
    get bool() {
      return t('ui.boolean')
    },
    get date() {
      return t('ui.date')
    },
    get time() {
      return t('ui.time')
    },
    get datetime() {
      return t('ui.dateAndTime')
    },
    get timestamp() {
      return t('ui.dateAndTime')
    },
  }
  return labels[value] || type || '-'
}
</script>

<style scoped lang="scss">
.dataset-dialog {
  width: min(1180px, 96vw);
  max-height: min(860px, 92vh);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.dialog-head {
  flex: 0 0 auto;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 24px 28px;
  border-bottom: 1px solid #e7ecf6;
}

.dialog-title {
  font-size: 20px;
  font-weight: 900;
}

.dialog-caption {
  color: #71809a;
}

.dialog-body {
  flex: 1 1 auto;
  min-height: 0;
  overflow: auto;
  padding: 20px 28px;
  display: grid;
  gap: 16px;
}

.dataset-dialog :deep(.q-card__actions) {
  flex: 0 0 auto;
  padding: 18px 28px;
  border-top: 1px solid #e7ecf6;
  background: #fff;
}

.dataset-form {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.dataset-form .q-field:nth-child(3),
.dataset-form .q-field:nth-child(4),
.dataset-form .sql-tools,
.dataset-form .manual-fields {
  grid-column: 1 / -1;
}

.sql-tools {
  display: flex;
  align-items: center;
  gap: 12px;
  color: #71809a;
  font-size: 12px;
}

.sql-note {
  grid-column: 1 / -1;
  color: #50607a;
  background: #f5f7ff;
  border: 1px solid #dfe5ff;
}

.field-preview {
  min-height: 0;
  display: grid;
  gap: 10px;
}

.field-preview-table {
  max-height: 300px;
  overflow: auto;
}

.preview-head {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.preview-head strong {
  font-size: 16px;
  font-weight: 900;
}

.preview-head span {
  color: #71809a;
  font-size: 12px;
}
</style>
