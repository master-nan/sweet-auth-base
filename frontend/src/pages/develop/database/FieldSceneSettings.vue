<template>
  <fieldset class="field-scenes">
    <legend>{{ t('document.fieldScenes') }}</legend>
    <div class="row q-col-gutter-sm">
      <q-select
        v-for="scene in scenes" :key="scene" class="col-12 col-sm-6" dense outlined
        emit-value map-options :options="options"
        :label="t(scene === 'insert' ? 'document.onInsert' : 'document.onUpdate')"
        :model-value="fieldAccess(settings, scene)" @update:model-value="setMode(scene, $event)"
      />
    </div>
    <div class="row q-gutter-x-md q-mt-sm">
      <q-checkbox dense :model-value="settings.is_list_show" :label="t('document.listVisible')"
        @update:model-value="emit('patch', { is_list_show: $event })" />
      <q-checkbox dense :model-value="isDetailFieldVisible(settings)" :label="t('document.detailVisible')"
        @update:model-value="emit('patch', { detail_visible: $event })" />
    </div>
  </fieldset>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { TableField } from '@/api/services/sys-table'
import { fieldAccess, isDetailFieldVisible } from '@/utils/field-layout'

const props = defineProps<{ draft: Partial<TableField> }>()
const emit = defineEmits<{ patch: [value: Partial<TableField>] }>()
const { t } = useI18n()
const scenes = ['insert', 'update'] as const
type Mode = ReturnType<typeof fieldAccess>
const modes: Mode[] = ['hidden', 'readonly', 'editable']
const options = computed(() => modes.map(value => ({ value, label: t('document.' + value) })))
const settings = computed(() => ({
  ...props.draft,
  is_list_show: props.draft.is_list_show ?? true,
  is_insert_show: props.draft.is_insert_show ?? true,
  is_update_show: props.draft.is_update_show ?? true,
}))
function setMode(scene: typeof scenes[number], mode: Mode) {
  const visible = mode !== 'hidden', readonly = mode === 'readonly'
  emit('patch', scene === 'insert'
    ? { is_insert_show: visible, is_insert_readonly: readonly }
    : { is_update_show: visible, is_update_readonly: readonly })
}
</script>

<style scoped>
.field-scenes {
  min-width: 0;
  margin-inline: 0;
  padding: 12px 16px 16px;
  border: 1px solid var(--app-border);
  border-radius: 6px;
}
.field-scenes legend { padding-inline: 5px; font-weight: 600; }
</style>
