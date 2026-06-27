<template>
  <base-content class="q-pa-sm configure-page" scrollable>
    <q-form ref="formRef" class="configure-shell" @submit.prevent="saveConfig">
      <section class="configure-hero">
        <div class="configure-hero__icon">
          <q-icon name="tune" />
        </div>
        <div class="configure-hero__content">
          <div class="configure-hero__title">配置管理</div>
          <div class="configure-hero__subtitle">系统参数、登录策略和邮件服务集中维护</div>
        </div>
        <div class="configure-hero__actions">
          <q-btn
            outline
            color="primary"
            icon="refresh"
            label="刷新"
            :loading="loading"
            @click="refreshConfig"
          />
        </div>
      </section>

      <div class="configure-layout">
        <aside class="configure-sidebar">
          <button
            v-for="section in configSections"
            :key="section.name"
            type="button"
            class="configure-nav-item"
            :class="{ 'configure-nav-item--active': activeTab === section.name }"
            @click="activeTab = section.name"
          >
            <span class="configure-nav-item__icon">
              <q-icon :name="section.icon" />
            </span>
            <span class="configure-nav-item__body">
              <span class="configure-nav-item__title">{{ section.label }}</span>
              <span class="configure-nav-item__caption">{{ section.caption }}</span>
            </span>
            <q-chip dense square :color="section.color" text-color="white">
              {{ section.status }}
            </q-chip>
          </button>

          <div class="configure-sidebar-note">
            <q-icon name="verified_user" />
            <div>
              <div class="configure-sidebar-note__title">运行时配置</div>
              <div class="configure-sidebar-note__text">
                保存后会刷新前端配置缓存，登录策略即时生效。
              </div>
            </div>
          </div>
        </aside>

        <main class="configure-main">
          <div class="configure-section-head">
            <div>
              <div class="configure-section-head__title">{{ currentSection.label }}</div>
              <div class="configure-section-head__caption">{{ currentSection.caption }}</div>
            </div>
            <q-icon :name="currentSection.icon" />
          </div>

          <div class="configure-overview">
            <div v-for="item in overviewItems" :key="item.label" class="configure-metric">
              <q-icon :name="item.icon" :color="item.color" />
              <div>
                <div class="configure-metric__value">{{ item.value }}</div>
                <div class="configure-metric__label">{{ item.label }}</div>
              </div>
            </div>
          </div>

          <section v-show="activeTab === 'security'" class="configure-panel">
            <div class="setting-row">
              <div class="setting-row__main">
                <span class="setting-row__icon">
                  <q-icon name="shield" />
                </span>
                <div>
                  <div class="setting-row__title">登录验证码</div>
                  <div class="setting-row__caption">开启后登录页会出现验证码输入与校验</div>
                </div>
              </div>
              <q-toggle
                v-model="formData.enable_captcha"
                color="primary"
                keep-color
                :label="formData.enable_captcha ? '已开启' : '已关闭'"
              />
            </div>

            <div class="field-grid security-field-grid">
              <q-select
                v-model="formData.password_policy"
                class="field-grid__wide"
                :options="policyOptions"
                label="密码策略"
                outlined
                dense
                hide-bottom-space
                emit-value
                map-options
                :rules="[(val) => String(val ?? '').trim().length > 0 || '请选择密码策略']"
                @update:model-value="syncPasswordPolicyPreset"
              >
                <template #option="scope">
                  <q-item v-bind="scope.itemProps">
                    <q-item-section>
                      <q-item-label>{{ scope.opt.label }}</q-item-label>
                      <q-item-label caption>{{ scope.opt.description }}</q-item-label>
                    </q-item-section>
                  </q-item>
                </template>
                <template #append>
                  <q-icon name="policy" />
                </template>
              </q-select>

              <div class="password-policy-card field-grid__wide">
                <div class="password-policy-card__head">
                  <q-chip dense square color="primary" text-color="white">
                    {{ currentPolicyPreset.shortLabel }}
                  </q-chip>
                  <div>
                    <div class="password-policy-card__title">{{ passwordPolicySummary }}</div>
                    <div class="password-policy-card__desc">
                      {{ currentPolicyPreset.description }}
                    </div>
                  </div>
                </div>
                <div class="password-policy-card__rule">
                  <span>校验表达式</span>
                  <code>{{ passwordPolicyRegex }}</code>
                </div>
              </div>

              <template v-if="isCustomPasswordPolicy">
                <q-input
                  v-model.number="formData.password_length"
                  type="number"
                  label="自定义最小长度"
                  outlined
                  dense
                  hide-bottom-space
                  :rules="[(val) => Number(val) >= 6 || '密码长度不能小于6位']"
                >
                  <template #append>
                    <q-icon name="password" />
                  </template>
                </q-input>

                <q-select
                  v-model="formData.password_complexity"
                  :options="complexityOptions"
                  label="自定义复杂度"
                  outlined
                  dense
                  hide-bottom-space
                  emit-value
                  map-options
                >
                  <template #append>
                    <q-icon name="security" />
                  </template>
                </q-select>
              </template>

              <q-input
                v-model.number="formData.password_expire_time"
                type="number"
                label="密码过期时间（天）"
                outlined
                dense
                hide-bottom-space
                :rules="[(val) => Number(val) >= 0 || '过期时间不能为负数']"
              >
                <template #append>
                  <q-icon name="timer" />
                </template>
              </q-input>

              <q-input
                v-model.number="formData.password_error_count"
                type="number"
                label="密码错误锁定次数"
                outlined
                dense
                hide-bottom-space
                :rules="[(val) => Number(val) > 0 || '错误次数必须大于0']"
              >
                <template #append>
                  <q-icon name="lock" />
                </template>
              </q-input>

              <q-input
                v-model.number="formData.password_lock_minutes"
                type="number"
                label="锁定时长（分钟）"
                outlined
                dense
                hide-bottom-space
                :rules="[(val) => Number(val) > 0 || '锁定时长必须大于0']"
              >
                <template #append>
                  <q-icon name="lock_clock" />
                </template>
              </q-input>

              <div class="login-lock-card field-grid__wide">
                <span class="login-lock-card__icon">
                  <q-icon name="lock_clock" />
                </span>
                <div class="login-lock-card__body">
                  <div class="login-lock-card__title">
                    连续失败 {{ formData.password_error_count }} 次后锁定
                    {{ formData.password_lock_minutes }} 分钟
                  </div>
                  <div class="login-lock-card__text">
                    到期会自动解锁；管理员也可以在用户管理行操作中解除锁定。
                  </div>
                </div>
              </div>
            </div>
          </section>

          <section v-show="activeTab === 'system'" class="configure-panel">
            <div class="system-brand-row">
              <div class="system-logo-preview">
                <img
                  v-if="systemConfig.system_logo"
                  :src="systemConfig.system_logo"
                  alt="系统Logo"
                />
                <q-icon v-else name="hub" />
              </div>
              <div>
                <div class="system-brand-row__title">
                  {{ systemConfig.system_name || 'Sweet Admin' }}
                </div>
                <div class="system-brand-row__caption">
                  {{ systemConfig.system_description || '通用低代码底座' }}
                </div>
              </div>
            </div>

            <div class="field-grid">
              <q-input
                v-model="systemConfig.system_name"
                label="系统名称"
                outlined
                dense
                hide-bottom-space
                :rules="[(val) => String(val ?? '').trim().length > 0 || '请输入系统名称']"
              >
                <template #append>
                  <q-icon name="business" />
                </template>
              </q-input>

              <q-input v-model="systemConfig.system_version" label="系统版本" outlined dense>
                <template #append>
                  <q-icon name="new_releases" />
                </template>
              </q-input>

              <q-input
                v-model="systemConfig.system_logo"
                class="field-grid__wide"
                label="系统Logo URL"
                outlined
                dense
              >
                <template #append>
                  <q-icon name="image" />
                </template>
              </q-input>

              <q-input
                v-model="systemConfig.system_description"
                class="field-grid__wide"
                type="textarea"
                label="系统描述"
                outlined
                dense
                autogrow
              >
                <template #append>
                  <q-icon name="description" />
                </template>
              </q-input>
            </div>
          </section>

          <section v-show="activeTab === 'email'" class="configure-panel">
            <div class="setting-row">
              <div class="setting-row__main">
                <span class="setting-row__icon">
                  <q-icon name="mark_email_read" />
                </span>
                <div>
                  <div class="setting-row__title">邮件服务</div>
                  <div class="setting-row__caption">
                    维护 SMTP 参数，用户密码重置后可自动发送临时密码通知
                  </div>
                </div>
              </div>
              <q-toggle
                v-model="emailConfig.enable_email"
                color="primary"
                keep-color
                :label="emailConfig.enable_email ? '已开启' : '已关闭'"
              />
            </div>

            <div class="field-grid">
              <q-input
                v-model="emailConfig.smtp_server"
                label="SMTP服务器"
                outlined
                dense
                hide-bottom-space
                :disable="!emailConfig.enable_email"
                :rules="[
                  (val) =>
                    !emailConfig.enable_email ||
                    String(val ?? '').trim().length > 0 ||
                    'SMTP服务器不能为空',
                ]"
              >
                <template #append>
                  <q-icon name="dns" />
                </template>
              </q-input>

              <q-input
                v-model.number="emailConfig.smtp_port"
                type="number"
                label="SMTP端口"
                outlined
                dense
                hide-bottom-space
                :disable="!emailConfig.enable_email"
                :rules="[
                  (val) => !emailConfig.enable_email || Number(val) > 0 || 'SMTP端口不能为空',
                ]"
              >
                <template #append>
                  <q-icon name="settings_ethernet" />
                </template>
              </q-input>

              <q-input
                v-model="emailConfig.sender_email"
                label="发件人邮箱"
                outlined
                dense
                hide-bottom-space
                :disable="!emailConfig.enable_email"
                :rules="[
                  (val) =>
                    !emailConfig.enable_email ||
                    String(val ?? '').trim().length > 0 ||
                    '发件人邮箱不能为空',
                ]"
              >
                <template #append>
                  <q-icon name="alternate_email" />
                </template>
              </q-input>

              <q-input
                v-model="emailConfig.sender_password"
                label="发件人密码/授权码"
                outlined
                dense
                :type="showPassword ? 'text' : 'password'"
                :disable="!emailConfig.enable_email"
                hint="留空保存时会沿用原授权码"
              >
                <template #append>
                  <q-icon
                    :name="showPassword ? 'visibility' : 'visibility_off'"
                    class="cursor-pointer"
                    @click="showPassword = !showPassword"
                  />
                </template>
              </q-input>
            </div>

            <div class="email-test-row">
              <q-input
                v-model="testEmailTo"
                label="测试收件邮箱"
                outlined
                dense
                :disable="!emailConfig.enable_email"
                hint="使用已保存的邮件配置发送；修改 SMTP 参数后请先保存"
              >
                <template #append>
                  <q-icon name="mail" />
                </template>
              </q-input>
              <q-btn
                color="primary"
                outline
                icon="outgoing_mail"
                label="发送测试邮件"
                type="button"
                :loading="loading"
                :disable="!emailConfig.enable_email"
                @click="sendTestEmail"
              />
            </div>
          </section>

          <div class="configure-actions">
            <div class="configure-actions__status">
              <q-icon name="schedule" />
              <span>{{ lastUpdatedText }}</span>
            </div>
            <div class="configure-actions__buttons">
              <q-btn
                color="primary"
                icon="save"
                label="保存配置"
                :loading="loading"
                unelevated
                type="submit"
              />
            </div>
          </div>
        </main>
      </div>
    </q-form>
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'develop_configure' })
import { computed, nextTick, ref } from 'vue'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import { type Configure, useBasicApi } from 'src/api/services/basic'
import { useQuasar, type QForm } from 'quasar'
import { useLoadingStore } from 'src/stores/loading'
import { storeToRefs } from 'pinia'
import { useConfigureStore } from 'stores/configure'
import {
  effectivePasswordPolicy,
  getPasswordPolicyPreset,
  normalizePasswordPolicy,
  passwordPolicyDescription,
  passwordPolicyOptions,
  passwordPolicyRegexText,
} from 'src/utils/passwordPolicy'

