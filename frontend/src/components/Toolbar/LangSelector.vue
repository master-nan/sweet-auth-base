<template>
  <q-select v-model="locale" hide-dropdown-icon :options="localeOptions" dense borderless emit-value map-options options-dense @update:model-value="handleSelectLanguage">
    <template v-slot:prepend>
      <q-icon size="sm" color="white" name="language"/>
    </template>
    <template v-slot:selected-item="scope">
      <div style="color: white"> {{ scope.opt.label }}</div>
    </template>
  </q-select>
</template>

<script lang="ts" setup>
import { LocalStorage } from 'quasar'
import { onMounted } from 'vue'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'LangSelector' })

const { locale } = useI18n({ useScope: 'global' })
const localeOptions = [
  { value: 'en-US', label: 'English' },
  { value: 'zh-CN', label: '简体中文' }
]

onMounted(() => {
  const language: string | null = LocalStorage.getItem('lang')
  if (language !== null) {
    locale.value = language
  }
})

const handleSelectLanguage = () => {
  LocalStorage.set('lang', locale.value)
}
</script>
