<template>
  <base-content class="q-pa-sm verification-page">
    <div class="verification-workbench">
      <aside class="verification-master">
        <div class="verification-toolbar">
          <q-input v-model="keyword" outlined dense clearable placeholder="搜索验证场景">
            <template #append><q-icon name="search" /></template>
          </q-input>
          <q-select
            v-model="category"
            :options="categoryOptions"
            outlined
            dense
            emit-value
            map-options
            label="场景分类"
          />
        </div>
        <q-scroll-area class="verification-master-scroll">
          <q-list separator>
            <q-item
              v-for="scenario in filteredScenarios"
              :key="scenario.id"
              clickable
              :active="scenario.id === selectedId"
              active-class="verification-item--active"
              @click="selectedId = scenario.id"
            >
              <q-item-section avatar>
                <q-icon :name="scenario.icon" color="primary" />
              </q-item-section>
              <q-item-section>
                <q-item-label>{{ scenario.title }}</q-item-label>
                <q-item-label caption lines="2">{{ scenario.summary }}</q-item-label>
              </q-item-section>
              <q-item-section side>
                <status-chip
                  :label="statusLabels[scenario.status].label"
                  :color="statusLabels[scenario.status].color"
                  :outline="scenario.status !== 'ready'"
                />
              </q-item-section>
            </q-item>
          </q-list>
          <div v-if="!filteredScenarios.length" class="verification-empty">
            <q-icon name="search_off" size="40px" />
            <span>没有匹配的验证场景</span>
          </div>
        </q-scroll-area>
      </aside>

      <main class="verification-detail">
        <q-banner dense class="verification-guide-note">
          本页是验证步骤说明，不会自动创建账号、业务数据或外部系统配置。标记为“需要配置”或“需要样例数据”的场景，请按准备项完成后再验证。
        </q-banner>
        <q-scroll-area v-if="selectedScenario" class="verification-detail-scroll">
          <div class="verification-detail-content">
            <header class="verification-detail-header">
              <div class="verification-detail-title-wrap">
                <q-icon :name="selectedScenario.icon" class="verification-detail-icon" />
                <div>
                  <div class="verification-detail-title">{{ selectedScenario.title }}</div>
                  <div class="verification-detail-summary">{{ selectedScenario.summary }}</div>
                </div>
              </div>
              <q-space />
              <q-btn
                v-if="selectedScenario.routeName"
                unelevated
                color="primary"
                icon-right="open_in_new"
                :label="selectedScenario.actionLabel || '打开相关页面'"
                @click="openScenario(selectedScenario)"
              />
            </header>

            <section class="verification-section">
              <h3>开始前准备</h3>
              <q-list dense>
                <q-item v-for="item in selectedScenario.prerequisites" :key="item">
                  <q-item-section avatar><q-icon name="check_circle_outline" color="primary" /></q-item-section>
                  <q-item-section>{{ item }}</q-item-section>
                </q-item>
              </q-list>
            </section>

            <section class="verification-section">
              <h3>操作步骤</h3>
              <ol class="verification-steps">
                <li v-for="step in selectedScenario.steps" :key="step">{{ step }}</li>
              </ol>
            </section>

            <section class="verification-section">
              <h3>预期结果</h3>
              <q-list dense>
                <q-item v-for="item in selectedScenario.expected" :key="item">
                  <q-item-section avatar><q-icon name="task_alt" color="positive" /></q-item-section>
                  <q-item-section>{{ item }}</q-item-section>
                </q-item>
              </q-list>
            </section>

            <q-banner v-if="selectedScenario.note" rounded class="verification-note">
              <template #avatar><q-icon name="info_outline" color="primary" /></template>
              {{ selectedScenario.note }}
            </q-banner>
          </div>
        </q-scroll-area>
      </main>
    </div>
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'develop_verification_page' })

import { computed, ref } from 'vue'
import { useQuasar } from 'quasar'
import { useRouter } from 'vue-router'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import StatusChip from 'src/components/Display/StatusChip.vue'

type ScenarioStatus = 'ready' | 'configuration' | 'data'

