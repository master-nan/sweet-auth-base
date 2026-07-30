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
import {
  writeUIPreferences,
  type SupportedLocale,
} from 'src/utils/ui-preferences'

defineOptions({ name: 'LangSelector' })

const { locale } = useI18n({ useScope: 'global' })
const localeOptions = [
  { value: 'en-US', label: 'English' },
  { value: 'zh-CN', label: '简体中文' },
]

const handleSelectLanguage = () => {
  writeUIPreferences({ locale: locale.value as SupportedLocale })
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
