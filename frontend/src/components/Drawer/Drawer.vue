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
    <div class="absolute-top app-drawer__brand">
      <toolbar-title
        :title="title"
        :subtitle="subtitle || '通用低代码底座'"
        :logo="logo || ''"
        class="full-width"
        :mini="is_drawer_mini"
      />
    </div>
    <base-menu class="app-drawer__menu" />
    <q-btn
      v-if="is_drawer_mini"
      class="app-drawer__close"
      flat
      round
      dense
      icon="keyboard_double_arrow_left"
      aria-label="隐藏侧栏"
      @click="closeDrawer"
    >
      <q-tooltip anchor="center right" self="center left">隐藏侧栏</q-tooltip>
    </q-btn>
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

defineOptions({ name: 'MyDrawer' })

const props = withDefaults(
  defineProps<{ modelValue: boolean; title: string; subtitle?: string; logo?: string }>(),
  {
    modelValue: false,
    title: '',
    subtitle: '通用低代码底座',
    logo: '',
  },
)

const emit = defineEmits<{ (e: 'update:modelValue'): void }>()

const _model_value = useVModel(props, 'modelValue', emit)

const is_mini_overlay = ref<boolean>(false)

const appStore = useAppStore()
const { is_drawer_mini } = storeToRefs(appStore)

const toggle_drawer_open = useToggle(_model_value)
const toggle_drawer_mini = useToggle(is_drawer_mini)

const toggleDrawer = () => {
  if (_model_value.value) {
    is_mini_overlay.value = false
    toggle_drawer_mini(!is_drawer_mini.value)
    return
  }
  toggle_drawer_mini(false)
  toggle_drawer_open(true)
}

const closeDrawer = () => {
  is_mini_overlay.value = false
  toggle_drawer_mini(false)
  toggle_drawer_open(false)
}

defineExpose({
  toggleDrawer,
  closeDrawer,
  toggleDrawerOpen: toggle_drawer_open,
  toggleDrawerMini: toggle_drawer_mini,
})
</script>

<style scoped lang="scss">
.app-drawer {
  background:
    linear-gradient(180deg, rgba($primary, 0.045) 0%, rgba($primary, 0) 180px),
    #fbfcff;
  border-right: 1px solid #e4e8f4;
}

.app-drawer__brand {
  height: 96px;
  padding: 10px 10px 8px;
  background: transparent;
}

.app-drawer__menu {
  height: calc(100vh - 96px);
  margin-top: 96px;
}

.app-drawer--mini {
  .app-drawer__brand {
    height: 64px;
    padding: 8px 6px;
  }

  .app-drawer__menu {
    height: calc(100vh - 116px);
    margin-top: 64px;
  }
}

.app-drawer__close {
  position: absolute;
  bottom: 12px;
  left: 50%;
  width: 34px;
  height: 34px;
  color: $primary;
  border: 1px solid rgba($primary, 0.2);
  background: rgba($primary, 0.08);
  transform: translateX(-50%);

  &:hover {
    background: rgba($primary, 0.14);
  }
}

</style>
