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
        <div class="join-datasets">
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
            :model-value="draft.join_type"
            dense
            outlined
            emit-value
            map-options
            :label="t('ui.relationMode')"
            :options="joinTypeOptions"
            @update:model-value="$emit('update:joinType', $event as ReportDatasetJoinType)"
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
        </div>

        <section class="condition-section">
          <div class="condition-head">
            <div>
              <q-icon name="compare_arrows" />
              <strong>{{ t('ui.associationConditions') }}</strong>
              <q-badge outline color="primary">{{ draft.conditions.length }}</q-badge>
            </div>
            <q-btn
              flat
              dense
              color="primary"
              icon="add"
              :label="t('ui.addCondition')"
              @click="addCondition"
            />
          </div>
          <div class="condition-list">
            <div v-for="(condition, index) in draft.conditions" :key="index" class="condition-row">
              <q-select
                :model-value="condition.left_field"
                dense
                outlined
                emit-value
                map-options
                :label="t('ui.leftField')"
                :options="leftFieldOptions"
                @update:model-value="updateCondition(index, 'left_field', String($event || ''))"
              />
              <div class="condition-equals" aria-hidden="true">=</div>
              <q-select
                :model-value="condition.right_field"
                dense
                outlined
                emit-value
                map-options
                :label="t('ui.rightField')"
                :options="rightFieldOptions"
                @update:model-value="updateCondition(index, 'right_field', String($event || ''))"
              />
              <q-btn
                flat
                round
                dense
                color="negative"
                icon="delete"
                :disable="draft.conditions.length <= 1"
                @click="removeCondition(index)"
              >
                <q-tooltip>{{ t('ui.delete') }}</q-tooltip>
              </q-btn>
            </div>
          </div>
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

import type { ReportDatasetJoinCondition, ReportDatasetJoinType } from '@/api/services/report'

const { t } = useI18n({ useScope: 'global' })

type Option<T = string> = { label: string; value: T }

const props = defineProps<{
  modelValue: boolean
  editing?: boolean
  draft: {
    left_dataset_id: string
    right_dataset_id: string
    join_type: ReportDatasetJoinType
    conditions: ReportDatasetJoinCondition[]
  }
  datasetOptions: Array<Option>
  leftFieldOptions: Array<Option>
  rightFieldOptions: Array<Option>
  joinTypeOptions: Array<Option<ReportDatasetJoinType>>
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'update:leftDatasetId': [value: string]
  'update:rightDatasetId': [value: string]
  'update:joinType': [value: ReportDatasetJoinType]
  'update:conditions': [value: ReportDatasetJoinCondition[]]
  confirm: []
}>()

function updateCondition(index: number, key: keyof ReportDatasetJoinCondition, value: string) {
  emit(
    'update:conditions',
    props.draft.conditions.map((condition, conditionIndex) =>
      conditionIndex === index ? { ...condition, [key]: value } : { ...condition },
    ),
  )
}

function addCondition() {
  emit('update:conditions', [
    ...props.draft.conditions.map((condition) => ({ ...condition })),
    {
      left_field: String(props.leftFieldOptions[0]?.value || ''),
      right_field: String(props.rightFieldOptions[0]?.value || ''),
    },
  ])
}

function removeCondition(index: number) {
  if (props.draft.conditions.length <= 1) return
  emit(
    'update:conditions',
    props.draft.conditions
      .filter((_, conditionIndex) => conditionIndex !== index)
      .map((condition) => ({ ...condition })),
  )
}
</script>

<style scoped lang="scss">
.join-dialog {
  width: min(900px, calc(100vw - 32px));
  max-width: 900px;
}

.dialog-head {
  display: flex;
  align-items: center;
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
  gap: 18px;
  padding: 20px;
}

.join-datasets {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 170px minmax(0, 1fr);
  gap: 12px;
}

.condition-section {
  display: grid;
  gap: 10px;
  padding-top: 14px;
  border-top: 1px solid #e7ecf6;
}

.condition-head,
.condition-head > div {
  display: flex;
  align-items: center;
  gap: 8px;
}

.condition-head {
  justify-content: space-between;
}

.condition-head .q-icon {
  color: var(--q-primary);
  font-size: 18px;
}

.condition-list {
  display: grid;
  gap: 8px;
}

.condition-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 28px minmax(0, 1fr) 32px;
  align-items: center;
  gap: 8px;
}

.condition-equals {
  color: #65738b;
  font-size: 18px;
  font-weight: 800;
  text-align: center;
}

.join-builder :deep(.q-field) {
  width: 100%;
  min-width: 0;
}

.join-builder :deep(.q-field--dense .q-field__control),
.join-builder :deep(.q-field--dense .q-field__marginal) {
  min-height: 42px;
  height: 42px;
}

.join-builder :deep(.q-field__native),
.join-builder :deep(.q-field__native > span) {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.join-dialog :deep(.q-card__actions) {
  min-height: 58px;
  padding: 10px 20px;
  border-top: 1px solid #e7ecf6;
}

@media (max-width: 760px) {
  .join-datasets,
  .condition-row {
    grid-template-columns: 1fr;
  }

  .condition-equals {
    display: none;
  }
}
</style>
