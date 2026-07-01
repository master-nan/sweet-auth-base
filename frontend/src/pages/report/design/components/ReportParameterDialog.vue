<template>
  <q-dialog :model-value="modelValue" persistent @update:model-value="$emit('update:modelValue', $event)">
    <q-card class="parameter-dialog">
      <q-card-section class="dialog-head">
        <div>
          <div class="dialog-title">{{ editing ? '编辑参数' : '新增参数' }}</div>
          <div class="dialog-caption">参数会显示在报表运行页顶部，并转换为绑定数据集字段的查询条件。</div>
        </div>
        <q-btn flat round dense icon="close" @click="$emit('update:modelValue', false)" />
      </q-card-section>

      <q-card-section class="dialog-form">
        <q-input
          :model-value="draft.label"
          dense
          outlined
          label="参数名称"
          @update:model-value="$emit('update:label', String($event || ''))"
        />
        <q-select
          :model-value="draft.type"
          dense
          outlined
          emit-value
          map-options
          label="控件类型"
          :options="typeOptions"
          @update:model-value="$emit('update:type', $event as ReportParameterType)"
        />
        <q-select
          :model-value="draft.dataset_id"
          dense
          outlined
          emit-value
          map-options
          label="数据集"
          :options="datasetOptions"
          @update:model-value="$emit('update:datasetId', String($event || ''))"
        />
        <q-select
          :model-value="draft.field"
          dense
          outlined
          emit-value
          map-options
          label="字段"
          :options="fieldOptions"
          @update:model-value="$emit('update:field', String($event || ''))"
        />
        <q-select
          :model-value="draft.operator"
          dense
          outlined
          emit-value
          map-options
          label="匹配方式"
          :options="operatorOptions"
          @update:model-value="$emit('update:operator', $event as ReportParameterOperator)"
        />
        <q-input
          :model-value="draft.placeholder"
          dense
          outlined
          label="占位提示"
          @update:model-value="$emit('update:placeholder', String($event || ''))"
        />
      </q-card-section>

      <q-card-actions align="right">
        <q-btn flat label="取消" @click="$emit('update:modelValue', false)" />
        <q-btn color="primary" unelevated icon="save" label="保存" @click="$emit('confirm')" />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import type {
  ReportParameterOperator,
  ReportParameterType,
} from 'src/api/services/report'

type Option<T = string> = { label: string; value: T }

defineProps<{
  modelValue: boolean
  editing: boolean
  draft: {
    id: string
    label: string
    dataset_id: string
    field: string
    type: ReportParameterType
    operator: ReportParameterOperator
    placeholder: string
  }
  datasetOptions: Array<Option>
  fieldOptions: Array<Option>
  typeOptions: Array<Option<ReportParameterType>>
  operatorOptions: Array<Option<ReportParameterOperator>>
}>()

defineEmits<{
  'update:modelValue': [value: boolean]
  'update:label': [value: string]
  'update:datasetId': [value: string]
  'update:field': [value: string]
  'update:type': [value: ReportParameterType]
  'update:operator': [value: ReportParameterOperator]
  'update:placeholder': [value: string]
  confirm: []
}>()
</script>

<style scoped lang="scss">
.parameter-dialog {
  width: min(720px, 92vw);
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

.dialog-form {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}
</style>
