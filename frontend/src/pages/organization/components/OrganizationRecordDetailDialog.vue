<template>
  <form-dialog-shell
    v-if="mode === 'dialog'"
    :model-value="modelValue"
    :title="title"
    :subtitle="subtitle || ''"
    :icon="icon || 'badge'"
    readonly
    :show-submit="false"
    :show-preview="false"
    width="min(1240px, calc(100vw - 48px))"
    max-height="min(88vh, 860px)"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <template v-if="statusLabel" #title-extra>
      <q-chip dense square outline :color="statusColor || 'positive'">
        {{ statusLabel }}
      </q-chip>
    </template>

    <template v-if="(topButtons || []).length" #header-actions>
      <div class="row q-gutter-sm">
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
    </template>

    <template v-if="normalizedSections.length > 1" #navigation>
      <detail-section-navigation
        :model-value="activeSectionKey"
        :items="normalizedSections"
        @update:model-value="activeSectionKey = $event"
      />
    </template>

    <div v-if="loading" class="row flex-center organization-detail-loading">
      <q-spinner color="primary" size="40px" />
    </div>
    <q-banner v-else-if="error" class="text-negative">
      <template #avatar><q-icon name="error_outline" /></template>
      {{ error }}
    </q-banner>
    <section v-else>
      <div class="text-subtitle1 text-weight-bold q-mb-lg">
        {{ activeSection?.label || '基本信息' }}
      </div>
      <detail-field-grid :items="activeSection?.items || []" variant="plain" />
      <slot name="section" :section-key="activeSectionKey" mode="dialog" />
      <slot mode="dialog" />
    </section>

    <template v-if="(bottomButtons || []).length" #footer-actions>
      <q-btn
        v-for="button in bottomButtons || []"
        :key="button.id || button.code"
        v-bind="menuButtonDisplayProps(button)"
        unelevated
        :color="button.color || 'primary'"
        :disable="isButtonDisabled(button)"
        @click="emit('button-click', button)"
      />
    </template>
  </form-dialog-shell>

  <organization-record-detail-content
    v-else-if="modelValue"
    mode="page"
    :title="title"
    :subtitle="subtitle || ''"
    :items="items || []"
    :sections="sections || []"
    :icon="icon || 'badge'"
    :status-label="statusLabel || ''"
    :status-color="statusColor || 'positive'"
    :loading="Boolean(loading)"
    :error="error || ''"
    :top-buttons="topButtons || []"
    :bottom-buttons="bottomButtons || []"
    :record-context="recordContext || null"
    @close="emit('update:modelValue', false)"
    @button-click="emit('button-click', $event)"
  >
    <template #section="slotProps">
      <slot name="section" v-bind="slotProps" />
    </template>
    <template #default="slotProps">
      <slot v-bind="slotProps" />
    </template>
  </organization-record-detail-content>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { MenuButton } from 'src/api/services/sys-menu'
import DetailFieldGrid from 'src/components/Detail/DetailFieldGrid.vue'
import DetailSectionNavigation from 'src/components/Detail/DetailSectionNavigation.vue'
import FormDialogShell from 'src/components/FormDialog/FormDialogShell.vue'
import { evaluateButtonDisabled } from 'src/utils/button-handlers'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'
import OrganizationRecordDetailContent from './OrganizationRecordDetailContent.vue'
import type { OrganizationDetailMode } from '../organization-detail-mode'
import type {
  OrganizationDetailItem,
  OrganizationDetailSection,
} from './organization-record-detail'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    mode?: OrganizationDetailMode
    title: string
    subtitle?: string
    items?: OrganizationDetailItem[]
    sections?: OrganizationDetailSection[]
    icon?: string
    avatarLabel?: string
    statusLabel?: string
    statusColor?: string
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
    sections: () => [],
    icon: 'badge',
    avatarLabel: '',
    statusLabel: '',
    statusColor: 'positive',
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

const normalizedSections = computed<OrganizationDetailSection[]>(() =>
  props.sections.length
    ? props.sections
    : [{ key: 'basic', label: '基础信息', items: props.items }],
)
const activeSectionKey = ref('')
const activeSection = computed(
  () =>
    normalizedSections.value.find((section) => section.key === activeSectionKey.value) ||
    normalizedSections.value[0],
)

watch(
  [() => props.modelValue, normalizedSections],
  ([visible, sectionList]) => {
    const sections = sectionList as OrganizationDetailSection[]
    if (
      visible &&
      (!activeSectionKey.value ||
        !sections.some((section) => section.key === activeSectionKey.value))
    ) {
      activeSectionKey.value = sections[0]?.key || ''
    }
  },
  { immediate: true },
)

const isButtonDisabled = (button: MenuButton) =>
  evaluateButtonDisabled(button, {
    row: props.recordContext || {},
    selection: props.recordContext ? [props.recordContext] : [],
    selectionCount: props.recordContext ? 1 : 0,
    query: {},
    params: {},
  })
</script>
