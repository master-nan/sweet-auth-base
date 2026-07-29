<template>
  <q-card flat bordered>
    <q-card-section class="row items-start no-wrap">
      <q-btn
        v-if="mode === 'page'"
        flat
        round
        dense
        icon="arrow_back"
        class="q-mr-sm"
        @click="emit('close')"
      >
        <q-tooltip>返回列表</q-tooltip>
      </q-btn>
      <div>
        <div class="text-h6">{{ title }}</div>
        <div v-if="subtitle" class="text-caption text-grey-7">{{ subtitle }}</div>
      </div>
      <q-space />
      <div v-if="(topButtons || []).length" class="row q-gutter-xs">
        <q-btn
          v-for="button in topButtons || []"
          :key="button.id || button.code"
          v-bind="menuButtonDisplayProps(button)"
          unelevated
          :color="button.color || 'primary'"
          :disable="isButtonDisabled(button)"
          @click="emit('button-click', button)"
        />
      </div>
      <q-btn v-if="mode === 'dialog'" flat round dense icon="close" @click="emit('close')">
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
    <template v-else>
      <q-card-section class="row q-col-gutter-xl q-row-gutter-lg">
        <div
          v-for="item in items"
          :key="item.label"
          class="col-12"
          :class="{ 'col-sm-6': !item.fullWidth }"
        >
          <div class="text-caption text-grey-7 q-mb-xs">{{ item.label }}</div>
          <q-chip
            v-if="item.chip"
            dense
            square
            outline
            :color="item.color || 'primary'"
          >
            {{ displayValue(item.value) }}
          </q-chip>
          <div v-else class="text-body2 text-weight-medium">
            {{ displayValue(item.value) }}
          </div>
        </div>
      </q-card-section>
      <slot />
    </template>

    <template v-if="(bottomButtons || []).length || mode === 'dialog'">
      <q-separator />
      <q-card-actions align="right">
        <q-btn
          v-for="button in bottomButtons || []"
          :key="button.id || button.code"
          v-bind="menuButtonDisplayProps(button)"
          unelevated
          :color="button.color || 'primary'"
          :disable="isButtonDisabled(button)"
          @click="emit('button-click', button)"
        />
        <q-btn v-if="mode === 'dialog'" flat color="primary" label="关闭" @click="emit('close')" />
      </q-card-actions>
    </template>
  </q-card>
</template>

<script setup lang="ts">
import type { MenuButton } from 'src/api/services/sys-menu'
import type { OrganizationDetailMode } from 'src/pages/organization/organization-detail-mode'
import type { OrganizationDetailItem } from './organization-record-detail'
import { evaluateButtonDisabled } from 'src/utils/button-handlers'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'

const props = withDefaults(
  defineProps<{
    mode: OrganizationDetailMode
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
  close: []
  'button-click': [button: MenuButton]
}>()

const displayValue = (value: OrganizationDetailItem['value']) => {
  if (value === null || value === undefined || value === '') return '-'
  if (typeof value === 'boolean') return value ? '是' : '否'
  return String(value)
}

const isButtonDisabled = (button: MenuButton) =>
  evaluateButtonDisabled(button, {
    row: props.recordContext || {},
    selection: props.recordContext ? [props.recordContext] : [],
    selectionCount: props.recordContext ? 1 : 0,
    query: {},
    params: {},
  })
</script>