type ConfigTab = 'security' | 'system' | 'email'

interface ConfigSection {
  name: ConfigTab
  label: string
  caption: string
  icon: string
  status: string
  color: string
}

const loadingStore = useLoadingStore()
const { loading } = storeToRefs(loadingStore)
const $q = useQuasar()
const basicApi = useBasicApi()
const configureStore = useConfigureStore()

const activeTab = ref<ConfigTab>('security')
const formRef = ref<QForm | null>(null)
const showPassword = ref(false)
const testEmailTo = ref('')

const complexityOptions = [
  { label: '低：长度校验', value: 1 },
  { label: '中：字母 + 数字', value: 2 },
  { label: '高：三类字符', value: 3 },
]

const policyOptions = passwordPolicyOptions

const defaultSection: ConfigSection = {
  name: 'security',
  label: '安全配置',
  caption: '登录策略与密码规则',
  icon: 'security',
  status: '配置',
  color: 'primary',
}

const formData = ref<Configure>({
  id: 0,
  enable_captcha: false,
  password_length: 6,
  password_complexity: 1,
  password_expire_time: 0,
  password_error_count: 5,
  password_lock_minutes: 15,
  password_policy: 'medium',
})

const systemConfig = ref({
  system_name: 'Sweet Admin',
  system_version: '0.1',
  system_logo: '',
  system_description: '通用低代码底座',
})

