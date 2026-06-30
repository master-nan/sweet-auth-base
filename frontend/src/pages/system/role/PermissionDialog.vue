<template>
  <q-dialog v-model="isOpen" persistent no-shake @hide="resetState">
    <q-card class="permission-dialog-card">
      <q-card-section class="permission-dialog-header">
        <div>
          <div class="text-h6 text-weight-bold">权限分配</div>
          <div class="text-caption text-grey-7">{{ role?.name }}</div>
        </div>
        <q-space />
        <div class="permission-summary">
          <q-chip dense square color="primary" text-color="white" icon="account_tree">
            {{ selectedMenuCount }}/{{ totalMenuCount }} 菜单
          </q-chip>
          <q-chip dense square color="indigo-1" text-color="primary" icon="touch_app">
            {{ selectedButtonCount }} 按钮
          </q-chip>
          <q-chip
            v-if="apiPermissionCount > 0"
            dense
            square
            color="grey-2"
            text-color="grey-8"
            icon="link"
          >
            {{ apiPermissionCount }} 接口权限
          </q-chip>
        </div>
        <q-btn flat round dense icon="close" :disable="loading" @click="isOpen = false" />
      </q-card-section>

      <q-card-section class="permission-dialog-body">
        <section class="permission-panel permission-menu-panel">
          <div class="permission-panel-head">
            <div>
              <div class="permission-panel-title">菜单权限</div>
              <div class="permission-panel-caption">勾选角色可访问的菜单，点菜单查看右侧按钮</div>
            </div>
            <q-badge color="primary" outline>{{ selectedMenuCount }} 已选</q-badge>
          </div>

          <div class="permission-menu-tools">
            <q-input
              v-model="menuKeyword"
              dense
              outlined
              clearable
              placeholder="搜索菜单名称 / 编码 / 路径"
              class="permission-menu-search"
            >
              <template #prepend>
                <q-icon name="search" />
              </template>
            </q-input>
            <q-btn flat round dense icon="unfold_more" color="primary" @click="expandAllMenus">
              <q-tooltip>展开全部</q-tooltip>
            </q-btn>
            <q-btn flat round dense icon="unfold_less" color="primary" @click="collapseAllMenus">
              <q-tooltip>收起全部</q-tooltip>
            </q-btn>
          </div>

          <q-scroll-area class="permission-menu-scroll">
            <q-tree
              v-if="filteredMenuTree.length"
              :nodes="filteredMenuTree"
              node-key="id"
              tick-strategy="leaf"
              selected-color="primary"
              no-connectors
              v-model:ticked="tickedMenus"
              v-model:selected="selectedMenuId"
              v-model:expanded="expandedKeys"
              @update:selected="onMenuSelect"
              @update:ticked="handleMenuTickedChange"
            >
              <template #default-header="prop">
                <div
                  class="permission-menu-node"
                  :class="{ 'permission-menu-node--active': selectedMenuId === prop.node.id }"
                >
                  <q-icon
                    :name="prop.node.icon || 'article'"
                    size="20px"
                    class="permission-menu-node-icon"
                  />
                  <div class="permission-menu-node-text">
                    <div class="permission-menu-node-title">{{ t(prop.node.title) }}</div>
                    <div class="permission-menu-node-code">
                      {{ prop.node.name || '-' }}
                      <span v-if="prop.node.path"> · {{ prop.node.path }}</span>
                    </div>
                  </div>
                  <q-badge
                    v-if="buttonSelectionCountForMenu(prop.node.id) > 0"
                    dense
                    color="primary"
                    text-color="white"
                  >
                    {{ buttonSelectionCountForMenu(prop.node.id) }}
                  </q-badge>
                </div>
              </template>
            </q-tree>
            <div v-else class="permission-empty">
              <q-icon name="search_off" />
              <span>没有匹配的菜单</span>
            </div>
          </q-scroll-area>
        </section>

        <section class="permission-panel permission-detail-panel">
          <template v-if="selectedMenu">
            <div class="permission-selected-head">
              <div class="permission-selected-icon">
                <q-icon :name="selectedMenu.icon || 'article'" size="28px" />
              </div>
              <div class="permission-selected-info">
                <div class="permission-selected-title">{{ t(selectedMenu.title) }}</div>
                <div class="permission-selected-meta">
                  <q-badge color="primary" outline>{{ selectedMenu.name }}</q-badge>
                  <q-badge v-if="selectedMenu.path" color="grey-6" outline>
                    {{ selectedMenu.path }}
                  </q-badge>
                  <q-badge v-if="selectedMenu.table_code" color="indigo-5" outline>
                    绑定表 {{ selectedMenu.table_code }}
                  </q-badge>
                </div>
              </div>
              <q-toggle
                :model-value="isSelectedMenuTicked"
                color="primary"
                label="菜单权限"
                @update:model-value="toggleSelectedMenu"
              />
            </div>

            <q-tabs
              v-model="activeDetailTab"
              class="permission-detail-tabs"
              active-color="primary"
              indicator-color="primary"
              align="left"
              narrow-indicator
            >
              <q-tab name="buttons" icon="touch_app" label="按钮权限" />
              <q-tab
                name="data_scope"
                icon="rule"
                label="数据权限"
                :disable="!isSelectedMenuDataScopeCapable"
              />
            </q-tabs>

            <q-tab-panels v-model="activeDetailTab" animated class="permission-tab-panels">
              <q-tab-panel name="buttons" class="q-pa-none permission-tab-panel">
                <div class="permission-button-toolbar">
                  <div>
                    <div class="permission-panel-title">按钮权限</div>
                    <div class="permission-panel-caption">
                      页面按钮 {{ selectedVisibleButtonCount }}/{{ visibleMenuButtons.length }}，接口权限
                      {{ selectedApiPermissionCount }}/{{ apiPermissionButtons.length }}
                    </div>
                  </div>
                  <div class="permission-button-actions">
                    <q-btn
                      flat
                      dense
                      color="primary"
                      icon="done_all"
                      label="全选按钮"
                      :disable="menuButtons.length === 0"
                      @click="selectButtonGroup(menuButtons)"
                    />
                    <q-btn
                      flat
                      dense
                      color="grey-7"
                      icon="remove_done"
                      label="清空"
                      :disable="menuButtons.length === 0"
                      @click="clearButtonGroup(menuButtons)"
                    />
                  </div>
                </div>

                <q-scroll-area class="permission-button-scroll">
                  <div v-if="menuButtons.length === 0" class="permission-empty permission-empty--large">
                    <q-icon name="touch_app" />
                    <span>该菜单没有可分配的按钮权限</span>
                  </div>

                  <div
                    v-for="group in buttonGroups"
                    :key="group.key"
                    class="permission-button-group"
                    :class="{ 'permission-button-group--dependency': group.key === 'api_permission' }"
                  >
                    <div class="permission-button-group-head">
                      <div class="permission-button-group-title">
                        <q-icon :name="group.icon" />
                        <span>{{ group.label }}</span>
                        <q-badge color="primary" outline>
                          {{ buttonSelectionCount(group.buttons) }}/{{ group.buttons.length }}
                        </q-badge>
                      </div>
                      <div>
                        <q-btn
                          flat
                          dense
                          color="primary"
                          label="全选"
                          @click="selectButtonGroup(group.buttons)"
                        />
                        <q-btn
                          flat
                          dense
                          color="grey-7"
                          label="清空"
                          @click="clearButtonGroup(group.buttons)"
                        />
                      </div>
                    </div>

                    <div class="permission-button-list">
                      <div
                        v-for="button in group.buttons"
                        :key="button.id"
                        class="permission-button-item"
                        :class="{
                          'permission-button-item--checked': tickedButtons.includes(button.id),
                        }"
                      >
                        <q-checkbox v-model="tickedButtons" :val="button.id" color="primary" />
                        <div class="permission-button-main">
                          <div class="permission-button-title">
                            <q-icon :name="button.icon || fallbackButtonIcon(button)" />
                            <span>{{ button.name }}</span>
                            <q-badge v-if="isApiPermission(button)" color="grey-7" outline>
                              接口
                              <q-tooltip>不在页面展示，用来控制查询、详情、元数据等后台接口权限</q-tooltip>
                            </q-badge>
                            <q-badge v-if="button.is_disabled" color="orange-7" outline>禁用</q-badge>
                          </div>
                          <div class="permission-button-code">{{ button.code }}</div>
                          <div v-if="button.api_path" class="permission-button-api">
                            {{ button.http_method || 'ANY' }} {{ button.api_path }}
                          </div>
                          <div v-else-if="derivedButtonApi(button)" class="permission-button-api">
                            {{ derivedButtonApi(button) }}
                          </div>
                        </div>
                        <q-badge color="grey-7" outline>
                          {{ positionLabel(button.position) }}
                        </q-badge>
                      </div>
                    </div>
                  </div>
                </q-scroll-area>
              </q-tab-panel>

              <q-tab-panel name="data_scope" class="q-pa-none permission-tab-panel">
                <div class="permission-button-toolbar">
                  <div>
                    <div class="permission-panel-title">数据权限</div>
                    <div class="permission-panel-caption">
                      {{ selectedRoleDataScopeRows.length }} 个绑定，{{ selectedEnabledDataScopeCount }} 已配置
                    </div>
                  </div>
                  <q-badge color="primary" outline>{{ selectedMenu?.table_code }}</q-badge>
                </div>

                <q-scroll-area class="permission-data-scope-scroll">
                  <div v-if="!isSelectedMenuDataScopeCapable" class="permission-empty permission-empty--large">
                    <q-icon name="rule" />
                    <span>当前菜单未绑定数据表</span>
                  </div>
                  <div v-else-if="selectedRoleDataScopeRows.length === 0" class="permission-empty permission-empty--large">
                    <q-icon name="rule_folder" />
                    <span>当前菜单尚未绑定数据权限维度</span>
                  </div>
                  <div v-else class="permission-data-scope-list">
                    <div
                      v-for="row in selectedRoleDataScopeRows"
                      :key="row.key"
                      class="permission-data-scope-item"
                      :class="{ 'permission-data-scope-item--enabled': row.enabled }"
                    >
                      <q-toggle
                        v-model="row.enabled"
                        color="primary"
                        @update:model-value="toggleRoleDataScope(row)"
                      />
                      <div class="permission-data-scope-main">
                        <div class="permission-button-title">
                          <q-icon name="rule" />
                          <span>{{ row.dimension_label }}</span>
                          <q-badge color="grey-7" outline>{{ row.field_code }}</q-badge>
                        </div>
                        <div class="permission-button-code">{{ row.dimension_code }}</div>
                      </div>
                      <div class="permission-data-scope-fields">
                        <sweet-select
                          v-model="row.strategy"
                          emit-value
                          map-options
                          options-dense
                          label="范围策略"
                          :disable="!row.enabled"
                          :options="dataPermissionStrategyOptions"
                          @update:model-value="onRoleStrategyChange(row)"
                        />
                        <scope-value-select
                          v-if="needsRoleScopeValues(row)"
                          v-model="row.scope_values"
                          :max-values="row.strategy === 'user_field' ? 1 : 0"
                          :label="row.strategy === 'user_field' ? '用户字段' : '范围值'"
                          class="permission-scope-value-select"
                          :disable="!row.enabled"
                          :loading="row.loading_options"
                          :options="scopeValueOptions(row)"
                          :free-input="row.strategy !== 'user_field' && row.dimension_source_type !== 'table'"
                          @focus="loadRoleDimensionOptions(row)"
                        />
                      </div>
                    </div>
                  </div>
                </q-scroll-area>
              </q-tab-panel>
            </q-tab-panels>
          </template>

          <div v-else class="permission-empty permission-empty--large">
            <q-icon name="ads_click" />
            <span>选择左侧菜单后配置按钮权限</span>
          </div>
        </section>
      </q-card-section>

      <q-card-actions class="permission-dialog-actions" align="right">
        <q-btn flat label="关闭" color="grey-7" :loading="loading" @click="isOpen = false" />
        <q-btn
          unelevated
          label="保存权限"
          color="primary"
          :loading="loading"
          @click="savePermission"
        />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import type { Role } from 'src/api/services/sys-role'
