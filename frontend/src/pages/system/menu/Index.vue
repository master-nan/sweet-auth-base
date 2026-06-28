<template>
  <base-content class="q-pa-sm menu-page">
    <master-detail-page
      class="menu-master-detail"
      :mode="SysMasterDetailMode.TABLE"
      master-title="菜单结构"
      :master-subtitle="menuSummaryText"
      :detail-title="selectedMenuTitle"
      :detail-subtitle="selectedMenuSubtitle"
      master-width="minmax(620px, 44%)"
      min-width="1180px"
      min-height="calc(100vh - 150px)"
    >
      <template #master-actions>
        <q-btn
          v-for="btn in master_top_buttons"
          :key="btn.id"
          v-bind="menuButtonDisplayProps(btn)"
          unelevated
          :color="btn.color || 'primary'"
          @click="handleButtonClick(btn)"
          :disable="loading"
        >
          <q-tooltip>{{ btn.name }}</q-tooltip>
        </q-btn>
        <q-btn round outline color="primary" icon="refresh" @click="fetchMenus" :loading="loading">
          <q-tooltip>刷新菜单</q-tooltip>
        </q-btn>
      </template>

      <template #master-toolbar>
        <div class="menu-master-toolbar">
          <q-input
            class="full-width"
            outlined
            v-model="searchText"
            placeholder="搜索菜单名称 / 编码 / 路径 / 组件"
            clearable
          >
            <template v-slot:append>
              <q-icon name="search" />
            </template>
          </q-input>
        </div>
      </template>

      <template #master-content>
        <tree-table
          class="fit sticky-header-table menu-tree-table"
          :data="filteredMenuData"
          :columns="columns"
          :selected-row-id="selectedMenu?.id ?? null"
          :loading="loading"
          :dark="$q.dark.isActive"
          :indent="!is_drawer_mini"
          bordered
          separator="cell"
          flat
          @node-selected="handleNodeSelected"
        >
          <template v-slot:body-cell-icon="props">
            <q-icon :name="(props as any).row.icon || 'folder'" size="sm" class="q-mr-xs" />
          </template>
          <template v-slot:body-cell-actions="props">
            <div class="q-gutter-xs text-center">
              <q-btn
                v-for="btn in master_line_buttons"
                :key="btn.id"
                v-bind="menuButtonDisplayProps(btn)"
                flat
                dense
                :color="btn.color || 'primary'"
                @click.stop="handleButtonClick(btn, props.row)"
                :loading="loading"
                size="sm"
              >
                <q-tooltip>{{ btn.name }}</q-tooltip>
              </q-btn>
            </div>
          </template>
        </tree-table>
      </template>

      <template #detail-context>
        <div v-if="selectedMenu" class="menu-detail-context">
          <div class="menu-icon-tile">
            <q-icon :name="selectedMenu.icon || 'menu'" />
          </div>
          <div class="menu-detail-main">
            <div class="menu-detail-title">{{ t(selectedMenu.title) }}</div>
            <div class="menu-detail-meta">
              <q-chip dense square color="primary" text-color="white">
                {{ selectedMenu.name }}
              </q-chip>
              <span v-if="selectedMenu.path">{{ selectedMenu.path }}</span>
              <span v-if="selectedMenu.component">{{ selectedMenu.component }}</span>
            </div>
          </div>
        </div>
        <div v-else class="menu-detail-context menu-detail-context--empty">
          <q-icon name="menu_open" size="28px" />
          <div>
            <div class="menu-detail-title">请选择菜单</div>
            <div class="menu-detail-meta">左侧选择一个菜单后维护它的按钮子表</div>
          </div>
        </div>
      </template>

      <template #detail-toolbar>
        <div v-if="selectedMenu" class="menu-detail-toolbar">
          <q-tabs
            v-model="activeTab"
            class="menu-detail-tabs"
            active-color="primary"
            indicator-color="primary"
            align="left"
            narrow-indicator
          >
            <q-tab name="buttons" label="按钮管理" icon="touch_app" />
            <q-tab name="preview" label="菜单预览" icon="visibility" />
          </q-tabs>
          <q-space />
          <div v-if="activeTab === 'buttons'" class="menu-detail-actions">
            <q-btn
              v-for="btn in detailToolbarButtons"
              :key="btn.id"
              v-bind="menuButtonDisplayProps(btn)"
              unelevated
              :color="btn.color || 'primary'"
              @click="handleButtonClick(btn)"
            />
          </div>
        </div>
      </template>

      <template #detail-content>
        <q-tab-panels v-if="selectedMenu" v-model="activeTab" animated class="menu-tab-panels">
          <q-tab-panel name="buttons" class="q-pa-none">
            <q-table
              class="fit sticky-header-table menu-button-table"
              :rows="menuButtons"
              :columns="buttonColumns"
              row-key="id"
              bordered
              flat
              :dark="$q.dark.isActive"
              :loading="loading"
              :pagination="{ rowsPerPage: 0 }"
              hide-pagination
            >
              <template v-slot:body-cell-actions="props">
                <q-td :props="props" class="q-gutter-xs">
                  <q-btn
                    v-for="btn in detailRowButtons"
                    :key="btn.id"
                    flat
                    v-bind="menuButtonDisplayProps(btn)"
                    :color="btn.color || 'primary'"
                    size="sm"
                    @click="handleButtonClick(btn, props.row)"
                  >
                    <q-tooltip>{{ btn.name }}</q-tooltip>
                  </q-btn>
                </q-td>
              </template>
              <template v-slot:no-data>
                <div class="full-width row flex-center text-grey-7 q-gutter-sm q-pa-lg">
                  <q-icon name="touch_app" size="28px" />
                  <span>当前菜单还没有配置按钮</span>
                </div>
              </template>
            </q-table>
          </q-tab-panel>

          <q-tab-panel name="preview" class="q-pa-none">
            <div class="preview-container">
              <div class="preview-sidebar">
                <div class="preview-header">应用导航</div>
                <q-scroll-area class="preview-scroll">
                  <q-list bordered separator>
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
                        <q-list padding>
                          <q-item
                            v-for="subMenu in menu.children"
                            :key="subMenu.id"
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
          </q-tab-panel>
        </q-tab-panels>

        <div v-else class="menu-empty-state">
          <q-icon name="account_tree" size="76px" />
          <div class="menu-empty-title">选择左侧菜单</div>
          <div class="menu-empty-desc">菜单是主表，按钮是当前菜单下的子表。</div>
        </div>
      </template>
    </master-detail-page>

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
      :title="isButtonEdit ? '编辑按钮' : '新增按钮'"
      :fields="buttonFields"
      :submit-btn-text="isButtonEdit ? '保存' : '创建'"
      table-code="sys_menu_button"
      @submit="handleButtonFormSubmit"
    />
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'system_menu' })
import { ref, computed, onMounted } from 'vue'
import { useQuasar } from 'quasar'
import { useRoute, useRouter } from 'vue-router'
import BaseContent from 'components/BaseContent/BaseContent.vue'
import TreeTable from 'components/TreeTable/TreeTable.vue'
import MasterDetailPage from 'src/components/MasterDetail/MasterDetailPage.vue'
import MenuRoutePreview from 'src/components/MenuPreview/MenuRoutePreview.vue'
import DynamicFormDialog from 'src/components/FormDialog/DynamicFormDialog.vue'
import { useLoadingStore } from 'src/stores/loading'
import { useAppStore } from 'src/stores/app'
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
  SysMasterDetailMode,
  SysMenuButtonEventAction,
  SysMenuButtonEventActionMap,
  SysMenuButtonPosition,
  SysTableFieldInputType,
  SysTableFieldType,
} from 'src/types/enum'
import { type QTableProps } from 'quasar'
import type { Query, TableColumn } from 'src/types/global'
import { useI18n } from 'vue-i18n'
import { useDictStore } from 'src/stores/dict'
import { buildTableColumns, buildRelationLookups } from 'src/utils/column-format'
import { useMasterDetailPageButtons } from 'src/composables/page-buttons'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'
import { useConfirmDialog } from 'src/composables/confirm-dialog'

