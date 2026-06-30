<template>
  <q-select
    ref="selectRef"
    class="sweet-select"
    :model-value="modelValue"
    outlined
    dense
    options-dense
    clear-icon="close"
    popup-content-class="sweet-select-menu"
    v-bind="$attrs"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <template v-for="(_, slotName) in $slots" #[slotName]="slotProps">
      <slot :name="slotName" v-bind="slotProps || {}" />
    </template>
  </q-select>
</template>

<script setup lang="ts">
import { ref } from 'vue'

defineOptions({
  inheritAttrs: false,
})

defineProps<{
  modelValue?: any
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: any): void
}>()

const selectRef = ref()

defineExpose({
  focus: () => selectRef.value?.focus?.(),
  blur: () => selectRef.value?.blur?.(),
  validate: () => selectRef.value?.validate?.(),
  resetValidation: () => selectRef.value?.resetValidation?.(),
})
</script>

<style scoped lang="scss">
.sweet-select :deep(.q-field__control) {
  border-radius: 6px;
}

.sweet-select :deep(.q-field__native) {
  min-width: 0;
}

.sweet-select :deep(.q-chip) {
  border-radius: 6px;
}
</style>

<style lang="scss">
.sweet-select-menu {
  border-radius: 6px;
  box-shadow: 0 10px 28px rgba(34, 43, 69, 0.16);
}
</style>
