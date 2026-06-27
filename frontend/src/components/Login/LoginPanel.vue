<template>
  <q-card flat class="login-card">
    <div class="login-card-top">
      <div class="login-brand">
        <div class="login-brand-logo">
          <img v-if="systemLogo" :src="systemLogo" alt="系统Logo" />
          <q-icon v-else name="admin_panel_settings" />
        </div>
        <div>
          <div class="login-brand-title">{{ systemName }}</div>
          <div class="login-brand-subtitle">{{ systemDescription || '通用低代码底座' }}</div>
        </div>
      </div>
      <div class="login-mode-toggle">
        <dark-mode />
      </div>
    </div>
    <div class="column items-stretch">
      <q-card-section class="login-body">
        <q-form
          ref="loginForm"
          class="login-form"
          @submit.prevent="onLoginClick"
        >
          <q-input
            v-model="username"
            placeholder="请输入账号"
            dense
            clearable
            outlined
            no-error-icon
            lazy-rules
            hide-bottom-space
            :rules="[(val) => !!val || '请输入账号']"
          >
            <template v-slot:prepend>
              <q-icon size="xs" name="person" />
            </template>
          </q-input>
          <q-input
            v-model="password"
            placeholder="请输入密码"
            dense
            outlined
            no-error-icon
            :type="isPwd ? 'password' : 'text'"
            lazy-rules
            hide-bottom-space
            :rules="[(val) => !!val || '请输入密码']"
            autocomplete="off"
          >
            <template v-slot:prepend>
              <q-icon size="xs" name="https" />
            </template>
            <template v-slot:append>
              <q-icon
                :name="isPwd ? 'visibility_off' : 'visibility'"
                class="cursor-pointer"
                @click="isPwd = !isPwd"
              />
            </template>
          </q-input>
          <q-input
            v-if="enableCaptcha"
            v-model="captcha"
            class="login-captcha-field"
            placeholder="请输入验证码"
            dense
            outlined
            no-error-icon
            lazy-rules
            hide-bottom-space
            :rules="[(val) => !!val || '请输入验证码']"
          >
            <template v-slot:prepend>
              <q-icon size="xs" name="security" />
            </template>
            <template v-slot:append>
              <q-img class="login-captcha-image" :src="captchaImage" @click="fetchCaptcha">
                <q-tooltip>点击刷新验证码</q-tooltip>
              </q-img>
            </template>
          </q-input>
          <q-btn
            class="login-submit full-width"
            size="md"
            unelevated
            color="primary"
            type="submit"
            :loading="loading"
          >
            登录
          </q-btn>
          <q-banner
            v-if="message !== ''"
            inline-actions
            dense
            rounded
            class="login-message text-red"
          >
            {{ message }}
          </q-banner>
        </q-form>
        <div v-if="systemDescription" class="login-desc text-caption text-grey-7">
          {{ systemDescription }}
        </div>
      </q-card-section>
    </div>
  </q-card>
</template>

<script lang="ts" setup>
import { computed, ref, onMounted } from 'vue'
import { useVModels } from '@vueuse/core'
import { useBasicApi } from 'src/api/services/basic'
import { QForm } from 'quasar'
import DarkMode from 'src/components/Toolbar/DarkMode.vue'
import { useConfigureStore } from 'stores/configure'

defineOptions({ name: 'MyLogin' })

interface Props {
  username: string | undefined
  password: string | undefined
  captcha: string | undefined
  captcha_id: string | undefined
  loading: boolean
  message: string
}

const captchaImage = ref('')

const configureStore = useConfigureStore()
const { captchaImg } = useBasicApi()
const systemName = computed(() => configureStore.getSystemName || 'Sweet Admin')
const systemLogo = computed(() => configureStore.getSystemLogo || '')
const systemDescription = computed(() => configureStore.getSystemDescription || '')

const fetchCaptcha = async () => {
  try {
    const response = await captchaImg()
    const captchaData = response.data
    const captcha_id = captchaData.captcha_id
    const image = captchaData.image
    // 创建一个 Blob URL 来显示验证码图片
    let base64Image = image
    if (!image.startsWith('data:image/png;base64,')) {
      base64Image = `data:image/png;base64,${image}`
    }
    const byteCharacters = atob(base64Image.split(',')[1] ?? '') // 去掉 base64 前缀
    const byteNumbers = new Array(byteCharacters.length)
    for (let i = 0; i < byteCharacters.length; i++) {
      byteNumbers[i] = byteCharacters.charCodeAt(i)
    }
    const byteArray = new Uint8Array(byteNumbers)
    const blob = new Blob([byteArray], { type: 'image/png' })
    const url = URL.createObjectURL(blob)
    captchaImage.value = url
    // 传递 captcha_id
    emit('update:captcha_id', captcha_id)
  } catch (error) {
    console.error('Failed to fetch captcha:', error)
  }
}

onMounted(async () => {
  // 退出登录不会刷新整个页面，boot 也不会重跑；这里强制拉一次配置，保证验证码开关是最新的
  await configureStore.fetchConfigure({ force: true })
  enableCaptcha.value = configureStore.getEnableCaptcha
  if (enableCaptcha.value) {
    await fetchCaptcha()
  }
})

const enableCaptcha = ref(false)

const props = withDefaults(defineProps<Props>(), {
  username: '',
  password: '',
  captcha: '',
  loading: false,
  message: '',
})

