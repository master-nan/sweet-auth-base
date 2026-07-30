<template>
  <base-content class="q-pa-sm menu-page" :class="{ 'menu-page--dark': $q.dark.isActive }">
    <q-card flat bordered class="menu-page-header-card">
      <q-card-section class="menu-page-header">
        <q-avatar
          class="menu-icon-tile"
          color="primary"
          text-color="white"
          icon="account_tree"
        />
        <div>
          <div class="menu-page-title">菜单管理</div>
          <div class="menu-page-subtitle">统一维护页面导航、操作按钮与接口权限</div>
        </div>
        <q-space />
        <div class="menu-page-description">菜单定义页面入口，按钮定义功能操作</div>
      </q-card-section>
    </q-card>

    <div class="menu-workbench">
      <div class="menu-workbench-body">
        <section class="menu-tree-pane">
          <div class="menu-pane-header">
            <div class="menu-pane-title">菜单结构</div>
            <q-badge rounded color="deep-purple-1" text-color="primary">
              {{ countMenus(filteredMenuData) }}
            </q-badge>
            <q-space />
            <q-btn
              round
              outline
              color="primary"
              icon="refresh"
              size="sm"
              :loading="loading"
              @click="fetchMenus"
            >
              <q-tooltip>刷新菜单</q-tooltip>
            </q-btn>
            <q-btn
              v-for="btn in master_top_buttons"
              :key="btn.id"
              v-bind="menuButtonDisplayProps(btn)"
              :color="btn.color || 'primary'"
              size="sm"
              :disable="loading"
              @click="handleButtonClick(btn)"
            >
              <q-tooltip>{{ btn.name }}</q-tooltip>
            </q-btn>
          </div>

          <div class="menu-tree-search">
            <q-input
              v-model="searchText"
              outlined
              dense
              clearable
              placeholder="搜索菜单名称、编码或路径"
            >
              <template #prepend>
                <q-icon name="search" />
              </template>
            </q-input>
          </div>

          <q-scroll-area class="menu-tree-scroll">
            <q-tree
              v-model:expanded="expandedMenuIds"
              :selected="selectedMenu?.id ?? null"
              :nodes="filteredMenuData"
              node-key="id"
              children-key="children"
              no-connectors
              class="menu-tree"
            >
              <template #default-header="{ node }">
                <div
                  class="menu-tree-node"
                  :class="{ 'menu-tree-node--selected': selectedMenu?.id === node.id }"
                  @click.stop="handleNodeSelected(node)"
                >
                  <q-icon
                    :name="node.icon || (node.children?.length ? 'folder' : 'insert_drive_file')"
                    class="menu-tree-node-icon"
                  />
                  <div class="menu-tree-node-main">
                    <div class="menu-tree-node-title">{{ t(node.title) }}</div>
                    <div class="menu-tree-node-meta">{{ menuNodeMeta(node) }}</div>
                  </div>
                  <span
                    v-if="selectedMenu?.id !== node.id"
                    class="menu-tree-state"
                    :class="{ 'menu-tree-state--disabled': !isMenuEnabled(node) }"
                  />
                  <div class="menu-tree-node-actions">
                    <q-btn
                      v-for="btn in treeLineButtons"
                      :key="btn.id"
                      v-bind="menuButtonDisplayProps(btn)"
                      :icon="treeActionIcon(btn)"
                      flat
                      dense
                      round
                      size="xs"
                      color="primary"
                      :loading="loading"
                      @click.stop="handleButtonClick(btn, node)"
                    >
                      <q-tooltip>{{ btn.name }}</q-tooltip>
                    </q-btn>
                  </div>
                </div>
              </template>
            </q-tree>
          </q-scroll-area>
        </section>

        <section v-if="selectedMenu" class="menu-detail-pane">
          <div class="menu-detail-header">
            <q-avatar
              class="menu-icon-tile"
              color="deep-purple-1"
              text-color="primary"
              :icon="selectedMenu.icon || 'menu'"
            />
            <div class="menu-detail-heading">
              <div class="menu-detail-title-row">
                <div class="menu-detail-title">{{ t(selectedMenu.title) }}</div>
                <div
                  class="menu-detail-status"
                  :class="{ 'menu-detail-status--disabled': !isMenuEnabled(selectedMenu) }"
                >
                  {{ isMenuEnabled(selectedMenu) ? '正常显示' : '已隐藏' }}
                </div>
              </div>
              <div class="menu-detail-meta">
                <q-badge color="deep-purple-1" text-color="primary">
                  {{ selectedMenu.name }}
                </q-badge>
                <span>{{ selectedMenu.path || '-' }}</span>
              </div>
            </div>
            <q-space />
            <q-btn
              v-if="selectedMenuUpdateButton"
              v-bind="menuButtonDisplayProps(selectedMenuUpdateButton)"
              icon="edit"
              label="编辑菜单"
              :round="false"
              :color="selectedMenuUpdateButton.color || 'primary'"
              size="sm"
              @click="handleButtonClick(selectedMenuUpdateButton, selectedMenu)"
            >
              <q-tooltip>{{ selectedMenuUpdateButton.name }}</q-tooltip>
            </q-btn>
          </div>

          <div class="menu-detail-summary">
            <div class="menu-summary-item menu-summary-item--wide">
              <div class="menu-summary-label">页面组件</div>
              <div class="menu-summary-value">{{ selectedMenu.component || '-' }}</div>
            </div>
            <div class="menu-summary-item">
              <div class="menu-summary-label">页面类型</div>
              <div class="menu-summary-value">{{ menuPageTypeLabel(selectedMenu.page_type) }}</div>
            </div>
            <div class="menu-summary-item">
              <div class="menu-summary-label">父级菜单</div>
              <div class="menu-summary-value">{{ selectedParentMenuTitle }}</div>
            </div>
            <div class="menu-summary-item">
              <div class="menu-summary-label">排序</div>
              <div class="menu-summary-value">{{ selectedMenu.sequence ?? '-' }}</div>
            </div>
          </div>

          <q-tabs
            v-model="activeTab"
            dense
            align="left"
            active-color="primary"
            indicator-color="primary"
            narrow-indicator
            class="menu-detail-tabs"
          >
            <q-tab name="page_buttons">
              <div class="menu-tab-label">
                <q-icon name="touch_app" />
                <span>页面按钮</span>
                <q-badge rounded color="deep-purple-1" text-color="primary">
                  {{ pageMenuButtons.length }}
                </q-badge>
              </div>
            </q-tab>
            <q-tab name="api_permissions">
              <div class="menu-tab-label">
                <q-icon name="verified_user" />
                <span>接口权限</span>
                <q-badge rounded color="deep-purple-1" text-color="primary">
                  {{ apiPermissionButtons.length }}
                </q-badge>
              </div>
            </q-tab>
            <q-tab name="preview">
              <div class="menu-tab-label">
                <q-icon name="visibility" />
                <span>菜单预览</span>
              </div>
            </q-tab>
          </q-tabs>

          <div v-if="activeTab !== 'preview'" class="menu-button-panel">
            <div class="menu-button-toolbar">
              <div>
                <div class="menu-button-toolbar-title">{{ activeButtonPanelTitle }}</div>
                <div class="menu-button-toolbar-subtitle">{{ activeButtonPanelSubtitle }}</div>
              </div>
              <q-space />
              <q-btn
                v-for="btn in detailToolbarButtons"
                :key="btn.id"
                v-bind="menuButtonDisplayProps(btn)"
                :label="activeToolbarButtonLabel(btn)"
                :color="btn.color || 'primary'"
                @click="handleButtonClick(btn)"
              >
                <q-tooltip>{{ activeToolbarButtonLabel(btn) }}</q-tooltip>
              </q-btn>
            </div>

            <q-scroll-area
              ref="menuButtonScrollAreaRef"
              class="menu-button-table-scroll"
              :delay="0"
              :vertical-offset="[6, 6]"
            >
              <q-table
                class="menu-button-table sticky-header-table"
                :rows="activeButtonRows"
                :columns="buttonDisplayColumns"
                row-key="id"
                flat
                :dark="$q.dark.isActive"
                :loading="loading"
                :pagination="{ rowsPerPage: 0 }"
                hide-pagination
              >
                <template #body-cell-feature="props">
                  <q-td :props="props">
                    <div class="menu-button-feature">
                      <q-avatar
                        class="menu-icon-tile"
                        size="30px"
                        color="deep-purple-1"
                        text-color="primary"
                        :icon="props.row.icon || 'touch_app'"
                      />
                      <div>
                        <div class="menu-button-name">{{ props.row.name }}</div>
                        <div class="menu-button-code">{{ props.row.code }}</div>
                      </div>
                    </div>
                  </q-td>
                </template>
                <template #body-cell-position="props">
                  <q-td :props="props">
                    <q-badge color="grey-3" text-color="grey-8">
                      {{ menuButtonPositionLabel(props.row.position) }}
                    </q-badge>
                  </q-td>
                </template>
                <template #body-cell-event_action="props">
                  <q-td :props="props">
                    <span class="menu-button-action">{{ props.row.event_action || '-' }}</span>
                  </q-td>
                </template>
                <template #body-cell-permission="props">
                  <q-td :props="props">
                    <div class="menu-button-permission">
                      <q-badge
                        v-if="props.row.http_method"
                        :color="httpMethodColor(props.row.http_method)"
                      >
                        {{ props.row.http_method.toUpperCase() }}
                      </q-badge>
                      <span class="menu-button-api-path">{{ props.row.api_path || '-' }}</span>
                    </div>
                  </q-td>
                </template>
                <template #body-cell-state="props">
                  <q-td :props="props">
                    <q-badge outline :color="isMenuButtonEnabled(props.row) ? 'positive' : 'grey'">
                      {{ isMenuButtonEnabled(props.row) ? '启用' : '停用' }}
                    </q-badge>
                  </q-td>
                </template>
                <template #body-cell-actions="props">
                  <q-td :props="props">
                    <div class="row items-center no-wrap q-gutter-xs">
                      <q-btn
                        v-for="btn in detailRowButtons"
                        :key="btn.id"
                        v-bind="menuButtonDisplayProps(btn)"
                        flat
                        dense
                        round
                        size="sm"
                        :color="btn.color || 'primary'"
                        @click="handleButtonClick(btn, props.row)"
                      >
                        <q-tooltip>{{ btn.name }}</q-tooltip>
                      </q-btn>
                    </div>
                  </q-td>
                </template>
                <template #no-data>
                  <div class="full-width row flex-center text-grey-7 q-gutter-sm q-pa-xl">
                    <q-icon
                      :name="activeTab === 'page_buttons' ? 'touch_app' : 'verified_user'"
                      size="28px"
                    />
                    <span>{{ activeButtonEmptyText }}</span>
                  </div>
                </template>
              </q-table>
            </q-scroll-area>
          </div>

          <div v-else class="preview-container">
            <div class="preview-sidebar">
              <div class="preview-header">应用导航</div>
              <q-scroll-area class="preview-scroll">
                <q-list bordered separator :dark="$q.dark.isActive">
                  <q-item-label header>菜单视图</q-item-label>
                  <template v-for="menu in previewMenus" :key="menu.id">
                    <q-expansion-item
                      v-if="menu.children && menu.children.length > 0"
                      :icon="menu.icon || 'folder'"
                      :label="t(menu.title)"
                      :default-opened="
                        selectedMenu &&
                        (selectedMenu.id === menu.id || selectedMenu.pid === menu.id)
                      "
                    >
                      <q-list padding :dark="$q.dark.isActive">
                        <q-item
                          v-for="subMenu in menu.children"
                          :key="subMenu.id"
                          class="q-pl-xl"
                          clickable
                          :active="selectedMenu && selectedMenu.id === subMenu.id"
                          active-class="bg-primary text-white"
                        >
                          <q-item-section avatar>
                            <q-icon :name="subMenu.icon || 'subdirectory_arrow_right'" />
                          </q-item-section>
                          <q-item-section>{{ t(subMenu.title) }}</q-item-section>
                        </q-item>
                      </q-list>
                    </q-expansion-item>

                    <q-item
                      v-else
                      clickable
                      :active="selectedMenu && selectedMenu.id === menu.id"
                      active-class="bg-primary text-white"
                    >
                      <q-item-section avatar>
                        <q-icon :name="menu.icon || 'folder'" />
                      </q-item-section>
                      <q-item-section>{{ t(menu.title) }}</q-item-section>
                    </q-item>
                  </template>
                </q-list>
              </q-scroll-area>
            </div>

            <div class="preview-content">
              <menu-route-preview :selectedMenu="selectedMenu" :menuTree="menuData" />
            </div>
          </div>
        </section>

        <section v-else class="menu-detail-empty">
          <q-icon name="account_tree" size="72px" />
          <div class="menu-empty-title">选择左侧菜单</div>
          <div class="menu-empty-desc">选择菜单后维护页面按钮与接口权限</div>
        </section>
      </div>
    </div>

    <!-- 菜单表单对话框 -->
    <dynamic-form-dialog
      v-model="showFormDialog"
      :edit-data="currentEditData"
      :title="
        currentEditData?.id ? '编辑菜单' : currentEditData?.pid ? '新增子菜单' : '新增顶级菜单'
      "
      :fields="menuFields"
      :submit-btn-text="currentEditData?.id ? '保存' : '创建'"
      @submit="handleFormSubmit"
    />

    <dynamic-form-dialog
      v-model="buttonDialogOpen"
      :edit-data="buttonEditData"
      :title="buttonDialogTitle"
      :fields="buttonFields"
      :submit-btn-text="isButtonEdit ? '保存' : '创建'"
      table-code="sys_menu_button"
      @submit="handleButtonFormSubmit"
    />
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'system_menu' })
import { ref, computed, nextTick, onMounted, watch } from 'vue'
import { useQuasar, type QTableProps } from 'quasar'
import { useRoute, useRouter } from 'vue-router'
import BaseContent from 'components/BaseContent/BaseContent.vue'
import MenuRoutePreview from 'src/components/MenuPreview/MenuRoutePreview.vue'
import DynamicFormDialog from 'src/components/FormDialog/DynamicFormDialog.vue'
import { useLoadingStore } from 'src/stores/loading'
import { storeToRefs } from 'pinia'
import {
  useMenuApi,
  type Menu,
  type MenuButton,
  type MenuCreateReq,
  type MenuButtonCreateReq,
  type MenuButtonUpdateReq,
} from 'src/api/services/sys-menu'
import { useTableApi, type TableField } from 'src/api/services/sys-table'
import {
  SysMenuButtonEventAction,
  SysMenuButtonEventActionMap,
  SysMenuButtonPosition,
  SysMenuButtonPositionMap,
  SysTableFieldInputType,
  SysTableFieldType,
} from 'src/types/enum'
import type { Query } from 'src/types/global'
import { useI18n } from 'vue-i18n'
import { useDictStore } from 'src/stores/dict'
import { useMasterDetailPageButtons } from 'src/composables/page-buttons'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'
import { isApiPermission, isPageButton } from 'src/utils/menu-button'
import { useConfirmDialog } from 'src/composables/confirm-dialog'

