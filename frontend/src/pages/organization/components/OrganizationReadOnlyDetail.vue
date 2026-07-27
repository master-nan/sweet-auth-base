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

    <div v-else-if="groups.length" class="organization-detail-groups">
      <q-card
        v-for="group in groups"
        :key="group.key"
        flat
        bordered
        class="organization-detail-group"
      >
        <q-card-section class="organization-detail-group__header">
          <q-icon :name="group.icon || 'subject'" size="18px" />
          <h3>{{ group.title }}</h3>
        </q-card-section>

        <q-separator />

        <q-card-section>
          <div class="organization-detail-grid">
            <article
              v-for="field in group.fields"
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
        </q-card-section>
      </q-card>
    </div>

    <div v-else-if="!loading" class="organization-detail-empty">
      <q-icon name="description" size="42px" />
      <div>{{ emptyText }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { OrganizationDetailGroup } from './organization-read-only-detail'

defineOptions({ name: 'OrganizationReadOnlyDetail' })

withDefaults(
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
</script>

<style scoped lang="scss">
.organization-readonly-detail {
  position: relative;
  min-height: 240px;
}

.organization-detail-groups {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  align-items: start;
  gap: 12px;
  padding-bottom: 8px;
}

.organization-detail-group {
  min-width: 0;
  border-color: #dfe5ee;
  border-radius: 8px;
  background: #fff;
}

.organization-detail-group__header {
  min-height: 50px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 14px;
  color: #40516c;
}

.organization-detail-group__header h3 {
  margin: 0;
  color: #26354d;
  font-size: 14px;
  font-weight: 700;
  line-height: 20px;
}

.organization-detail-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 14px;
}

.organization-detail-field {
  min-width: 0;
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

@media (max-width: 1200px) {
  .organization-detail-groups {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
