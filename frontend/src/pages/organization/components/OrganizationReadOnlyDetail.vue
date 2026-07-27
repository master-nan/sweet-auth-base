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
            :color="field.color || 'grey-6'"
            text-color="white"
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
defineOptions({ name: 'OrganizationReadOnlyDetail' })

interface OrganizationDetailField {
  key: string
  label: string
  value: string
  kind?: 'text' | 'code' | 'status'
  color?: string
  wide?: boolean
}

withDefaults(
  defineProps<{
    fields: OrganizationDetailField[]
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
</script>

<style scoped lang="scss">
.organization-readonly-detail {
  position: relative;
  height: 100%;
  min-height: 240px;
  overflow: auto;
}

.organization-detail-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0;
}

.organization-detail-field {
  min-width: 0;
  padding: 16px;
  border-bottom: 1px solid #edf0f5;
}

.organization-detail-field:nth-child(odd) {
  border-right: 1px solid #edf0f5;
}

.organization-detail-field--wide {
  grid-column: 1 / -1;
  border-right: 0;
}

.organization-detail-label {
  margin-bottom: 7px;
  color: #657189;
  font-size: 12px;
}

.organization-detail-value {
  overflow-wrap: anywhere;
  color: #172033;
  font-size: 14px;
  line-height: 1.6;
}

.organization-detail-value code {
  color: #172033;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
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

@media (max-width: 1200px) {
  .organization-detail-grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .organization-detail-field,
  .organization-detail-field:nth-child(odd) {
    border-right: 0;
  }
}
</style>
