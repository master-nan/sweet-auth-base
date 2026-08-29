<template>
  <form-dialog-shell
    v-model="visible"
    title="接口定义详情"
    :subtitle="detail ? `${detail.interface_code} · v${detail.version}` : '正在读取技术契约'"
    icon="api"
    readonly
    :loading="loading"
    width="min(960px, calc(100vw - 48px))"
  >
    <div v-if="detail" class="interface-detail">
      <section>
        <div class="interface-detail__title">基础信息</div>
        <detail-field-grid :items="basicItems" />
      </section>
      <q-separator />
      <section>
        <div class="interface-detail__title">技术契约</div>
        <detail-field-grid :items="contractItems" />
      </section>
      <q-separator />
      <section>
        <div class="interface-detail__title">请求参数契约</div>
        <div class="interface-detail__hint">
          这里展示接口允许接收的参数名称、位置和类型，不展示任何一次执行的真实参数值。
        </div>
        <q-markup-table
          v-if="detail.input_contract.parameters.length"
          flat
          bordered
          dense
          class="interface-detail__parameters"
        >
          <thead>
            <tr>
              <th class="text-left">参数</th>
              <th class="text-left">位置</th>
              <th class="text-left">类型</th>
              <th class="text-center">必填</th>
              <th class="text-center">允许多值</th>
              <th class="text-right">最大长度</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="parameter in detail.input_contract.parameters"
              :key="`${parameter.location}:${parameter.code}`"
            >
              <td>
                <div class="text-weight-medium">{{ parameter.name || parameter.code }}</div>
                <div v-if="parameter.name" class="text-caption text-grey-7 text-mono">
                  {{ parameter.code }}
                </div>
              </td>
              <td>{{ locationLabels[parameter.location] }}</td>
              <td>{{ dataTypeLabels[parameter.data_type] }}</td>
              <td class="text-center">{{ parameter.required ? '是' : '否' }}</td>
              <td class="text-center">{{ parameter.allow_multiple ? '是' : '否' }}</td>
              <td class="text-right">{{ parameter.max_length || '-' }}</td>
            </tr>
          </tbody>
        </q-markup-table>
        <div v-else class="interface-detail__empty">该接口没有声明可变请求参数。</div>
      </section>
      <q-separator />
      <section>
        <div class="interface-detail__title">响应处理</div>
        <div class="interface-detail__hint">
          平台按“响应大小限制”读取结果。执行详情只展示 HTTP 状态、响应大小、Hash
          和安全摘要，原始响应体不作为管理页面内容保存。
        </div>
      </section>
    </div>
    <div v-else class="interface-detail__loading">
      <q-spinner-dots color="primary" size="36px" />
    </div>
  </form-dialog-shell>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import FormDialogShell from 'src/components/FormDialog/FormDialogShell.vue'
import DetailFieldGrid from 'src/components/Detail/DetailFieldGrid.vue'
import type { DetailFieldItem } from 'src/components/Detail/types'
import {
  type InterfaceDefinitionDetail,
  type InterfaceInputDataType,
  type InterfaceInputLocation,
  useIntegrationApi,
} from 'src/api/services/integration'

const props = defineProps<{ modelValue: boolean; id: number }>()
const emit = defineEmits<{ (event: 'update:modelValue', value: boolean): void }>()
const api = useIntegrationApi()
const detail = ref<InterfaceDefinitionDetail | null>(null)
const loading = ref(false)
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})
const statusLabels = { draft: '草稿', enabled: '已启用', disabled: '已停用' }
const locationLabels: Record<InterfaceInputLocation, string> = {
  path: 'Path',
  query: 'Query',
  header: 'Header',
  body: 'JSON Body',
}
const dataTypeLabels: Record<InterfaceInputDataType, string> = {
  string: '字符串',
  integer: '整数',
  number: '数值',
  boolean: '布尔',
  object: '对象',
  array: '数组',
}
const basicItems = computed<DetailFieldItem[]>(() =>
  detail.value
    ? [
        {
          label: '所属系统',
          value: `${detail.value.external_system.name}（${detail.value.external_system.system_code}）`,
        },
        { label: '接口编码', value: detail.value.interface_code },
        { label: '接口名称', value: detail.value.name },
        { label: '版本', value: `v${detail.value.version}` },
        { label: '状态', value: statusLabels[detail.value.status] },
        { label: '更新时间', value: detail.value.gmt_modify },
      ]
    : [],
)
const contractItems = computed<DetailFieldItem[]>(() =>
  detail.value
    ? [
        { label: '协议', value: detail.value.protocol.toUpperCase() },
        { label: 'HTTP Method', value: detail.value.http_method },
        { label: '相对路径', value: detail.value.relative_path },
        { label: '超时', value: `${detail.value.timeout_seconds} 秒` },
        {
          label: '响应大小限制',
          value: `${(detail.value.response_limit / 1024).toLocaleString()} KiB`,
        },
        {
          label: '认证引用',
          value: detail.value.credential
            ? `${detail.value.credential.name}（${detail.value.credential.credential_code}）`
            : '未配置',
        },
        { label: '凭证状态', value: detail.value.credential?.effective_status || '-' },
        {
          label: '重试策略',
          value: detail.value.retry_policy
            ? `${detail.value.retry_policy.policy_name}（${detail.value.retry_policy.policy_code} · v${detail.value.retry_policy.version}）`
            : '未配置',
        },
        {
          label: '策略状态',
          value: detail.value.retry_policy ? statusLabels[detail.value.retry_policy.status] : '-',
        },
        { label: '描述', value: detail.value.description || '-', fullWidth: true },
      ]
    : [],
)

const load = async () => {
  if (!props.id) return
  loading.value = true
  try {
    detail.value = (await api.getInterfaceDefinition(props.id)).data
  } finally {
    loading.value = false
  }
}
watch(
  () => [props.modelValue, props.id] as const,
  ([open]) => {
    if (open) void load()
    else detail.value = null
  },
  { immediate: true },
)
</script>

<style scoped lang="scss">
.interface-detail {
  display: grid;
  gap: 24px;
  padding: 4px 6px 20px;
}
.interface-detail__title {
  margin-bottom: 16px;
  font-size: 16px;
  font-weight: 700;
}
.interface-detail__hint {
  margin: -6px 0 14px;
  color: var(--app-text-muted);
  font-size: 13px;
  line-height: 1.7;
}
.interface-detail__parameters {
  background: var(--app-surface);
}
.interface-detail__empty {
  padding: 18px;
  border: 1px dashed var(--app-border);
  color: var(--app-text-muted);
  text-align: center;
}
.interface-detail__loading {
  min-height: 260px;
  display: grid;
  place-items: center;
}
</style>
