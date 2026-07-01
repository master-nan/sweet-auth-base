<template>
  <q-dialog :model-value="modelValue" persistent @update:model-value="$emit('update:modelValue', $event)">
    <q-card class="dataset-dialog">
      <q-card-section class="dialog-head">
        <div>
          <div class="dialog-title">{{ editingDataset ? '编辑数据集' : '新增数据集' }}</div>
          <div class="dialog-caption">表数据集可直接选择系统表，SQL 数据集先维护 SQL 与字段结构。</div>
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
            label="数据集类型"
            :disable="!!editingDataset"
            :options="datasetTypeOptions"
            @update:model-value="$emit('update:type', $event as ReportDatasetType)"
          />
          <q-input
            :model-value="draft.name"
            dense
            outlined
            label="数据集名称"
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
            label="来源表"
            :options="dataSources"
            @update:model-value="$emit('update:sourceCode', String($event || ''))"
          />
          <q-input
            v-else
            :model-value="draft.sql"
            outlined
            autogrow
            label="SQL 语句"
            placeholder="select company_name, amount from tms_waybill where ..."
            @update:model-value="$emit('update:sql', String($event || ''))"
          />
          <q-input
            v-if="draft.type === 'sql'"
            :model-value="draft.fieldsText"
            dense
            outlined
            label="字段"
            hint="逗号分隔，例如 company_name,amount"
            @update:model-value="$emit('update:fieldsText', String($event || ''))"
          />
        </div>

        <div class="field-preview">
          <div class="preview-head">
            <strong>字段预览</strong>
            <span>{{ previewFields.length }} 个字段</span>
          </div>
          <q-table
            flat
            bordered
            dense
            row-key="code"
            separator="cell"
            :rows="previewFields"
            :columns="fieldColumns"
            :pagination="{ rowsPerPage: 8 }"
          />
        </div>
      </q-card-section>

      <q-card-actions align="right">
        <q-btn flat label="取消" @click="$emit('update:modelValue', false)" />
        <q-btn
          color="primary"
          unelevated
          icon="save"
          :label="editingDataset ? '保存' : '添加'"
          @click="$emit('confirm')"
        />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { QTableProps } from 'quasar'
import type {
  ReportDataSource,
  ReportDataset,
  ReportDatasetType,
  ReportField,
} from 'src/api/services/report'

defineProps<{
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
}>()

defineEmits<{
  'update:modelValue': [value: boolean]
  'update:type': [value: ReportDatasetType]
  'update:name': [value: string]
  'update:sourceCode': [value: string]
  'update:sql': [value: string]
  'update:fieldsText': [value: string]
  confirm: []
}>()

const fieldColumns = computed<QTableProps['columns']>(() => [
  { name: 'name', field: 'name', label: '字段名称', align: 'left' },
  { name: 'code', field: 'code', label: '字段编码', align: 'left' },
  { name: 'type', field: 'type', label: '类型', align: 'left' },
  { name: 'role', field: (row: ReportField) => row.role || '-', label: '角色', align: 'left' },
])
</script>

<style scoped lang="scss">
.dataset-dialog {
  width: min(880px, 94vw);
}

.dialog-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
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
  display: grid;
  gap: 16px;
}

.dataset-form {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.dataset-form .q-field:nth-child(3),
.dataset-form .q-field:nth-child(4) {
  grid-column: 1 / -1;
}

.field-preview {
  display: grid;
  gap: 10px;
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
