<template>
  <q-dialog :model-value="modelValue" @update:model-value="emit('update:modelValue', $event)">
    <q-card style="width: 720px; max-width: 92vw">
      <q-card-section class="row items-start no-wrap">
        <div>
          <div class="text-h6">{{ title }}</div>
          <div v-if="subtitle" class="text-caption text-grey-7">{{ subtitle }}</div>
        </div>
        <q-space />
        <q-btn v-close-popup flat round dense icon="close">
          <q-tooltip>关闭</q-tooltip>
        </q-btn>
      </q-card-section>
      <q-separator />

      <q-card-section v-if="loading" class="row justify-center q-pa-xl">
        <q-spinner color="primary" size="32px" />
      </q-card-section>
      <q-banner v-else-if="error" class="text-negative">
        <template #avatar><q-icon name="error_outline" /></template>
        {{ error }}
      </q-banner>
      <q-list v-else separator>
        <q-item v-for="item in items" :key="item.label">
          <q-item-section>
            <q-item-label caption>{{ item.label }}</q-item-label>
            <q-item-label>
              <q-chip
                v-if="item.chip"
                dense
                square
                outline
                :color="item.color || 'primary'"
              >
                {{ displayValue(item.value) }}
              </q-chip>
              <span v-else class="text-body2">{{ displayValue(item.value) }}</span>
            </q-item-label>
          </q-item-section>
        </q-item>
      </q-list>

      <q-card-actions align="right">
        <q-btn v-close-popup flat color="primary" label="关闭" />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
export interface OrganizationDetailItem {
  label: string
  value?: string | number | boolean | null
  chip?: boolean
  color?: string
}

withDefaults(
  defineProps<{
    modelValue: boolean
    title: string
    subtitle?: string
    items?: OrganizationDetailItem[]
    loading?: boolean
    error?: string
  }>(),
  {
    subtitle: '',
    items: () => [],
    loading: false,
    error: '',
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const displayValue = (value: OrganizationDetailItem['value']) => {
  if (value === null || value === undefined || value === '') return '-'
  if (typeof value === 'boolean') return value ? '是' : '否'
  return String(value)
}
</script>
