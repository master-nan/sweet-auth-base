<template>
  <q-dialog v-model="isOpen" persistent no-shake @hide="resetState">
    <q-card class="data-permission-dialog-card">
      <q-card-section class="data-permission-dialog-header">
        <div>
          <div class="data-permission-dialog-title">数据权限 - {{ user?.user_name }}</div>
          <div class="data-permission-dialog-subtitle">配置当前用户相对角色权限的覆盖范围</div>
        </div>
        <q-space />
        <q-badge color="primary" class="data-permission-dialog-badge">
          {{ enabledPointCount }} 覆盖
        </q-badge>
        <q-btn flat round dense icon="close" :disable="busy" @click="isOpen = false">
          <q-tooltip>关闭</q-tooltip>
        </q-btn>
      </q-card-section>
      <q-separator />

      <q-card-section class="data-permission-dialog-body">
        <div class="data-permission-filter">
          <q-select
            v-model="selectedMenuId"
            dense
            outlined
            clearable
            emit-value
            map-options
            label="菜单"
            :options="menuOptions"
          />
          <q-input v-model="keyword" dense outlined clearable placeholder="搜索维度 / 字段 / 菜单">
            <template #prepend>
              <q-icon name="search" />
            </template>
          </q-input>
        </div>

        <q-scroll-area class="data-permission-scroll">
          <q-list v-if="filteredPoints.length" separator class="data-permission-list">
            <q-item v-for="point in filteredPoints" :key="point.key" class="data-permission-item">
              <q-item-section avatar class="data-permission-item-icon">
                <q-toggle v-model="point.enabled" color="primary" />
              </q-item-section>
              <q-item-section class="data-permission-item-main">
                <q-item-label class="data-permission-item-title">
                  {{ displayMenuTitle(point.menu) }}
                </q-item-label>
                <q-item-label caption class="data-permission-item-code">
                  {{ point.menu.table_code }} · {{ point.binding.dimension?.name || point.binding.dimension_code }}
                  · {{ point.binding.field_code }}
                </q-item-label>
              </q-item-section>
              <q-item-section side class="data-permission-item-fields">
                <q-select
                  v-model="point.override_mode"
                  dense
                  outlined
                  emit-value
                  map-options
                  label="覆盖模式"
                  :disable="busy || !point.enabled"
                  :options="dataPermissionOverrideModeOptions"
                  @update:model-value="normalizeDenyPoint(point)"
                />
                <q-select
                  v-model="point.strategy"
                  dense
                  outlined
                  emit-value
                  map-options
                  label="范围策略"
                  :disable="busy || !point.enabled || point.override_mode === 'deny'"
                  :options="dataPermissionStrategyOptions"
                  @update:model-value="onStrategyChange(point)"
                />
                <q-select
                  v-if="needsValues(point)"
                  v-model="point.scope_values"
                  dense
                  outlined
                  multiple
                  use-input
                  use-chips
                  new-value-mode="add-unique"
                  emit-value
                  map-options
                  label="范围值"
                  :disable="busy || !point.enabled"
                  :loading="point.loading_options"
                  :options="point.option_items"
                  @focus="loadDimensionOptions(point)"
                />
                <q-input
                  v-model="point.expire_at"
                  dense
                  outlined
                  clearable
                  label="过期时间"
                  placeholder="YYYY-MM-DD HH:mm:ss"
                  :disable="busy || !point.enabled"
                />
              </q-item-section>
            </q-item>
          </q-list>

          <div v-else class="data-permission-empty">
            <q-icon name="rule" size="34px" />
            <span>暂无可覆盖的数据权限点</span>
          </div>
        </q-scroll-area>
      </q-card-section>

      <q-separator />
      <q-card-actions align="right" class="data-permission-dialog-actions">
        <q-btn flat color="grey-7" label="关闭" :disable="busy" @click="isOpen = false" />
        <q-btn color="primary" unelevated icon="save" label="保存" :loading="busy" @click="save" />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useQuasar } from 'quasar'
import { useI18n } from 'vue-i18n'
import type { Menu } from 'src/api/services/sys-menu'
import { useMenuApi } from 'src/api/services/sys-menu'
import type { User } from 'src/api/services/sys-user'
import { useLoadingStore } from 'src/stores/loading'
import {
  dataPermissionOverrideModeOptions,
  dataPermissionStrategyOptions,
  type DataPermissionBinding,
  type DataPermissionOption,
  type UserDataPermissionOverride,
  type UserDataPermissionOverrideSaveItem,
  useDataPermissionApi,
} from 'src/api/services/data-permission'

