<template>
  <q-dialog v-model="openSettingPanel" full-height position="right">
    <q-card class="theme-setting-card">
      <q-card-section class="row">
        <div class="text-weight-bold text-h6">{{ t('themeSetting.title') }}</div>
        <q-space />
        <q-btn icon="close" flat round dense v-close-popup />
      </q-card-section>
      <q-separator />
      <q-card-section class="q-gutter-sm">
        <div class="row items-center justify-between no-wrap q-gutter-sm">
          <div class="text-weight-bold text-subtitle2">{{ t('themeSetting.themeColor') }}</div>
          <q-btn
            outline
            dense
            no-caps
            color="primary"
            icon="restart_alt"
            :label="t('themeSetting.resetColor')"
            :disable="isDefaultThemeColor"
            @click="resetThemeColor"
          />
        </div>
        <q-color v-model="hex" flat bordered no-header-tabs @change="handleColorChange" />
      </q-card-section>
      <q-separator />
      <q-card-section class="q-gutter-sm">
        <div class="text-weight-bold text-subtitle2">{{ t('themeSetting.layoutMode') }}</div>
        <q-btn-toggle
          v-model="layoutMode"
          class="theme-setting-layout-toggle"
          spread
          no-caps
          unelevated
          toggle-color="primary"
          :options="layoutOptions"
        />
      </q-card-section>
      <q-separator />
      <q-card-section>
        <div class="text-weight-bold text-subtitle2 q-mb-sm">
          {{ t('themeSetting.setting') }}
        </div>
        <q-list bordered separator class="rounded-borders">
          <q-item>
            <q-item-section avatar>
              <q-icon name="dark_mode" color="primary" />
            </q-item-section>
            <q-item-section>
              <q-item-label>{{ t('themeSetting.darkMode') }}</q-item-label>
              <q-item-label caption>{{ t('themeSetting.darkModeHint') }}</q-item-label>
            </q-item-section>
            <q-item-section side>
              <DarkMode />
            </q-item-section>
          </q-item>
          <q-item>
            <q-item-section avatar>
              <q-icon name="view_sidebar" color="primary" />
            </q-item-section>
            <q-item-section>
              <q-item-label>{{ t('themeSetting.drawerMini') }}</q-item-label>
              <q-item-label caption>{{ t('themeSetting.drawerMiniHint') }}</q-item-label>
            </q-item-section>
            <q-item-section side>
              <q-toggle v-model="drawerMini" color="primary" />
            </q-item-section>
          </q-item>
        </q-list>
      </q-card-section>
    </q-card>
  </q-dialog>

</template>

<script setup lang="ts">
defineOptions({ name: 'ThemeSetting' })
import { computed, ref } from 'vue'
import { useToggle } from '@vueuse/shared'
import { useThemeStore } from 'src/stores/theme'
import { useAppStore } from 'src/stores/app'
import DarkMode from '../Toolbar/DarkMode.vue'
import { useI18n } from 'vue-i18n'
import { DEFAULT_PRIMARY_COLOR, type LayoutMode } from 'src/utils/ui-preferences'

const { t } = useI18n()
const openSettingPanel = ref<boolean>(false)
const toggleSettingPanel = useToggle(openSettingPanel)

const themeStore = useThemeStore()
const appStore = useAppStore()

const hex = ref<string>(themeStore.primaryColor)
const isDefaultThemeColor = computed(
  () => themeStore.primaryColor.toLowerCase() === DEFAULT_PRIMARY_COLOR.toLowerCase(),
)
const layoutMode = computed<LayoutMode>({
  get: () => themeStore.layoutMode,
  set: (value) => themeStore.setLayoutMode(value),
})
const drawerMini = computed({
  get: () => appStore.is_drawer_mini,
  set: (value: boolean) => appStore.setDrawerMini(value),
})
const layoutOptions = computed(() => [
  {
    label: t('themeSetting.layoutSplit'),
    value: 'split',
    icon: 'view_sidebar',
  },
  {
    label: t('themeSetting.layoutFull'),
    value: 'full',
    icon: 'web_asset',
  },
])

defineExpose({ toggleSettingPanel })

const handleColorChange = (colorHex: string) => {
  themeStore.setThemeColor(colorHex)
}

const resetThemeColor = () => {
  themeStore.resetThemeColor()
  hex.value = DEFAULT_PRIMARY_COLOR
}
</script>

<style scoped lang="scss">
.theme-setting-card {
  width: 380px;
  max-width: 100vw;
}

.theme-setting-layout-toggle {
  width: 100%;
  border: 1px solid var(--app-primary-border);
  border-radius: 8px;
  overflow: hidden;
  background: var(--app-primary-soft);
}
</style>