const { t } = useI18n()

// 全局状态
const loadingStore = useLoadingStore()
const { loading } = storeToRefs(loadingStore)

// API 和工具
const $q = useQuasar()
const { confirmDanger } = useConfirmDialog($q)
const route = useRoute()
const router = useRouter()
const menuApi = useMenuApi()
const tableApi = useTableApi()
const dictStore = useDictStore()

// 菜单数据相关
const menuData = ref<Menu[]>([])
const searchText = ref('')
const selectedMenu = ref<Menu | null>(null)
const activeTab = ref<'page_buttons' | 'api_permissions' | 'preview'>('page_buttons')
const expandedMenuIds = ref<number[]>([])
type ButtonTableTab = 'page_buttons' | 'api_permissions'
type ButtonTableScrollAreaRef = { getScrollTarget?: () => HTMLElement }
const menuButtonScrollAreaRef = ref<ButtonTableScrollAreaRef | null>(null)
const buttonTableScrollPositions: Record<ButtonTableTab, number> = {
  page_buttons: 0,
  api_permissions: 0,
}

const isButtonTableTab = (tab: typeof activeTab.value): tab is ButtonTableTab =>
  tab === 'page_buttons' || tab === 'api_permissions'

const getButtonTableScrollElement = () => menuButtonScrollAreaRef.value?.getScrollTarget?.() || null

