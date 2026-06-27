<template>
  <base-content scrollable>
    <div class="q-pa-md column q-gutter-md">
      <div class="row q-col-gutter-md metric-grid">
        <div v-for="item in metricCards" :key="item.label" class="col-12 col-sm-6 col-md-3">
          <q-card flat bordered :class="['metric-card', `metric-card--${item.tone}`]">
            <q-card-section class="metric-card__body">
              <div class="metric-card__icon">
                <q-icon :name="item.icon" size="24px" />
              </div>
              <div class="metric-card__content">
                <div class="metric-card__value">
                  <count-to :start-value="0" :end-value="item.value" />
                </div>
                <div class="metric-card__label">{{ item.label }}</div>
                <div class="metric-card__hint">{{ item.hint }}</div>
              </div>
            </q-card-section>
          </q-card>
        </div>
      </div>

      <div class="row q-col-gutter-md">
        <div class="col-12 col-lg-7">
          <q-card flat bordered>
            <q-card-section class="row items-center justify-between">
              <div class="text-subtitle1 text-weight-medium">权限覆盖</div>
              <q-chip dense square color="primary" text-color="white">
                {{ visibleActionCount }} 个可见操作
              </q-chip>
            </q-card-section>
            <q-separator />
            <q-card-section class="column q-gutter-md">
              <div v-for="group in menuGroups" :key="group.name">
                <div class="row items-center justify-between q-mb-xs">
                  <div class="text-body2">{{ formatMenuTitle(group.name) }}</div>
                  <div class="text-caption text-grey-7">
                    {{ group.visibleMenus }} 菜单 / {{ group.actions }} 操作
                  </div>
                </div>
                <q-linear-progress
                  rounded
                  size="10px"
                  :value="groupRatio(group)"
                  :color="group.color"
                  track-color="grey-3"
                />
              </div>
              <q-banner v-if="menuGroups.length === 0" rounded class="bg-grey-2 text-grey-8">
                当前用户暂无可访问菜单。
              </q-banner>
            </q-card-section>
          </q-card>
        </div>

        <div class="col-12 col-lg-5">
          <q-card flat bordered>
            <q-card-section class="row items-center justify-between">
              <div class="text-subtitle1 text-weight-medium">低代码页面</div>
              <q-chip dense square color="secondary" text-color="white">
                {{ lowCodeMenus.length }} 个
              </q-chip>
            </q-card-section>
            <q-separator />
            <q-list separator>
              <q-item v-for="menu in lowCodeMenus.slice(0, 6)" :key="menu.id">
                <q-item-section avatar>
                  <q-icon :name="menu.icon || 'dynamic_form'" color="secondary" />
                </q-item-section>
                <q-item-section>
                  <q-item-label>{{ formatMenuTitle(menu.title) }}</q-item-label>
                  <q-item-label caption>{{ menu.table_code }}</q-item-label>
                </q-item-section>
                <q-item-section side>
                  <q-btn round dense flat icon="open_in_new" :to="lowCodeRoute(menu)">
                    <q-tooltip>打开</q-tooltip>
                  </q-btn>
                </q-item-section>
              </q-item>
              <q-item v-if="lowCodeMenus.length === 0">
                <q-item-section class="text-grey-7">暂无已发布低代码页面。</q-item-section>
              </q-item>
            </q-list>
          </q-card>
        </div>
      </div>

      <q-card flat bordered>
        <q-card-section class="row items-center justify-between">
          <div>
            <div class="text-subtitle1 text-weight-medium">最近审计</div>
            <div class="text-caption text-grey-7">
              {{
                auditAvailable
                  ? `最近 ${recentLogs.length} 条操作，成功率 ${successRate}%`
                  : '当前角色未开放审计概览'
              }}
            </div>
          </div>
          <q-btn
            v-if="auditAvailable"
            dense
            flat
            icon="refresh"
            :loading="loading"
            @click="fetchRecentAudit"
          >
            <q-tooltip>刷新审计</q-tooltip>
          </q-btn>
        </q-card-section>
        <q-separator />
        <q-markup-table flat separator="horizontal" class="dashboard-table">
          <thead>
            <tr>
              <th class="text-left">时间</th>
              <th class="text-left">用户</th>
              <th class="text-left">动作</th>
              <th class="text-left">资源</th>
              <th class="text-left">状态</th>
              <th class="text-right">耗时</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="log in recentLogs" :key="log.id">
              <td>{{ log.gmt_create || '-' }}</td>
              <td>{{ log.user_name || '-' }}</td>
              <td>{{ log.action || log.method }}</td>
              <td>{{ log.resource_code || log.url }}</td>
              <td>
                <q-chip
                  dense
                  square
                  :color="log.success ? 'positive' : 'negative'"
                  text-color="white"
                >
                  {{ log.success ? '成功' : '失败' }}
                </q-chip>
              </td>
              <td class="text-right">{{ log.duration_ms }}ms</td>
            </tr>
            <tr v-if="recentLogs.length === 0">
              <td colspan="6" class="text-center text-grey-7 q-pa-lg">
                {{ auditAvailable ? '暂无审计记录。' : '没有审计菜单权限。' }}
              </td>
            </tr>
          </tbody>
        </q-markup-table>
      </q-card>
    </div>
  </base-content>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from 'vue'
import { storeToRefs } from 'pinia'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import CountTo from 'src/components/CountTo/CountTo.vue'
import { useMenuApi, type Menu } from 'src/api/services/sys-menu'
import { useAccessLogApi, type AccessLog } from 'src/api/services/access-log'
import { useUserStore } from 'src/stores/user'
import { useLoadingStore } from 'src/stores/loading'
import { isApiPermission, isPageButton } from 'src/utils/menu-button'

defineOptions({ name: 'DashboardIndex' })

