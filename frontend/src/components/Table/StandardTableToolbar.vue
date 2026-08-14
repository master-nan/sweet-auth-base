<template>
  <div class="row items-center q-gutter-xs full-width">
    <div class="col-grow row items-center q-gutter-xs">
      <slot name="quick-search" />
      <slot name="advanced-trigger" />
      <slot name="column-selector" />
      <slot name="left-actions" />
    </div>
    <q-space />
    <div class="row items-center q-gutter-xs">
      <slot name="right-actions" />
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
