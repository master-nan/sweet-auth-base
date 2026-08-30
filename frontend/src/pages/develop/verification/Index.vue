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
                  :label="scenarioChip(scenario).label"
                  :color="scenarioChip(scenario).color"
                  :outline="scenarioChip(scenario).outline"
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
          带“准备样例”按钮的场景会创建独立的功能验证数据，支持重复准备和清理；其他场景仍按页面中的准备项手工验证。生产环境不会开放样例操作。
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
              <div class="verification-actions">
                <q-btn
                  v-if="selectedScenario.sampleId"
                  flat
                  round
                  color="primary"
                  icon="refresh"
                  :loading="loadingStatuses"
                  @click="loadSampleStatuses"
                >
                  <q-tooltip>刷新样例状态</q-tooltip>
                </q-btn>
                <q-btn
                  v-if="
                    selectedScenario.sampleId && sampleStatus(selectedScenario)?.state !== 'empty'
                  "
                  outline
                  color="negative"
                  icon="delete_sweep"
                  label="清理样例"
                  :loading="cleaningId === selectedScenario.sampleId"
                  :disable="sampleStatus(selectedScenario)?.available === false"
                  @click="confirmCleanup(selectedScenario)"
                />
                <q-btn
                  v-if="selectedScenario.sampleId"
                  unelevated
                  color="primary"
                  icon="playlist_add_check"
                  :label="
                    sampleStatus(selectedScenario)?.state === 'ready' ? '重新准备' : '准备样例'
                  "
                  :loading="preparingId === selectedScenario.sampleId"
                  :disable="sampleStatus(selectedScenario)?.available === false"
                  @click="prepareSample(selectedScenario)"
                />
                <q-btn
                  v-if="
                    selectedScenario.id === 'integration-call' &&
                    sampleStatus(selectedScenario)?.state === 'ready'
                  "
                  outline
                  color="primary"
                  icon="play_arrow"
                  label="运行连通性测试"
                  :loading="runningIntegrationSample"
                  @click="runIntegrationSample"
                />
                <q-btn
                  v-if="selectedScenario.routeName"
                  unelevated
                  color="primary"
                  icon-right="open_in_new"
                  :label="selectedScenario.actionLabel || '打开相关页面'"
                  @click="openScenario(selectedScenario)"
                />
              </div>
            </header>

            <section v-if="selectedScenario.sampleId" class="verification-sample-status">
              <div class="verification-sample-status__main">
                <status-chip
                  :label="scenarioChip(selectedScenario).label"
                  :color="scenarioChip(selectedScenario).color"
                  :outline="scenarioChip(selectedScenario).outline"
                />
                <span>{{ sampleStatus(selectedScenario)?.summary || '正在读取样例状态' }}</span>
              </div>
              <div
                v-if="sampleStatus(selectedScenario)?.details?.length"
                class="verification-sample-facts"
              >
                <div v-for="detail in sampleStatus(selectedScenario)?.details" :key="detail.label">
                  <span>{{ detail.label }}</span>
                  <strong>{{ detail.value }}</strong>
                </div>
              </div>
            </section>

            <section
              v-if="selectedScenario.sampleFiles?.length"
              class="verification-sample-files"
            >
              <div>
                <strong>测试文件</strong>
                <span>下载后直接按下方步骤上传，不需要自行寻找样例。</span>
              </div>
              <div class="verification-sample-files__actions">
                <q-btn
                  v-for="file in selectedScenario.sampleFiles"
                  :key="file.fileName"
                  outline
                  color="primary"
                  icon="download"
                  :label="file.label"
                  @click="downloadSampleFile(file)"
                />
              </div>
            </section>

            <section class="verification-section">
              <h3>开始前准备</h3>
              <q-list dense>
                <q-item v-for="item in selectedScenario.prerequisites" :key="item">
                  <q-item-section avatar
                    ><q-icon name="check_circle_outline" color="primary"
                  /></q-item-section>
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
                  <q-item-section avatar
                    ><q-icon name="task_alt" color="positive"
                  /></q-item-section>
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

    <q-dialog v-model="accountDialog" persistent>
      <q-card class="verification-account-dialog">
        <q-card-section class="row items-start q-gutter-md">
          <q-icon name="password" color="primary" size="36px" />
          <div>
            <div class="text-h6">功能验证账号</div>
            <div class="text-caption text-grey-7">密码只在本次准备完成后显示，请先记下再关闭。</div>
          </div>
        </q-card-section>
        <q-separator />
        <q-card-section class="q-pa-none">
          <q-list separator>
            <q-item v-for="account in preparedAccounts" :key="account.user_name" class="q-py-md">
              <q-item-section>
                <q-item-label class="text-weight-bold">{{ account.user_name }}</q-item-label>
                <q-item-label caption>{{ account.role }}</q-item-label>
                <q-item-label caption>{{ account.expected }}</q-item-label>
              </q-item-section>
              <q-item-section side>
                <div class="verification-password">
                  <code>{{ account.password }}</code>
                  <q-btn
                    flat
                    round
                    dense
                    icon="content_copy"
                    color="primary"
                    @click="copyPassword(account)"
                  >
                    <q-tooltip>复制账号和密码</q-tooltip>
                  </q-btn>
                </div>
              </q-item-section>
            </q-item>
          </q-list>
        </q-card-section>
        <q-separator />
        <q-card-actions align="right">
          <q-btn flat color="primary" label="我已记下" @click="accountDialog = false" />
        </q-card-actions>
      </q-card>
    </q-dialog>
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'develop_verification_page' })