import type { Menu, MenuButton } from 'src/api/services/sys-menu'
import { useMenuApi } from 'src/api/services/sys-menu'
import { useTableApi } from 'src/api/services/sys-table'
import { SysMenuButtonEventAction, SysMenuButtonPosition, SysMenuButtonPositionMap } from 'src/types/enum'
import { type Query } from 'src/types/global'
import { useI18n } from 'vue-i18n'
import { useLoadingStore } from 'stores/loading'
import { storeToRefs } from 'pinia'
import { isApiPermission, isPageButton } from 'src/utils/menu-button'
import { useQuasar } from 'quasar'
import {
  dataPermissionStrategyOptions,
  type DataPermissionBinding,
  type DataPermissionOption,
  type RoleDataPermission,
  type RoleDataPermissionSaveItem,
  useDataPermissionApi,
} from 'src/api/services/data-permission'
import ScopeValueSelect from 'src/components/DataPermission/ScopeValueSelect.vue'
import SweetSelect from 'src/components/Select/SweetSelect.vue'

type ButtonGroup = {
  key: string
  label: string
  icon: string
  buttons: MenuButton[]
}

type RoleDataScopeRow = {
  key: string
  menu_id: number
  table_code: string
  dimension_code: string
  dimension_label: string
  dimension_source_type: string
  field_code: string
  enabled: boolean
  strategy: string
  scope_values: string[]
  option_items: DataPermissionOption[]
  loading_options: boolean
}