const resetButtonTableScroll = () => {
  buttonTableScrollPositions.page_buttons = 0
  buttonTableScrollPositions.api_permissions = 0
  const scrollElement = getButtonTableScrollElement()
  if (scrollElement) scrollElement.scrollTop = 0
}

watch(activeTab, async (nextTab, previousTab) => {
  const scrollElement = getButtonTableScrollElement()
  if (scrollElement && isButtonTableTab(previousTab)) {
    buttonTableScrollPositions[previousTab] = scrollElement.scrollTop
  }

  await nextTick()

  if (!isButtonTableTab(nextTab)) return
  const nextScrollElement = getButtonTableScrollElement()
  if (nextScrollElement) {
    nextScrollElement.scrollTop = buttonTableScrollPositions[nextTab]
  }
})

const isMenuButtonChildAction = (btn: MenuButton) => {
  return (
    btn.event_action === 'create_button' ||
    btn.event_action === 'update_button' ||
    btn.event_action === 'delete_button' ||
    btn.event_action === 'query_button' ||
    btn.event_action === 'button_metadata' ||
    btn.code.includes('_button_') ||
    btn.api_path.includes('/admin/menu/button')
  )
}

const {
  master_line_buttons,
  master_top_buttons,
  detail_line_buttons,
  detail_top_buttons,
  detail_form_top_buttons,
  detail_form_bottom_buttons,
} = useMasterDetailPageButtons('system_menu', isMenuButtonChildAction)

const detailToolbarButtons = computed(() => [
  ...detail_top_buttons.value,
  ...detail_form_top_buttons.value,
])

const detailRowButtons = computed(() => [
  ...detail_line_buttons.value,
  ...detail_form_bottom_buttons.value,
])

const selectedMenuUpdateButton = computed(() =>
  master_line_buttons.value.find((btn) => btn.event_action === 'update'),
)

const treeActionOrder: Record<string, number> = {
  create_child: 10,
  update: 20,
  duplicate: 30,
  delete: 40,
}

const treeLineButtons = computed(() =>
  [...master_line_buttons.value].sort(
    (a, b) => (treeActionOrder[a.event_action] ?? 100) - (treeActionOrder[b.event_action] ?? 100),
  ),
)

const treeActionIcon = (button: MenuButton) => {
  const iconMap: Record<string, string> = {
    create_child: 'add',
    update: 'edit',
    duplicate: 'content_copy',
    delete: 'delete_outline',
  }
  return iconMap[button.event_action] || button.icon || 'more_horiz'
}

const countMenus = (menus: Menu[]): number =>
  menus.reduce((total, menu) => total + 1 + countMenus(menu.children || []), 0)

const findMenu = (menus: Menu[], id: number): Menu | undefined => {
  for (const menu of menus) {
    if (menu.id === id) return menu
    const found = findMenu(menu.children || [], id)
    if (found) return found
  }
  return undefined
}

const findParentMenu = (menus: Menu[], id: number): Menu | undefined => {
  for (const menu of menus) {
    if (menu.children?.some((child) => child.id === id)) return menu
    const found = findParentMenu(menu.children || [], id)
    if (found) return found
  }
  return undefined
}

const findMenuByName = (menus: Menu[], name: string): Menu | undefined => {
  for (const menu of menus) {
    if (menu.name === name) return menu
    const found = findMenuByName(menu.children || [], name)
    if (found) return found
  }
  return undefined
}