import { computed, onMounted, ref } from 'vue'
import { copyToClipboard, useQuasar } from 'quasar'
import { useRouter } from 'vue-router'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import StatusChip from 'src/components/Display/StatusChip.vue'
import {
  useDevelopmentVerificationApi,
  type VerificationSampleAccount,
  type VerificationSampleScenario,
  type VerificationSampleStatus,
} from 'src/api/services/development-verification'
import { useIntegrationApi } from 'src/api/services/integration'

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
  sampleId?: VerificationSampleScenario
  sampleFiles?: VerificationSampleFile[]
}

interface VerificationSampleFile {
  label: string
  fileName: string
  url?: string
  generatedSizeMiB?: number
}

const router = useRouter()
const $q = useQuasar()
const verificationApi = useDevelopmentVerificationApi()
const integrationApi = useIntegrationApi()
const keyword = ref('')
const category = ref('all')
const selectedId = ref('permission-page')
const loadingStatuses = ref(false)
const preparingId = ref<VerificationSampleScenario | null>(null)
const cleaningId = ref<VerificationSampleScenario | null>(null)
const statuses = ref<Partial<Record<VerificationSampleScenario, VerificationSampleStatus>>>({})
const accountDialog = ref(false)
const preparedAccounts = ref<VerificationSampleAccount[]>([])
const runningIntegrationSample = ref(false)

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