const emailConfig = ref({
  enable_email: false,
  smtp_server: '',
  smtp_port: 465,
  sender_email: '',
  sender_password: '',
})

const normalizedPasswordPolicy = computed(() =>
  normalizePasswordPolicy(formData.value.password_policy),
)

const isCustomPasswordPolicy = computed(() => normalizedPasswordPolicy.value === 'custom')

const currentPolicyPreset = computed(() => getPasswordPolicyPreset(normalizedPasswordPolicy.value))

const effectivePolicy = computed(() => effectivePasswordPolicy(formData.value))

const passwordPolicySummary = computed(() => passwordPolicyDescription(formData.value))

const passwordPolicyRegex = computed(() => passwordPolicyRegexText(formData.value))

const policyLabel = computed(() => currentPolicyPreset.value.label)

const lastUpdatedText = computed(() => {
  return formData.value.gmt_modify
    ? `最近修改：${formData.value.gmt_modify}`
    : '配置尚未保存修改记录'
})

const configSections = computed<ConfigSection[]>(() => [
  {
    name: 'security',
    label: '安全配置',
    caption: `验证码${formData.value.enable_captcha ? '开启' : '关闭'}，${passwordPolicySummary.value}`,
    icon: 'security',
    status: `${formData.value.password_error_count}次`,
    color: 'primary',
  },
  {
    name: 'email',
    label: '邮件配置',
    caption: emailConfig.value.enable_email
      ? emailConfig.value.smtp_server || '待配置服务器'
      : '邮件服务未启用',
    icon: 'email',
    status: emailConfig.value.enable_email ? '开启' : '关闭',
    color: emailConfig.value.enable_email ? 'positive' : 'grey-6',
  },
  {
    name: 'system',
    label: '系统配置',
    caption: systemConfig.value.system_name || 'Sweet Admin',
    icon: 'settings',
    status: systemConfig.value.system_version || '版本',
    color: 'primary',
  },
])