const collectExpandableMenuIds = (menus: Menu[]): number[] =>
  menus.flatMap((menu) => [
    ...(menu.children?.length ? [menu.id] : []),
    ...collectExpandableMenuIds(menu.children || []),
  ])

const menuNodeMeta = (menu: Menu) => {
  if (menu.children?.length) return `${menu.name} · ${menu.children.length} 个子菜单`
  return [menu.name, menu.path].filter(Boolean).join(' · ') || '-'
}

const isMenuEnabled = (menu: Menu) => menu.state !== false && !menu.is_hidden

const isMenuButtonEnabled = (button: MenuButton) =>
  button.state !== false && button.is_hidden !== true && button.is_disabled !== true

const menuPageTypeLabel = (pageType?: string) => {
  const labels: Record<string, string> = {
    directory: '目录',
    fixed: '固定页面',
    low_code: '低代码页面',
  }
  return pageType ? labels[pageType] || pageType : '-'
}

const menuButtonPositionLabel = (position: SysMenuButtonPosition) =>
  SysMenuButtonPositionMap[position] || '-'

const httpMethodColor = (method: string) => {
  const colors: Record<string, string> = {
    GET: 'blue-7',
    POST: 'primary',
    PUT: 'orange-7',
    PATCH: 'deep-orange-7',
    DELETE: 'negative',
  }
  return colors[method.toUpperCase()] || 'grey-7'
}

const selectedParentMenuTitle = computed(() => {
  if (!selectedMenu.value) return '-'
  const parent = findParentMenu(menuData.value, selectedMenu.value.id)
  return parent ? t(parent.title) : '顶级菜单'
})

const firstMenu = (menus: Menu[]): Menu | null => {
  for (const menu of menus) {
    return menu
  }
  return null
}

const findCurrentRouteMenu = (menus: Menu[]) => {
  const routeName = typeof route.name === 'string' ? route.name : ''
  if (!routeName) return undefined
  return findMenuByName(menus, routeName)
}

const action_handlers: Record<string, (row?: any) => void> = {
  create: () => openAddDialog(0),
  create_child: (row) => row && openAddDialog(row.id),
  update: (row) => row && openEditDialog(row),
  detail: (row) => row && openRecordDetail('sys_menu', row.id),
  duplicate: (row) => row && duplicateMenu(row),
  delete: (row) => row && confirmDeleteMenu(row.id || row),
  create_button: () => openAddButtonDialog(),
  update_button: (row) => row && openEditButtonDialog(row),
  delete_button: (row) => row && confirmDeleteButton(row),
}

const handleButtonClick = (btn: MenuButton, row?: any) => {
  const action = (btn.event_action || '').trim()
  const handler = action_handlers[action]
  if (handler) handler(row)
}

const openRecordDetail = (tableCode: string, id: number) => {
  void router.push({
    name: 'record_detail',
    params: {
      source: 'generalization',
      table_code: tableCode,
      id,
    },
  })
}

// 菜单表单相关
const showFormDialog = ref(false)
const currentEditData = ref<Menu | null>(null)
const menuFields = ref<TableField[]>([])

// 按钮相关
type MenuButtonFormData = (MenuButtonCreateReq | MenuButtonUpdateReq) & {
  is_button?: boolean
}

const menuButtons = ref<MenuButton[]>([])
const buttonDialogOpen = ref(false)
const buttonFieldMeta = ref<TableField[]>([])
const buttonEditData = ref<MenuButtonFormData>({
  menu_id: 0,
  name: '',
  code: '',
  memo: '',
  position: SysMenuButtonPosition.LINE,
  event_type: '',
  event_action: '',
  icon: '',
  color: '',
  sequence: 0,
  api_path: '',
  http_method: '',
  disable_when: '',
  params_schema: '',
  confirm_text: '',
  is_hidden: false,
  is_button: true,
  is_disabled: false,
})
const isButtonEdit = computed(
  () => 'id' in buttonEditData.value && Boolean(buttonEditData.value.id),
)
const currentButtonIsPageButton = computed(() =>
  normalizeBooleanValue(buttonEditData.value.is_button, true),
)
const buttonDialogTitle = computed(() => {
  const kind = currentButtonIsPageButton.value ? '页面按钮' : '接口权限'
  return `${isButtonEdit.value ? '编辑' : '新增'}${kind}`
})

const pageMenuButtons = computed(() =>
  menuButtons.value.filter(isPageButton).sort((a, b) => (a.sequence || 0) - (b.sequence || 0)),
)

const apiPermissionButtons = computed(() =>
  menuButtons.value.filter(isApiPermission).sort((a, b) => (a.sequence || 0) - (b.sequence || 0)),
)

const activeButtonRows = computed(() =>
  activeTab.value === 'api_permissions' ? apiPermissionButtons.value : pageMenuButtons.value,
)

const activeButtonPanelTitle = computed(() =>
  activeTab.value === 'api_permissions' ? '接口权限' : '页面按钮',
)

const activeButtonPanelSubtitle = computed(() =>
  activeTab.value === 'api_permissions'
    ? '维护当前页面使用但不直接显示为按钮的接口权限'
    : '控制当前页面中可见的操作及其接口权限',
)

const activeButtonEmptyText = computed(() =>
  activeTab.value === 'api_permissions'
    ? '当前菜单还没有配置接口权限'
    : '当前菜单还没有配置页面按钮',
)

const activeToolbarButtonLabel = (button: MenuButton) =>
  activeTab.value === 'api_permissions' && button.event_action === 'create_button'
    ? '新增接口权限'
    : button.name

const normalizeBooleanValue = (value: unknown, fallback = false): boolean => {
  if (value === undefined || value === null || value === '') return fallback
  if (typeof value === 'boolean') return value
  if (typeof value === 'number') return value !== 0
  if (typeof value !== 'string') return fallback
  const normalized = value.trim().toLowerCase()
  if (['true', '1', 't', 'yes', 'y', '是'].includes(normalized)) return true
  if (['false', '0', 'f', 'no', 'n', '否'].includes(normalized)) return false
  return fallback
}

// 默认空查询
const emptyAdvancedQuery = (): Query => ({
  page: 1,
  num: 15,
  expressions: [
    {
      rules: [{ field: '', value: null }],
      nested: [],
    },
  ],
})

// 查询参数
const query = ref<Query>({
  page: 1,
  num: 15,
  order: {
    field: 'sequence',
    is_asc: true,
  },
  table_code: 'sys_menu',
  expressions: emptyAdvancedQuery().expressions,
  quick_query: {
    keyword: '',
  },
  include_deleted: false,
})

