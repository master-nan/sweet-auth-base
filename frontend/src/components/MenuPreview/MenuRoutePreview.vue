<template>
  <div class="menu-route-preview" :class="{ 'menu-route-preview--dark': $q.dark.isActive }">
    <div class="route-header">
      <div class="route-breadcrumbs">
        <q-breadcrumbs separator="/">
          <template v-for="(breadcrumb, index) in breadcrumbs" :key="index">
            <q-breadcrumbs-el :label="t(breadcrumb)" />
          </template>
        </q-breadcrumbs>
      </div>
    </div>

    <div class="route-content">
      <div class="route-placeholder">
        <q-icon :name="selectedMenu?.icon || 'article'" size="50px" color="primary" />
        <div class="text-h6 q-mt-sm">
          {{ selectedMenu?.title ? t(selectedMenu?.title) : t('ui.pageContents') }}
        </div>
        <div class="route-description q-mt-xs">{{ routeDescription }}</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineOptions({ name: 'MenuRoutePreview' })
import { ref, computed, watchEffect } from 'vue'
import type { Menu } from 'src/api/services/sys-menu'
import { useI18n } from 'vue-i18n'
import { useQuasar } from 'quasar'

const { t } = useI18n()
const $q = useQuasar()

const props = defineProps({
  selectedMenu: {
    type: Object as () => Menu | null,
    default: null,
  },
  menuTree: {
    type: Array as () => Menu[],
    default: () => [],
  },
})

const breadcrumbs = ref<string[]>([])

// 递归查找菜单的完整路径
const findBreadcrumbPath = (menuId: number, menus: Menu[], path: string[] = []): string[] => {
  for (const menu of menus) {
    // 检查当前菜单
    if (menu.id === menuId) {
      return [...path, menu.title]
    }

    // 检查子菜单
    if (menu.children && menu.children.length) {
      const found = findBreadcrumbPath(menuId, menu.children, [...path, menu.title])
      if (found.length) return found
    }
  }

  return []
}

// 根据选中的菜单生成描述
const routeDescription = computed(() => {
  if (!props.selectedMenu) return t('ui.selectAMenuToViewPreview')

  if (props.selectedMenu.children && props.selectedMenu.children.length > 0) {
    return t('ui.thisIsAParentMenuWithSubmenuItems', { value1: props.selectedMenu.children.length })
  }

  return t('ui.routeComponent', {
    value1: props.selectedMenu.path,
    value2: props.selectedMenu.component || t('ui.noComponent'),
  })
})

// 监听选中菜单的变化，更新面包屑
watchEffect(() => {
  if (props.selectedMenu && props.menuTree.length) {
    breadcrumbs.value = findBreadcrumbPath(props.selectedMenu.id, props.menuTree)

    // 如果未找到，可能是因为菜单不在顶层
    if (!breadcrumbs.value.length && props.selectedMenu) {
      breadcrumbs.value = [props.selectedMenu.title]
    }
  } else {
    breadcrumbs.value = [t('ui.homePage')]
  }
})
</script>

<style scoped lang="scss">
.menu-route-preview {
  --preview-surface: #fff;
  --preview-subtle: #fafafa;
  --preview-border: #e4e9f2;
  --preview-text: #172033;
  --preview-muted: #738097;
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--preview-border);
  border-radius: 8px;
  color: var(--preview-text);
  background: var(--preview-surface);
}

.menu-route-preview--dark {
  --preview-surface: #1f2636;
  --preview-subtle: #232b3d;
  --preview-border: #39445a;
  --preview-text: #f1f4fa;
  --preview-muted: #aab5c9;
}

.route-header {
  padding: 12px 16px;
  border-bottom: 1px solid var(--preview-border);
  background: var(--preview-subtle);
}

.route-breadcrumbs {
  font-size: 14px;
}

.route-breadcrumbs :deep(.q-breadcrumbs__el) {
  color: var(--preview-muted);
}

.route-content {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  background: var(--preview-surface);
}

.route-placeholder {
  text-align: center;
  max-width: 400px;
}

.route-description {
  color: var(--preview-muted);
}
</style>
