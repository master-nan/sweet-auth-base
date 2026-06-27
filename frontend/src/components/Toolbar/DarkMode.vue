<template>
  <q-btn round dense flat :icon="$q.dark.isActive ? 'light_mode' : 'dark_mode'" @click="toggle">
    <q-tooltip>{{ $q.dark.isActive ? t('layout.lightMode') : t('layout.darkMode') }}</q-tooltip>
  </q-btn>
</template>

<script lang="ts" setup>
defineOptions({ name: 'DarkMode' })

import { LocalStorage, Dark } from 'quasar'
import { onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useQuasar } from 'quasar'

const $q = useQuasar()

onMounted(() => {
  const dark: boolean | null = LocalStorage.getItem('dark')
  if (dark !== null) {
    Dark.set(dark)
  }
})
const { t } = useI18n()

const toggle = () => {
  Dark.toggle()
  LocalStorage.set('dark', Dark.isActive)
}
</script>
