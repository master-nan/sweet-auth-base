<template>
  <div class="row items-center q-gutter-sm full-width">
    <div v-if="$slots['scheme-selector']" class="row items-center q-gutter-xs">
      <slot name="scheme-selector" />
    </div>
    <q-separator
      v-if="$slots['scheme-selector'] && ($slots['quick-presets'] || $slots['quick-search'] || $slots['advanced-trigger'])"
      vertical
      inset
    />
    <div class="col-grow row items-center q-gutter-xs">
      <slot name="quick-presets" />
      <slot name="quick-search" />
      <slot name="advanced-trigger" />
      <slot name="save-scheme" />
      <slot name="left-actions" />
    </div>
    <q-space />
    <div class="row items-center q-gutter-xs">
      <slot name="right-actions" />
      <q-separator
        v-if="$slots['right-actions']"
        vertical
        inset
      />
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