const buttonDisplayColumns: QTableProps['columns'] = [
  {
    name: 'feature',
    label: '功能按钮',
    field: 'name',
    align: 'left',
    style: 'width: 22%',
    headerStyle: 'width: 22%',
  },
  {
    name: 'position',
    label: '展示位置',
    field: 'position',
    align: 'left',
    style: 'width: 13%',
    headerStyle: 'width: 13%',
  },
  {
    name: 'event_action',
    label: '触发动作',
    field: 'event_action',
    align: 'left',
    style: 'width: 16%',
    headerStyle: 'width: 16%',
  },
  {
    name: 'permission',
    label: '接口权限',
    field: 'api_path',
    align: 'left',
  },
  {
    name: 'state',
    label: '状态',
    field: 'state',
    align: 'left',
    style: 'width: 9%',
    headerStyle: 'width: 9%',
  },
  {
    name: 'actions',
    label: '操作',
    field: 'id',
    align: 'left',
    style: 'width: 90px',
    headerStyle: 'width: 90px',
  },
]

const normalizeMenuFields = (fields: TableField[] = []) => {
  const labelMap: Record<string, string> = {
    page_type: '页面类型',
    table_code: '绑定表编码',
    option: '扩展配置',
  }
  return fields.map((field) => ({
    ...field,
    field_name: labelMap[field.field_code] || field.field_name,
  }))
}

const menuButtonEventActionOptions = Object.values(SysMenuButtonEventAction).map((value) => ({
  label: `${SysMenuButtonEventActionMap[value]}（${value}）`,
  value,
}))

// 从字段元数据中提取 position 字典选项
const buttonFields = computed(() =>
  buttonFieldMeta.value
    .filter((field) => field.is_insert_show || field.is_update_show)
    .map((field) => {
      if (field.field_code === 'menu_id') {
        return { ...field, is_insert_show: false, is_update_show: false }
      }
      if (field.field_code === 'position') {
        return {
          ...field,
          input_type: SysTableFieldInputType.SELECT,
          dict_code: field.dict_code || 'sys_menu_button_position',
          field_name: currentButtonIsPageButton.value ? '展示位置' : '接口归类',
        }
      }
      if (field.field_code === 'display_mode') {
        return {
          ...field,
          input_type: SysTableFieldInputType.SELECT,
          dict_code: field.dict_code || 'sys_menu_button_display_mode',
          field_name: '展示方式',
        }
      }
      if (field.field_code === 'event_action') {
        return {
          ...field,
          input_type: SysTableFieldInputType.SELECT,
          dict_code: field.dict_code || 'sys_menu_button_event_action',
          field_name: currentButtonIsPageButton.value ? '事件动作' : '权限动作',
          options: menuButtonEventActionOptions,
        }
      }
      if (field.field_code === 'http_method') {
        return {
          ...field,
          input_type: SysTableFieldInputType.SELECT,
          dict_code: field.dict_code || 'http_method',
          field_name: '请求方法',
        }
      }
      if (
        field.field_code === 'params_schema' ||
        field.field_code === 'disable_when' ||
        field.field_code === 'before_hooks' ||
        field.field_code === 'after_hooks'
      ) {
        return {
          ...field,
          field_type: SysTableFieldType.JSON,
          input_type: SysTableFieldInputType.JSON_EDITOR,
        }
      }
      if (field.field_code === 'is_button') {
        return {
          ...field,
          field_name: '是否页面按钮',
          default_value: 'true',
        }
      }
      if (field.field_code === 'name') {
        return { ...field, field_name: currentButtonIsPageButton.value ? '按钮名称' : '接口名称' }
      }
      if (field.field_code === 'code') {
        return { ...field, field_name: currentButtonIsPageButton.value ? '按钮编码' : '接口编码' }
      }
      if (field.field_code === 'is_hidden') {
        return { ...field, field_name: '是否隐藏', default_value: 'false' }
      }
      if (field.field_code === 'is_disabled') {
        return { ...field, field_name: '是否禁用', default_value: 'false' }
      }
      return field
    })
    .filter((field) => field.is_insert_show || field.is_update_show),
)

// 获取菜单字段定义
const fetchMenuTableFields = async () => {
  // 并行加载菜单和按钮的字段元数据
  const [menuRes, buttonRes] = await Promise.all([
    tableApi.queryTableByCode('sys_menu'),
    tableApi.queryTableByCode('sys_menu_button'),
  ])

  // 收集所有字典编码
  const allDictCodes: string[] = []
  const menuFields_ = menuRes.success
    ? normalizeMenuFields(menuRes.data?.table_fields || [])
    : undefined
  const btnFields = buttonRes.data?.table_fields

  if (btnFields) {
    buttonFieldMeta.value = btnFields
    allDictCodes.push(...btnFields.map((f) => f.dict_code).filter((c): c is string => !!c))
  }
  if (menuFields_) {
    allDictCodes.push(...menuFields_.map((f) => f.dict_code).filter((c): c is string => !!c))
  }

  await dictStore.loadDicts([...new Set(allDictCodes)])

  // 处理菜单字段元数据
  if (menuFields_) {
    menuFields.value = menuFields_.filter((field) => field.is_insert_show || field.is_update_show)
  }
}

// 获取菜单列表
const fetchMenus = async () => {
  const res = await menuApi.queryMenu(query.value)
  if (res.success) {
    menuData.value = Array.isArray(res.data) ? res.data : []
    if (expandedMenuIds.value.length === 0) {
      expandedMenuIds.value = menuData.value
        .filter((menu) => Boolean(menu.children?.length))
        .map((menu) => menu.id)
    }
    const routeMenu = findCurrentRouteMenu(menuData.value)
    const nextSelected = selectedMenu.value
      ? findMenu(menuData.value, selectedMenu.value.id) || routeMenu || firstMenu(menuData.value)
      : routeMenu || firstMenu(menuData.value)

    if (nextSelected) {
      selectedMenu.value = nextSelected
      await fetchMenuButtons(nextSelected.id)
    } else {
      selectedMenu.value = null
      menuButtons.value = []
    }
  }
}

// 过滤菜单数据
const filteredMenuData = computed(() => {
  if (!searchText.value) return menuData.value
  const searchLower = searchText.value.toLowerCase()
  const filterNodes = (nodes: Menu[]): Menu[] => {
    return nodes.filter((node) => {
      const translatedTitle = t(node.title).toLowerCase()
      const matches =
        node.title.toLowerCase().includes(searchLower) ||
        translatedTitle.includes(searchLower) ||
        node.name.toLowerCase().includes(searchLower) ||
        node.path.toLowerCase().includes(searchLower) ||
        (node.component && node.component.toLowerCase().includes(searchLower))

      if (node.children && node.children.length) {
        const filteredChildren = filterNodes(node.children)
        node = { ...node, children: filteredChildren }
        return filteredChildren.length > 0 || matches
      }
      return matches
    })
  }
  return filterNodes([...menuData.value])
})

