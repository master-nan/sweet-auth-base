<template>
  <div class="organization-detail-page">
    <header class="organization-detail-page-header">
      <div class="row items-center no-wrap">
        <q-icon :name="icon" class="organization-detail-page-icon" />
        <div>
          <div class="organization-detail-page-title">{{ title }}</div>
          <div class="row items-center q-gutter-sm text-grey-7 q-mt-xs">
            <span v-if="subtitle">{{ subtitle }}</span>
            <q-chip v-if="statusLabel" dense square :color="statusColor" text-color="white">
              {{ statusLabel }}
            </q-chip>
          </div>
        </div>
      </div>
      <q-space />
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
        <q-btn
          flat
          color="primary"
          icon="arrow_back"
          :label="t('ui.backToList')"
          @click="emit('close')"
        />
        <q-btn
          outline
          color="primary"
          icon="refresh"
          :label="t('ui.refresh')"
          :loading="loading"
          @click="emit('refresh')"
        />
      </div>
    </header>

    <div v-if="loading" class="row flex-center organization-detail-loading">
      <q-spinner color="primary" size="40px" />
    </div>
    <q-banner v-else-if="error" class="text-negative">
      <template #avatar><q-icon name="error_outline" /></template>
      {{ error }}
    </q-banner>
    <template v-else>
      <section
        v-for="section in normalizedSections"
        :key="section.key"
        class="organization-detail-page-panel"
      >
        <div class="text-subtitle1 text-weight-bold q-mb-md">{{ section.label }}</div>
        <detail-field-grid v-if="section.items?.length" :items="section.items" variant="card" />
        <slot name="section" :section-key="section.key" :mode="mode" />
      </section>

      <slot :mode="mode" />

      <section v-if="(bottomButtons || []).length" class="organization-detail-page-panel">
        <div class="text-subtitle1 text-weight-bold q-mb-md">{{ t('ui.detailActions') }}</div>
        <div class="row q-gutter-sm">
          <q-btn
            v-for="button in bottomButtons || []"
            :key="button.id || button.code"
            v-bind="menuButtonDisplayProps(button)"
            unelevated
            :color="button.color || 'primary'"
            :disable="isButtonDisabled(button)"
            @click="emit('button-click', button)"
          />
        </div>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { computed } from 'vue'
import type { MenuButton } from '@/api/services/sys-menu'
import DetailFieldGrid from '@/components/Detail/DetailFieldGrid.vue'
import type { OrganizationDetailMode } from '@/pages/organization/organization-detail-mode'
import type {
  OrganizationDetailItem,
  OrganizationDetailSection,
} from './organization-record-detail'
import { evaluateButtonDisabled } from '@/utils/button-handlers'
import { menuButtonDisplayProps } from '@/utils/menu-button-display'

const { t } = useI18n({ useScope: 'global' })

const props = withDefaults(
  defineProps<{
    mode: OrganizationDetailMode
    title: string
    subtitle?: string
    items?: OrganizationDetailItem[]
    sections?: OrganizationDetailSection[]
    icon?: string
    statusLabel?: string
    statusColor?: string
    loading?: boolean
    error?: string
    topButtons?: MenuButton[]
    bottomButtons?: MenuButton[]
    recordContext?: object | null
  }>(),
  {
    subtitle: '',
    items: () => [],
    sections: () => [],
    icon: 'badge',
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
  close: []
  refresh: []
  'button-click': [button: MenuButton]
}>()

const normalizedSections = computed<OrganizationDetailSection[]>(() =>
  props.sections.length
    ? props.sections
    : [
        {
          key: 'basic',
          get label() {
            return t('ui.basicInfo')
          },
          items: props.items,
        },
      ],
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

<style scoped>
.organization-detail-page {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 14px;
  width: 100%;
  max-width: 1480px;
  margin: 0 auto;
}

.organization-detail-page-header,
.organization-detail-page-panel {
  border: 1px solid var(--app-border);
  background: var(--app-surface);
  border-radius: 8px;
}

.organization-detail-page-header {
  position: sticky;
  top: 0;
  z-index: 2;
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px 18px;
  box-shadow: 0 2px 8px rgba(31, 42, 68, 0.08);
}

.organization-detail-page-icon {
  width: 46px;
  height: 46px;
  margin-right: 14px;
  border-radius: 8px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 28px;
  color: white;
  background: var(--q-primary);
}

.organization-detail-page-title {
  font-size: 24px;
  font-weight: 700;
  line-height: 1.2;
}

.organization-detail-page-panel {
  padding: 20px 22px;
}

@media (max-width: 700px) {
  .organization-detail-page-header {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
