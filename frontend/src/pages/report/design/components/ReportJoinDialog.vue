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

        <div class="join-arrow" aria-hidden="true">
          <q-icon name="arrow_forward" />
        </div>

        <div class="join-operator">
          <div class="join-side__title">
            <q-icon name="compare_arrows" />
            <span>{{ t('ui.relationMode') }}</span>
          </div>
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

        <div class="join-arrow" aria-hidden="true">
          <q-icon name="arrow_forward" />
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
  width: min(920px, calc(100vw - 32px));
  max-width: 920px;
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
  grid-template-columns:
    minmax(220px, 1fr) 28px minmax(150px, 180px) 28px
    minmax(220px, 1fr);
  align-items: center;
  gap: 10px;
  padding: 20px;
}

.join-side {
  min-width: 0;
  display: grid;
  gap: 10px;
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
  min-width: 0;
  display: grid;
  gap: 10px;
}

.join-arrow {
  display: grid;
  place-items: center;
  color: var(--q-primary);
}

.join-arrow .q-icon {
  font-size: 20px;
}

.join-side :deep(.q-field),
.join-operator :deep(.q-field) {
  width: 100%;
  min-width: 0;
}

.join-side :deep(.q-field__native),
.join-side :deep(.q-field__native > span),
.join-operator :deep(.q-field__native),
.join-operator :deep(.q-field__native > span) {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.join-side :deep(.q-field--dense .q-field__control),
.join-side :deep(.q-field--dense .q-field__marginal),
.join-operator :deep(.q-field--dense .q-field__control),
.join-operator :deep(.q-field--dense .q-field__marginal) {
  min-height: 42px;
  height: 42px;
}

.join-dialog :deep(.q-card__actions) {
  min-height: 58px;
  padding: 10px 20px;
  border-top: 1px solid #e7ecf6;
}

@media (max-width: 760px) {
  .join-builder {
    grid-template-columns: 1fr;
  }

  .join-arrow .q-icon {
    transform: rotate(90deg);
  }
}
</style>