const currentSection = computed<ConfigSection>(() => {
  return configSections.value.find((item) => item.name === activeTab.value) ?? defaultSection
})

const overviewItems = computed(() => [
  {
    label: '验证码',
    value: formData.value.enable_captcha ? '开启' : '关闭',
    icon: 'verified_user',
    color: formData.value.enable_captcha ? 'positive' : 'grey-7',
  },
  {
    label: '密码策略',
    value: policyLabel.value,
    icon: 'lock',
    color: 'primary',
  },
  {
    label: '登录锁定',
    value: `${formData.value.password_error_count}次/${formData.value.password_lock_minutes}分钟`,
    icon: 'lock_clock',
    color: 'warning',
  },
])

const fetchConfig = async () => {
  const response = await basicApi.configureDetail()
  if (response.data) {
    formData.value = {
      ...formData.value,
      ...response.data,
      password_policy: normalizePasswordPolicy(response.data.password_policy || 'medium'),
    }

    systemConfig.value = {
      system_name: response.data.system_name || 'Sweet Admin',
      system_version: response.data.system_version || '0.1',
      system_logo: response.data.system_logo || '',
      system_description: response.data.system_description || '通用低代码底座',
    }

    emailConfig.value = {
      enable_email: response.data.enable_email ?? false,
      smtp_server: response.data.smtp_server || '',
      smtp_port: response.data.smtp_port || 465,
      sender_email: response.data.sender_email || '',
      sender_password: '',
    }
  }
}
fetchConfig()

