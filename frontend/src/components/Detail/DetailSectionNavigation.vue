<template>
  <aside class="detail-section-navigation">
    <button
      v-for="item in items"
      :key="item.key"
      type="button"
      class="detail-section-navigation__item"
      :class="{ 'is-active': modelValue === item.key }"
      @click="emit('update:modelValue', item.key)"
    >
      <span class="detail-section-navigation__copy">
        <strong>{{ item.label }}</strong>
        <small v-if="item.caption">{{ item.caption }}</small>
      </span>
      <em v-if="item.count !== undefined">{{ item.count }}</em>
      <q-icon v-else name="chevron_right" size="18px" />
    </button>

    <div v-if="$slots.footer" class="detail-section-navigation__footer">
      <slot name="footer" />
    </div>
  </aside>
</template>

<script setup lang="ts">
import type { DetailSectionNavigationItem } from './types'

withDefaults(
  defineProps<{
    modelValue: string
    items?: DetailSectionNavigationItem[]
  }>(),
  {
    items: () => [],
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()
</script>

<style scoped>
.detail-section-navigation {
  min-width: 0;
  min-height: 0;
  height: 100%;
  padding: 18px 14px;
  border-right: 1px solid var(--app-border);
  display: flex;
  flex-direction: column;
  gap: 10px;
  background: var(--app-surface-muted);
}

.detail-section-navigation__item {
  width: 100%;
  min-height: 72px;
  padding: 12px 14px;
  border: 1px solid transparent;
  border-radius: 8px;
  display: flex;
  align-items: center;
  gap: 12px;
  background: transparent;
  color: var(--app-text-muted);
  text-align: left;
  cursor: pointer;
}

.detail-section-navigation__copy {
  min-width: 0;
  flex: 1;
}

.detail-section-navigation__item strong {
  display: block;
  color: var(--app-text-strong);
  font-size: 15px;
  line-height: 1.2;
  font-weight: 700;
}

.detail-section-navigation__item small {
  display: block;
  margin-top: 5px;
  color: var(--app-text-muted);
  font-size: 12px;
  line-height: 1.45;
}

.detail-section-navigation__item em {
  min-width: 30px;
  height: 28px;
  padding: 0 8px;
  border-radius: 15px;
  display: grid;
  place-items: center;
  background: var(--app-primary-soft);
  color: var(--q-primary);
  font-style: normal;
  font-weight: 700;
}

.detail-section-navigation__item.is-active {
  border-color: var(--app-primary-border);
  background: var(--app-surface);
  color: var(--q-primary);
  box-shadow: 0 8px 18px rgb(105 87 237 / 10%);
}

.detail-section-navigation__item.is-active strong {
  color: var(--q-primary);
}

.detail-section-navigation__footer {
  margin-top: auto;
}

</style>