interface VerificationScenario {
  id: string
  title: string
  category: string
  icon: string
  status: ScenarioStatus
  summary: string
  prerequisites: string[]
  steps: string[]
  expected: string[]
  note?: string
  routeName?: string
  actionLabel?: string
}

const router = useRouter()
const $q = useQuasar()
const keyword = ref('')
const category = ref('all')
const selectedId = ref('permission-page')

const categoryOptions = [
  { label: '全部场景', value: 'all' },
  { label: '权限与安全', value: 'permission' },
  { label: '数据与低代码', value: 'data' },
  { label: '组织与集成', value: 'integration' },
  { label: '文件与内容', value: 'file' },
  { label: '查询与消息', value: 'query' },
  { label: '报表', value: 'report' },
]

const statusLabels: Record<ScenarioStatus, { label: string; color: string }> = {
  ready: { label: '可直接验证', color: 'positive' },
  configuration: { label: '需要配置', color: 'warning' },
  data: { label: '需要样例数据', color: 'info' },
}

const scenarios: VerificationScenario[] = [
  {
    id: 'permission-page',
    title: '菜单、页面与按钮权限',
    category: 'permission',
    icon: 'admin_panel_settings',
    status: 'ready',
    summary: '使用管理员、只读用户和无权限用户验证页面及业务按钮边界。',
    prerequisites: [
      '准备三个测试账号，并分别分配管理员、只读和无业务菜单角色。',
      '不知道测试账号原密码时，由管理员在用户管理执行“重置密码”，保存弹窗中只显示一次的临时密码。',
    ],
    steps: [
      '管理员登录，确认可以看到用户、角色、菜单页面及其管理按钮。',
      '只读用户登录，确认页面可打开，但新增、编辑、删除等按钮不可用。',
      '无权限用户登录，确认左侧菜单不出现，直接输入页面地址也无法绕过权限。',
    ],
    expected: ['页面可见性与角色菜单一致。', '按钮可见性与菜单按钮授权一致。', '直接调用未授权 API 会被拒绝。'],
    routeName: 'system_role',
    actionLabel: '打开角色管理',
  },
  {
    id: 'data-permission',
    title: '数据权限范围',
    category: 'permission',
    icon: 'rule',
    status: 'configuration',
    summary: '为同一个业务页面配置不同数据范围，比较两个用户看到的数据。',
    prerequisites: ['业务表已发布且至少有两组可区分的数据。', '业务记录包含组织、公司或所有者等可用于过滤的字段。'],
    steps: [
      '在“数据资源”登记业务表，并配置资源主键和需要保护的操作。',
      '在“归属定义”把业务字段映射到当前用户、组织或公司维度。',
      '建立权限策略和规则，再把策略授权给测试角色或测试用户。',
      '分别使用管理员和受限用户打开同一业务列表并执行相同查询。',
    ],
    expected: ['管理员看到完整数据。', '受限用户只看到规则允许的数据。', '详情、编辑和删除接口同样不能越权。'],
    note: '数据权限测试不依赖组织同步本身，但按公司或组织过滤时，必须先有真实组织、用户任职和业务记录归属数据。',
    routeName: 'system_data_permission',
    actionLabel: '打开数据权限',
  },
  {
    id: 'tms-company-scope',
    title: 'TMS 车辆公司范围',
    category: 'permission',
    icon: 'local_shipping',
    status: 'configuration',
    summary: '验证车辆列表默认受当前用户公司范围约束，多公司用户可以查看授权公司集合。',
    prerequisites: ['已发布 tms_vehicle 低代码页面。', '车辆表有稳定的公司字段。', '测试用户已通过任职或授权绑定一个或多个公司。'],
    steps: [
      '把车辆资源的公司字段配置为数据权限维度，不要只在前端写死公司下拉框。',
      '给单公司用户授权公司 A，打开车辆页面并检查查询条件和返回数据。',
      '给多公司用户授权公司 A、B，再次打开车辆页面并查询。',
      '手工修改请求条件尝试查询未授权公司 C。',
    ],
    expected: ['单公司用户默认只返回公司 A。', '多公司用户可返回 A、B 的并集。', '公司 C 即使通过请求参数传入也不会返回。'],
    note: '查询条件可以展示当前授权公司，但真正的限制必须由后端 Data Permission 与页面条件做 AND 组合。',
    routeName: 'system_data_permission',
    actionLabel: '配置车辆数据权限',
  },
  {
    id: 'metadata-low-code',
    title: 'Metadata 与低代码页面',
    category: 'data',
    icon: 'dynamic_form',
    status: 'data',
    summary: '验证字段配置如何影响列表、查询、新增、编辑和详情。',
    prerequisites: ['准备一张非生产样例表。', '表中包含文本、数字、日期、布尔、字典和 Relation 字段。'],
    steps: [
      '在数据管理中初始化表元数据并逐项配置字段。',
      '设置列表、表单、详情、查询、排序可见性以及 Input Type。',
      '配置字典、Relation 和必要索引后发布页面。',
      '从生成的页面完成查询、新增、编辑、详情和删除。',
    ],
    expected: ['页面只显示元数据允许的字段。', '字典和 Relation 显示业务名称而不是编码或外键。', '字段校验与后端类型约束一致。'],
    routeName: 'develop_database',
    actionLabel: '打开数据管理',
  },
  {
    id: 'organization-sync',
    title: '组织、人事与岗位同步',
    category: 'integration',
    icon: 'account_tree',
    status: 'configuration',
    summary: '从真实 HR 接口同步法人架构、管理架构、岗位、人员和任职。',
    prerequisites: ['已配置外部系统、凭证、接口定义和同步任务。', '源数据具有稳定 SourceKey、版本时间和明确的父子关系。'],
    steps: [
      '先同步法人主体和法人组织，再同步管理公司、管理组织及两套架构关系。',
      '同步岗位和人员档案，最后同步人员任职。',
      '在同步批次查看总数、成功数、失败数和 Checkpoint。',
      '在组织架构、人员与任职、岗位页面核对结果。',
    ],
    expected: ['法人架构和管理架构可以分别浏览。', '同一组织事实不会因重复同步产生重复记录。', '父级晚到、旧事件和失败记录具有明确处理结果。'],
    routeName: 'organization_sync_batch',
    actionLabel: '打开同步批次',
  },
  {
    id: 'integration-call',
    title: '外部接口调用',
    category: 'integration',
    icon: 'hub',
    status: 'configuration',
    summary: '验证系统、凭证、接口、重试策略和执行记录的完整调用过程。',
    prerequisites: ['准备不含真实生产密钥的测试接口。', '按顺序配置外部系统、凭证、接口定义和重试策略。'],
    steps: [
      '在接口定义中配置 Method、相对路径、参数和响应约束。',
      '发起一次手工执行，并在执行记录查看输入安全摘要和状态。',
      '进入 Attempt 调用日志核对 HTTP 状态、耗时和错误分类。',
      '制造一次可重试技术错误，确认按策略生成下一次 Attempt。',
    ],
    expected: ['凭证正文不会出现在列表、详情或日志。', '执行详情可滚动并完整展示安全摘要。', '业务失败与技术重试不会混淆。'],
    routeName: 'integration_execution',
    actionLabel: '打开执行记录',
  },
  {
    id: 'file-upload',
    title: '文件上传、下载与删除',
    category: 'file',
    icon: 'upload_file',
    status: 'data',
    summary: '通过低代码文件字段验证普通上传、权限访问和删除生命周期。',
    prerequisites: ['样例业务表已配置文件字段并发布。', '测试账号具有该记录的新增、详情和删除权限。'],
    steps: [
      '在新增表单选择一个小文件并完成普通上传。',
      '保存记录后从详情页预览或下载文件。',
      '使用无记录访问权限的账号尝试打开相同文件。',
      '删除业务记录并检查文件引用和物理文件清理结果。',
    ],
    expected: ['有权限用户可以按用途预览或下载。', '无权限用户不能仅凭文件 ID 访问。', '删除失败时数据库与存储不会留下半完成状态。'],
    routeName: 'develop_database',
    actionLabel: '配置文件字段',
  },
  {
    id: 'video-preview',
    title: '视频与大文件预览',
    category: 'file',
    icon: 'movie',
    status: 'data',
    summary: '验证视频 Range 请求、预览权限以及大文件分片上传。',
    prerequisites: ['准备无敏感内容的 MP4 测试文件。', '样例页面已配置文件字段。'],
    steps: [
      '上传超过普通上传阈值的测试文件，观察分片、合并和进度。',
      '从详情页打开视频预览并拖动播放进度。',
      '刷新页面后再次预览，确认签名地址可按规则重新获取。',
      '中断一次上传，确认过期分片会由清理机制处理。',
    ],
    expected: ['分片只在全部上传完成后合并。', '视频拖动时 Range 响应正常。', 'preview 与 download 用途不能互换。'],
    routeName: 'develop_database',
    actionLabel: '打开数据管理',
  },
  {
    id: 'query-center',
    title: '关键词与高级查询',
    category: 'query',
    icon: 'manage_search',
    status: 'ready',
    summary: '验证关键词、高级条件、排序和分页使用统一 Query 协议。',
    prerequisites: ['选择一个已有数据的标准列表页面。'],
    steps: [
      '输入关键词并查询，记录结果。',
      '打开高级查询，设置一组包含嵌套 AND/OR 的条件。',
      '保留高级条件再提交关键词查询。',
      '修改排序和分页后重复查询。',
    ],
    expected: ['关键词与高级条件做 AND 组合。', '高级条件不会被关键词查询清空。', '条件预览按层级展示真实逻辑。'],
    routeName: 'system_application',
    actionLabel: '打开标准列表',
  },
  {
    id: 'query-scheme',
    title: '查询方案',
    category: 'query',
    icon: 'bookmark',
    status: 'ready',
    summary: '验证个人、公共、角色和页面默认方案的保存、应用与权限。',
    prerequisites: ['当前账号可访问至少一个启用 Query Scope 的页面。'],
    steps: [
      '设置关键词、高级条件、排序和动态绑定后另存为个人方案。',
      '修改当前条件，检查 Dirty 状态并保存修改。',
      '从查询方案管理页查看方案详情、默认状态和可见范围。',
      '切换账号验证个人隔离和角色方案可见性。',
    ],
    expected: ['方案保存完整 Query 状态。', '个人方案不会被其他用户看到。', '无权限目标页面不能通过“使用方案”绕过。'],
    routeName: 'query_scheme_manager',
    actionLabel: '管理查询方案',
  },
  {
    id: 'notification',
    title: '消息通知',
    category: 'query',
    icon: 'notifications',
    status: 'data',
    summary: '验证未读数、最近消息、已读状态、用户隔离和安全跳转。',
    prerequisites: ['通过后端 NotificationService 为两个测试用户分别发送通知。'],
    steps: [
      '登录收件用户，检查 Header 未读数字和最近通知。',
      '标记一条已读，再执行全部已读。',
      '打开消息中心验证全部、未读和已读筛选。',
      '点击有权限和无权限的 Action 消息。',
    ],
    expected: ['用户只能看到自己的通知。', '已读操作幂等且未读数同步更新。', '无权限 Action 不跳转也不泄露目标数据。'],
    routeName: 'notification_center',
    actionLabel: '打开消息中心',
  },
  {
    id: 'report-runtime',
    title: '报表发布与运行',
    category: 'report',
    icon: 'analytics',
    status: 'configuration',
    summary: '验证当前轻量 Sheet 报表的设计、发布、运行、导出和执行日志。',
    prerequisites: ['准备一个只读 table 或受控 SQL 数据集。', '测试角色具有对应报表和数据权限。'],
    steps: [
      '在报表管理新建定义并进入设计器配置数据集、单元格和绑定。',
      '发布报表后从报表中心运行。',
      '验证运行结果、CSV 导出和执行日志。',
      '使用受限账号确认报表查询仍受菜单和数据权限约束。',
    ],
    expected: ['设计版本与已发布运行版本隔离。', '导出内容与运行结果一致。', '未授权用户不能运行或读取报表数据。'],
    routeName: 'report_manage',
    actionLabel: '打开报表管理',
  },
]