type MenuGroup = {
  name: string
  visibleMenus: number
  actions: number
  color: string
}

const menuApi = useMenuApi()
const accessLogApi = useAccessLogApi()
const userStore = useUserStore()
const loadingStore = useLoadingStore()
const { loading } = storeToRefs(loadingStore)

const menus = ref<Menu[]>([])
const recentLogs = ref<AccessLog[]>([])

const auditAvailable = computed(() => userStore.buttons.includes('system_audit_query'))

const flatMenus = computed(() => flattenMenus(menus.value))
const visibleMenus = computed(() => flatMenus.value.filter((menu) => !menu.is_hidden))
const lowCodeMenus = computed(() =>
  visibleMenus.value.filter((menu) => menu.page_type === 'low_code' && !!menu.table_code),
)
const allButtons = computed(() => flatMenus.value.flatMap((menu) => menu.menu_buttons || []))
const visibleActionCount = computed(
  () => allButtons.value.filter((button) => isPageButton(button) && !button.is_disabled).length,
)
const apiPermissionCount = computed(
  () => allButtons.value.filter(isApiPermission).length,
)
const successRate = computed(() => {
  if (recentLogs.value.length === 0) return 0
  const successCount = recentLogs.value.filter((log) => log.success).length
  return Math.round((successCount / recentLogs.value.length) * 100)
})

const metricCards = computed(() => [
  {
    label: '可访问菜单',
    hint: '当前账号范围',
    value: visibleMenus.value.length,
    icon: 'menu',
    tone: 'primary',
  },
  {
    label: '低代码页面',
    hint: '已发布配置',
    value: lowCodeMenus.value.length,
    icon: 'dynamic_form',
    tone: 'neutral',
  },
  {
    label: '可见操作',
    hint: '按钮与动作权限',
    value: visibleActionCount.value,
    icon: 'ads_click',
    tone: 'positive',
  },
  {
    label: '接口权限',
    hint: '后台接口权限',
    value: apiPermissionCount.value,
    icon: 'lock',
    tone: 'warning',
  },
])

const menuGroups = computed<MenuGroup[]>(() => {
  const colors = ['primary', 'secondary', 'positive', 'info', 'warning', 'deep-purple']
  return menus.value
    .filter((menu) => !menu.is_hidden)
    .map((menu, index) => {
      const descendants = flattenMenus(menu.children || [])
      const items = [menu, ...descendants].filter((item) => !item.is_hidden)
      const actions = items
        .flatMap((item) => item.menu_buttons || [])
        .filter(isPageButton).length
      return {
        name: menu.title || menu.name,
        visibleMenus: items.length,
        actions,
        color: colors[index % colors.length] || 'primary',
      }
    })
})

function flattenMenus(source: Menu[]): Menu[] {
  const result: Menu[] = []
  for (const menu of source) {
    result.push(menu)
    if (menu.children?.length) {
      result.push(...flattenMenus(menu.children))
    }
  }
  return result
}

function groupRatio(group: MenuGroup) {
  const max = Math.max(...menuGroups.value.map((item) => item.visibleMenus + item.actions), 1)
  return Math.min((group.visibleMenus + group.actions) / max, 1)
}

function formatMenuTitle(title?: string) {
  if (!title) return '-'
  if (title.startsWith('router.')) return title.split('.').pop() || title
  return title
}

function lowCodeRoute(menu: Menu) {
  return {
    path: `/admin/develop/${menu.path}`,
  }
}

async function fetchMenus() {
  const res = await menuApi.queryMyMenu()
  if (res.success && res.data) {
    menus.value = res.data
  }
}

async function fetchRecentAudit() {
  if (!auditAvailable.value) return
  const res = await accessLogApi.queryAccessLogs({
    page: 1,
    num: 8,
    expressions: [],
  })
  if (res.success && res.data) {
    recentLogs.value = res.data
  }
}

onMounted(async () => {
  await fetchMenus()
  await fetchRecentAudit()
})
</script>

<style scoped>
.metric-card {
  min-height: 116px;
  border-radius: 8px;
  overflow: hidden;
  transition:
    transform 0.18s ease,
    box-shadow 0.18s ease,
    border-color 0.18s ease;
}

.metric-card:hover {
  transform: translateY(-1px);
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.08);
}

.metric-card__body {
  min-height: 116px;
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 18px 20px;
}

.metric-card__icon {
  width: 48px;
  height: 48px;
  flex: 0 0 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
}

.metric-card__content {
  min-width: 0;
}

.metric-card__label {
  margin-top: 2px;
  color: #475569;
  font-size: 14px;
  line-height: 18px;
  font-weight: 500;
}

.metric-card__value {
  color: #111827;
  font-size: 30px;
  line-height: 34px;
  font-weight: 700;
}

.metric-card__hint {
  margin-top: 4px;
  color: #94a3b8;
  font-size: 12px;
  line-height: 16px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.metric-card--primary {
  border-left: 4px solid #1976d2;
}

.metric-card--primary .metric-card__icon {
  color: #1976d2;
  background: rgba(25, 118, 210, 0.1);
}

.metric-card--neutral {
  border-left: 4px solid #607d8b;
}

.metric-card--neutral .metric-card__icon {
  color: #455a64;
  background: rgba(96, 125, 139, 0.1);
}

.metric-card--positive {
  border-left: 4px solid #21ba45;
}

.metric-card--positive .metric-card__icon {
  color: #1b8f38;
  background: rgba(33, 186, 69, 0.1);
}

.metric-card--warning {
  border-left: 4px solid #f2c037;
}

.metric-card--warning .metric-card__icon {
  color: #a66d00;
  background: rgba(242, 192, 55, 0.16);
}

.dashboard-table td,
.dashboard-table th {
  white-space: nowrap;
}
</style>