const props = defineProps({
  role: {
    type: Object as () => Role,
    required: true,
  },
  open: {
    type: Boolean,
    required: true,
  },
})

const { t } = useI18n()
const $q = useQuasar()

const emit = defineEmits<{
  (event: 'update:open', value: boolean): void
  (event: 'save', value: {
    roleId: number
    menuIds: number[]
    buttonIds: number[]
    menuButtonMap: { menuId: number; buttonIds: number[] }[]
    dataPermissions: RoleDataPermissionSaveItem[]
  }): void
}>()

const loadingStore = useLoadingStore()
const { loading } = storeToRefs(loadingStore)
const { queryMenu } = useMenuApi()
const tableApi = useTableApi()
const dataPermissionApi = useDataPermissionApi()

const isOpen = computed({
  get: () => props.open,
  set: (val) => emit('update:open', val),
})

const menuTree = ref<Menu[]>([])
const menuKeyword = ref('')
const tickedMenus = ref<number[]>([])
const expandedKeys = ref<number[]>([])
const selectedMenu = ref<Menu | null>(null)
const selectedMenuId = ref<number | null>(null)
const menuButtons = ref<MenuButton[]>([])
const tickedButtons = ref<number[]>([])
const menuButtonSelections = ref<Map<number, number[]>>(new Map())
const activeDetailTab = ref<'buttons' | 'data_scope'>('buttons')
const savedRoleDataScopes = ref<RoleDataPermission[]>([])
const roleDataScopeRows = ref<RoleDataScopeRow[]>([])
const userFieldOptions = ref<DataPermissionOption[]>([])
const dimensionOptionCache = ref<Map<string, DataPermissionOption[]>>(new Map())
const dimensionOptionRequests = new Map<string, Promise<DataPermissionOption[]>>()

const flattenedMenus = computed(() => flattenMenus(menuTree.value))
const totalMenuCount = computed(() => flattenedMenus.value.length)
const selectedMenuCount = computed(() => tickedMenus.value.length)