watch(searchText, (value) => {
  if (!value.trim()) return
  expandedMenuIds.value = collectExpandableMenuIds(filteredMenuData.value)
})

// 获取按钮列表
const fetchMenuButtons = async (menuId: number) => {
  const res = await menuApi.queryMenuButtons(menuId)
  if (res.success) {
    menuButtons.value = res.data
  }
}

// 选择菜单节点
const handleNodeSelected = (menu: Menu) => {
  if (selectedMenu.value?.id === menu.id) return
  selectedMenu.value = menu
  resetButtonTableScroll()
  void fetchMenuButtons(menu.id)
}

// 获取下一个序号
const getNextSequence = (pid: number): number => {
  let maxSequence = 0

  const findMaxSequence = (menus: Menu[], parentId: number) => {
    for (const menu of menus) {
      if (menu.pid === parentId) {
        maxSequence = Math.max(maxSequence, menu.sequence || 0)
      }

      if (menu.children && menu.children.length > 0) {
        findMaxSequence(menu.children, parentId)
      }
    }
  }

  findMaxSequence(menuData.value, pid)
  return maxSequence + 10 // 增加10作为步长
}

// 打开添加菜单对话框
const openAddDialog = (pid: number) => {
  // 创建一个新的菜单对象
  currentEditData.value = {
    id: 0,
    pid: pid,
    name: '',
    path: '',
    component: '',
    title: '',
    is_hidden: false,
    sequence: getNextSequence(pid),
    page_type: 'fixed',
    table_code: '',
    option: '',
    icon: '',
    redirect: '',
  }

  showFormDialog.value = true
}

// 打开编辑菜单对话框
const openEditDialog = (menu: Menu) => {
  currentEditData.value = { ...menu }
  showFormDialog.value = true
}

// 复制菜单
const duplicateMenu = (menu: Menu) => {
  currentEditData.value = {
    ...menu,
    id: 0,
    title: `${menu.title} (复制)`,
  }
  showFormDialog.value = true
}

// 确认删除菜单
const confirmDeleteMenu = (menuId: number) => {
  confirmDanger({
    message: '确定要删除该菜单吗？删除后将无法恢复，且所有子菜单和按钮也将被删除。',
  }).onOk(() => {
    void (async () => {
      const res = await menuApi.deleteMenu(menuId)
      if (res.success) {
        if (selectedMenu.value && selectedMenu.value.id === menuId) {
          selectedMenu.value = null
        }
        await fetchMenus()
      }
    })()
  })
}

// 处理表单提交
const handleFormSubmit = async (formPayload: { data: Menu; isEdit: boolean; id?: number }) => {
  if (formPayload.isEdit && formPayload.id) {
    // 编辑模式
    const res = await menuApi.updateMenu({
      ...formPayload.data,
      id: formPayload.id,
    })
    if (res.success) {
      showFormDialog.value = false

      // 如果更新的是当前选中的菜单，更新选中数据
      if (selectedMenu.value && selectedMenu.value.id === formPayload.id) {
        selectedMenu.value = { ...formPayload.data, id: formPayload.id }
      }
    }
  } else {
    // 创建模式
    const res = await menuApi.createMenu(formPayload.data as MenuCreateReq)
    if (res.success) {
      showFormDialog.value = false

      // 如果创建的是所选菜单的子菜单，刷新后自动选中父菜单
      if (selectedMenu.value && formPayload.data.pid === selectedMenu.value.id) {
        const parentId = selectedMenu.value.id
        await fetchMenus()
        const parent = findMenu(menuData.value, parentId)
        if (parent) {
          selectedMenu.value = parent
          fetchMenuButtons(parent.id)
        }
        return
      }
    }
  }

  // 刷新菜单列表
  await fetchMenus()
}

// 添加按钮
const openAddButtonDialog = () => {
  if (!selectedMenu.value) return
  const createAsPageButton = activeTab.value !== 'api_permissions'

  buttonEditData.value = {
    menu_id: selectedMenu.value.id,
    name: '',
    code: '',
    memo: '',
    position: SysMenuButtonPosition.LINE,
    event_type: '',
    event_action: '',
    icon: '',
    color: '',
    sequence: 0,
    api_path: '',
    http_method: '',
    disable_when: '',
    params_schema: '',
    confirm_text: '',
    is_hidden: false,
    is_button: createAsPageButton,
    is_disabled: false,
    display_mode: 'auto',
  }

  buttonDialogOpen.value = true
}

// 编辑按钮
const openEditButtonDialog = (button: MenuButton) => {
  buttonEditData.value = {
    ...button,
    is_button: normalizeBooleanValue(button.is_button, true),
    is_hidden: normalizeBooleanValue(button.is_hidden, false),
    is_disabled: normalizeBooleanValue(button.is_disabled, false),
  }
  buttonDialogOpen.value = true
}

// 确认删除按钮
const confirmDeleteButton = (button: MenuButton) => {
  confirmDanger({
    message: `确定要删除按钮 "${button.name}" 吗？`,
    loading: loading.value,
    disable: loading.value,
  }).onOk(() => {
    void (async () => {
      const res = await menuApi.deleteMenuButton(button.id)
      if (res.success) {
        if (selectedMenu.value) {
          await fetchMenuButtons(selectedMenu.value.id)
        }
      }
    })()
  })
}

const buildMenuButtonPayload = (
  button: Partial<MenuButtonFormData>,
): MenuButtonCreateReq | MenuButtonUpdateReq => {
  const toJSONText = (value: unknown) => {
    if (value === undefined || value === null || value === '') return ''
    if (typeof value === 'string') return value
    return JSON.stringify(value)
  }

  const payload: MenuButtonCreateReq = {
    menu_id: Number(button.menu_id || selectedMenu.value?.id || 0),
    name: String(button.name || ''),
    code: String(button.code || ''),
    memo: String(button.memo || ''),
    position: Number(button.position || SysMenuButtonPosition.LINE) as SysMenuButtonPosition,
    event_type: String(button.event_type || ''),
    event_action: String(button.event_action || ''),
    icon: String(button.icon || ''),
    color: String(button.color || ''),
    sequence: Number(button.sequence || 0),
    api_path: String(button.api_path || ''),
    http_method: String(button.http_method || ''),
    params_schema: toJSONText(button.params_schema),
    confirm_text: String(button.confirm_text || ''),
    disable_when: toJSONText(button.disable_when),
    is_button: normalizeBooleanValue(button.is_button, true),
    is_hidden: normalizeBooleanValue(button.is_hidden, false),
    is_disabled: normalizeBooleanValue(button.is_disabled, false),
  }
  if (button.display_mode !== undefined) payload.display_mode = button.display_mode
  if (button.before_hooks !== undefined) payload.before_hooks = toJSONText(button.before_hooks)
  if (button.after_hooks !== undefined) payload.after_hooks = toJSONText(button.after_hooks)
  return 'id' in button && button.id ? { ...payload, id: button.id } : payload
}

