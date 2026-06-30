<template>
  <q-dialog v-model="isOpen" persistent no-shake @hide="resetState">
    <q-card class="data-permission-dialog-card">
      <q-card-section class="data-permission-dialog-header">
        <div>
          <div class="data-permission-dialog-title">数据权限 - {{ user?.user_name }}</div>
          <div class="data-permission-dialog-subtitle">基础归属用于用户自己的公司/部门等范围，特殊授权只处理例外</div>
        </div>
        <q-space />
        <q-badge color="primary" class="data-permission-dialog-badge">
          {{ enabledOwnershipCount }} 归属 / {{ enabledPointCount }} 特殊
        </q-badge>
        <q-btn flat round dense icon="close" :disable="busy" @click="isOpen = false">
          <q-tooltip>关闭</q-tooltip>
        </q-btn>
      </q-card-section>
      <q-separator />

      <q-card-section class="data-permission-dialog-body">
        <q-tabs
          v-model="activeTab"
          class="data-permission-tabs"
          active-color="primary"
          indicator-color="primary"
          align="left"
        >
          <q-tab name="ownership" icon="badge" label="基础归属" />
          <q-tab name="overrides" icon="rule" label="特殊授权" />
        </q-tabs>

        <q-tab-panels v-model="activeTab" animated class="data-permission-tab-panels">
          <q-tab-panel name="ownership" class="q-pa-none data-permission-tab-panel">
            <div class="data-permission-filter data-permission-filter--single">
              <q-input v-model="ownershipKeyword" dense outlined clearable placeholder="搜索维度编码 / 名称">
                <template #prepend>
                  <q-icon name="search" />
                </template>
              </q-input>
            </div>

            <q-scroll-area class="data-permission-scroll">
              <q-list v-if="filteredOwnerships.length" separator class="data-permission-list">
                <q-item v-for="item in filteredOwnerships" :key="item.key" class="data-permission-item">
                  <q-item-section avatar class="data-permission-item-icon">
                    <q-toggle v-model="item.enabled" color="primary" @update:model-value="toggleOwnership(item)" />
                  </q-item-section>
                  <q-item-section class="data-permission-item-main">
                    <q-item-label class="data-permission-item-title">
                      {{ item.dimension.name }}
                    </q-item-label>
                    <q-item-label caption class="data-permission-item-code">
                      {{ item.dimension.code }}
                      <span v-if="item.dimension.source_code"> · {{ item.dimension.source_code }}</span>
                    </q-item-label>
                  </q-item-section>
                  <q-item-section side class="data-permission-ownership-fields">
                    <q-select
                      v-model="item.scope_values"
                      class="data-permission-value-select"
                      dense
                      outlined
                      multiple
                      use-input
                      new-value-mode="add-unique"
                      emit-value
                      map-options
                      options-dense
                      label="归属值"
                      :disable="busy || !item.enabled"
                      :loading="item.loading_options"
                      :options="item.option_items"
                      :display-value="scopeValuesDisplay(item.scope_values, item.option_items)"
                      :hint="ownershipValueHint(item)"
                      @focus="loadDimensionOptionsFor(item)"
                    >
                      <q-tooltip v-if="scopeValuesTooltip(item.scope_values, item.option_items)">
                        {{ scopeValuesTooltip(item.scope_values, item.option_items) }}
                      </q-tooltip>
                    </q-select>
                  </q-item-section>
                </q-item>
              </q-list>

              <div v-else class="data-permission-empty">
                <q-icon name="badge" size="34px" />
                <span>暂无可配置的数据维度</span>
              </div>
            </q-scroll-area>
          </q-tab-panel>

          <q-tab-panel name="overrides" class="q-pa-none data-permission-tab-panel">
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
                      class="data-permission-value-select"
                      dense
                      outlined
                      multiple
                      use-input
                      new-value-mode="add-unique"
                      emit-value
                      map-options
                      options-dense
                      label="范围值"
                      :disable="busy || !point.enabled"
                      :loading="point.loading_options"
                      :options="point.option_items"
                      :display-value="scopeValuesDisplay(point.scope_values, point.option_items)"
                      @focus="loadDimensionOptionsFor(point)"
                    >
                      <q-tooltip v-if="scopeValuesTooltip(point.scope_values, point.option_items)">
                        {{ scopeValuesTooltip(point.scope_values, point.option_items) }}
                      </q-tooltip>
                    </q-select>
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
          </q-tab-panel>
        </q-tab-panels>
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
  type DataPermissionDimension,
  type DataPermissionOption,
  type UserDataPermissionOverride,
  type UserDataPermissionOverrideSaveItem,
  type UserDimensionValue,
  type UserDimensionValueSaveItem,
  useDataPermissionApi,
} from 'src/api/services/data-permission'
import { compactSelectionDisplay, compactSelectionTooltip } from 'src/utils/select-display'

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