const filteredMenuTree = computed(() => {
  const keyword = menuKeyword.value.trim().toLowerCase()
  if (!keyword) return menuTree.value
  return filterMenus(menuTree.value, keyword)
})

const isSelectedMenuTicked = computed(() => {
  return !!selectedMenu.value && tickedMenus.value.includes(selectedMenu.value.id)
})

const isSelectedMenuDataScopeCapable = computed(() => {
  return !!selectedMenu.value?.table_code && !selectedMenu.value?.is_hidden
})

const selectedRoleDataScopeRows = computed(() => {
  if (!selectedMenu.value?.id) return []
  return roleDataScopeRows.value.filter((row) => row.menu_id === selectedMenu.value?.id)
})

const selectedEnabledDataScopeCount = computed(() =>
  selectedRoleDataScopeRows.value.filter((row) => row.enabled).length,
)

const selectedButtonCount = computed(() => {
  const ids = new Set<number>()
  menuButtonSelections.value.forEach((buttonIds, menuId) => {
    if (!tickedMenus.value.includes(menuId)) return
    buttonIds.forEach((id) => ids.add(id))
  })
  if (selectedMenu.value?.id && tickedMenus.value.includes(selectedMenu.value.id)) {
    tickedButtons.value.forEach((id) => ids.add(id))
  }
  return ids.size
})

const apiPermissionCount = computed(() =>
  flattenedMenus.value.reduce((total, menu) => {
    return total + (menu.menu_buttons || []).filter(isApiPermission).length
  }, 0),
)

const sortButtons = (buttons: MenuButton[]) => {
  return [...buttons].sort((left, right) => {
    const sequenceDiff = Number(left.sequence || 0) - Number(right.sequence || 0)
    if (sequenceDiff !== 0) return sequenceDiff
    return left.id - right.id
  })
}

const visibleMenuButtons = computed(() =>
  sortButtons(menuButtons.value.filter(isPageButton)),
)
const apiPermissionButtons = computed(() =>
  sortButtons(menuButtons.value.filter(isApiPermission)),
)

const selectedVisibleButtonCount = computed(() =>
  visibleMenuButtons.value.filter((button) => tickedButtons.value.includes(button.id)).length,
)
const selectedApiPermissionCount = computed(() =>
  apiPermissionButtons.value.filter((button) => tickedButtons.value.includes(button.id)).length,
)

const buttonGroups = computed<ButtonGroup[]>(() => {
  const positions = [
    { key: 'top', value: SysMenuButtonPosition.TOP, icon: 'vertical_align_top' },
    { key: 'line', value: SysMenuButtonPosition.LINE, icon: 'table_rows' },
    { key: 'bottom', value: SysMenuButtonPosition.BOTTOM, icon: 'vertical_align_bottom' },
    { key: 'form_top', value: SysMenuButtonPosition.FORM_TOP, icon: 'web_asset' },
    { key: 'form_bottom', value: SysMenuButtonPosition.FORM_BOTTOM, icon: 'browser_updated' },
    { key: 'detail_top', value: SysMenuButtonPosition.DETAIL_TOP, icon: 'article' },
    { key: 'detail_bottom', value: SysMenuButtonPosition.DETAIL_BOTTOM, icon: 'fact_check' },
  ]

  const groups = positions
    .map((position) => ({
      key: position.key,
      label: SysMenuButtonPositionMap[position.value],
      icon: position.icon,
      buttons: visibleMenuButtons.value.filter((button) => button.position === position.value),
    }))
    .filter((group) => group.buttons.length > 0)

  if (apiPermissionButtons.value.length > 0) {
    groups.push({
      key: 'api_permission',
      label: '接口权限',
      icon: 'link',
      buttons: apiPermissionButtons.value,
    })
  }

  return groups
})

watch(
  () => props.open,
  (newValue) => {
    if (newValue) {
      resetState()
      void fetchMenuTree()
    }
  },
)

watch(
  tickedButtons,
  (buttonIds) => {
    if (!selectedMenu.value?.id) return
    menuButtonSelections.value.set(selectedMenu.value.id, [...buttonIds])
    if (buttonIds.length > 0) {
      ensureMenuTicked(selectedMenu.value.id)
    }
  },
  { deep: true },
)

const resetState = () => {
  menuKeyword.value = ''
  tickedMenus.value = []
  expandedKeys.value = []
  selectedMenu.value = null
  selectedMenuId.value = null
  menuButtons.value = []
  tickedButtons.value = []
  menuButtonSelections.value = new Map()
  activeDetailTab.value = 'buttons'
  savedRoleDataScopes.value = []
  roleDataScopeRows.value = []
  userFieldOptions.value = []
  dimensionOptionCache.value = new Map()
  dimensionOptionRequests.clear()
}