const validateMenuButtonPayload = (button: MenuButtonCreateReq | MenuButtonUpdateReq) => {
  if (!button.event_action) {
    $q.notify({ type: 'warning', position: 'top-right', message: '请选择事件动作' })
    return false
  }
  if (
    !Object.values(SysMenuButtonEventAction).includes(
      button.event_action as SysMenuButtonEventAction,
    )
  ) {
    $q.notify({ type: 'warning', position: 'top-right', message: '事件动作不在系统枚举范围内' })
    return false
  }
  return true
}

const handleButtonFormSubmit = async (formPayload: {
  data: MenuButtonFormData
  isEdit: boolean
  id?: number
}) => {
  const data =
    formPayload.isEdit && formPayload.id
      ? { ...formPayload.data, id: formPayload.id }
      : formPayload.data
  await saveButton(data)
}

// 保存按钮
const saveButton = async (button: MenuButtonCreateReq | MenuButtonUpdateReq) => {
  const payload = buildMenuButtonPayload(button)
  if (!validateMenuButtonPayload(payload)) return

  if ('id' in payload && payload.id) {
    const res = await menuApi.updateMenuButton(payload as MenuButtonUpdateReq)
    if (res.success) {
      buttonDialogOpen.value = false
    }
  } else {
    const res = await menuApi.createMenuButton(payload as MenuButtonCreateReq)
    if (res.success) {
      buttonDialogOpen.value = false
    }
  }
  if (selectedMenu.value) {
    await fetchMenuButtons(selectedMenu.value.id)
  }
}

// 完善预览菜单部分
const buildPreviewMenuTree = (menus: Menu[]): Menu[] => {
  return menus
    .filter((menu) => !menu.is_hidden)
    .map((menu) => ({
      ...menu,
      children:
        menu.children && menu.children.length > 0 ? buildPreviewMenuTree(menu.children) : [],
    }))
}

// 预览可见菜单
const previewMenus = computed(() => {
  return buildPreviewMenuTree(menuData.value)
})

onMounted(async () => {
  await fetchMenuTableFields()
  await fetchMenus()
})
</script>

<style scoped lang="scss">
.menu-page {
  display: flex;
  flex-direction: column;
  gap: 10px;
  overflow: hidden;
  --menu-surface: #fff;
  --menu-subtle: #fbfcff;
  --menu-border: #e3e8f2;
  --menu-text: #172033;
  --menu-muted: #738097;
  --menu-hover: #f5f6fb;
  --menu-selected: #f0edff;
  --menu-header-bg:
    linear-gradient(135deg, rgba(115, 103, 240, 0.08), rgba(0, 184, 169, 0.05)), #fff;
}

.menu-page--dark {
  --menu-surface: #1f2636;
  --menu-subtle: #232b3d;
  --menu-border: #39445a;
  --menu-text: #f1f4fa;
  --menu-muted: #aab5c9;
  --menu-hover: #293247;
  --menu-selected: #343158;
  --menu-header-bg:
    linear-gradient(135deg, rgba(115, 103, 240, 0.16), rgba(0, 184, 169, 0.06)), #1f2636;
}

.menu-page--dark :deep(.bg-deep-purple-1) {
  background: #343158 !important;
}

.menu-page-header-card {
  flex: 0 0 auto;
  overflow: hidden;
  border-color: var(--menu-border);
  border-radius: 8px;
  color: var(--menu-text);
  background: var(--menu-header-bg);
}

.menu-workbench {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  color: var(--menu-text);
}

.menu-page-header {
  min-height: 70px;
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 12px 20px;
}

.menu-page-title {
  font-size: 20px;
  font-weight: 800;
}

.menu-page-subtitle,
.menu-page-description {
  color: var(--menu-muted);
  font-size: 12px;
}

.menu-page-subtitle {
  margin-top: 4px;
}

.menu-icon-tile {
  border-radius: 7px !important;
}

.menu-workbench-body {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(380px, 420px) minmax(0, 1fr);
  gap: 10px;
}

.menu-tree-pane {
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-rows: 58px 58px minmax(0, 1fr);
  overflow: hidden;
  border: 1px solid var(--menu-border);
  border-radius: 8px;
  background: var(--menu-subtle);
}

.menu-pane-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 14px;
  border-bottom: 1px solid var(--menu-border);
  background: var(--menu-surface);
}

.menu-pane-title {
  font-size: 16px;
  font-weight: 800;
}

.menu-tree-search {
  padding: 9px 12px;
  border-bottom: 1px solid var(--menu-border);
}

.menu-tree-scroll {
  min-height: 0;
  height: 100%;
}

.menu-tree {
  padding: 8px 0;
}

.menu-tree :deep(.q-tree__node-header) {
  position: relative;
  min-height: 0;
  padding: 2px 0;
  border-radius: 0;
  transition: background-color 0.15s ease;
}

.menu-tree :deep(.q-tree__node-header:hover) {
  background: var(--menu-hover);
}

.menu-tree :deep(.q-tree__node-header.q-tree__node--selected) {
  color: var(--menu-text);
  background: var(--menu-selected);
}

.menu-tree :deep(.q-tree__node-header.q-tree__node--selected::before) {
  position: absolute;
  z-index: 1;
  top: 8px;
  bottom: 8px;
  left: 0;
  width: 4px;
  border-radius: 0 999px 999px 0;
  background: var(--q-primary);
  content: '';
}

.menu-tree
  :deep(.q-tree__children > .q-tree__node > .q-tree__node-header) {
  width: calc(100% + 47px);
  margin-left: -47px;
  padding-left: 51px;
}

.menu-tree
  :deep(
    .q-tree__children
      .q-tree__children
      > .q-tree__node
      > .q-tree__node-header
  ) {
  width: calc(100% + 94px);
  margin-left: -94px;
  padding-left: 98px;
}

.menu-tree
  :deep(
    .q-tree__children
      .q-tree__children
      .q-tree__children
      > .q-tree__node
      > .q-tree__node-header
  ) {
  width: calc(100% + 141px);
  margin-left: -141px;
  padding-left: 145px;
}

.menu-tree :deep(.q-tree__node-header-content) {
  min-width: 0;
  flex: 1 1 auto;
}

.menu-tree :deep(.q-tree__arrow) {
  color: var(--menu-muted);
  font-size: 20px;
}

