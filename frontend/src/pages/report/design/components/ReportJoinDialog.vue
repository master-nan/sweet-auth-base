<template>
  <q-dialog
    :model-value="modelValue"
    persistent
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <q-card class="join-dialog">
      <q-card-section class="dialog-head">
        <div class="dialog-title">
          {{ editing ? t('ui.dataSetAssociation') : t('ui.newDataSetAssociation') }}
        </div>
        <q-btn flat round dense icon="close" @click="$emit('update:modelValue', false)" />
      </q-card-section>

      <q-card-section class="dialog-form">
        <q-select
          :model-value="draft.left_dataset_id"
          dense
          outlined
          emit-value
          map-options
          :label="t('ui.leftDataSet')"
          :options="datasetOptions"
          @update:model-value="$emit('update:leftDatasetId', String($event || ''))"
        />
        <q-select
          :model-value="draft.left_field"
          dense
          outlined
          emit-value
          map-options
          :label="t('ui.leftField')"
          :options="leftFieldOptions"
          @update:model-value="$emit('update:leftField', String($event || ''))"
        />
        <q-select
          :model-value="draft.right_dataset_id"
          dense
          outlined
          emit-value
          map-options
          :label="t('ui.rightDataSet')"
          :options="datasetOptions"
          @update:model-value="$emit('update:rightDatasetId', String($event || ''))"
        />
        <q-select
          :model-value="draft.right_field"
          dense
          outlined
          emit-value
          map-options
          :label="t('ui.rightField')"
          :options="rightFieldOptions"
          @update:model-value="$emit('update:rightField', String($event || ''))"
        />
        <q-select
          :model-value="draft.join_type"
          dense
          outlined
          emit-value
          map-options
          :label="t('ui.relationMode')"
          :options="joinTypeOptions"
          @update:model-value="$emit('update:joinType', $event as ReportDatasetJoinType)"
        />
      </q-card-section>

      <q-card-actions align="right">
        <q-btn flat :label="t('ui.cancel')" @click="$emit('update:modelValue', false)" />
        <q-btn
          color="primary"
          unelevated
          icon="add_link"
          :label="editing ? t('ui.save') : t('ui.addAssociation')"
          @click="$emit('confirm')"
        />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import type { ReportDatasetJoinType } from 'src/api/services/report'

const { t } = useI18n({ useScope: 'global' })

type Option<T = string> = { label: string; value: T }

defineProps<{
  modelValue: boolean
  editing?: boolean
  draft: {
    left_dataset_id: string
    left_field: string
    right_dataset_id: string
    right_field: string
    join_type: ReportDatasetJoinType
  }
  datasetOptions: Array<Option>
  leftFieldOptions: Array<Option>
  rightFieldOptions: Array<Option>
  joinTypeOptions: Array<Option<ReportDatasetJoinType>>
}>()

defineEmits<{
  'update:modelValue': [value: boolean]
  'update:leftDatasetId': [value: string]
  'update:leftField': [value: string]
  'update:rightDatasetId': [value: string]
  'update:rightField': [value: string]
  'update:joinType': [value: ReportDatasetJoinType]
  confirm: []
}>()
</script>

<style scoped lang="scss">
.join-dialog {
  width: min(760px, 92vw);
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

.dialog-form {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.dialog-form .q-field:last-child {
  grid-column: 1 / -1;
}
</style>