const fetchMenuTree = async () => {
  const query: Query = {
    page: 1,
    num: 1000,
    order: {
      field: 'sequence',
    },
    expressions: [
      {
        rules: [
          {
            field: '',
            value: null,
          },
        ],
      },
    ],
    quick_query: {
      keyword: '',
    },
    include_deleted: false,
  }

  const [result, scopeResult] = await Promise.all([
    queryMenu(query),
    dataPermissionApi.getRoleDataPermissions(props.role?.id || 0),
  ])
  await loadUserFieldOptions()
  if (!result.success) return
  savedRoleDataScopes.value = scopeResult.success ? scopeResult.data || [] : []

  menuTree.value = result.data
  const roleMenuIds = props.role?.menus?.map((menu) => menu.id) || []
  tickedMenus.value = roleMenuIds
  initializeRoleButtonSelections()
  expandSelectedMenuPaths(menuTree.value, roleMenuIds)

  const initialMenuId =
    roleMenuIds.find((id) => !!findMenuById(menuTree.value, id)) || firstMenuId(menuTree.value)
  if (initialMenuId) {
    selectedMenuId.value = initialMenuId
    onMenuSelect(initialMenuId)
  }
}

const initializeRoleButtonSelections = () => {
  const selections = new Map<number, number[]>()
  if (props.role?.buttons) {
    props.role.buttons.forEach((button) => {
      const existing = selections.get(button.menu_id) || []
      existing.push(button.id)
      selections.set(button.menu_id, existing)
    })
  }
  menuButtonSelections.value = selections
}

const expandSelectedMenuPaths = (nodes: Menu[], selectedIds: number[]) => {
  const expandSet = new Set<number>()

  const findParentPaths = (menuNodes: Menu[], targetIds: number[], parentIds: number[] = []) => {
    for (const node of menuNodes) {
      const currentPath = [...parentIds, node.id]
      if (targetIds.includes(node.id)) {
        currentPath.forEach((id) => expandSet.add(id))
      }
      if (node.children?.length) {
        findParentPaths(node.children, targetIds, currentPath)
      }
    }
  }

  findParentPaths(nodes, selectedIds)
  expandedKeys.value = Array.from(expandSet)
}

const onMenuSelect = (menuId: number) => {
  if (!menuId) return
  persistCurrentMenuButtons()
  fetchMenuButtons(menuId)
}

const fetchMenuButtons = (menuId: number) => {
  loadingStore.setLoading(true)
  const menu = findMenuById(menuTree.value, menuId)
  if (menu) {
    selectedMenu.value = menu
    menuButtons.value = menu.menu_buttons || []
    tickedButtons.value = menuButtonSelections.value.get(menuId) || []
    if (!isDataScopeCapableMenu(menu)) {
      activeDetailTab.value = 'buttons'
    }
    void ensureRoleDataScopeRows(menu)
  }
  loadingStore.setLoading(false)
}

const persistCurrentMenuButtons = () => {
  if (!selectedMenu.value?.id) return
  menuButtonSelections.value.set(selectedMenu.value.id, [...tickedButtons.value])
}

const handleMenuTickedChange = (menuIds: readonly number[]) => {
  const selectedIds = new Set(menuIds)
  menuButtonSelections.value.forEach((_, menuId) => {
    if (!selectedIds.has(menuId)) {
      menuButtonSelections.value.set(menuId, [])
    }
  })
  roleDataScopeRows.value = roleDataScopeRows.value.map((row) =>
    selectedIds.has(row.menu_id) ? row : { ...row, enabled: false },
  )
  if (selectedMenu.value?.id && !selectedIds.has(selectedMenu.value.id)) {
    tickedButtons.value = []
  }
}

const toggleSelectedMenu = (checked: boolean) => {
  if (!selectedMenu.value?.id) return
  if (checked) {
    ensureMenuTicked(selectedMenu.value.id)
    return
  }
  tickedMenus.value = tickedMenus.value.filter((id) => id !== selectedMenu.value?.id)
  tickedButtons.value = []
  menuButtonSelections.value.set(selectedMenu.value.id, [])
  roleDataScopeRows.value = roleDataScopeRows.value.map((row) =>
    row.menu_id === selectedMenu.value?.id ? { ...row, enabled: false } : row,
  )
}

const ensureMenuTicked = (menuId: number) => {
  if (!tickedMenus.value.includes(menuId)) {
    tickedMenus.value = [...tickedMenus.value, menuId]
  }
}

const selectButtonGroup = (buttons: MenuButton[]) => {
  if (!selectedMenu.value?.id) return
  const ids = new Set(tickedButtons.value)
  buttons.forEach((button) => ids.add(button.id))
  tickedButtons.value = Array.from(ids)
  if (buttons.length > 0) {
    ensureMenuTicked(selectedMenu.value.id)
  }
}

const clearButtonGroup = (buttons: MenuButton[]) => {
  const removeIds = new Set(buttons.map((button) => button.id))
  tickedButtons.value = tickedButtons.value.filter((id) => !removeIds.has(id))
}

const buttonSelectionCount = (buttons: MenuButton[]) => {
  return buttons.filter((button) => tickedButtons.value.includes(button.id)).length
}

const buttonSelectionCountForMenu = (menuId: number) => {
  if (selectedMenu.value?.id === menuId) {
    return tickedButtons.value.length
  }
  return menuButtonSelections.value.get(menuId)?.length || 0
}

const expandAllMenus = () => {
  expandedKeys.value = flattenedMenus.value.map((menu) => menu.id)
}

const collapseAllMenus = () => {
  expandedKeys.value = selectedMenu.value?.id ? [selectedMenu.value.id] : []
}

const flattenMenus = (menus: Menu[]): Menu[] => {
  return menus.flatMap((menu) => [menu, ...flattenMenus(menu.children || [])])
}