const refreshConfig = async () => {
  await fetchConfig()
  $q.notify({ type: 'positive', position: 'top-right', message: '配置已刷新' })
}

const syncPasswordPolicyPreset = () => {
  if (isCustomPasswordPolicy.value) return
  formData.value.password_length = effectivePolicy.value.minLen
  formData.value.password_complexity = effectivePolicy.value.complexity
}

const sendTestEmail = async () => {
  const to = testEmailTo.value.trim()
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(to)) {
    $q.notify({ type: 'warning', position: 'top-right', message: '请输入正确的测试收件邮箱' })
    return
  }
  await basicApi.testConfigureEmail(to)
}

const firstInvalidTab = (): ConfigTab | null => {
  if (
    (isCustomPasswordPolicy.value && Number(formData.value.password_length) < 6) ||
    Number(formData.value.password_expire_time) < 0 ||
    Number(formData.value.password_error_count) <= 0 ||
    Number(formData.value.password_lock_minutes) <= 0 ||
    String(formData.value.password_policy ?? '').trim().length === 0
  ) {
    return 'security'
  }

  if (String(systemConfig.value.system_name ?? '').trim().length === 0) {
    return 'system'
  }

  if (
    emailConfig.value.enable_email &&
    (String(emailConfig.value.smtp_server ?? '').trim().length === 0 ||
      Number(emailConfig.value.smtp_port) <= 0 ||
      String(emailConfig.value.sender_email ?? '').trim().length === 0)
  ) {
    return 'email'
  }

  return null
}

const saveConfig = async () => {
  const invalidTab = firstInvalidTab()
  if (invalidTab) {
    activeTab.value = invalidTab
    await nextTick()
  }

  const valid = await formRef.value?.validate(true)
  if (valid !== true) {
    $q.notify({
      color: 'negative',
      position: 'top-right',
      message: '请先完善必填项后再保存',
    })
    return
  }

  const completeConfig = {
    ...formData.value,
    password_length: effectivePolicy.value.minLen,
    password_complexity: effectivePolicy.value.complexity,
    password_lock_minutes: Number(formData.value.password_lock_minutes) || 15,
    password_policy: normalizedPasswordPolicy.value,
    system_name: systemConfig.value.system_name,
    system_version: systemConfig.value.system_version,
    system_logo: systemConfig.value.system_logo,
    system_description: systemConfig.value.system_description,
    enable_email: emailConfig.value.enable_email,
    smtp_server: emailConfig.value.smtp_server,
    smtp_port: emailConfig.value.smtp_port,
    sender_email: emailConfig.value.sender_email,
    sender_password: emailConfig.value.sender_password,
  }

  const result = await basicApi.updateConfigure(completeConfig)
  if (result.success) {
    configureStore.setConfigure(completeConfig)
    emailConfig.value.sender_password = ''
  }
}
</script>

<style scoped lang="scss">
.configure-page {
  --configure-border: rgba(115, 103, 240, 0.16);
  --configure-muted: #6f7d95;
  --configure-ink: #1f2a44;
}

.configure-shell {
  min-width: 0;
}

.configure-hero {
  display: flex;
  align-items: center;
  gap: 14px;
  min-height: 72px;
  padding: 12px 16px;
  border: 1px solid var(--configure-border);
  border-radius: 8px;
  background: linear-gradient(135deg, rgba(115, 103, 240, 0.1), rgba(0, 184, 169, 0.07)), #fff;
}

.configure-hero__icon {
  display: grid;
  width: 48px;
  height: 48px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 8px;
  background: $primary;
  color: #fff;
  box-shadow: 0 10px 22px rgba(115, 103, 240, 0.22);
}

.configure-hero__icon .q-icon {
  font-size: 25px;
}

.configure-hero__content {
  min-width: 0;
  flex: 1;
}

