<template>
  <router-view />
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useLegacyUiLocalizer } from 'src/i18n/legacy-localizer'

defineOptions({ name: 'App' })

const { locale } = useI18n({ useScope: 'global' })
useLegacyUiLocalizer(locale)
</script>
