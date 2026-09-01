<template>
  <q-dialog
    :model-value="modelValue"
    maximized
    @hide="handleDialogValue(false)"
    @update:model-value="handleDialogValue"
  >
    <q-card class="runtime-dialog">
      <report-runtime-view
        :active="modelValue"
        :report="report"
        :default-page-size="defaultPageSize"
        :allow-export="allowExport ?? true"
        :mode="mode || 'center'"
        show-close
        @close="handleDialogValue(false)"
      />
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import type { Report } from 'src/api/services/report'
import ReportRuntimeView from './ReportRuntimeView.vue'

withDefaults(
  defineProps<{
    modelValue: boolean
    report: Report | null
    defaultPageSize?: number | undefined
    allowExport?: boolean
    mode?: 'center' | 'manage'
  }>(),
  {
    allowExport: true,
    mode: 'center',
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

function handleDialogValue(value: boolean) {
  emit('update:modelValue', value)
}
</script>

<style scoped>
.runtime-dialog {
  display: flex;
  flex-direction: column;
}
</style>