.configure-hero__title {
  color: var(--configure-ink);
  font-size: 22px;
  font-weight: 700;
  line-height: 1.25;
}

.configure-hero__subtitle {
  margin-top: 2px;
  color: var(--configure-muted);
  font-size: 13px;
}

.configure-hero__actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.configure-layout {
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr);
  gap: 10px;
  margin-top: 10px;
  min-height: 0;
}

.configure-sidebar,
.configure-main {
  border: 1px solid var(--configure-border);
  border-radius: 8px;
  background: #fff;
}

.configure-sidebar {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px;
}

.configure-nav-item {
  width: 100%;
  min-height: 64px;
  display: grid;
  grid-template-columns: 40px minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  padding: 8px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: var(--configure-ink);
  cursor: pointer;
  text-align: left;
  transition:
    border-color 0.18s ease,
    background 0.18s ease,
    box-shadow 0.18s ease;
}

.configure-nav-item:hover,
.configure-nav-item--active {
  border-color: rgba(115, 103, 240, 0.2);
  background: rgba(115, 103, 240, 0.06);
}

.configure-nav-item--active {
  box-shadow: inset 3px 0 0 $primary;
}

.configure-nav-item__icon {
  display: grid;
  width: 36px;
  height: 36px;
  place-items: center;
  border-radius: 8px;
  background: rgba(115, 103, 240, 0.12);
  color: $primary;
}

.configure-nav-item__icon .q-icon {
  font-size: 20px;
}

.configure-nav-item__body {
  min-width: 0;
  display: grid;
  gap: 2px;
}

.configure-nav-item__title {
  overflow: hidden;
  color: var(--configure-ink);
  font-size: 14px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.configure-nav-item__caption {
  overflow: hidden;
  color: var(--configure-muted);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.configure-sidebar-note {
  display: flex;
  gap: 10px;
  margin-top: auto;
  padding: 10px;
  border-radius: 8px;
  background: #f7f8ff;
  color: var(--configure-muted);
}

.configure-sidebar-note .q-icon {
  margin-top: 2px;
  color: $primary;
  font-size: 20px;
}

.configure-sidebar-note__title {
  color: var(--configure-ink);
  font-size: 13px;
  font-weight: 700;
}

.configure-sidebar-note__text {
  margin-top: 2px;
  font-size: 12px;
  line-height: 1.45;
}

.configure-main {
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.configure-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--configure-border);
}

.configure-section-head__title {
  color: var(--configure-ink);
  font-size: 18px;
  font-weight: 700;
}

.configure-section-head__caption {
  margin-top: 2px;
  color: var(--configure-muted);
  font-size: 13px;
}

.configure-section-head > .q-icon {
  color: rgba(115, 103, 240, 0.55);
  font-size: 26px;
}

.configure-overview {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
  padding: 8px 16px;
  border-bottom: 1px solid var(--configure-border);
  background: #fbfcff;
}

.configure-metric {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 54px;
  padding: 9px 10px;
  border: 1px solid #e5ebf6;
  border-radius: 8px;
  background: #fff;
}

.configure-metric > .q-icon {
  display: grid;
  width: 34px;
  height: 34px;
  place-items: center;
  border-radius: 8px;
  background: rgba(115, 103, 240, 0.08);
  font-size: 20px;
}

.configure-metric__value {
  overflow: hidden;
  max-width: 220px;
  color: var(--configure-ink);
  font-size: 16px;
  font-weight: 700;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.configure-metric__label {
  margin-top: 1px;
  color: var(--configure-muted);
  font-size: 12px;
}

.configure-panel {
  flex: 1;
  padding: 14px;
}

.setting-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 12px;
  padding: 10px 12px;
  border: 1px solid #e4e9f4;
  border-radius: 8px;
  background: #fbfcff;
}

.setting-row__main {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.setting-row__icon {
  display: grid;
  width: 34px;
  height: 34px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 8px;
  background: rgba(115, 103, 240, 0.1);
  color: $primary;
}

.setting-row__icon .q-icon {
  font-size: 20px;
}

.setting-row__title {
  color: var(--configure-ink);
  font-size: 15px;
  font-weight: 700;
}

.setting-row__caption {
  margin-top: 2px;
  color: var(--configure-muted);
  font-size: 12px;
}

.field-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px 14px;
}

.field-grid__wide {
  grid-column: 1 / -1;
}

.security-field-grid {
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: 10px 12px;
}

.security-field-grid > .q-field:not(.field-grid__wide) {
  grid-column: span 2;
}

.password-policy-card {
  display: grid;
  gap: 8px;
  padding: 10px 12px;
  border: 1px solid #e3e9f5;
  border-radius: 8px;
  background: #fbfcff;
}

.password-policy-card__head {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.password-policy-card__title {
  color: var(--configure-ink);
  font-size: 14px;
  font-weight: 800;
}

.password-policy-card__desc {
  margin-top: 2px;
  color: var(--configure-muted);
  font-size: 12px;
  line-height: 1.35;
}

.password-policy-card__rule {
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr);
  align-items: start;
  gap: 8px;
  padding-top: 7px;
  border-top: 1px dashed #d9e1f0;
  color: var(--configure-muted);
  font-size: 12px;
}

.password-policy-card__rule code {
  min-width: 0;
  overflow-wrap: anywhere;
  color: #405078;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  line-height: 1.35;
}

.login-lock-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border: 1px solid #e3e9f5;
  border-radius: 8px;
  background: #fbfcff;
}