const { t } = useI18n()

// 全局状态
const loadingStore = useLoadingStore()
const { loading } = storeToRefs(loadingStore)
const appStore = useAppStore()
const { is_drawer_mini } = storeToRefs(appStore)

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
const activeTab = ref('buttons')

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
  master_has_line_buttons,
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

const findMenuByName = (menus: Menu[], name: string): Menu | undefined => {
  for (const menu of menus) {
    if (menu.name === name) return menu
    const found = findMenuByName(menu.children || [], name)
    if (found) return found
  }
  return undefined
}

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

const menuSummaryText = computed(() => {
  const total = countMenus(menuData.value)
  const visible = countMenus(filteredMenuData.value)
  return searchText.value ? `${visible} / ${total} 个菜单` : `${total} 个菜单`
})

const selectedMenuTitle = computed(() =>
  selectedMenu.value ? t(selectedMenu.value.title) : '按钮子表',
)

const selectedMenuSubtitle = computed(() => {
  if (!selectedMenu.value) return '选择一个菜单后维护按钮配置'
  return selectedMenu.value.path || selectedMenu.value.component || selectedMenu.value.name
})

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
const isButtonEdit = computed(() => 'id' in buttonEditData.value && Boolean(buttonEditData.value.id))

const normalizeBooleanValue = (value: unknown, fallback = false): boolean => {
  if (value === undefined || value === null || value === '') return fallback
  if (typeof value === 'boolean') return value
  if (typeof value === 'number') return value !== 0
  const normalized = String(value).trim().toLowerCase()
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

// 保存查询到的表字段，供表单使用
const tableFields = ref<TableField[]>([])

// 表格列定义 - 修改这部分
const columns = ref<TableColumn[]>([])

const table_fields_advanced = ref<TableField[]>([])
const visibleColumns = ref<string[]>([])

const buttonColumns = ref<QTableProps['columns']>([])

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
          field_name: '按钮位置',
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
          field_name: '事件动作',
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

  // 并行加载字典和关联表查找映射
  const [, btnRelationLookups, menuRelationLookups] = await Promise.all([
    dictStore.loadDicts([...new Set(allDictCodes)]),
    btnFields ? buildRelationLookups(btnFields) : Promise.resolve({}),
    menuFields_ ? buildRelationLookups(menuFields_) : Promise.resolve({}),
  ])

  // 用 buildTableColumns 从元数据自动构建按钮列
  if (btnFields) {
    const { columns: btnCols } = buildTableColumns(btnFields, {
      getDictLabel: dictStore.getDictLabel,
      relationLookups: btnRelationLookups,
    })
    buttonColumns.value = detailRowButtons.value.length
      ? btnCols
      : btnCols.filter((c) => c.name !== 'actions')
  }

  // 处理菜单字段元数据
  if (menuFields_) {
    tableFields.value = menuFields_

    const { columns: cols, advancedFields } = buildTableColumns(menuFields_, {
      getDictLabel: dictStore.getDictLabel,
      relationLookups: menuRelationLookups,
    })
    columns.value = master_has_line_buttons.value ? cols : cols.filter((c) => c.name !== 'actions')
    table_fields_advanced.value = advancedFields
    visibleColumns.value = columns.value.map((c) => c.name)
    menuFields.value = menuFields_.filter((field) => field.is_insert_show || field.is_update_show)
  }
}

