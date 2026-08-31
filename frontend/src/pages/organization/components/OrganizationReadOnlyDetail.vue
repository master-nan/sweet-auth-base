<template>
  <div class="organization-readonly-detail">
    <q-inner-loading :showing="loading" :dark="Dark.isActive">
      <q-spinner color="primary" size="38px" />
    </q-inner-loading>

    <q-banner v-if="error" :dark="Dark.isActive" rounded class="q-ma-md">
      <template #avatar>
        <q-icon name="error_outline" color="negative" />
      </template>
      {{ error }}
    </q-banner>

    <div v-else-if="fields.length" class="organization-detail-grid">
      <article
        v-for="field in fields"
        :key="field.key"
        class="organization-detail-field q-py-sm"
        :class="{ 'organization-detail-field--wide': field.wide }"
      >
        <div class="text-caption text-grey-7 q-mb-xs">{{ field.label }}</div>
        <div class="text-body2 text-weight-medium">
          <q-chip
            v-if="field.kind === 'status'"
            dense
            square
            outline
            :color="field.color || 'grey-6'"
          >
            {{ field.value }}
          </q-chip>
          <code v-else-if="field.kind === 'code'" class="text-body2 text-weight-medium">
            {{ field.value }}
          </code>
          <span v-else>{{ field.value }}</span>
        </div>
        <q-separator class="q-mt-sm" />
      </article>
    </div>

    <div v-else-if="!loading" class="absolute-full column flex-center q-gutter-sm text-grey-6">
      <q-icon name="description" size="42px" />
      <div>{{ displayEmptyText }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { computed } from 'vue'
import { Dark } from 'quasar'
import type { OrganizationDetailGroup } from './organization-read-only-detail'

const { t } = useI18n({ useScope: 'global' })

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
    emptyText: '',
  },
)

const displayEmptyText = computed(() => props.emptyText || t('ui.pleaseSelectARecord'))

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
  overflow-wrap: anywhere;
}

.organization-detail-field--wide {
  grid-column: 1 / -1;
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
