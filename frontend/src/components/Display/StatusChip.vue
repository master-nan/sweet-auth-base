<template>
  <q-chip
    dense
    square
    :outline="outline"
    :color="color"
    :text-color="textColor || (outline ? undefined : 'white')"
    :icon="icon || undefined"
  >
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
    textColor?: string
  }>(),
  {
    icon: '',
    outline: true,
    textColor: '',
  },
)
</script>
