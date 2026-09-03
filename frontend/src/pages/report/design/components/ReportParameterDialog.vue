<template>
  <q-dialog
    :model-value="modelValue"
    persistent
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <q-card class="parameter-dialog">
      <q-card-section class="dialog-head">
        <div>
          <div class="dialog-title">
            {{ editing ? t('ui.editParameters') : t('ui.addParameter') }}
          </div>
          <div class="dialog-caption">{{ t('ui.parametersAreDisplayedAtTheTopOfTheRunPage') }}</div>
        </div>
        <q-btn flat round dense icon="close" @click="$emit('update:modelValue', false)" />
      </q-card-section>

      <q-card-section class="dialog-form">
        <q-input
          :model-value="draft.label"
          dense
          outlined
          :label="t('ui.parameterName')"
          @update:model-value="$emit('update:label', String($event || ''))"
        />
        <q-select
          :model-value="draft.type"
          dense
          outlined
          emit-value
          map-options
          :label="t('ui.controlType')"
          :options="typeOptions"
          @update:model-value="$emit('update:type', $event as ReportParameterType)"
        />
        <q-select
          :model-value="draft.dataset_id"
          dense
          outlined
          emit-value
          map-options
          :label="t('ui.dataset')"
          :options="datasetOptions"
          @update:model-value="$emit('update:datasetId', String($event || ''))"
        />
        <q-select
          :model-value="draft.field"
          dense
          outlined
          emit-value
          map-options
          :label="t('ui.field')"
          :options="fieldOptions"
          @update:model-value="$emit('update:field', String($event || ''))"
        />
        <q-select
          :model-value="draft.operator"
          dense
          outlined
          emit-value
          map-options
          :label="t('ui.matchingMethod')"
          :options="operatorOptions"
          @update:model-value="$emit('update:operator', $event as ReportParameterOperator)"
        />
        <q-input
          :model-value="draft.placeholder"
          dense
          outlined
          :label="t('ui.placeholderTip')"
          @update:model-value="$emit('update:placeholder', String($event || ''))"
        />
        <q-input
          :model-value="draft.default_value"
          dense
          outlined
          clearable
          :label="t('ui.defaultValue')"
          :hint="t('ui.autoIntakeWhenRunningPagesOpenDateRangesSeparatedFrom')"
          @update:model-value="$emit('update:defaultValue', String($event || ''))"
        />
      </q-card-section>

      <q-card-actions align="right">
        <q-btn flat :label="t('ui.cancel')" @click="$emit('update:modelValue', false)" />
        <q-btn
          color="primary"
          unelevated
          icon="save"
          :label="t('ui.save')"
          @click="$emit('confirm')"
        />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import type { ReportParameterOperator, ReportParameterType } from '@/api/services/report'

const { t } = useI18n({ useScope: 'global' })

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
    default_value: string
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
  'update:defaultValue': [value: string]
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