.menu-tree-node {
  width: 100%;
  min-width: 0;
  min-height: 54px;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 8px;
  cursor: pointer;
}

.menu-tree-node:hover {
  background: transparent;
}

.menu-tree-node--selected {
  background: transparent;
}

.menu-tree-node--selected .menu-tree-node-title {
  color: var(--menu-text);
}

.menu-tree-node-icon {
  flex: 0 0 auto;
  color: var(--menu-muted);
  font-size: 21px;
}

.menu-tree-node--selected .menu-tree-node-icon {
  color: var(--q-primary);
}

.menu-tree-node-main {
  flex: 1;
  min-width: 0;
}

.menu-tree-node-title {
  overflow: hidden;
  font-size: 14px;
  font-weight: 750;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.menu-tree-node-meta {
  margin-top: 3px;
  overflow: hidden;
  color: var(--menu-muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.menu-tree-state {
  width: 8px;
  height: 8px;
  flex: 0 0 8px;
  margin-right: 8px;
  border-radius: 50%;
  background: var(--q-positive);
  transition: opacity 0.15s ease;
}

.menu-tree-state--disabled {
  background: var(--q-grey-6);
}

.menu-tree-node-actions {
  position: absolute;
  right: 8px;
  display: flex;
  align-items: center;
  gap: 1px;
  opacity: 0;
  pointer-events: none;
  visibility: hidden;
  transition: opacity 0.15s ease;
}

.menu-tree-node:hover .menu-tree-node-actions,
.menu-tree-node--selected .menu-tree-node-actions {
  opacity: 1;
  pointer-events: auto;
  visibility: visible;
}

.menu-tree-node:hover .menu-tree-state {
  opacity: 0;
}

.menu-tree-node:hover .menu-tree-node-main,
.menu-tree-node--selected .menu-tree-node-main {
  padding-right: 94px;
}

.menu-tree-node-actions :deep(.q-btn) {
  width: 22px;
  min-width: 22px;
  height: 22px;
  min-height: 22px;
  padding: 0;
}

.menu-tree-node-actions :deep(.q-icon) {
  font-size: 15px;
}

.menu-detail-pane {
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-rows: 102px 68px 44px minmax(0, 1fr);
  overflow: hidden;
  border: 1px solid var(--menu-border);
  border-radius: 8px;
  background: var(--menu-surface);
}

.menu-detail-header {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 0 22px;
  border-bottom: 1px solid var(--menu-border);
}

.menu-detail-heading {
  min-width: 0;
}

.menu-detail-title-row {
  display: flex;
  align-items: center;
  gap: 9px;
}

.menu-detail-title {
  overflow: hidden;
  font-size: 20px;
  font-weight: 800;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.menu-detail-status {
  display: flex;
  align-items: center;
  gap: 5px;
  color: var(--q-positive);
  font-size: 12px;
  font-weight: 700;
}

.menu-detail-status::before {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: currentColor;
  content: '';
}

.menu-detail-status--disabled {
  color: var(--q-grey-6);
}

.menu-detail-meta {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
  color: var(--menu-muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
}

.menu-detail-meta span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.menu-detail-summary {
  display: grid;
  grid-template-columns: 1.6fr 0.8fr 0.8fr 0.6fr;
  padding: 0 22px;
  border-bottom: 1px solid var(--menu-border);
  background: var(--menu-subtle);
}

.menu-summary-item {
  min-width: 0;
  display: flex;
  flex-direction: column;
  justify-content: center;
  margin: 10px 0;
  padding: 0 18px;
  border-right: 1px solid var(--menu-border);
}

.menu-summary-item:first-child {
  padding-left: 0;
}

.menu-summary-item:last-child {
  padding-right: 0;
  border-right: 0;
}

.menu-summary-label {
  color: var(--menu-muted);
  font-size: 11px;
}

.menu-summary-value {
  margin-top: 5px;
  overflow: hidden;
  font-size: 12px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.menu-detail-tabs {
  padding: 0 14px;
  border-bottom: 1px solid var(--menu-border);
}

.menu-detail-tabs :deep(.q-tab) {
  min-height: 44px;
}

.menu-tab-label {
  display: flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
}

.menu-tab-label > .q-icon {
  font-size: 19px;
}

.menu-button-panel {
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-rows: 68px minmax(0, 1fr);
  padding: 0 20px 18px;
}

.menu-button-toolbar {
  display: flex;
  align-items: center;
}

.menu-button-toolbar-title {
  font-size: 15px;
  font-weight: 800;
}

.menu-button-toolbar-subtitle {
  margin-top: 4px;
  color: var(--menu-muted);
  font-size: 11px;
}

.menu-button-table-scroll {
  min-width: 0;
  min-height: 0;
  height: 100%;
  overflow: hidden;
  border: 1px solid var(--menu-border);
  border-radius: 4px;
}

.menu-button-table.sticky-header-table {
  min-height: 100%;
  height: auto;
  border: 0;
  border-radius: 0;
}

.menu-button-table :deep(.q-table__middle) {
  max-height: none;
  overflow: visible;
}

.menu-button-table :deep(th) {
  height: 46px;
  color: var(--menu-muted);
  background: var(--menu-subtle);
  font-weight: 800;
}

.menu-button-table :deep(td) {
  height: 58px;
}

.menu-button-feature,
.menu-button-permission {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
}

.menu-button-name {
  font-weight: 750;
}

.menu-button-code,
.menu-button-action,
.menu-button-api-path {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}

.menu-button-code {
  margin-top: 2px;
  color: var(--menu-muted);
  font-size: 10px;
}

.menu-button-api-path {
  overflow: hidden;
  color: var(--menu-muted);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.menu-detail-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--menu-border);
  border-radius: 8px;
  color: var(--menu-muted);
  background: var(--menu-surface);
  text-align: center;
}

.menu-empty-title {
  margin-top: 14px;
  color: var(--menu-text);
  font-size: 18px;
  font-weight: 800;
}

.menu-empty-desc {
  margin-top: 6px;
  font-size: 13px;
}

.preview-container {
  min-height: 0;
  display: flex;
  overflow: hidden;
}

.preview-sidebar {
  width: 280px;
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--menu-border);
  background: var(--menu-subtle);
}

.preview-header {
  padding: 12px 16px;
  border-bottom: 1px solid var(--menu-border);
  font-weight: 800;
  background: var(--menu-surface);
}

.preview-scroll {
  flex: 1;
  min-height: 0;
}

.preview-content {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  background: var(--menu-surface);
}

@media (max-width: 1280px) {
  .menu-workbench-body {
    grid-template-columns: minmax(350px, 390px) minmax(0, 1fr);
  }

  .menu-page-description {
    display: none;
  }
}
</style>