const sampleStateLabels = {
  empty: { label: '未准备', color: 'grey-7', outline: true },
  partial: { label: '需要补齐', color: 'warning', outline: true },
  ready: { label: '样例已就绪', color: 'positive', outline: false },
  unavailable: { label: '当前环境不可用', color: 'grey-7', outline: true },
} as const

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
    expected: [
      '页面可见性与角色菜单一致。',
      '按钮可见性与菜单按钮授权一致。',
      '直接调用未授权 API 会被拒绝。',
    ],
    routeName: 'system_role',
    actionLabel: '打开角色管理',
  },
  {
    id: 'data-permission',
    sampleId: 'data-permission',
    title: '数据权限范围',
    category: 'permission',
    icon: 'rule',
    status: 'data',
    summary: '为同一个业务页面配置不同数据范围，比较两个用户看到的数据。',
    prerequisites: [
      '点击“准备样例”，记下弹窗中的两个临时账号。',
      '重新登录后从“开发管理”打开功能验证订单页面。',
    ],
    steps: [
      '先用华东账号登录，打开“功能验证-数据权限订单”，确认只能看到 EAST 的两条订单。',
      '退出后使用全部订单账号登录，打开同一页面，确认可以看到三条订单。',
      '分别打开订单详情，确认列表和详情使用同一数据范围。',
      '回到管理员账号，在数据权限页面查看样例资源、归属、策略和授权是如何连接的。',
    ],
    expected: [
      '管理员看到完整数据。',
      '受限用户只看到规则允许的数据。',
      '详情、编辑和删除接口同样不能越权。',
    ],
    note: '数据权限测试不依赖组织同步本身，但按公司或组织过滤时，必须先有真实组织、用户任职和业务记录归属数据。',
    routeName: 'lowcode_verify_permission_order',
    actionLabel: '打开样例订单',
  },
  {
    id: 'tms-company-scope',
    sampleId: 'tms-company-scope',
    title: 'TMS 车辆公司范围',
    category: 'permission',
    icon: 'local_shipping',
    status: 'data',
    summary: '验证车辆列表默认受当前用户公司范围约束，多公司用户可以查看授权公司集合。',
    prerequisites: [
      '确保 tms_company 和 tms_vehicle 已在数据管理初始化并发布。',
      '点击“准备样例”，记下两个临时账号。',
    ],
    steps: [
      '使用华东账号登录 TMS 车辆页面，确认只看到华东公司的两辆车。',
      '使用多公司账号登录同一页面，确认看到华东和华南共三辆车。',
      '在公司查询条件中选择华西公司，确认未授权车辆仍不会返回。',
      '回到管理员账号查看 verify_tms_vehicle 资源，理解 company_id 与策略值如何组合。',
    ],
    expected: [
      '单公司用户默认只返回公司 A。',
      '多公司用户可返回 A、B 的并集。',
      '公司 C 即使通过请求参数传入也不会返回。',
    ],
    note: '查询条件可以展示当前授权公司，但真正的限制必须由后端 Data Permission 与页面条件做 AND 组合。',
    routeName: 'lowcode_tms_vehicle',
    actionLabel: '打开 TMS 车辆',
  },
  {
    id: 'metadata-low-code',
    sampleId: 'metadata-low-code',
    title: 'Metadata 与低代码页面',
    category: 'data',
    icon: 'dynamic_form',
    status: 'data',
    summary: '验证字段配置如何影响列表、查询、新增、编辑和详情。',
    prerequisites: [
      '点击“准备样例”，系统会创建两张 verify 表、一个字典和两条记录。',
      '样例页面首次发布后需要重新登录一次，以加载新菜单。',
    ],
    steps: [
      '打开“功能验证-低代码记录”，观察文本、数字、日期、布尔、字典和 Relation 的展示。',
      '新增一条记录，检查日期组件、状态下拉和分类下拉是否可用。',
      '编辑并打开详情，确认字典与 Relation 显示名称而不是原始编码和外键。',
      '在数据管理查看 verify_lowcode_record 字段配置，再回到页面对照效果。',
    ],
    expected: [
      '页面只显示元数据允许的字段。',
      '字典和 Relation 显示业务名称而不是编码或外键。',
      '字段校验与后端类型约束一致。',
    ],
    routeName: 'lowcode_verify_lowcode_record',
    actionLabel: '打开低代码样例',
  },
  {
    id: 'organization-sync',
    sampleId: 'organization-sync',
    title: '组织、人事与岗位同步',
    category: 'integration',
    icon: 'account_tree',
    status: 'configuration',
    summary: '通过本地 HR 接口按真实同步链路验证法人架构、管理架构、岗位和人员。',
    prerequisites: [
      '点击“准备样例”，系统会创建独立的 verify_hr_source、7 个接口和 7 个手工同步任务。',
      '默认 Docker 环境会启用 Integration Worker、同步 Runner 和组织 HR Consumer。',
    ],
    steps: [
      '打开同步任务，按法人公司、管理公司、法人部门、管理部门、岗位、员工、离职员工的顺序逐个点击手工执行。',
      '在同步批次等待每个任务完成，查看总数、成功数、忽略数、失败数和 Checkpoint。',
      '在组织架构切换法人架构与管理架构，核对两棵树；再到岗位和人员页面核对档案。',
      '重复执行一遍全部任务，确认相同 SourceKey 不产生重复组织、岗位或人员。',
    ],
    expected: [
      '法人架构和管理架构可以分别浏览。',
      '同一组织事实不会因重复同步产生重复记录。',
      '同步批次和执行详情能够追溯每次真实 HTTP 调用及消费结果。',
    ],
    note: '当前 HR Consumer 合同覆盖法人公司、管理公司、两类部门、岗位、在职和离职人员。人员任职仍是独立业务合同，本样例不伪造任职关系。',
    routeName: 'organization_sync_batch',
    actionLabel: '打开同步批次',
  },
  {
    id: 'integration-call',
    sampleId: 'integration-call',
    title: '外部接口调用',
    category: 'integration',
    icon: 'hub',
    status: 'configuration',
    summary: '用本地 JSON 接口验证系统、加密凭证、接口、重试策略和执行记录。',
    prerequisites: [
      '点击“准备样例”，系统会创建 verify_integration_source 及其测试凭证、重试策略和 verify_ping 接口。',
      '样例只连接默认 Docker 内的静态 JSON 地址，不使用真实生产地址或密钥。',
    ],
    steps: [
      '打开接口定义，搜索 verify_ping，核对 Method、相对路径、凭证和重试策略。',
      '点击接口的手工执行按钮，并在执行记录查看输入安全摘要和状态。',
      '进入 Attempt 调用日志核对 HTTP 状态、耗时和错误分类。',
      '可复制接口后临时填写一个不存在的路径，验证技术错误和重试 Attempt；不要修改已启用的样例接口。',
    ],
    expected: [
      '凭证正文不会出现在列表、详情或日志。',
      '执行详情可滚动并完整展示安全摘要。',
      '业务失败与技术重试不会混淆。',
    ],
    routeName: 'integration_execution',
    actionLabel: '打开执行记录',
  },
  {
    id: 'file-upload',
    sampleId: 'file-upload',
    title: '文件上传、下载与删除',
    category: 'file',
    icon: 'upload_file',
    status: 'data',
    summary: '通过低代码文件字段验证普通上传、权限访问和删除生命周期。',
    prerequisites: ['点击“准备样例”，系统会发布功能验证文件页面并创建一条可编辑记录。'],
    steps: [
      '下载页面提供的小文件，打开样例页面并编辑“功能验证-上传测试”记录。',
      '在普通文件字段选择该文件，保存并完成普通上传。',
      '保存记录后从详情页预览或下载文件。',
      '复制文件地址后退出登录再访问，确认不能仅凭文件地址越权读取。',
      '删除或替换文件，确认页面引用随记录更新。',
    ],
    expected: [
      '有权限用户可以按用途预览或下载。',
      '无权限用户不能仅凭文件 ID 访问。',
      '删除或替换后旧引用不再出现在记录中。',
    ],
    sampleFiles: [
      {
        label: '下载小文件',
        fileName: 'sweet-admin-file-upload-sample.txt',
        url: '/verification-fixtures/files/sample-small.txt',
      },
      {
        label: '生成 6 MiB 文件',
        fileName: 'sweet-admin-chunk-upload-sample.txt',
        generatedSizeMiB: 6,
      },
    ],
    routeName: 'lowcode_verify_file_record',
    actionLabel: '打开文件样例',
  },
  {
    id: 'video-preview',
    sampleId: 'video-preview',
    title: '视频与大文件预览',
    category: 'file',
    icon: 'movie',
    status: 'data',
    summary: '验证视频 Range 请求、预览权限以及大文件分片上传。',
    prerequisites: [
      '点击“准备样例”，系统会发布与普通文件共用的低代码样例页面。',
      '下载页面提供的无敏感内容 MP4；该文件超过样例字段 0.1 MiB 的分片阈值。',
    ],
    steps: [
      '下载 MP4，打开样例页面并编辑“功能验证-上传测试”记录。',
      '在 MP4 视频字段上传该文件，观察分片、合并和进度。',
      '从详情页打开视频预览并拖动播放进度。',
      '刷新页面后再次预览，确认签名地址可按规则重新获取。',
      '重新选择文件并中断上传，确认未完成上传不会写入业务记录。',
    ],
    expected: [
      '分片只在全部上传完成后合并。',
      '视频拖动时 Range 响应正常。',
      'preview 与 download 用途不能互换。',
    ],
    sampleFiles: [
      {
        label: '下载 MP4 视频',
        fileName: 'sweet-admin-video-preview-sample.mp4',
        url: '/verification-fixtures/files/sample-video.mp4',
      },
    ],
    routeName: 'lowcode_verify_file_record',
    actionLabel: '打开视频样例',
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
    expected: [
      '关键词与高级条件做 AND 组合。',
      '高级条件不会被关键词查询清空。',
      '条件预览按层级展示真实逻辑。',
    ],
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
    expected: [
      '方案保存完整 Query 状态。',
      '个人方案不会被其他用户看到。',
      '无权限目标页面不能通过“使用方案”绕过。',
    ],
    routeName: 'query_scheme_manager',
    actionLabel: '管理查询方案',
  },
  {
    id: 'notification',
    sampleId: 'notification',
    title: '消息通知',
    category: 'query',
    icon: 'notifications',
    status: 'data',
    summary: '验证未读数、最近消息、已读状态、用户隔离和安全跳转。',
    prerequisites: ['点击“准备样例”，系统会给当前账号发送系统、业务和提醒三类通知。'],
    steps: [
      '观察 Header 未读数字和最近通知是否立即增加。',
      '标记一条已读，再执行全部已读。',
      '打开消息中心验证全部、未读和已读筛选。',
      '点击有权限和无权限的 Action 消息。',
    ],
    expected: [
      '用户只能看到自己的通知。',
      '已读操作幂等且未读数同步更新。',
      '无权限 Action 不跳转也不泄露目标数据。',
    ],
    routeName: 'notification_center',
    actionLabel: '打开消息中心',
  },
  {
    id: 'report-runtime',
    title: '报表发布与运行',
    category: 'report',
    icon: 'analytics',
    status: 'ready',
    summary: '验证当前轻量 Sheet 报表的设计、发布、运行、导出和执行日志。',
    prerequisites: [
      '系统初始化已内置并发布“访问日志概览”报表，不需要额外准备数据集。',
      '先在系统中进行几次查询或保存操作，为访问日志产生可观察的数据。',
    ],
    steps: [
      '在报表中心打开“访问日志概览”，设置关键字后运行报表。',
      '验证运行结果、CSV 导出和执行日志。',
      '在报表管理打开该定义，对照 table 数据源、查询参数、Sheet 单元格和绑定配置。',
      '复制一份报表进行修改并发布，确认草稿不会影响原已发布版本。',
    ],
    expected: [
      '设计版本与已发布运行版本隔离。',
      '导出内容与运行结果一致。',
      '未授权用户不能运行或读取报表数据。',
    ],
    routeName: 'report_center',
    actionLabel: '打开报表中心',
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

const sampleStatus = (scenario: VerificationScenario) => {
  if (!scenario.sampleId) return undefined
  return statuses.value[scenario.sampleId]
}

const scenarioChip = (scenario: VerificationScenario) => {
  if (scenario.sampleId) {
    const status = sampleStatus(scenario)
    if (!status) return { label: '读取中', color: 'grey-7', outline: true }
    return sampleStateLabels[status.state]
  }
  const status = statusLabels[scenario.status]
  return { ...status, outline: scenario.status !== 'ready' }
}

const loadSampleStatuses = async () => {
  loadingStatuses.value = true
  try {
    const response = await verificationApi.statuses()
    const next: Partial<Record<VerificationSampleScenario, VerificationSampleStatus>> = {}
    for (const status of response.data || []) next[status.scenario_id] = status
    statuses.value = next
  } finally {
    loadingStatuses.value = false
  }
}

const prepareSample = async (scenario: VerificationScenario) => {
  if (!scenario.sampleId) return
  preparingId.value = scenario.sampleId
  try {
    const response = await verificationApi.prepare(scenario.sampleId)
    if (!response.data) return
    statuses.value = { ...statuses.value, [scenario.sampleId]: response.data.status }
    preparedAccounts.value = response.data.accounts || []
    accountDialog.value = preparedAccounts.value.length > 0
    $q.notify({ type: 'positive', message: `${scenario.title}样例已准备` })
  } finally {
    preparingId.value = null
  }
}

const cleanupSample = async (scenario: VerificationScenario) => {
  if (!scenario.sampleId) return
  cleaningId.value = scenario.sampleId
  try {
    const response = await verificationApi.cleanup(scenario.sampleId)
    if (response.data) statuses.value = { ...statuses.value, [scenario.sampleId]: response.data }
    $q.notify({ type: 'positive', message: `${scenario.title}样例已清理` })
  } finally {
    cleaningId.value = null
  }
}

const confirmCleanup = (scenario: VerificationScenario) => {
  $q.dialog({
    title: '清理功能验证样例',
    message:
      '只会删除本工作台创建的 verify_ 配置、数据和功能验证账号，不会删除现有业务数据。文件与视频共用同一张样例表，清理其中一个会同时清理两项。',
    cancel: true,
    persistent: true,
  }).onOk(() => {
    void cleanupSample(scenario)
  })
}

const copyPassword = async (account: VerificationSampleAccount) => {
  await copyToClipboard(`账号：${account.user_name}\n密码：${account.password}`)
  $q.notify({ type: 'positive', message: `已复制 ${account.user_name} 的账号和密码` })
}

const downloadSampleFile = async (file: VerificationSampleFile) => {
  try {
    let blob: Blob
    if (file.generatedSizeMiB) {
      const bytes = Math.max(1, Math.round(file.generatedSizeMiB * 1024 * 1024))
      const marker = 'Sweet Admin chunk upload verification sample.\n'
      blob = new Blob([marker.repeat(Math.ceil(bytes / marker.length)).slice(0, bytes)], {
        type: 'text/plain;charset=utf-8',
      })
    } else if (file.url) {
      const response = await fetch(file.url)
      if (!response.ok) throw new Error(`HTTP ${response.status}`)
      blob = await response.blob()
    } else {
      throw new Error('样例文件地址未配置')
    }
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = file.fileName
    document.body.appendChild(link)
    link.click()
    link.remove()
    URL.revokeObjectURL(url)
  } catch {
    $q.notify({ type: 'negative', message: '样例文件下载失败，请确认前端静态资源已更新' })
  }
}

const runIntegrationSample = async () => {
  runningIntegrationSample.value = true
  try {
    const systems = await integrationApi.queryExternalSystems({
      page: 1,
      num: 100,
      expressions: [],
    })
    const system = systems.data?.find((item) => item.system_code === 'verify_integration_source')
    if (!system) throw new Error('找不到功能验证外部系统')

    const definitions = await integrationApi.queryInterfaceDefinitions({
      page: 1,
      num: 100,
      expressions: [],
      external_system_id: system.id,
    })
    const definition = definitions.data?.find((item) => item.interface_code === 'verify_ping')
    if (!definition) throw new Error('找不到功能验证接口')

    const result = await integrationApi.createExecution({
      external_system_id: system.id,
      interface_definition_id: definition.id,
      trigger_source: 'manual',
      idempotency_scope: 'development_verification',
      idempotency_key: `verify-ping-${Date.now()}`,
      input: { path_params: {}, query_params: {}, headers: {} },
    })
    if (!result.success || !result.data) throw new Error('创建执行记录失败')
    $q.notify({ type: 'positive', message: `已创建连通性测试 ${result.data.execution_no}` })
    await router.push({ name: 'integration_execution_detail_page', params: { id: result.data.id } })
  } catch (error) {
    $q.notify({
      type: 'negative',
      message: error instanceof Error ? error.message : '运行连通性测试失败',
    })
  } finally {
    runningIntegrationSample.value = false
  }
}

const openScenario = async (scenario: VerificationScenario) => {
  if (!scenario.routeName || !router.hasRoute(scenario.routeName)) {
    const prepared = scenario.sampleId && sampleStatus(scenario)?.state === 'ready'
    $q.notify({
      type: 'warning',
      message: prepared ? '样例页面刚创建，请重新登录以加载新菜单' : '当前账号没有可打开的相关页面',
    })
    return
  }
  await router.push({ name: scenario.routeName })
}

onMounted(loadSampleStatuses)
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

.verification-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  flex-wrap: wrap;
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

.verification-sample-status {
  padding: 16px;
  margin-top: 16px;
  border: 1px solid var(--app-border);
  border-radius: 6px;
  background: var(--app-surface-muted);
}

.verification-sample-status__main {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--app-text-strong);
}

