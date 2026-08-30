<template>
  <base-content scrollable>
    <div class="platform-dashboard q-pa-md">
      <header class="dashboard-header">
        <div class="dashboard-header__mark">
          <q-icon name="space_dashboard" size="24px" />
        </div>
        <div class="dashboard-header__copy">
          <h1>平台概览</h1>
          <p>查看组织基础数据、登录会话、集成异常和最近操作。</p>
        </div>
        <q-space />
        <div class="dashboard-header__meta">
          <span>{{ userStore.user_name || '当前用户' }}</span>
          <span>更新于 {{ refreshedAt || '-' }}</span>
        </div>
        <q-btn round flat icon="refresh" :loading="refreshing" @click="loadDashboard">
          <q-tooltip>刷新概览</q-tooltip>
        </q-btn>
      </header>

      <q-banner v-if="partialFailure" dense class="dashboard-warning">
        <template #avatar><q-icon name="info_outline" color="warning" /></template>
        部分概览数据暂时无法读取，其他模块仍可正常使用。
      </q-banner>

      <section class="metric-strip" aria-label="平台关键指标">
        <article v-for="item in metrics" :key="item.label" class="metric-item">
          <div class="metric-item__icon" :class="`metric-item__icon--${item.tone}`">
            <q-icon :name="item.icon" size="21px" />
          </div>
          <div class="metric-item__body">
            <div class="metric-item__label">{{ item.label }}</div>
            <div class="metric-item__value">{{ displayMetric(item.value) }}</div>
            <div class="metric-item__hint">{{ item.hint }}</div>
          </div>
        </article>
      </section>

      <div class="dashboard-workspace">
        <section class="dashboard-section dashboard-section--organization">
          <div class="section-heading">
            <div>
              <h2>组织基础数据</h2>
              <p>法人架构、管理架构、人员和岗位的当前档案数量。</p>
            </div>
            <q-btn
              v-if="canOpenOrganization"
              flat
              color="primary"
              icon-right="arrow_forward"
              label="查看组织架构"
              :to="{ name: 'organization_structure' }"
            />
          </div>
          <div class="organization-summary">
            <article v-for="item in organizationItems" :key="item.label">
              <q-icon :name="item.icon" size="22px" :color="item.color" />
              <div>
                <span>{{ item.label }}</span>
                <strong>{{ displayMetric(item.value) }}</strong>
              </div>
              <q-btn
                v-if="item.route && item.available"
                round
                dense
                flat
                icon="chevron_right"
                :to="{ name: item.route }"
              >
                <q-tooltip>打开{{ item.label }}</q-tooltip>
              </q-btn>
            </article>
          </div>
        </section>

        <section class="dashboard-section dashboard-section--attention">
          <div class="section-heading">
            <div>
              <h2>需要关注</h2>
              <p>只统计当前账号有权查看的运行异常。</p>
            </div>
          </div>
          <div class="attention-list">
            <article v-for="item in attentionItems" :key="item.label">
              <div class="attention-list__icon" :class="{ 'is-clear': item.value === 0 }">
                <q-icon :name="item.value === 0 ? 'check' : item.icon" />
              </div>
              <div class="attention-list__copy">
                <strong>{{ item.label }}</strong>
                <span>{{ item.description }}</span>
              </div>
              <q-chip
                v-if="item.value !== null"
                dense
                square
                :color="item.value ? 'negative' : 'positive'"
                text-color="white"
              >
                {{ item.value }}
              </q-chip>
              <span v-else class="text-caption text-grey-6">无查看权限</span>
              <q-btn
                v-if="item.route && item.value !== null"
                round
                dense
                flat
                icon="chevron_right"
                :to="{ name: item.route }"
              >
                <q-tooltip>查看详情</q-tooltip>
              </q-btn>
            </article>
          </div>
        </section>
      </div>

      <section class="dashboard-section dashboard-section--audit">
        <div class="section-heading">
          <div>
            <h2>最近操作</h2>
            <p>
              {{
                auditAvailable
                  ? '最近发生的后台操作，便于快速发现失败请求。'
                  : '当前账号没有审计日志查看权限。'
              }}
            </p>
          </div>
          <q-btn
            v-if="auditAvailable"
            flat
            color="primary"
            icon-right="arrow_forward"
            label="查看全部"
            :to="{ name: 'system_audit' }"
          />
        </div>
        <q-markup-table flat separator="horizontal" class="dashboard-table">
          <thead>
            <tr>
              <th class="text-left">时间</th>
              <th class="text-left">用户</th>
              <th class="text-left">动作</th>
              <th class="text-left">资源</th>
              <th class="text-left">结果</th>
              <th class="text-right">耗时</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="log in recentLogs" :key="log.id">
              <td>{{ log.gmt_create || '-' }}</td>
              <td>{{ log.user_name || '-' }}</td>
              <td>{{ log.action || log.method || '-' }}</td>
              <td>{{ log.resource_code || log.url || '-' }}</td>
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
                {{ auditAvailable ? '暂无审计记录' : '无查看权限' }}
              </td>
            </tr>
          </tbody>
        </q-markup-table>
      </section>

      <nav class="quick-entry" aria-label="常用入口">
        <span>常用入口</span>
        <q-btn
          v-for="item in quickEntries"
          :key="item.route"
          flat
          no-caps
          :icon="item.icon"
          :label="item.label"
          :to="{ name: item.route }"
        />
      </nav>
    </div>
  </base-content>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive, ref } from 'vue'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import { useUserStore } from 'src/stores/user'
