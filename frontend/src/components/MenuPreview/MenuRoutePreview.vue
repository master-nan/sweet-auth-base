<template>
  <div class="menu-route-preview">
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
          {{ selectedMenu?.title ? t(selectedMenu?.title) : '页面内容' }}
        </div>
        <div class="text-grey-7 q-mt-xs">{{ routeDescription }}</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineOptions({ name: 'MenuRoutePreview' })
import { ref, computed, watchEffect } from 'vue'
import type { Menu } from 'src/api/services/sys-menu'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

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
  if (!props.selectedMenu) return '选择一个菜单查看预览'

  if (props.selectedMenu.children && props.selectedMenu.children.length > 0) {
    return `这是一个父级菜单，包含 ${props.selectedMenu.children.length} 个子菜单项`
  }

  return `路由: ${props.selectedMenu.path} | 组件: ${props.selectedMenu.component || '无组件'}`
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
    breadcrumbs.value = ['首页']
  }
})
</script>

<style scoped lang="scss">
.menu-route-preview {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  border: 1px solid #eaeaea;
  border-radius: 8px;
  overflow: hidden;
}

.route-header {
  padding: 12px 16px;
  border-bottom: 1px solid #f0f0f0;
  background-color: #fafafa;
}

.route-breadcrumbs {
  font-size: 14px;
}

.route-content {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.route-placeholder {
  text-align: center;
  max-width: 400px;
}
</style>
