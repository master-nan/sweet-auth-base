<template>
  <div
    class="app-scrollable-table"
    :class="wrapperClass"
    :style="wrapperStyle"
  >
    <div v-if="$slots.top" class="q-table__top app-scrollable-table__top">
      <slot name="top" />
    </div>

    <q-scroll-area
      class="app-scrollable-table__area"
      :delay="0"
      :vertical-offset="[4, 4]"
      :horizontal-offset="[4, 4]"
    >
      <q-table
        v-bind="tableAttrs"
        :rows="props.rows"
        class="app-scrollable-table__table"
        flat
        hide-bottom
      >
        <template
          v-for="slotName in tableSlotNames"
          :key="slotName"
          #[slotName]="slotProps"
        >
          <slot :name="slotName" v-bind="slotProps || {}" />
        </template>
      </q-table>
    </q-scroll-area>

    <div v-if="$slots.bottom" class="q-table__bottom app-scrollable-table__bottom">
      <slot name="bottom" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, useAttrs, useSlots } from 'vue'
import type { HTMLAttributes, StyleValue } from 'vue'
import type { QTableProps } from 'quasar'

defineOptions({
  name: 'ScrollableTable',
  inheritAttrs: false,
})

const attrs = useAttrs()
const slots = useSlots()
const props = defineProps<{
  rows: QTableProps['rows']
}>()

const wrapperClass = computed(() => attrs.class as HTMLAttributes['class'])
const wrapperStyle = computed(() => attrs.style as StyleValue)
const tableAttrs = computed(() =>
  Object.fromEntries(
    Object.entries(attrs).filter(([name]) => !['class', 'style', 'bordered'].includes(name)),
  ),
)
const tableSlotNames = computed(() =>
  Object.keys(slots).filter((slotName) => slotName !== 'top' && slotName !== 'bottom'),
)
</script>

<style scoped lang="scss">
.app-scrollable-table {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr) auto;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  border: 1px solid rgba(0, 0, 0, 0.12);
  border-radius: 4px;
  background: var(--q-card-background, #fff);
}

.app-scrollable-table__top,
.app-scrollable-table__bottom {
  position: relative;
  z-index: 3;
  flex: none;
}

.app-scrollable-table__area {
  min-width: 0;
  min-height: 0;
  height: 100%;
  overflow: hidden;
}

.app-scrollable-table__area :deep(.q-scrollarea__content) {
  min-width: 100%;
  max-width: none;
}

.app-scrollable-table__table {
  width: max-content;
  min-width: 100%;
  min-height: 100%;
  height: auto;
  border: 0;
  border-radius: 0;
}

.app-scrollable-table__table :deep(.q-table__middle) {
  max-height: none;
  overflow: visible;
}

.app-scrollable-table__table :deep(table) {
  width: 100%;
}

.app-scrollable-table__table :deep(thead tr th) {
  position: sticky;
  top: 0;
  z-index: 2;
}

.body--dark .app-scrollable-table {
  border-color: var(--app-dark-border);
  background: var(--app-dark-surface);
}
</style>
