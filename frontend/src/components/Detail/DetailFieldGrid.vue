<template>
  <div
    class="detail-field-grid"
    :class="{
      'detail-field-grid--plain': variant === 'plain',
      'detail-field-grid--card': variant === 'card',
    }"
  >
    <article
      v-for="item in items"
      :key="item.label"
      class="detail-field-grid__item"
      :class="{ 'detail-field-grid__item--full': item.fullWidth }"
    >
      <div class="detail-field-grid__label">
        <span>{{ item.label }}</span>
        <q-chip v-if="item.meta" dense square>{{ item.meta }}</q-chip>
      </div>
      <div class="detail-field-grid__value">
        <q-chip
          v-if="item.chip"
          dense
          square
          :outline="item.outline ?? variant === 'plain'"
          :color="item.color || 'primary'"
          :text-color="
            item.textColor || ((item.outline ?? variant === 'plain') ? undefined : 'white')
          "
        >
          {{ displayValue(item.value) }}
        </q-chip>
        <span v-else>{{ displayValue(item.value) }}</span>
      </div>
    </article>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import type { DetailFieldItem } from './types'

const { t } = useI18n({ useScope: 'global' })

withDefaults(
  defineProps<{
    items?: DetailFieldItem[]
    variant?: 'plain' | 'card'
  }>(),
  {
    items: () => [],
    variant: 'plain',
  },
)

const displayValue = (value: DetailFieldItem['value']) => {
  if (value === null || value === undefined || value === '') return '-'
  if (typeof value === 'boolean') return value ? t('ui.yes') : t('ui.no')
  return String(value)
}
</script>

<style scoped>
.detail-field-grid {
  display: grid;
}

.detail-field-grid--plain {
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 38px 52px;
}

.detail-field-grid--card {
  grid-template-columns: repeat(auto-fill, minmax(290px, 1fr));
  gap: 12px;
}

.detail-field-grid__item {
  min-width: 0;
}

.detail-field-grid--card .detail-field-grid__item {
  padding: 12px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.detail-field-grid__item--full {
  grid-column: 1 / -1;
}

.detail-field-grid__label {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  color: var(--app-text-muted);
  font-size: 12px;
}

.detail-field-grid__value {
  margin-top: 8px;
  color: var(--app-text-strong);
  font-size: 15px;
  font-weight: 600;
  overflow-wrap: anywhere;
}

@media (max-width: 700px) {
  .detail-field-grid--plain,
  .detail-field-grid--card {
    grid-template-columns: 1fr;
  }
}
</style>