.login-lock-card__icon {
  display: grid;
  width: 38px;
  height: 38px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 8px;
  background: rgba(115, 103, 240, 0.1);
  color: $primary;
}

.login-lock-card__icon .q-icon {
  font-size: 21px;
}

.login-lock-card__body {
  min-width: 0;
}

.login-lock-card__title {
  color: var(--configure-ink);
  font-size: 14px;
  font-weight: 800;
}

.login-lock-card__text {
  margin-top: 2px;
  color: var(--configure-muted);
  font-size: 12px;
}

.email-test-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: start;
  gap: 12px;
  margin-top: 12px;
}

.system-brand-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  padding: 10px 12px;
  border: 1px solid #e4e9f4;
  border-radius: 8px;
  background: linear-gradient(135deg, rgba(115, 103, 240, 0.08), rgba(0, 207, 232, 0.05));
}

.system-logo-preview {
  display: grid;
  width: 42px;
  height: 42px;
  flex: 0 0 auto;
  place-items: center;
  overflow: hidden;
  border-radius: 8px;
  background: $primary;
  color: #fff;
}

.system-logo-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.system-logo-preview .q-icon {
  font-size: 24px;
}

.system-brand-row__title {
  color: var(--configure-ink);
  font-size: 16px;
  font-weight: 700;
}

.system-brand-row__caption {
  margin-top: 2px;
  color: var(--configure-muted);
  font-size: 13px;
}

.configure-actions {
  position: sticky;
  bottom: 0;
  z-index: 2;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: auto;
  padding: 10px 16px;
  border-top: 1px solid var(--configure-border);
  border-radius: 0 0 8px 8px;
  background: rgba(255, 255, 255, 0.96);
  box-shadow: 0 -10px 24px rgba(31, 42, 68, 0.05);
}

.configure-actions__status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  color: var(--configure-muted);
  font-size: 12px;
}

.configure-actions__status span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.configure-actions__buttons {
  display: flex;
  gap: 8px;
}

@media (max-width: 1100px) {
  .configure-layout {
    grid-template-columns: 1fr;
  }

  .security-field-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .security-field-grid > .q-field:not(.field-grid__wide) {
    grid-column: auto;
  }

  .configure-sidebar {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .configure-sidebar-note {
    display: none;
  }
}

@media (max-width: 720px) {
  .configure-hero,
  .configure-actions,
  .setting-row {
    align-items: stretch;
    flex-direction: column;
  }

  .configure-hero__actions,
  .configure-actions__buttons {
    width: 100%;
  }

  .configure-hero__actions .q-btn,
  .configure-actions__buttons .q-btn {
    flex: 1;
  }

  .configure-sidebar {
    grid-template-columns: 1fr;
  }

  .configure-overview,
  .field-grid,
  .email-test-row {
    grid-template-columns: 1fr;
  }

  .security-field-grid > .q-field:not(.field-grid__wide) {
    grid-column: 1 / -1;
  }
}
</style>
