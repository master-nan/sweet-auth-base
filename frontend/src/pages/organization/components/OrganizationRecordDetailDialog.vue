<template>
  <q-dialog
    v-if="mode === 'dialog'"
    :model-value="modelValue"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <organization-record-detail-content
      style="width: 960px; max-width: 92vw"
      mode="dialog"
      :title="title"
      :subtitle="subtitle || ''"
      :items="items || []"
      :loading="Boolean(loading)"
      :error="error || ''"
      :top-buttons="topButtons || []"
      :bottom-buttons="bottomButtons || []"
      :record-context="recordContext || null"
      @close="emit('update:modelValue', false)"
      @button-click="emit('button-click', $event)"
    >
      <slot />
    </organization-record-detail-content>
  </q-dialog>
  <organization-record-detail-content
    v-else-if="modelValue"
    mode="page"
    :title="title"
    :subtitle="subtitle || ''"
    :items="items || []"
    :loading="Boolean(loading)"
    :error="error || ''"
    :top-buttons="topButtons || []"
    :bottom-buttons="bottomButtons || []"
    :record-context="recordContext || null"
    @close="emit('update:modelValue', false)"
    @button-click="emit('button-click', $event)"
  >
    <slot />
  </organization-record-detail-content>
</template>

<script setup lang="ts">
import type { MenuButton } from 'src/api/services/sys-menu'
import OrganizationRecordDetailContent from './OrganizationRecordDetailContent.vue'
import type { OrganizationDetailMode } from '../organization-detail-mode'
import type { OrganizationDetailItem } from './organization-record-detail'

withDefaults(
  defineProps<{
    modelValue: boolean
    mode?: OrganizationDetailMode
    title: string
    subtitle?: string
    items?: OrganizationDetailItem[]
    loading?: boolean
    error?: string
    topButtons?: MenuButton[]
    bottomButtons?: MenuButton[]
    recordContext?: object | null
  }>(),
  {
    mode: 'dialog',
    subtitle: '',
    items: () => [],
    loading: false,
    error: '',
    topButtons: () => [],
    bottomButtons: () => [],
    recordContext: null,
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'button-click': [button: MenuButton]
}>()
</script>
