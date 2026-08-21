<template>
  <div class="standard-table-toolbar row items-center no-wrap q-gutter-sm full-width">
    <div
      v-if="$slots['query-controls']"
      class="standard-table-toolbar__query col-grow row items-center no-wrap q-gutter-xs"
    >
      <slot name="query-controls" />
    </div>
    <div v-else class="standard-table-toolbar__query col-grow row items-center no-wrap q-gutter-xs">
      <slot name="quick-search" />
      <slot name="left-actions" />
    </div>
    <q-space />
    <div class="standard-table-toolbar__actions row items-center no-wrap q-gutter-xs">
      <slot name="right-actions" />
      <q-separator v-if="$slots['right-actions']" vertical inset />
      <div v-if="$slots['column-selector']" class="standard-table-toolbar__column-selector">
        <slot name="column-selector" />
      </div>
      <slot name="extra" />
      <q-btn
        flat
        dense
        round
        icon="refresh"
        color="primary"
        aria-label="刷新当前视图"
        :loading="refreshing"
        :disable="disabled"
        @click="$emit('refresh')"
      >
        <q-tooltip>刷新当前视图</q-tooltip>
      </q-btn>
    </div>
  </div>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    refreshing?: boolean
    disabled?: boolean
  }>(),
  {
    refreshing: false,
    disabled: false,
  },
)

defineEmits<{
  refresh: []
}>()
</script>

<style scoped>
.standard-table-toolbar__query {
  min-width: 0;
}

.standard-table-toolbar__actions {
  flex: none;
}

.standard-table-toolbar__column-selector {
  width: 150px;
}

.standard-table-toolbar__column-selector :deep(.q-field) {
  width: 100%;
}
</style>
