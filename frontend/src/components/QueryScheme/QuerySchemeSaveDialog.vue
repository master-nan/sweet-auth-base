<template>
  <q-dialog v-model="visible" :maximized="$q.screen.lt.sm">
    <q-card style="width: 520px; max-width: 100%">
      <q-card-section class="row items-center">
        <div class="text-h6">{{ t('ui.saveQueryScheme') }}</div>
        <q-space />
        <q-btn
          v-close-popup
          flat
          round
          dense
          icon="close"
          :aria-label="t('ui.closeSaveProgramWindow')"
        >
          <q-tooltip>{{ t('ui.close') }}</q-tooltip>
        </q-btn>
      </q-card-section>
      <q-separator />
      <q-card-section class="q-gutter-md">
        <q-input
          v-model="name"
          outlined
          dense
          :label="t('ui.schemeName')"
          maxlength="64"
          counter
          autofocus
        />
        <q-checkbox v-model="isDefault" :label="t('ui.setAsMyDefaultScheme')" />
        <div v-if="canUpdate" class="text-caption text-grey-7">
          {{ t('ui.savesPairs') }}{{ source?.name
          }}{{ t('ui.changesMayAlsoBeSavedAsNewIndividualProgrammes') }}
        </div>
      </q-card-section>
      <q-card-actions align="right" class="q-pa-md">
        <q-btn v-close-popup flat :label="t('ui.cancel')" />
        <q-btn
          v-if="canUpdate"
          outline
          color="primary"
          :label="t('ui.saveChanges')"
          :loading="loading"
          :disable="!valid"
          @click="submit(false)"
        />
        <q-btn
          color="primary"
          :label="saveAsLabel"
          :loading="loading"
          :disable="!valid"
          @click="submit(true)"
        />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { computed, ref, watch } from 'vue'
import { useQuasar } from 'quasar'
import { QuerySchemeType, type QuerySchemeSource } from '@/modules/query-scheme/types'

const { t } = useI18n({ useScope: 'global' })

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    source?: QuerySchemeSource | null
    loading?: boolean
  }>(),
  { source: null, loading: false },
)
const $q = useQuasar()
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  save: [value: { name: string; isDefault: boolean; saveAs: boolean }]
}>()
const name = ref('')
const isDefault = ref(false)
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})
const canUpdate = computed(() => props.source?.type === QuerySchemeType.PERSONAL)
const saveAsLabel = computed(() => {
  if (canUpdate.value) return t('ui.saveAs')
  return props.source ? t('ui.saveAsMyScheme') : t('ui.save')
})
const valid = computed(() => name.value.trim().length > 0 && name.value.trim().length <= 64)

watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    name.value = props.source?.name || ''
    isDefault.value = props.source?.is_default || false
  },
  { immediate: true },
)

const submit = (saveAs: boolean) => {
  if (!valid.value) return
  emit('save', { name: name.value.trim(), isDefault: isDefault.value, saveAs })
}
</script>