const filteredScenarios = computed(() => {
  const needle = keyword.value.trim().toLowerCase()
  return scenarios.filter((scenario) => {
    if (category.value !== 'all' && scenario.category !== category.value) return false
    if (!needle) return true
    return `${scenario.title} ${scenario.summary}`.toLowerCase().includes(needle)
  })
})

const selectedScenario = computed(() => {
  const visible = filteredScenarios.value
  return visible.find((scenario) => scenario.id === selectedId.value) || visible[0] || null
})

const openScenario = async (scenario: VerificationScenario) => {
  if (!scenario.routeName || !router.hasRoute(scenario.routeName)) {
    $q.notify({ type: 'warning', message: '当前账号没有可打开的相关页面' })
    return
  }
  await router.push({ name: scenario.routeName })
}
</script>

<style scoped lang="scss">
.verification-page {
  overflow: hidden;
}

.verification-workbench {
  display: grid;
  grid-template-columns: minmax(320px, 390px) minmax(0, 1fr);
  height: 100%;
  min-height: 0;
  overflow: hidden;
  border: 1px solid var(--app-border);
  background: var(--app-surface);
}

.verification-master,
.verification-detail {
  min-width: 0;
  min-height: 0;
}

.verification-detail {
  display: flex;
  flex-direction: column;
}