// 获取菜单列表
const fetchMenus = async () => {
  const res = await menuApi.queryMenu(query.value)
  if (res.success) {
    menuData.value = Array.isArray(res.data) ? res.data : []
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
      const matches =
        node.title.toLowerCase().includes(searchLower) ||
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

// 获取按钮列表
const fetchMenuButtons = async (menuId: number) => {
  const res = await menuApi.queryMenuButtons(menuId)
  if (res.success) {
    menuButtons.value = res.data
  }
}

// 选择菜单节点
const handleNodeSelected = (menu: Menu) => {
  selectedMenu.value = menu
  fetchMenuButtons(menu.id)
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
    is_button: true,
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
  if (!Object.values(SysMenuButtonEventAction).includes(button.event_action as SysMenuButtonEventAction)) {
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
  const data = formPayload.isEdit && formPayload.id
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
.menu-master-toolbar {
  padding: 12px 14px;
  border-bottom: 1px solid #e3e8f2;
  background: #fbfcff;
}

.menu-tree-table,
.menu-button-table {
  :deep(.q-table__top) {
    padding: 0;
  }

  :deep(th) {
    height: 48px;
    background: #fbfcff;
    color: #172033;
    font-weight: 800;
  }

  :deep(td) {
    height: 48px;
  }
}

.menu-detail-context {
  min-height: 78px;
  display: flex;
  align-items: center;
  gap: 14px;
}

.menu-detail-context--empty {
  color: #657189;
}

.menu-icon-tile {
  width: 48px;
  height: 48px;
  flex: 0 0 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  background: linear-gradient(135deg, #4f6fe8, #7664f6);
  color: #fff;
  font-size: 24px;
  box-shadow: 0 10px 24px rgba(79, 111, 232, 0.25);
}

.menu-detail-main {
  min-width: 0;
  flex: 1;
}

.menu-detail-title {
  overflow: hidden;
  color: #172033;
  font-size: 18px;
  font-weight: 800;
  line-height: 1.3;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.menu-detail-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 7px;
  overflow: hidden;
  color: #657189;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  white-space: nowrap;
}

.menu-detail-meta span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.menu-detail-toolbar {
  min-height: 56px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 14px;
  border-bottom: 1px solid #e3e8f2;
  background: #fbfcff;
}

.menu-detail-tabs {
  min-width: 260px;
}

.menu-detail-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.menu-tab-panels {
  height: 100%;
  background: #fff;
}

.menu-tab-panels :deep(.q-panel),
.menu-tab-panels :deep(.q-tab-panel) {
  height: 100%;
  min-height: 0;
}

.menu-empty-state {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #7b8498;
  text-align: center;
}

.menu-empty-title {
  margin-top: 14px;
  color: #172033;
  font-size: 18px;
  font-weight: 800;
}

.menu-empty-desc {
  margin-top: 6px;
  font-size: 13px;
}

.preview-container {
  display: flex;
  flex-direction: row;
  height: 100%;
  min-height: 0;
  border: 1px solid #e3e8f2;
  overflow: hidden;
}

.preview-sidebar {
  width: 280px;
  border-right: 1px solid #e3e8f2;
  background-color: #fbfcff;
  display: flex;
  flex-direction: column;
}

.preview-header {
  padding: 12px 16px;
  color: #172033;
  font-weight: 800;
  background-color: #fff;
  border-bottom: 1px solid #e3e8f2;
}

.preview-scroll {
  flex: 1;
  min-height: 0;
}

.preview-content {
  flex: 1;
  background-color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

</style>
