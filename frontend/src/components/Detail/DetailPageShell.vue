<template>
  <base-content scrollable class="q-pa-sm detail-page-shell">
    <div class="detail-page-shell__content">
      <header class="detail-page-shell__header">
        <div class="detail-page-shell__title-wrap">
          <q-icon :name="icon" class="detail-page-shell__icon" />
          <div class="detail-page-shell__heading">
            <div class="detail-page-shell__title">{{ title }}</div>
            <div v-if="subtitle || $slots.subtitle" class="detail-page-shell__subtitle">
              <slot name="subtitle">{{ subtitle }}</slot>
            </div>
          </div>
        </div>
        <q-space />
        <div v-if="$slots.actions" class="detail-page-shell__actions">
          <slot name="actions" />
        </div>
      </header>

      <q-inner-loading :showing="loading">
        <q-spinner color="primary" size="42px" />
      </q-inner-loading>

      <q-banner v-if="error" rounded class="detail-page-shell__error">
        <template #avatar>
          <q-icon name="error_outline" color="negative" />
        </template>
        {{ error }}
        <template v-if="retryable" #action>
          <q-btn flat color="negative" :label="displayRetryLabel" @click="emit('retry')" />
        </template>
      </q-banner>

      <slot />
    </div>
  </base-content>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

import BaseContent from 'src/components/BaseContent/BaseContent.vue'

const { t } = useI18n({ useScope: 'global' })

defineOptions({ name: 'DetailPageShell' })

const props = withDefaults(
  defineProps<{
    title: string
    subtitle?: string
    icon: string
    loading?: boolean
    error?: string
    retryable?: boolean
    retryLabel?: string
  }>(),
  {
    subtitle: '',
    loading: false,
    error: '',
    retryable: false,
    retryLabel: '',
  },
)

const displayRetryLabel = computed(() => props.retryLabel || t('ui.reload'))

const emit = defineEmits<{
  retry: []
}>()
</script>

<style scoped lang="scss">
.detail-page-shell {
  background: var(--app-surface-muted);
}

.detail-page-shell__content {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 14px;
  width: 100%;
  max-width: 1480px;
  margin: 0 auto;
  box-sizing: border-box;
}

.detail-page-shell__header,
:deep(.detail-page-section) {
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-surface);
}

.detail-page-shell__header {
  position: sticky;
  top: 0;
  z-index: 6;
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px 18px;
  box-shadow: 0 6px 18px rgba(26, 35, 58, 0.06);
}

.detail-page-shell__title-wrap {
  display: flex;
  align-items: center;
  gap: 14px;
  min-width: 0;
}

.detail-page-shell__heading {
  min-width: 0;
}

.detail-page-shell__icon {
  display: inline-flex;
  flex: 0 0 46px;
  align-items: center;
  justify-content: center;
  width: 46px;
  height: 46px;
  border-radius: 8px;
  color: #fff;
  background: var(--q-primary);
  font-size: 28px;
}

.detail-page-shell__title {
  color: var(--app-text-strong);
  font-size: 24px;
  font-weight: 700;
  line-height: 1.2;
}

.detail-page-shell__subtitle {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  margin-top: 6px;
  color: var(--app-text-muted);
}

.detail-page-shell__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.detail-page-shell__error {
  color: var(--q-negative);
  background: rgba(193, 0, 21, 0.08);
}

:deep(.detail-page-section) {
  padding: 20px 22px;
}

:deep(.detail-page-section h3) {
  margin: 0;
  color: var(--app-text-strong);
  font-size: 18px;
  font-weight: 700;
}

:deep(.detail-page-section p) {
  margin: 6px 0 0;
  color: var(--app-text-muted);
}

:deep(.detail-page-section__head) {
  display: flex;
  align-items: center;
  margin-bottom: 16px;
}

@media (max-width: 720px) {
  .detail-page-shell__header {
    flex-direction: column;
    align-items: stretch;
  }

  .detail-page-shell__actions {
    justify-content: flex-end;
  }
}
</style>