import { usePageButtons } from 'src/composables/page-buttons'
import { queryOrganizationOptions, type OrganizationSelectorType } from 'src/api/services/org'
import { useUserSessionApi } from 'src/api/services/user-session'
import { useIntegrationApi } from 'src/api/services/integration'
import { useAccessLogApi, type AccessLog } from 'src/api/services/access-log'

defineOptions({ name: 'DashboardIndex' })

type SummaryValue = number | null

const userStore = useUserStore()
const { hasGrantedCapability } = usePageButtons('home')
const sessionApi = useUserSessionApi()
const integrationApi = useIntegrationApi()
const accessLogApi = useAccessLogApi()

const refreshing = ref(false)
const partialFailure = ref(false)
const refreshedAt = ref('')
const recentLogs = ref<AccessLog[]>([])
const summary = reactive({
  onlineUsers: null as SummaryValue,
  onlineSessions: null as SummaryValue,
  legalEntities: null as SummaryValue,
  orgUnits: null as SummaryValue,
  employees: null as SummaryValue,
  positions: null as SummaryValue,
  failedExecutions: null as SummaryValue,
  expiredCredentials: null as SummaryValue,
})

const auditAvailable = computed(() => hasGrantedCapability('system_audit_query'))
const canOpenOrganization = computed(
  () =>
    hasGrantedCapability('organization_legal_entity_options') ||
    hasGrantedCapability('organization_unit_options'),
)

const combinedValue = (...values: SummaryValue[]): SummaryValue => {
  const available = values.filter((value): value is number => value !== null)
  return available.length ? available.reduce((total, value) => total + value, 0) : null
}

const attentionSummary = computed(() => {
  const parts: string[] = []
  if (summary.failedExecutions !== null) parts.push(`执行失败 ${summary.failedExecutions}`)
  if (summary.expiredCredentials !== null) parts.push(`凭证过期 ${summary.expiredCredentials}`)
  return parts.length ? parts.join(' · ') : '无集成运行查看权限'
})

const metrics = computed(() => [
  {
    label: '当前在线用户',
    value: summary.onlineUsers,
    hint:
      summary.onlineSessions === null
        ? '无登录会话查看权限'
        : `${summary.onlineSessions} 个活跃会话`,
    icon: 'group',
    tone: 'primary',
  },
  {
    label: '法人主体',
    value: summary.legalEntities,
    hint: summary.orgUnits === null ? '无组织档案查看权限' : `${summary.orgUnits} 个管理组织`,
    icon: 'corporate_fare',
    tone: 'neutral',
  },
  {
    label: '人员档案',
    value: summary.employees,
    hint: summary.positions === null ? '无人员档案查看权限' : `${summary.positions} 个岗位`,
    icon: 'badge',
    tone: 'positive',
  },
  {
    label: '待处理异常',
    value: combinedValue(summary.failedExecutions, summary.expiredCredentials),
    hint: attentionSummary.value,
    icon: 'error_outline',
    tone: 'warning',
  },
])

const organizationItems = computed(() => [
  {
    label: '法人主体',
    value: summary.legalEntities,
    icon: 'account_balance',
    color: 'primary',
    route: 'organization_structure',
    available: hasGrantedCapability('organization_legal_entity_options'),
  },
  {
    label: '管理组织',
    value: summary.orgUnits,
    icon: 'account_tree',
    color: 'teal',
    route: 'organization_structure',
    available: hasGrantedCapability('organization_unit_options'),
  },
  {
    label: '人员档案',
    value: summary.employees,
    icon: 'badge',
    color: 'positive',
    route: 'organization_employee',
    available: hasGrantedCapability('organization_employee_options'),
  },
  {
    label: '岗位档案',
    value: summary.positions,
    icon: 'work_outline',
    color: 'orange-8',
    route: 'organization_position',
    available: hasGrantedCapability('organization_position_options'),
  },
])