const emit = defineEmits<{
  (e: 'update:username'): void
  (e: 'update:password'): void
  (e: 'update:captcha'): void
  (e: 'update:captcha_id', captcha_id: string): void
  (e: 'update:loading'): void
  (e: 'onLoginClick'): void
}>()

const isPwd = ref<boolean>(true)
const loginForm = ref<QForm | null>(null)
const { username, password, loading, captcha, message } = useVModels(props, emit)

const onLoginClick = async () => {
  const success = await loginForm.value?.validate()
  if (success) {
    emit('onLoginClick')
  }
}
</script>

<style scoped lang="scss">
.login-card {
  width: 100%;
  max-width: 460px;
  position: relative;
  overflow: hidden;
  border: 1px solid rgba(15, 23, 42, 0.1);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.98);
  color: #111827;
  box-shadow: 0 22px 55px rgba(15, 23, 42, 0.14);
  backdrop-filter: blur(14px);
}

.login-card::before {
  content: '';
  position: absolute;
  inset: 0 0 auto;
  height: 4px;
  background: linear-gradient(90deg, #1976d2, #26a69a, #f6a623);
}

.login-card-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 38px 38px 22px;
}

.login-brand {
  display: flex;
  align-items: center;
  gap: 16px;
}

.login-brand-logo {
  width: 50px;
  height: 50px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  color: #ffffff;
  background: linear-gradient(135deg, #111827, #243044);
  box-shadow: 0 14px 28px rgba(17, 24, 39, 0.2);
}

.login-brand-logo .q-icon {
  font-size: 28px;
}

.login-brand-logo img {
  width: 100%;
  height: 100%;
  border-radius: inherit;
  object-fit: cover;
}

.login-brand-title {
  color: #111827;
  font-size: 26px;
  font-weight: 700;
  line-height: 1.2;
}

.login-brand-subtitle {
  margin-top: 4px;
  color: #64748b;
  font-size: 14px;
}

.login-mode-toggle {
  width: 34px;
  height: 34px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  background: #f1f5f9;
  border: 1px solid rgba(100, 116, 139, 0.18);
}

.login-body {
  padding: 0 38px 30px;
}

.login-form {
  display: grid;
  gap: 12px;
}

.login-form :deep(.q-field__control) {
  height: 46px;
  min-height: 46px;
  border-radius: 8px;
  background: #ffffff;
}

.login-form :deep(.q-field__marginal) {
  height: 46px;
  min-height: 46px;
  align-items: center;
}

.login-form :deep(.q-field__bottom) {
  min-height: 18px;
  padding: 4px 12px 0;
  color: var(--q-negative);
  font-size: 12px;
  line-height: 16px;
}

.login-form :deep(.q-field__messages),
.login-form :deep(.q-field__messages > div) {
  line-height: 16px;
}

.login-form :deep(.q-field--outlined .q-field__control::before) {
  border-color: rgba(100, 116, 139, 0.28);
}

.login-form :deep(.q-field--focused .q-field__control::after) {
  border-width: 1px;
}

.login-captcha-field :deep(.q-field__append) {
  padding-left: 8px;
}

.login-captcha-image {
  width: 112px;
  height: 34px;
  cursor: pointer;
  border-radius: 6px;
  overflow: hidden;
  border: 1px solid rgba(100, 116, 139, 0.24);
  background: #f8fafc;
}

.login-captcha-image :deep(.q-img__image) {
  object-fit: contain !important;
}

.login-form :deep(.q-field__native),
.login-form :deep(.q-field__prefix),
.login-form :deep(.q-field__suffix),
.login-form :deep(.q-field__input) {
  min-height: 46px;
  line-height: 22px;
}

.login-form :deep(.q-icon) {
  color: #111827;
}

.login-form :deep(.q-field__label),
.login-form :deep(input::placeholder) {
  color: #64748b;
}

.login-submit {
  height: 50px;
  border-radius: 8px;
  font-size: 16px;
  font-weight: 600;
  letter-spacing: 0;
  box-shadow: 0 14px 24px rgba(25, 118, 210, 0.18);
}

.login-message {
  border-radius: 8px;
  background: rgba(244, 67, 54, 0.08);
}

.login-desc {
  margin-top: 18px;
  text-align: center;
  line-height: 1.4;
}

.login-card.q-dark {
  border-color: rgba(255, 255, 255, 0.1);
  background: rgba(17, 24, 39, 0.94);
  color: #f8fafc;
}

.login-card.q-dark .login-captcha-image {
  border-color: rgba(203, 213, 225, 0.34);
  background: rgba(248, 250, 252, 0.96);
  box-shadow: inset 0 0 0 1px rgba(15, 23, 42, 0.06);
}

.login-card.q-dark .login-form :deep(.q-field__native),
.login-card.q-dark .login-form :deep(.q-field__prefix),
.login-card.q-dark .login-form :deep(.q-field__suffix),
.login-card.q-dark .login-form :deep(.q-field__input),
.login-card.q-dark .login-form :deep(.q-icon) {
  color: #f8fafc;
}

.login-card.q-dark .login-form :deep(.q-field__label),
.login-card.q-dark .login-form :deep(input::placeholder) {
  color: #cbd5e1;
}

@media (max-width: 599px) {
  .login-card {
    box-shadow: 0 16px 42px rgba(30, 41, 59, 0.14);
  }

  .login-card-top {
    padding: 30px 24px 18px;
  }

  .login-body {
    padding: 0 24px 26px;
  }
}
</style>
