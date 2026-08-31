<template>
  <q-btn
    class="dark-mode-btn"
    dense
    flat
    :icon="$q.dark.isActive ? 'light_mode' : 'dark_mode'"
    :aria-label="$q.dark.isActive ? t('layout.lightMode') : t('layout.darkMode')"
    @click="toggle"
  >
    <q-tooltip>{{ $q.dark.isActive ? t('layout.lightMode') : t('layout.darkMode') }}</q-tooltip>
  </q-btn>
</template>

<script lang="ts" setup>
defineOptions({ name: 'DarkMode' })

import { Dark } from 'quasar'
import { useI18n } from 'vue-i18n'
import { useQuasar } from 'quasar'
import { useThemeStore } from 'src/stores/theme'

const $q = useQuasar()
const themeStore = useThemeStore()
themeStore.applyPreferences()
const { t } = useI18n()

const toggle = () => {
  themeStore.setDarkMode(!Dark.isActive)
}
</script>
