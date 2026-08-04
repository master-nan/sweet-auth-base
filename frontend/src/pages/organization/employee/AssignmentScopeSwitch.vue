<template>
  <div
    class="assignment-scope-switch"
    role="tablist"
    aria-label="任职记录时间范围"
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
import { computed } from 'vue'
import type { AssignmentTimeScope } from 'src/api/services/org'

const options: Array<{ label: string; value: AssignmentTimeScope }> = [
  { label: '当前', value: 'current' },
  { label: '历史', value: 'history' },
  { label: '未来', value: 'future' },
  { label: '时间轴', value: 'timeline' },
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
  border: 1px solid rgb(112 122 142 / 22%);
  border-radius: 8px;
  background: rgb(112 122 142 / 8%);
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
  background: #ffffff;
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
  color: #424b5d;
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

:global(body.body--dark) .assignment-scope-switch {
  border-color: rgb(255 255 255 / 12%);
  background: rgb(255 255 255 / 5%);
}

:global(body.body--dark) .assignment-scope-switch__indicator {
  background: #30384b;
}

:global(body.body--dark) .assignment-scope-switch__option {
  color: #c4ccdb;
}

:global(body.body--dark) .assignment-scope-switch__option:hover:not(:disabled),
:global(body.body--dark) .assignment-scope-switch__option.is-active {
  color: var(--q-primary);
}

@media (max-width: 720px) {
  .assignment-scope-switch {
    width: 100%;
    min-width: 0;
  }
}
</style>
