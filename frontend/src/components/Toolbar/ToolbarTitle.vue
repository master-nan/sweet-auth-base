<template>
  <q-btn
    class="toolbar-title"
    :class="{ 'toolbar-title--mini': mini }"
    flat
    no-caps
    no-wrap
    @click="router.push('/')"
  >
    <span class="toolbar-title__logo">
      <img v-if="logo" :src="logo" alt="系统Logo" />
      <q-icon v-else name="admin_panel_settings" />
    </span>
    <span v-if="!mini" class="toolbar-title__text">
      <span class="toolbar-title__name">{{ title }}</span>
      <span class="toolbar-title__subtitle">{{ subtitle }}</span>
    </span>
  </q-btn>
</template>

<script lang="ts" setup>
import { useRouter } from 'vue-router'
const router = useRouter()

defineOptions({ name: 'ToolbarTitle' })
withDefaults(defineProps<{ title?: string; subtitle?: string; mini?: boolean; logo?: string }>(), {
  title: '',
  subtitle: '通用低代码底座',
  mini: false,
  logo: '',
})
</script>

<style scoped lang="scss">
.toolbar-title {
  width: 100%;
  height: 78px;
  justify-content: flex-start;
  padding: 0 14px;
  border: 1px solid #e8ecf7;
  border-radius: 16px;
  color: #111827;
  background: rgba(255, 255, 255, 0.84);

  :deep(.q-btn__content) {
    justify-content: flex-start;
    gap: 12px;
  }

  &:hover {
    background: #ffffff;
    border-color: rgba($primary, 0.22);
  }
}

.toolbar-title__logo {
  display: inline-grid;
  width: 38px;
  height: 38px;
  flex: 0 0 auto;
  place-items: center;
  overflow: hidden;
  border-radius: 12px;
  color: #111827;
  background: rgba($primary, 0.1);

  .q-icon {
    font-size: 24px;
  }
}

.toolbar-title__logo img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.toolbar-title__text {
  min-width: 0;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
}

.toolbar-title__name {
  max-width: 138px;
  overflow: hidden;
  color: #111827;
  font-size: 20px;
  font-weight: 800;
  line-height: 1.1;
  text-overflow: ellipsis;
}

.toolbar-title__subtitle {
  color: #7b8aa6;
  font-size: 12px;
  font-weight: 600;
  line-height: 1.2;
}

.toolbar-title--mini {
  height: 44px;
  justify-content: center;
  padding: 0;
  border-radius: 12px;

  :deep(.q-btn__content) {
    justify-content: center;
  }

  .toolbar-title__logo {
    width: 30px;
    height: 30px;

    .q-icon {
      font-size: 21px;
    }
  }
}
</style>