.verification-guide-note {
  flex: 0 0 auto;
  border-bottom: 1px solid var(--app-border);
  background: var(--app-surface-muted);
  color: var(--app-text-secondary);
}

.verification-master {
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--app-border);
}

.verification-toolbar {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 132px;
  gap: 8px;
  padding: 12px;
  border-bottom: 1px solid var(--app-border);
}

.verification-master-scroll,
.verification-detail-scroll {
  flex: 1;
  height: 100%;
  min-height: 0;
}

.verification-item--active {
  color: var(--q-primary);
  background: var(--app-primary-soft);
  box-shadow: inset 3px 0 0 var(--q-primary);
}

.verification-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 48px 16px;
  color: var(--app-text-muted);
}

.verification-detail-content {
  max-width: 1100px;
  margin: 0 auto;
  padding: 20px 24px 40px;
}

.verification-detail-header {
  display: flex;
  align-items: center;
  gap: 16px;
  padding-bottom: 18px;
  border-bottom: 1px solid var(--app-border);
}

.verification-detail-title-wrap {
  display: flex;
  align-items: center;
  gap: 14px;
  min-width: 0;
}

.verification-detail-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: 8px;
  color: #fff;
  background: var(--q-primary);
  font-size: 26px;
}

.verification-detail-title {
  color: var(--app-text-strong);
  font-size: 22px;
  font-weight: 700;
}

.verification-detail-summary {
  margin-top: 4px;
  color: var(--app-text-muted);
  line-height: 1.5;
}

.verification-section {
  padding: 20px 0;
  border-bottom: 1px solid var(--app-border);
}

.verification-section h3 {
  margin: 0 0 12px;
  color: var(--app-text-strong);
  font-size: 17px;
}

.verification-steps {
  display: grid;
  gap: 12px;
  margin: 0;
  padding-left: 28px;
  color: var(--app-text-strong);
  line-height: 1.65;
}

.verification-note {
  margin-top: 20px;
  border: 1px solid var(--app-primary-border);
  color: var(--app-text-strong);
  background: var(--app-primary-soft);
}

@media (max-width: 900px) {
  .verification-workbench {
    grid-template-columns: minmax(280px, 36%) minmax(0, 1fr);
  }

  .verification-toolbar {
    grid-template-columns: 1fr;
  }

  .verification-detail-header {
    align-items: flex-start;
    flex-wrap: wrap;
  }
}
</style>
