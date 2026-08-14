<template>
  <q-chip dense square :outline="outline" :color="color" :icon="icon || undefined">
    {{ label }}
  </q-chip>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    label: string
    color: string
    icon?: string
    outline?: boolean
  }>(),
  {
    icon: '',
    outline: true,
  },
)
</script>