type OwnershipPoint = {
  key: string
  dimension: DataPermissionDimension
  enabled: boolean
  scope_values: string[]
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
const ownershipKeyword = ref('')
const selectedMenuId = ref<number | null>(null)
const activeTab = ref<'ownership' | 'overrides'>('ownership')
const menuOptions = ref<Array<{ label: string; value: number }>>([])
const points = ref<PermissionPoint[]>([])
const ownerships = ref<OwnershipPoint[]>([])
const dimensionOptionCache = ref<Map<string, DataPermissionOption[]>>(new Map())
const dimensionOptionRequests = new Map<string, Promise<DataPermissionOption[]>>()
const busy = computed(() => loading.value || saving.value)
const enabledPointCount = computed(() => points.value.filter((point) => point.enabled).length)
const enabledOwnershipCount = computed(() => ownerships.value.filter((item) => item.enabled).length)

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

const filteredOwnerships = computed(() => {
  const normalizedKeyword = ownershipKeyword.value.trim().toLowerCase()
  if (!normalizedKeyword) return ownerships.value
  return ownerships.value.filter((item) =>
    [item.dimension.code, item.dimension.name, item.dimension.source_code, item.dimension.memo]
      .filter(Boolean)
      .join(' ')
      .toLowerCase()
      .includes(normalizedKeyword),
  )
})

const displayMenuTitle = (menu: Menu) => {
  const title = menu.title || menu.name
  return title.startsWith('router.') ? t(title) : title
}

const flattenMenus = (menus: Menu[]): Menu[] =>
  menus.flatMap((menu) => [menu, ...flattenMenus(menu.children || [])])

const isDataScopeCapableMenu = (menu: Menu) => !!menu.table_code && !menu.is_hidden

const pointKey = (menuId: number, dimensionCode: string) => `${menuId}:${dimensionCode}`

const resetState = () => {
  saving.value = false
  keyword.value = ''
  ownershipKeyword.value = ''
  selectedMenuId.value = null
  activeTab.value = 'ownership'
  menuOptions.value = []
  points.value = []
  ownerships.value = []
  dimensionOptionCache.value = new Map()
  dimensionOptionRequests.clear()
}

const load = async () => {
  if (!props.user?.id) return
  saving.value = true
  try {
    const [menusResult, overridesResult, dimensionsResult, ownershipResult] = await Promise.all([
      menuApi.queryUserMenus(props.user.id),
      dataPermissionApi.getUserDataPermissionOverrides(props.user.id),
      dataPermissionApi.queryDimensions({
        page: 1,
        num: 1000,
        expressions: [{ rules: [{ field: '', value: null }], nested: [] }],
        quick_query: { keyword: '' },
        include_deleted: false,
      }),
      dataPermissionApi.getUserDimensionValues(props.user.id),
    ])
    const userMenus = menusResult.success ? flattenMenus(menusResult.data || []).filter(isDataScopeCapableMenu) : []
    const overrides = overridesResult.success ? overridesResult.data || [] : []
    const dimensions = dimensionsResult.success ? dimensionsResult.data || [] : []
    const ownershipValues = ownershipResult.success ? ownershipResult.data || [] : []
    const overrideMap = new Map(overrides.map((item) => [pointKey(item.menu_id, item.dimension_code), item]))
    const ownershipMap = new Map(ownershipValues.map((item) => [item.dimension_code, item]))

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
    ownerships.value = dimensions
      .filter((dimension) => dimension.state !== false)
      .map((dimension) => ownershipFromDimension(dimension, ownershipMap.get(dimension.code)))
    ownerships.value
      .filter((item) => item.enabled && item.scope_values.length > 0)
      .forEach((item) => {
        void loadDimensionOptionsFor(item)
      })
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

const ownershipFromDimension = (
  dimension: DataPermissionDimension,
  value?: UserDimensionValue,
): OwnershipPoint => ({
  key: dimension.code,
  dimension,
  enabled: !!value && value.state !== false,
  scope_values: value?.scope_values || [],
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
  void loadDimensionOptionsFor(point)
}

const dimensionCodeForOptionTarget = (target: PermissionPoint | OwnershipPoint) => {
  return 'binding' in target ? target.binding.dimension_code : target.dimension.code
}

const setOptionTargetLoading = (dimensionCode: string, loadingOptions: boolean) => {
  points.value.forEach((point) => {
    if (point.binding.dimension_code === dimensionCode) point.loading_options = loadingOptions
  })
  ownerships.value.forEach((item) => {
    if (item.dimension.code === dimensionCode) item.loading_options = loadingOptions
  })
}

const setOptionTargetOptions = (dimensionCode: string, options: DataPermissionOption[]) => {
  points.value.forEach((point) => {
    if (point.binding.dimension_code === dimensionCode) point.option_items = options
  })
  ownerships.value.forEach((item) => {
    if (item.dimension.code === dimensionCode) item.option_items = options
  })
}

const loadDimensionOptionsFor = async (target: PermissionPoint | OwnershipPoint) => {
  const dimensionCode = dimensionCodeForOptionTarget(target)
  if (target.option_items.length) return
  const cached = dimensionOptionCache.value.get(dimensionCode)
  if (cached) {
    setOptionTargetOptions(dimensionCode, cached)
    return
  }
  setOptionTargetLoading(dimensionCode, true)
  try {
    let request = dimensionOptionRequests.get(dimensionCode)
    if (!request) {
      request = dataPermissionApi.getDimensionOptions(dimensionCode).then((result) =>
        result.success ? result.data || [] : [],
      )
      dimensionOptionRequests.set(dimensionCode, request)
    }
    const options = await request
    dimensionOptionCache.value.set(dimensionCode, options)
    setOptionTargetOptions(dimensionCode, options)
  } finally {
    dimensionOptionRequests.delete(dimensionCode)
    setOptionTargetLoading(dimensionCode, false)
  }
}

const toggleOwnership = (item: OwnershipPoint) => {
  if (!item.enabled) {
    item.scope_values = []
    return
  }
  void loadDimensionOptionsFor(item)
}

const scopeValuesDisplay = (values: string[], options: DataPermissionOption[]) => {
  return compactSelectionDisplay(values, options, 2)
}

const scopeValuesTooltip = (values: string[], options: DataPermissionOption[]) => {
  return compactSelectionTooltip(values, options)
}

const ownershipValueHint = (item: OwnershipPoint) => {
  if (item.dimension.source_type === 'table') {
    return item.dimension.source_code ? `来自 ${item.dimension.source_code}` : ''
  }
  return item.enabled ? '无来源维度可直接输入后回车' : ''
}

const validate = () => {
  for (const item of ownerships.value.filter((entry) => entry.enabled)) {
    if (item.scope_values.length === 0) {
      $q.notify({
        type: 'warning',
        position: 'top-right',
        message: `${item.dimension.name || item.dimension.code} 需要归属值`,
      })
      activeTab.value = 'ownership'
      return false
    }
  }
  for (const point of points.value.filter((item) => item.enabled)) {
    if (needsValues(point) && point.scope_values.length === 0) {
      $q.notify({
        type: 'warning',
        position: 'top-right',
        message: `${displayMenuTitle(point.menu)} / ${point.binding.dimension_code} 需要范围值`,
      })
      activeTab.value = 'overrides'
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
    const ownershipValues = ownerships.value
      .filter((item) => item.enabled)
      .map<UserDimensionValueSaveItem>((item) => ({
        dimension_code: item.dimension.code,
        scope_values: item.scope_values,
        state: true,
      }))
    const [ownershipResult, overrideResult] = await Promise.all([
      dataPermissionApi.saveUserDimensionValues(props.user.id, ownershipValues),
      dataPermissionApi.saveUserDataPermissionOverrides(props.user.id, overrides),
    ])
    if (ownershipResult.success && overrideResult.success) {
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

.data-permission-tabs {
  flex-shrink: 0;
  padding: 0 14px;
  border-bottom: 1px solid #e6ebf5;
  background: #fff;
}

.data-permission-tab-panels {
  flex: 1;
  min-height: 0;
  background: transparent;
}

.data-permission-tab-panel {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.data-permission-filter {
  flex-shrink: 0;
  display: grid;
  grid-template-columns: minmax(260px, 360px) minmax(260px, 1fr);
  gap: 12px;
  padding: 14px 18px;
  border-bottom: 1px solid #e6ebf5;
  background: #fff;
}

.data-permission-filter--single {
  grid-template-columns: minmax(260px, 420px);
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
  width: min(560px, 54%);
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  align-items: start;
}

.data-permission-item-fields .data-permission-value-select {
  grid-column: 1 / -1;
}

.data-permission-ownership-fields {
  width: min(420px, 42%);
}

.data-permission-value-select :deep(.q-field__native) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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

  .data-permission-ownership-fields {
    width: 100%;
  }
}
</style>