.verification-sample-files {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 16px;
  margin-top: 12px;
  border: 1px solid var(--app-border);
  border-radius: 6px;
}

.verification-sample-files > div:first-child {
  display: grid;
  gap: 4px;
}

.verification-sample-files span {
  color: var(--app-text-muted);
  font-size: 12px;
}

.verification-sample-files__actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.verification-sample-facts {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  margin-top: 14px;
}

.verification-sample-facts > div {
  display: grid;
  gap: 4px;
  min-width: 0;
}

.verification-sample-facts span {
  color: var(--app-text-muted);
  font-size: 12px;
}

.verification-sample-facts strong {
  overflow-wrap: anywhere;
  color: var(--app-text-strong);
  font-weight: 600;
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

.verification-account-dialog {
  width: min(720px, calc(100vw - 32px));
  max-width: 720px;
}

.verification-password {
  display: flex;
  align-items: center;
  gap: 4px;
}

.verification-password code {
  padding: 5px 8px;
  border: 1px solid var(--app-border);
  border-radius: 4px;
  color: var(--app-text-strong);
  background: var(--app-surface-muted);
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

  .verification-actions {
    width: 100%;
    justify-content: flex-start;
  }

  .verification-sample-facts {
    grid-template-columns: 1fr;
  }

  .verification-sample-files {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