const filterMenus = (menus: Menu[], keyword: string): Menu[] => {
  return menus.reduce<Menu[]>((result, menu) => {
    const children = filterMenus(menu.children || [], keyword)
    const searchText = [menu.title, menu.name, menu.path, menu.component, menu.table_code, menu.option]
      .filter(Boolean)
      .join(' ')
      .toLowerCase()
    if (searchText.includes(keyword) || children.length > 0) {
      result.push({ ...menu, children })
    }
    return result
  }, [])
}

const firstMenuId = (menus: Menu[]): number | null => {
  for (const menu of menus) {
    return menu.id
  }
  return null
}

const findMenuById = (menus: Menu[], id: number): Menu | null => {
  for (const menu of menus) {
    if (menu.id === id) {
      return menu
    }
    if (menu.children?.length) {
      const found = findMenuById(menu.children, id)
      if (found) {
        return found
      }
    }
  }
  return null
}

const positionLabel = (position: SysMenuButtonPosition) => {
  return SysMenuButtonPositionMap[position] || '未分组'
}

const fallbackButtonIcon = (button: MenuButton) => {
  if (isApiPermission(button)) return 'link'
  switch (button.position) {
    case SysMenuButtonPosition.LINE:
      return 'edit_note'
    case SysMenuButtonPosition.TOP:
      return 'add_circle'
    case SysMenuButtonPosition.BOTTOM:
      return 'vertical_align_bottom'
    case SysMenuButtonPosition.FORM_TOP:
      return 'web_asset'
    case SysMenuButtonPosition.FORM_BOTTOM:
      return 'browser_updated'
    case SysMenuButtonPosition.DETAIL_TOP:
      return 'article'
    case SysMenuButtonPosition.DETAIL_BOTTOM:
      return 'fact_check'
    default:
      return 'touch_app'
  }
}

const derivedButtonApi = (button: MenuButton) => {
  const action = (button.event_action || '').trim()
  if (action === SysMenuButtonEventAction.DETAIL && !button.api_path) {
    return '打开当前记录详情页'
  }
  return ''
}

const isDataScopeCapableMenu = (menu: Menu) => {
  return !!menu.table_code && !menu.is_hidden
}

const roleDataScopeKey = (menuId: number, dimensionCode: string) => `${menuId}:${dimensionCode}`

const normalizeScopeValues = (values: unknown) => {
  const list = Array.isArray(values) ? values : values === null || values === undefined || values === '' ? [] : [values]
  return list
    .map((item) => String(item).trim())
    .filter(Boolean)
    .filter((item, index, array) => array.indexOf(item) === index)
}

const findSavedRoleScope = (menuId: number, dimensionCode: string) => {
  return savedRoleDataScopes.value.find(
    (scope) => scope.menu_id === menuId && scope.dimension_code === dimensionCode,
  )
}

const ensureRoleDataScopeRows = async (menu: Menu) => {
  if (!isDataScopeCapableMenu(menu)) return
  const existingKeys = new Set(roleDataScopeRows.value.map((row) => row.key))
  if (roleDataScopeRows.value.some((row) => row.menu_id === menu.id)) return
  const result = await dataPermissionApi.getMenuBindings(menu.id)
  const bindings = result.success ? result.data || [] : []
  const rows = bindings
    .filter((binding) => binding.state !== false)
    .map((binding) => roleDataScopeRowFromBinding(menu, binding, findSavedRoleScope(menu.id, binding.dimension_code)))
    .filter((row) => !existingKeys.has(row.key))
  if (rows.length > 0) {
    roleDataScopeRows.value = [...roleDataScopeRows.value, ...rows]
    rows
      .filter((row) => row.enabled && needsRoleScopeValues(row) && row.scope_values.length > 0)
      .forEach((row) => {
        void loadRoleDimensionOptions(row)
      })
  }
}

const roleDataScopeRowFromBinding = (
  menu: Menu,
  binding: DataPermissionBinding,
  saved?: RoleDataPermission,
): RoleDataScopeRow => ({
  key: roleDataScopeKey(menu.id, binding.dimension_code),
  menu_id: menu.id,
  table_code: menu.table_code || binding.table_code,
  dimension_code: binding.dimension_code,
  dimension_label: binding.dimension?.name || binding.dimension_code,
  dimension_source_type: binding.dimension?.source_type || 'none',
  field_code: binding.field_code,
  enabled: !!saved,
  strategy: saved?.strategy || 'specified',
  scope_values: normalizeScopeValues(saved?.scope_values),
  option_items: [],
  loading_options: false,
})

const needsRoleScopeValues = (row: RoleDataScopeRow) => {
  return row.strategy === 'specified' || row.strategy === 'tree' || row.strategy === 'user_field'
}

const toggleRoleDataScope = (row: RoleDataScopeRow) => {
  if (row.enabled) {
    ensureMenuTicked(row.menu_id)
    if (needsRoleScopeValues(row)) {
      void loadRoleDimensionOptions(row)
    }
  }
}

const onRoleStrategyChange = (row: RoleDataScopeRow) => {
  row.scope_values = []
  if (row.strategy === 'user_field') {
    row.option_items = userFieldOptions.value
    return
  }
  if (!needsRoleScopeValues(row)) {
    return
  }
  void loadRoleDimensionOptions(row)
}