const attentionItems = computed(() => [
  {
    label: '失败的集成执行',
    value: summary.failedExecutions,
    description:
      summary.failedExecutions === 0 ? '当前没有失败执行' : '检查接口调用和同步处理结果',
    icon: 'sync_problem',
    route: 'integration_execution',
  },
  {
    label: '已过期集成凭证',
    value: summary.expiredCredentials,
    description:
      summary.expiredCredentials === 0 ? '当前没有过期凭证' : '更新凭证后再恢复接口调用',
    icon: 'key_off',
    route: 'integration_credential',
  },
])

const quickEntries = computed(() => {
  const entries = [
    {
      label: '组织架构',
      icon: 'account_tree',
      route: 'organization_structure',
      visible: canOpenOrganization.value,
    },
    {
      label: '在线用户',
      icon: 'devices',
      route: 'system_online_session',
      visible: hasGrantedCapability('system_online_session_query'),
    },
    {
      label: '执行记录',
      icon: 'play_circle',
      route: 'integration_execution',
      visible: hasGrantedCapability('integration_execution_query'),
    },
    {
      label: '审计日志',
      icon: 'manage_search',
      route: 'system_audit',
      visible: auditAvailable.value,
    },
  ]
  return entries.filter((entry) => entry.visible)
})

const displayMetric = (value: SummaryValue) => {
  if (value !== null) return value.toLocaleString('zh-CN')
  return refreshing.value ? '...' : '--'
}

const loadOrganizationTotal = async (
  type: OrganizationSelectorType,
  target: keyof typeof summary,
) => {
  const result = await queryOrganizationOptions(type, { page: 1, num: 1 })
  summary[target] = result.total
}

const loadDashboard = async () => {
  if (refreshing.value) return
  refreshing.value = true
  partialFailure.value = false
  const tasks: Promise<unknown>[] = []

  if (hasGrantedCapability('system_online_session_query')) {
    tasks.push(
      sessionApi.query({ keyword: '', status: 'online', page: 1, num: 1 }).then((response) => {
        summary.onlineUsers = response.data?.online_users || 0
        summary.onlineSessions =
          response.data?.online_sessions || response.data?.online_devices || 0
      }),
    )
  }
  if (hasGrantedCapability('organization_legal_entity_options')) {
    tasks.push(loadOrganizationTotal('legal_entity', 'legalEntities'))
  }
  if (hasGrantedCapability('organization_unit_options')) {
    tasks.push(loadOrganizationTotal('org_unit', 'orgUnits'))
  }
  if (hasGrantedCapability('organization_employee_options')) {
    tasks.push(loadOrganizationTotal('employee', 'employees'))
  }
  if (hasGrantedCapability('organization_position_options')) {
    tasks.push(loadOrganizationTotal('position', 'positions'))
  }
  if (hasGrantedCapability('integration_execution_query')) {
    tasks.push(
      integrationApi
        .queryExecutions({ page: 1, num: 1, expressions: [], status: 'failed' })
        .then((response) => {
          summary.failedExecutions = response.total || 0
        }),
    )
  }
  if (hasGrantedCapability('integration_credential_query')) {
    tasks.push(
      integrationApi
        .queryCredentials({ page: 1, num: 1, expressions: [], status: 'expired' })
        .then((response) => {
          summary.expiredCredentials = response.total || 0
        }),
    )
  }
  if (auditAvailable.value) {
    tasks.push(
      accessLogApi
        .queryAccessLogs({ page: 1, num: 6, expressions: [] })
        .then((response) => {
          recentLogs.value = response.data || []
        }),
    )
  }

  const results = await Promise.allSettled(tasks)
  partialFailure.value = results.some((result) => result.status === 'rejected')
  refreshedAt.value = new Date().toLocaleTimeString('zh-CN', { hour12: false })
  refreshing.value = false
}

onMounted(() => void loadDashboard())
</script>

<style scoped>
.platform-dashboard {
  display: grid;
  gap: 16px;
  max-width: 1600px;
  margin: 0 auto;
}

.dashboard-header {
  min-height: 76px;
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 16px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-surface, #fff);
}

.dashboard-header__mark {
  width: 44px;
  height: 44px;
  flex: 0 0 44px;
  display: grid;
  place-items: center;
  border-radius: 8px;
  color: var(--q-primary);
  background: var(--app-primary-soft);
}

.dashboard-header__copy h1,
.section-heading h2 {
  margin: 0;
  letter-spacing: 0;
  color: var(--app-text-strong);
}

.dashboard-header__copy h1 {
  font-size: 21px;
  line-height: 28px;
}

.dashboard-header__copy p,
.section-heading p {
  margin: 3px 0 0;
  color: var(--app-text-muted);
  font-size: 13px;
}

.dashboard-header__meta {
  display: grid;
  justify-items: end;
  color: var(--app-text-muted);
  font-size: 12px;
}

