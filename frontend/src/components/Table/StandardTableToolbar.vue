<template>
  <div class="row items-center q-gutter-sm full-width">
    <div v-if="$slots['query-controls']" class="col-grow row items-center q-gutter-xs">
      <slot name="query-controls" />
    </div>
    <div v-else class="col-grow row items-center q-gutter-xs">
      <slot name="quick-search" />
      <slot name="left-actions" />
    </div>
    <q-space />
    <div class="row items-center q-gutter-xs">
      <slot name="right-actions" />
      <q-separator v-if="$slots['right-actions']" vertical inset />
      <slot name="column-selector" />
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