const loadRoleDimensionOptions = async (row: RoleDataScopeRow) => {
  if (row.strategy === 'user_field') {
    row.option_items = userFieldOptions.value
    return
  }
  if (row.option_items.length) return
  const cachedOptions = dimensionOptionCache.value.get(row.dimension_code)
  if (cachedOptions) {
    row.option_items = cachedOptions
    return
  }

  row.loading_options = true
  try {
    let request = dimensionOptionRequests.get(row.dimension_code)
    if (!request) {
      request = dataPermissionApi.getDimensionOptions(row.dimension_code).then((result) =>
        result.success ? result.data || [] : [],
      )
      dimensionOptionRequests.set(row.dimension_code, request)
    }
    const options = await request
    dimensionOptionCache.value.set(row.dimension_code, options)
    roleDataScopeRows.value.forEach((item) => {
      if (item.dimension_code === row.dimension_code) {
        item.option_items = options
      }
    })
  } catch {
    row.option_items = []
  } finally {
    dimensionOptionRequests.delete(row.dimension_code)
    roleDataScopeRows.value.forEach((item) => {
      if (item.dimension_code === row.dimension_code) {
        item.loading_options = false
      }
    })
  }
}

const loadUserFieldOptions = async () => {
  const result = await tableApi.queryTableByCode('sys_user')
  const fields = result.success ? result.data?.table_fields || [] : []
  userFieldOptions.value = fields
    .filter((field) => !['password', 'access_tokens'].includes(field.field_code))
    .map((field) => ({
      label: `${field.field_name || field.field_code} (${field.field_code})`,
      value: field.field_code,
    }))
}

const scopeValueOptions = (row: RoleDataScopeRow) => {
  return row.strategy === 'user_field' ? userFieldOptions.value : row.option_items
}

const validateRoleDataScopes = () => {
  for (const row of roleDataScopeRows.value.filter((item) => item.enabled)) {
    if (needsRoleScopeValues(row) && row.scope_values.length === 0) {
      $q.notify({
        type: 'warning',
        position: 'top-right',
        message: row.strategy === 'user_field' ? `${row.dimension_label} 需要选择用户字段` : `${row.dimension_label} 需要范围值`,
      })
      activeDetailTab.value = 'data_scope'
      return false
    }
  }
  return true
}

const buildRoleDataPermissionPayload = () => {
  const selectedMenuIds = new Set(tickedMenus.value)
  const loadedKeys = new Set(roleDataScopeRows.value.map((row) => row.key))
  const payload = roleDataScopeRows.value
    .filter((row) => row.enabled && selectedMenuIds.has(row.menu_id))
    .map<RoleDataPermissionSaveItem>((row) => ({
      menu_id: row.menu_id,
      table_code: row.table_code,
      dimension_code: row.dimension_code,
      strategy: row.strategy,
      scope_values: needsRoleScopeValues(row) ? normalizeScopeValues(row.scope_values) : [],
      state: true,
    }))

  savedRoleDataScopes.value.forEach((scope) => {
    const key = roleDataScopeKey(scope.menu_id, scope.dimension_code)
    if (loadedKeys.has(key) || !selectedMenuIds.has(scope.menu_id)) return
    payload.push({
      menu_id: scope.menu_id,
      table_code: scope.table_code,
      dimension_code: scope.dimension_code,
      strategy: scope.strategy,
      scope_values: normalizeScopeValues(scope.scope_values),
      state: scope.state !== false,
    })
  })

  return payload
}

const savePermission = () => {
  persistCurrentMenuButtons()
  if (!validateRoleDataScopes()) return

  const selectedButtonIds: number[] = []
  const menuButtonMap: { menuId: number; buttonIds: number[] }[] = []

  tickedMenus.value.forEach((menuId) => {
    const buttons = menuButtonSelections.value.get(menuId) || []
    if (buttons.length > 0) {
      selectedButtonIds.push(...buttons)
      menuButtonMap.push({
        menuId,
        buttonIds: buttons,
      })
    }
  })

  emit('save', {
    roleId: props.role?.id,
    menuIds: tickedMenus.value,
    buttonIds: selectedButtonIds,
    menuButtonMap,
    dataPermissions: buildRoleDataPermissionPayload(),
  })
  isOpen.value = false
}

onMounted(() => {
  if (props.open) {
    void fetchMenuTree()
  }
})
</script>

<style scoped lang="scss">
.permission-dialog-card {
  width: 1280px;
  max-width: 94vw;
  height: 82vh;
  display: flex;
  flex-direction: column;
  border-radius: 8px;
  overflow: hidden;
}

.permission-dialog-header,
.permission-dialog-actions {
  flex-shrink: 0;
  background: #f7f8ff;
  border-bottom: 1px solid #e5e7f4;
}

.permission-dialog-header {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px 20px;
}

.permission-dialog-actions {
  border-top: 1px solid #e5e7f4;
  border-bottom: 0;
  padding: 12px 20px;
}

.permission-summary {
  display: flex;
  gap: 8px;
  align-items: center;
}

.permission-dialog-body {
  flex: 1;
  min-height: 0;
  padding: 16px;
  background: #f5f7fb;
}

.permission-layout,
.permission-dialog-body {
  display: flex;
  gap: 16px;
}