.dashboard-header__meta span:first-child {
  color: var(--app-text-strong);
  font-size: 14px;
  font-weight: 600;
}

.dashboard-warning {
  border: 1px solid rgba(242, 192, 55, 0.45);
  border-radius: 6px;
  background: rgba(242, 192, 55, 0.08);
}

.metric-strip {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-surface, #fff);
  overflow: hidden;
}

.metric-item {
  min-width: 0;
  min-height: 112px;
  display: flex;
  align-items: center;
  gap: 13px;
  padding: 18px;
  border-right: 1px solid var(--app-border);
}

.metric-item:last-child {
  border-right: 0;
}

.metric-item__icon {
  width: 40px;
  height: 40px;
  flex: 0 0 40px;
  display: grid;
  place-items: center;
  border-radius: 8px;
}

.metric-item__icon--primary {
  color: #2563eb;
  background: #eff6ff;
}

.metric-item__icon--neutral {
  color: #475569;
  background: #f1f5f9;
}

.metric-item__icon--positive {
  color: #16803a;
  background: #ecfdf3;
}

.metric-item__icon--warning {
  color: #b45309;
  background: #fff7ed;
}

.metric-item__body {
  min-width: 0;
}

.metric-item__label,
.metric-item__hint {
  color: var(--app-text-muted);
  font-size: 12px;
}

.metric-item__value {
  margin: 2px 0;
  color: var(--app-text-strong);
  font-size: 26px;
  line-height: 31px;
  font-weight: 700;
}

.metric-item__hint {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-workspace {
  display: grid;
  grid-template-columns: minmax(0, 1.6fr) minmax(360px, 1fr);
  gap: 16px;
}

.dashboard-section {
  min-width: 0;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-surface, #fff);
  overflow: hidden;
}

.section-heading {
  min-height: 70px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 18px;
  border-bottom: 1px solid var(--app-border);
}

.section-heading h2 {
  font-size: 16px;
  line-height: 22px;
}

.organization-summary {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.organization-summary article {
  min-width: 0;
  min-height: 86px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 18px;
  border-right: 1px solid var(--app-border);
  border-bottom: 1px solid var(--app-border);
}

.organization-summary article:nth-child(even) {
  border-right: 0;
}

.organization-summary article:nth-last-child(-n + 2) {
  border-bottom: 0;
}

.organization-summary article > div {
  min-width: 0;
  flex: 1;
  display: grid;
}

.organization-summary span {
  color: var(--app-text-muted);
  font-size: 12px;
}

.organization-summary strong {
  margin-top: 2px;
  font-size: 20px;
}

.attention-list article {
  min-height: 86px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 18px;
  border-bottom: 1px solid var(--app-border);
}

.attention-list article:last-child {
  border-bottom: 0;
}

.attention-list__icon {
  width: 34px;
  height: 34px;
  flex: 0 0 34px;
  display: grid;
  place-items: center;
  border-radius: 50%;
  color: #b42318;
  background: #fef3f2;
}

.attention-list__icon.is-clear {
  color: #16803a;
  background: #ecfdf3;
}

.attention-list__copy {
  min-width: 0;
  flex: 1;
  display: grid;
}

.attention-list__copy strong {
  font-size: 14px;
}

.attention-list__copy span {
  color: var(--app-text-muted);
  font-size: 12px;
}

.dashboard-section--audit {
  overflow-x: auto;
}

.dashboard-table {
  min-width: 760px;
}

.dashboard-table td,
.dashboard-table th {
  white-space: nowrap;
}

.quick-entry {
  min-height: 52px;
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 10px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-surface, #fff);
}

.quick-entry > span {
  padding: 0 10px;
  color: var(--app-text-muted);
  font-size: 12px;
}

@media (max-width: 1100px) {
  .metric-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .metric-item:nth-child(2) {
    border-right: 0;
  }

  .metric-item:nth-child(-n + 2) {
    border-bottom: 1px solid var(--app-border);
  }

  .dashboard-workspace {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 650px) {
  .platform-dashboard {
    padding: 8px;
  }

  .dashboard-header__meta {
    display: none;
  }

  .metric-strip {
    grid-template-columns: 1fr;
  }

  .metric-item {
    border-right: 0;
    border-bottom: 1px solid var(--app-border);
  }

  .metric-item:last-child {
    border-bottom: 0;
  }

  .organization-summary {
    grid-template-columns: 1fr;
  }

  .organization-summary article {
    border-right: 0;
    border-bottom: 1px solid var(--app-border);
  }

  .organization-summary article:nth-last-child(2) {
    border-bottom: 1px solid var(--app-border);
  }

  .quick-entry {
    overflow-x: auto;
  }
}
</style>
