<template>
  <q-select
    v-model="locale"
    class="language-selector"
    hide-dropdown-icon
    :options="localeOptions"
    dense
    borderless
    emit-value
    map-options
    options-dense
    :aria-label="t('layout.language')"
    @update:model-value="handleSelectLanguage"
  >
    <template v-slot:prepend>
      <q-icon size="sm" name="language" />
    </template>
    <template v-slot:selected-item="scope">
      <div>{{ scope.opt.label }}</div>
    </template>
  </q-select>
</template>

<script lang="ts" setup>
import { useI18n } from 'vue-i18n'
import { writeUIPreferences, type SupportedLocale } from 'src/utils/ui-preferences'
import { applyQuasarLanguage } from 'src/i18n/runtime/quasar'

defineOptions({ name: 'LangSelector' })

const { locale, t } = useI18n({ useScope: 'global' })
const localeOptions = [
  { value: 'en-US', label: 'English' },
  {
    value: 'zh-CN',
    get label() {
      return t('ui.simplifiedChinese')
    },
  },
]

const handleSelectLanguage = () => {
  const value = locale.value as SupportedLocale
  applyQuasarLanguage(value)
  writeUIPreferences({ locale: value })
}
</script>

<style scoped lang="scss">
.language-selector {
  min-width: 112px;
  color: inherit;

  :deep(.q-field__native),
  :deep(.q-field__prepend) {
    color: inherit;
  }
}
</style>
