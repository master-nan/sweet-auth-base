<template>
  <div class="organization-readonly-detail">
    <q-inner-loading :showing="loading">
      <q-spinner color="primary" size="38px" />
    </q-inner-loading>

    <q-banner v-if="error" class="organization-detail-error">
      <template #avatar>
        <q-icon name="error_outline" color="negative" />
      </template>
      {{ error }}
    </q-banner>

    <div v-else-if="fields.length" class="organization-detail-grid">
      <article
        v-for="field in fields"
        :key="field.key"
        class="organization-detail-field"
        :class="{ 'organization-detail-field--wide': field.wide }"
      >
        <div class="organization-detail-label">{{ field.label }}</div>
        <div class="organization-detail-value">
          <q-chip
            v-if="field.kind === 'status'"
            dense
            square
            outline
            :color="field.color || 'grey-6'"
          >
            {{ field.value }}
          </q-chip>
          <code v-else-if="field.kind === 'code'">{{ field.value }}</code>
          <span v-else>{{ field.value }}</span>
        </div>
      </article>
    </div>

    <div v-else-if="!loading" class="organization-detail-empty">
      <q-icon name="description" size="42px" />
      <div>{{ emptyText }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { OrganizationDetailGroup } from './organization-read-only-detail'

defineOptions({ name: 'OrganizationReadOnlyDetail' })

const props = withDefaults(
  defineProps<{
    groups: OrganizationDetailGroup[]
    loading?: boolean
    error?: string
    emptyText?: string
  }>(),
  {
    loading: false,
    error: '',
    emptyText: '请选择一条记录',
  },
)

const fields = computed(() => props.groups.flatMap((group) => group.fields))
</script>

<style scoped lang="scss">
.organization-readonly-detail {
  position: relative;
  min-height: 240px;
}

.organization-detail-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  column-gap: 36px;
  row-gap: 0;
}

.organization-detail-field {
  min-width: 0;
  min-height: 78px;
  padding: 14px 0 12px;
  border-bottom: 1px solid #edf0f5;
}

.organization-detail-field--wide {
  grid-column: 1 / -1;
}

.organization-detail-label {
  margin-bottom: 5px;
  color: #77839a;
  font-size: 12px;
  line-height: 18px;
}

.organization-detail-value {
  overflow-wrap: anywhere;
  color: #1b2940;
  font-size: 14px;
  font-weight: 500;
  line-height: 1.55;
}

.organization-detail-value code {
  color: #3b4b63;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 13px;
  font-weight: 500;
}

.organization-detail-error {
  margin: 16px;
  border: 1px solid #ffcdd2;
  background: #fff5f5;
  color: #b71c1c;
}

.organization-detail-empty {
  height: 100%;
  min-height: 240px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: #8792a6;
}

:global(.body--dark .organization-detail-field) {
  border-bottom-color: var(--app-dark-border);
}

:global(.body--dark .organization-detail-label),
:global(.body--dark .organization-detail-empty) {
  color: var(--app-dark-muted);
}

:global(.body--dark .organization-detail-value),
:global(.body--dark .organization-detail-value code) {
  color: var(--app-dark-text);
}

:global(.body--dark .organization-detail-error) {
  border-color: rgba(239, 83, 80, 0.45);
  background: rgba(239, 83, 80, 0.12);
  color: #ffb4b4;
}

@media (max-width: 1200px) {
  .organization-detail-grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .organization-detail-field--wide {
    grid-column: auto;
  }
}
</style>
