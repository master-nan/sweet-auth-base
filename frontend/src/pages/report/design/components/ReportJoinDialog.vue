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

      <q-card-section class="join-builder">
        <section class="join-side">
          <div class="join-side__title">
            <q-icon name="table_chart" />
            <span>{{ t('ui.leftDataSet') }}</span>
          </div>
          <q-select
            :model-value="draft.left_dataset_id"
            dense
            outlined
            emit-value
            map-options
            :label="t('ui.dataset')"
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
        </section>

        <div class="join-operator">
          <q-icon name="compare_arrows" />
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
        </div>

        <section class="join-side">
          <div class="join-side__title">
            <q-icon name="table_chart" />
            <span>{{ t('ui.rightDataSet') }}</span>
          </div>
          <q-select
            :model-value="draft.right_dataset_id"
            dense
            outlined
            emit-value
            map-options
            :label="t('ui.dataset')"
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
        </section>
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
  font-size: 18px;
  font-weight: 900;
}

.join-builder {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 132px minmax(0, 1fr);
  align-items: center;
  gap: 12px;
  padding: 18px;
}

.join-side {
  min-width: 0;
  display: grid;
  gap: 10px;
  padding: 12px;
  border: 1px solid #e1e7f1;
  border-radius: 8px;
  background: #fbfcff;
}

.join-side__title {
  display: flex;
  align-items: center;
  gap: 7px;
  color: #263248;
  font-size: 13px;
  font-weight: 800;
}

.join-side__title .q-icon {
  color: var(--q-primary);
  font-size: 18px;
}

.join-operator {
  display: grid;
  gap: 8px;
  text-align: center;
}

.join-operator > .q-icon {
  justify-self: center;
  color: var(--q-primary);
  font-size: 24px;
}

@media (max-width: 680px) {
  .join-builder {
    grid-template-columns: 1fr;
  }

  .join-operator {
    grid-template-columns: 28px minmax(0, 1fr);
    align-items: center;
    text-align: left;
  }
}
</style>
