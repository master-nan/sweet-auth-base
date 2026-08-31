<template>
  <div
    class="assignment-scope-switch"
    role="tablist"
    :aria-label="t('ui.assignmentDateRange')"
    :style="indicatorStyle"
  >
    <span class="assignment-scope-switch__indicator" aria-hidden="true" />
    <button
      v-for="option in options"
      :key="option.value"
      class="assignment-scope-switch__option"
      :class="{ 'is-active': option.value === modelValue }"
      type="button"
      role="tab"
      :aria-selected="option.value === modelValue"
      :disabled="loading"
      @click="selectScope(option.value)"
    >
      {{ option.label }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { computed } from 'vue'
import type { AssignmentTimeScope } from 'src/api/services/org'

const { t } = useI18n({ useScope: 'global' })

const options: Array<{ label: string; value: AssignmentTimeScope }> = [
  {
    get label() {
      return t('ui.current')
    },
    value: 'current',
  },
  {
    get label() {
      return t('ui.history')
    },
    value: 'history',
  },
  {
    get label() {
      return t('ui.theFuture')
    },
    value: 'future',
  },
  {
    get label() {
      return t('ui.axisOfTime')
    },
    value: 'timeline',
  },
]

const props = withDefaults(
  defineProps<{
    modelValue: AssignmentTimeScope
    loading?: boolean
  }>(),
  {
    loading: false,
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: AssignmentTimeScope]
}>()

const activeIndex = computed(() =>
  Math.max(
    options.findIndex((option) => option.value === props.modelValue),
    0,
  ),
)

const indicatorStyle = computed(() => ({
  '--assignment-scope-index': activeIndex.value,
}))

const selectScope = (scope: AssignmentTimeScope) => {
  if (scope !== props.modelValue && !props.loading) emit('update:modelValue', scope)
}
</script>

<style scoped>
.assignment-scope-switch {
  --assignment-scope-index: 0;

  position: relative;
  isolation: isolate;
  display: grid;
  grid-template-columns: repeat(4, minmax(62px, 1fr));
  min-width: 272px;
  padding: 4px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-surface-muted);
}

.assignment-scope-switch__indicator {
  position: absolute;
  z-index: 0;
  top: 4px;
  left: 4px;
  width: calc((100% - 8px) / 4);
  height: calc(100% - 8px);
  border: 1px solid var(--app-primary-border);
  border-radius: 6px;
  background: var(--app-surface);
  box-shadow: 0 4px 10px var(--app-primary-shadow);
  transform: translateX(calc(var(--assignment-scope-index) * 100%));
  transition:
    transform 220ms cubic-bezier(0.2, 0.8, 0.2, 1),
    box-shadow 180ms ease;
}

.assignment-scope-switch__option {
  position: relative;
  z-index: 1;
  min-width: 0;
  height: 34px;
  padding: 0 10px;
  border: 0;
  border-radius: 6px;
  color: var(--app-text-strong);
  background: transparent;
  cursor: pointer;
  font-weight: 500;
  transition: color 180ms ease;
}

.assignment-scope-switch__option:hover:not(:disabled),
.assignment-scope-switch__option.is-active {
  color: var(--q-primary);
}

.assignment-scope-switch__option:disabled {
  cursor: wait;
  opacity: 0.6;
}

@media (max-width: 720px) {
  .assignment-scope-switch {
    width: 100%;
    min-width: 0;
  }
}
</style>