.permission-panel {
  min-height: 0;
  background: #fff;
  border: 1px solid #e2e7f2;
  border-radius: 8px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.permission-menu-panel {
  width: 420px;
  flex-shrink: 0;
}

.permission-detail-panel {
  flex: 1;
  min-width: 0;
}

.permission-panel-head,
.permission-selected-head,
.permission-button-toolbar,
.permission-button-group-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.permission-panel-head {
  padding: 16px;
  border-bottom: 1px solid #e8ecf5;
}

.permission-panel-title {
  font-size: 16px;
  font-weight: 700;
  color: #111827;
}

.permission-panel-caption {
  margin-top: 4px;
  font-size: 12px;
  color: #718096;
}

.permission-menu-tools {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  border-bottom: 1px solid #e8ecf5;
}

.permission-menu-search {
  flex: 1;
}

.permission-menu-scroll,
.permission-button-scroll,
.permission-data-scope-scroll {
  flex: 1;
  min-height: 0;
}

.permission-menu-scroll {
  height: 100%;
  padding: 8px 10px 12px;
}

.permission-button-scroll,
.permission-data-scope-scroll {
  height: 100%;
}

.permission-detail-tabs {
  flex-shrink: 0;
  padding: 0 14px;
  border-bottom: 1px solid #e8ecf5;
  background: #fff;
}

.permission-tab-panels {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: #fff;
}

.permission-tab-panel {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 0;
}

.permission-menu-node {
  width: 100%;
  min-height: 42px;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 8px;
  border-radius: 6px;
}

.permission-menu-node--active {
  background: rgba($primary, 0.08);
}

.permission-menu-node-icon {
  color: $primary;
}

.permission-menu-node-text {
  min-width: 0;
  flex: 1;
}

.permission-menu-node-title {
  font-weight: 600;
  color: #1f2937;
  line-height: 18px;
}

.permission-menu-node-code,
.permission-button-code,
.permission-button-api {
  font-size: 12px;
  color: #718096;
  line-height: 18px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.permission-selected-head {
  flex-shrink: 0;
  padding: 18px 20px;
  border-bottom: 1px solid #e8ecf5;
  background: #fbfcff;
}

.permission-selected-icon {
  width: 54px;
  height: 54px;
  border-radius: 8px;
  display: grid;
  place-items: center;
  color: #fff;
  background: $primary;
  flex-shrink: 0;
}

.permission-selected-info {
  min-width: 0;
  flex: 1;
}

.permission-selected-title {
  font-size: 20px;
  font-weight: 800;
  color: #111827;
}

.permission-selected-meta {
  margin-top: 8px;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.permission-button-toolbar {
  flex-shrink: 0;
  padding: 14px 20px;
  border-bottom: 1px solid #e8ecf5;
}

.permission-button-actions {
  display: flex;
  gap: 8px;
}

.permission-button-group {
  padding: 16px 20px 4px;
}

.permission-button-group + .permission-button-group {
  border-top: 1px solid #edf1f7;
}

.permission-button-group--dependency {
  background: #fbfcff;
}

.permission-button-group-head {
  margin-bottom: 10px;
}

.permission-button-group-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 700;
  color: #1f2937;
}

.permission-button-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 10px;
}

.permission-button-item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  min-width: 0;
  padding: 12px;
  border: 1px solid #e4e9f3;
  border-radius: 8px;
  background: #fff;
  transition:
    border-color 0.2s ease,
    background 0.2s ease,
    box-shadow 0.2s ease;
}

.permission-button-item--checked {
  border-color: rgba($primary, 0.6);
  background: rgba($primary, 0.04);
  box-shadow: 0 4px 14px rgba($primary, 0.08);
}

.permission-button-main {
  flex: 1;
  min-width: 0;
}

.permission-button-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 700;
  color: #111827;
}

.permission-data-scope-list {
  display: grid;
  gap: 8px;
  padding: 12px 16px;
}

.permission-data-scope-item {
  display: grid;
  grid-template-columns: auto minmax(180px, 1fr) minmax(360px, 0.95fr);
  gap: 10px;
  align-items: center;
  min-height: 64px;
  padding: 8px 10px;
  border: 1px solid #e4e9f3;
  border-radius: 8px;
  background: #fff;
}

.permission-data-scope-item--enabled {
  border-color: rgba($primary, 0.55);
  background: rgba($primary, 0.04);
}

.permission-data-scope-main {
  min-width: 0;
}

.permission-data-scope-fields {
  display: grid;
  grid-template-columns: minmax(140px, 0.65fr) minmax(180px, 1fr);
  gap: 8px;
}

@media (max-width: 1120px) {
  .permission-data-scope-item {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .permission-data-scope-fields {
    grid-column: 1 / -1;
    grid-template-columns: 1fr;
  }
}

.permission-empty {
  min-height: 160px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: #8a94a6;
  font-size: 14px;
}

.permission-empty .q-icon {
  font-size: 38px;
}

.permission-empty--large {
  height: 100%;
  min-height: 320px;
}

:deep(.q-tree__node-header) {
  padding: 2px 0;
}

:deep(.q-tree__tickbox) {
  color: $primary;
}

:deep(.q-scrollarea__content) {
  min-width: 100%;
}
</style>