const props = defineProps<{
  open: boolean
  user: User | null
}>()

const emit = defineEmits<{
  (event: 'update:open', value: boolean): void
  (event: 'saved'): void
}>()

type PermissionPoint = {
  key: string
  menu: Menu
  binding: DataPermissionBinding
  enabled: boolean
  strategy: string
  scope_values: string[]
  override_mode: string
  expire_at: string
  option_items: DataPermissionOption[]
  loading_options: boolean
}

const $q = useQuasar()
const { t } = useI18n()
const menuApi = useMenuApi()
const dataPermissionApi = useDataPermissionApi()
const loadingStore = useLoadingStore()
const { loading } = storeToRefs(loadingStore)

const saving = ref(false)
const keyword = ref('')
const selectedMenuId = ref<number | null>(null)
const menuOptions = ref<Array<{ label: string; value: number }>>([])
const points = ref<PermissionPoint[]>([])
const busy = computed(() => loading.value || saving.value)
const enabledPointCount = computed(() => points.value.filter((point) => point.enabled).length)

const isOpen = computed({
  get: () => props.open,
  set: (value) => emit('update:open', value),
})

const filteredPoints = computed(() => {
  const normalizedKeyword = keyword.value.trim().toLowerCase()
  return points.value.filter((point) => {
    if (selectedMenuId.value && point.menu.id !== selectedMenuId.value) return false
    if (!normalizedKeyword) return true
    return [
      displayMenuTitle(point.menu),
      point.menu.name,
      point.menu.table_code,
      point.binding.dimension?.name,
      point.binding.dimension_code,
      point.binding.field_code,
    ]
      .filter(Boolean)
      .join(' ')
      .toLowerCase()
      .includes(normalizedKeyword)
  })
})

const displayMenuTitle = (menu: Menu) => {
  const title = menu.title || menu.name
  return title.startsWith('router.') ? t(title) : title
}

const flattenMenus = (menus: Menu[]): Menu[] =>
  menus.flatMap((menu) => [menu, ...flattenMenus(menu.children || [])])

const isLowCodeMenu = (menu: Menu) => menu.page_type === 'low_code' && !!menu.table_code && !menu.is_hidden

const pointKey = (menuId: number, dimensionCode: string) => `${menuId}:${dimensionCode}`

const resetState = () => {
  saving.value = false
  keyword.value = ''
  selectedMenuId.value = null
  menuOptions.value = []
  points.value = []
}

const load = async () => {
  if (!props.user?.id) return
  saving.value = true
  try {
    const [menusResult, overridesResult] = await Promise.all([
      menuApi.queryUserMenus(props.user.id),
      dataPermissionApi.getUserDataPermissionOverrides(props.user.id),
    ])
    const userMenus = menusResult.success ? flattenMenus(menusResult.data || []).filter(isLowCodeMenu) : []
    const overrides = overridesResult.success ? overridesResult.data || [] : []
    const overrideMap = new Map(overrides.map((item) => [pointKey(item.menu_id, item.dimension_code), item]))

    menuOptions.value = userMenus.map((menu) => ({
      label: `${displayMenuTitle(menu)} (${menu.table_code})`,
      value: menu.id,
    }))

    const bindingGroups = await Promise.all(
      userMenus.map(async (menu) => {
        const result = await dataPermissionApi.getMenuBindings(menu.id)
        return {
          menu,
          bindings: result.success ? result.data || [] : [],
        }
      }),
    )

    points.value = bindingGroups.flatMap(({ menu, bindings }) =>
      bindings
        .filter((binding) => binding.state !== false)
        .map((binding) => pointFromBinding(menu, binding, overrideMap.get(pointKey(menu.id, binding.dimension_code)))),
    )
  } finally {
    saving.value = false
  }
}

const pointFromBinding = (
  menu: Menu,
  binding: DataPermissionBinding,
  override?: UserDataPermissionOverride,
): PermissionPoint => ({
  key: pointKey(menu.id, binding.dimension_code),
  menu,
  binding,
  enabled: !!override,
  strategy: override?.strategy || 'specified',
  scope_values: override?.scope_values || [],
  override_mode: override?.override_mode || 'replace',
  expire_at: override?.expire_at ? String(override.expire_at).replace('T', ' ').slice(0, 19) : '',
  option_items: [],
  loading_options: false,
})

const needsValues = (point: PermissionPoint) => {
  return point.override_mode !== 'deny' && (point.strategy === 'specified' || point.strategy === 'tree')
}

