<template>
  <div class="report-filter-bar">
    <q-input
      :model-value="keyword"
      dense
      outlined
      clearable
      class="filter-keyword"
      :placeholder="t('ui.searchForReportNameCodeOrResponsiblePerson')"
      @update:model-value="$emit('update:keyword', String($event || ''))"
      @keyup.enter="$emit('search')"
    >
      <template #prepend>
        <q-icon name="search" />
      </template>
    </q-input>

    <q-select
      :model-value="status"
      dense
      outlined
      emit-value
      map-options
      :label="t('ui.status')"
      :options="statusOptions"
      @update:model-value="$emit('update:status', String($event || ''))"
    />

    <q-select
      :model-value="category"
      dense
      outlined
      emit-value
      map-options
      :label="t('ui.category')"
      :options="categoryOptions"
      @update:model-value="$emit('update:category', String($event || ''))"
    />

    <div class="filter-actions">
      <q-btn
        outline
        color="primary"
        icon="restart_alt"
        :label="t('ui.reset')"
        @click="$emit('reset')"
      />
      <q-btn
        color="primary"
        unelevated
        icon="search"
        :label="t('ui.query')"
        @click="$emit('search')"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n({ useScope: 'global' })
defineProps<{
  keyword: string
  status: string
  category: string
  statusOptions: Array<{ label: string; value: string }>
  categoryOptions: Array<{ label: string; value: string }>
}>()

defineEmits<{
  'update:keyword': [value: string]
  'update:status': [value: string]
  'update:category': [value: string]
  search: []
  reset: []
}>()
</script>

<style scoped lang="scss">
.report-filter-bar {
  display: grid;
  grid-template-columns: minmax(280px, 1fr) 160px 180px auto;
  gap: 10px;
  align-items: center;
}

.filter-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}
</style>
