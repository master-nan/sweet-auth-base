<template>
  <q-dialog v-model="visible" :maximized="$q.screen.lt.sm">
    <q-card style="width: 520px; max-width: 100%">
      <q-card-section class="row items-center">
        <div class="text-h6">保存查询方案</div>
        <q-space />
        <q-btn v-close-popup flat round dense icon="close" aria-label="关闭保存方案窗口">
          <q-tooltip>关闭</q-tooltip>
        </q-btn>
      </q-card-section>
      <q-separator />
      <q-card-section class="q-gutter-md">
        <q-input v-model="name" outlined dense label="方案名称" maxlength="64" counter autofocus />
        <q-checkbox v-model="isDefault" label="设为我的默认方案" />
        <div v-if="canUpdate" class="text-caption text-grey-7">
          可保存对“{{ source?.name }}”的修改，也可另存为新的个人方案。
        </div>
      </q-card-section>
      <q-card-actions align="right" class="q-pa-md">
        <q-btn v-close-popup flat label="取消" />
        <q-btn
          v-if="canUpdate"
          outline
          color="primary"
          label="保存修改"
          :loading="loading"
          :disable="!valid"
          @click="submit(false)"
        />
        <q-btn color="primary" :label="saveAsLabel" :loading="loading" :disable="!valid" @click="submit(true)" />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useQuasar } from 'quasar'
import { QuerySchemeType, type QuerySchemeSource } from 'src/modules/query-scheme/types'

const props = withDefaults(defineProps<{
  modelValue: boolean
  source?: QuerySchemeSource | null
  loading?: boolean
}>(), { source: null, loading: false })
const $q = useQuasar()
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  save: [value: { name: string; isDefault: boolean; saveAs: boolean }]
}>()
const name = ref('')
const isDefault = ref(false)
const visible = computed({ get: () => props.modelValue, set: (value) => emit('update:modelValue', value) })
const canUpdate = computed(() => props.source?.type === QuerySchemeType.PERSONAL)
const saveAsLabel = computed(() => {
  if (canUpdate.value) return '另存为'
  return props.source ? '另存为我的方案' : '保存'
})
const valid = computed(() => name.value.trim().length > 0 && name.value.trim().length <= 64)

watch(() => props.modelValue, (open) => {
  if (!open) return
  name.value = props.source?.name || ''
  isDefault.value = props.source?.is_default || false
}, { immediate: true })

const submit = (saveAs: boolean) => {
  if (!valid.value) return
  emit('save', { name: name.value.trim(), isDefault: isDefault.value, saveAs })
}
</script>