const normalizeDenyPoint = (point: PermissionPoint) => {
  if (point.override_mode === 'deny') {
    point.strategy = 'none'
    point.scope_values = []
  }
}

const onStrategyChange = (point: PermissionPoint) => {
  if (!needsValues(point)) {
    point.scope_values = []
    return
  }
  void loadDimensionOptions(point)
}

const loadDimensionOptions = async (point: PermissionPoint) => {
  if (point.option_items.length || point.loading_options) return
  point.loading_options = true
  try {
    const result = await dataPermissionApi.getDimensionOptions(point.binding.dimension_code)
    point.option_items = result.success ? result.data || [] : []
  } finally {
    point.loading_options = false
  }
}

const validate = () => {
  for (const point of points.value.filter((item) => item.enabled)) {
    if (needsValues(point) && point.scope_values.length === 0) {
      $q.notify({
        type: 'warning',
        position: 'top-right',
        message: `${displayMenuTitle(point.menu)} / ${point.binding.dimension_code} 需要范围值`,
      })
      return false
    }
  }
  return true
}

const save = async () => {
  if (!props.user?.id || !validate()) return
  saving.value = true
  try {
    const overrides = points.value
      .filter((point) => point.enabled)
      .map<UserDataPermissionOverrideSaveItem>((point) => ({
        menu_id: point.menu.id,
        ...(point.menu.table_code ? { table_code: point.menu.table_code } : {}),
        dimension_code: point.binding.dimension_code,
        strategy: point.override_mode === 'deny' ? 'none' : point.strategy,
        scope_values: needsValues(point) ? point.scope_values : [],
        override_mode: point.override_mode,
        ...(point.expire_at ? { expire_at: point.expire_at } : {}),
        state: true,
      }))
    const result = await dataPermissionApi.saveUserDataPermissionOverrides(props.user.id, overrides)
    if (result.success) {
      emit('saved')
      isOpen.value = false
    }
  } finally {
    saving.value = false
  }
}

watch(
  () => props.open,
  (open) => {
    if (open) void load()
  },
)
</script>

<style scoped lang="scss">
.data-permission-dialog-card {
  width: 1180px;
  max-width: 94vw;
  height: 82vh;
  display: flex;
  flex-direction: column;
  border-radius: 8px;
  overflow: hidden;
  background: #fff;
}

.data-permission-dialog-header,
.data-permission-dialog-actions {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 12px;
  background: #f7f8ff;
}

.data-permission-dialog-header {
  padding: 18px 22px;
  border-bottom: 1px solid #e6ebf5;
}

.data-permission-dialog-title {
  color: #172033;
  font-size: 20px;
  line-height: 1.2;
  font-weight: 800;
}

.data-permission-dialog-subtitle {
  margin-top: 5px;
  color: #748098;
  font-size: 13px;
}

.data-permission-dialog-badge {
  padding: 6px 10px;
  border-radius: 8px;
  font-weight: 700;
}

.data-permission-dialog-body {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 0;
  background: #f5f7fc;
}

.data-permission-filter {
  display: grid;
  grid-template-columns: minmax(260px, 360px) minmax(260px, 1fr);
  gap: 12px;
  padding: 14px 18px;
  border-bottom: 1px solid #e6ebf5;
  background: #fff;
}

.data-permission-scroll {
  flex: 1;
  min-height: 0;
}

.data-permission-list {
  padding: 14px;
}

.data-permission-item {
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
  padding: 14px;
  border: 1px solid #dde5f3;
  border-radius: 8px;
  background: #fff;
}

.data-permission-item-icon {
  width: 54px;
  min-width: 54px;
  align-items: center;
}

.data-permission-item-main {
  min-width: 180px;
}

.data-permission-item-title {
  color: #172033;
  font-size: 15px;
  font-weight: 800;
}

.data-permission-item-code {
  margin-top: 4px;
  color: #7a869f;
  word-break: break-all;
}

.data-permission-item-fields {
  width: min(680px, 62%);
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  align-items: start;
}

.data-permission-empty {
  height: 100%;
  min-height: 280px;
  display: flex;
  gap: 10px;
  align-items: center;
  justify-content: center;
  color: #7a869f;
}

.data-permission-dialog-actions {
  padding: 14px 22px;
  border-top: 1px solid #e6ebf5;
}

@media (max-width: 1023px) {
  .data-permission-filter {
    grid-template-columns: 1fr;
  }

  .data-permission-item {
    flex-wrap: wrap;
  }

  .data-permission-item-fields {
    width: 100%;
    grid-template-columns: 1fr;
  }
}
</style>
