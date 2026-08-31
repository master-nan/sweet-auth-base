<template>
  <q-drawer
    v-model="_model_value"
    class="app-drawer"
    :class="{ 'app-drawer--mini': is_drawer_mini }"
    :mini="is_drawer_mini"
    :width="230"
    :mini-width="58"
    show-if-above
    bordered
    :mini-to-overlay="is_mini_overlay"
  >
    <div class="app-drawer__shell">
      <div class="app-drawer__brand">
        <toolbar-title
          :title="title"
          :subtitle="subtitle || t('layout.defaultDescription')"
          :logo="logo || ''"
          class="full-width"
          :mini="is_drawer_mini"
        />
      </div>
      <base-menu class="app-drawer__menu" />
      <div v-if="is_drawer_mini" class="app-drawer__footer">
        <q-btn
          class="app-drawer__close"
          flat
          round
          dense
          icon="keyboard_double_arrow_left"
          :aria-label="t('layout.hideSidebar')"
          @click="closeDrawer"
        >
          <q-tooltip anchor="center right" self="center left">
            {{ t('layout.hideSidebar') }}
          </q-tooltip>
        </q-btn>
      </div>
    </div>
  </q-drawer>
</template>

<script lang="ts" setup>
import { ref } from 'vue'
import { useToggle } from '@vueuse/shared'
import { useVModel } from '@vueuse/core'
import ToolbarTitle from 'src/components/Toolbar/ToolbarTitle.vue'
import BaseMenu from 'src/components/Menu/BaseMenu.vue'
import { useAppStore } from 'src/stores/app'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'MyDrawer' })

const { t } = useI18n({ useScope: 'global' })

const props = withDefaults(
  defineProps<{ modelValue: boolean; title: string; subtitle?: string; logo?: string }>(),
  {
    modelValue: false,
    title: '',
    subtitle: '',
    logo: '',
  },
)

const emit = defineEmits<{ (e: 'update:modelValue'): void }>()

const _model_value = useVModel(props, 'modelValue', emit)

const is_mini_overlay = ref<boolean>(false)

const appStore = useAppStore()
const { is_drawer_mini } = storeToRefs(appStore)

const toggle_drawer_open = useToggle(_model_value)

const toggleDrawer = () => {
  if (_model_value.value) {
    is_mini_overlay.value = false
    appStore.setDrawerMini(!is_drawer_mini.value)
    return
  }
  appStore.setDrawerMini(false)
  toggle_drawer_open(true)
}

const closeDrawer = () => {
  is_mini_overlay.value = false
  appStore.setDrawerMini(false)
  toggle_drawer_open(false)
}

defineExpose({
  toggleDrawer,
  closeDrawer,
  toggleDrawerOpen: toggle_drawer_open,
  toggleDrawerMini: () => appStore.setDrawerMini(!is_drawer_mini.value),
})
</script>

<style scoped lang="scss">
.app-drawer {
  --app-drawer-surface: #fbfcff;

  border-right: 1px solid #e4e8f4;
  background: var(--app-drawer-surface);
  overflow: hidden;

  :deep(.q-drawer__content) {
    overflow: hidden;
  }
}

.app-drawer__shell {
  display: flex;
  width: 100%;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
  background: var(--app-drawer-surface);
}

.app-drawer__brand {
  position: relative;
  z-index: 1;
  flex: 0 0 96px;
  padding: 10px 10px 8px;
  background: var(--app-drawer-surface);
}

.app-drawer__menu {
  width: 100%;
  height: auto;
  min-height: 0;
  flex: 1 1 auto;
  overflow: hidden;
}

.app-drawer--mini {
  .app-drawer__brand {
    flex-basis: 64px;
    padding: 8px 6px;
  }

  .app-drawer__footer {
    display: flex;
    flex: 0 0 54px;
    align-items: center;
    justify-content: center;
    background: var(--app-drawer-surface);
  }
}

.app-drawer__close {
  width: 34px;
  height: 34px;
  color: var(--q-primary);
  border: 1px solid var(--app-primary-border);
  background: var(--app-primary-soft);

  &:hover {
    background: var(--app-primary-soft-strong);
  }
}

:global(.body--dark .app-drawer) {
  --app-drawer-surface: #1b2130;
}

</style>
