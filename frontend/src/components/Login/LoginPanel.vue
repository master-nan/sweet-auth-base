<template>
  <q-card flat class="login-card">
    <div class="login-mobile-brand">
      <div class="login-brand">
        <div class="login-brand-logo">
          <img v-if="systemLogo" :src="systemLogo" :alt="t('login.systemLogoAlt')" />
          <q-icon v-else name="admin_panel_settings" />
        </div>
        <div>
          <div class="login-brand-title">{{ systemName }}</div>
          <div class="login-brand-subtitle">
            {{ systemDescription || t('login.defaultDescription') }}
          </div>
        </div>
      </div>
    </div>

    <login-flow-visual class="login-flow-visual" />

    <q-card-section class="login-body">
      <div class="login-kicker">{{ t('login.accountLogin') }}</div>
      <div class="login-welcome">
        <h2>{{ t('login.welcome') }}</h2>
        <p>{{ t('login.continueHint') }}</p>
      </div>

      <q-form ref="loginForm" class="login-form" @submit.prevent="onLoginClick">
        <div class="login-field">
          <label for="login-username">{{ t('login.username') }}</label>
          <q-input
            id="login-username"
            v-model="username"
            :placeholder="t('login.usernamePlaceholder')"
            dense
            clearable
            outlined
            no-error-icon
            lazy-rules
            hide-bottom-space
            :rules="[(val) => !!val || t('login.usernameRequired')]"
          >
            <template v-slot:prepend>
              <q-icon size="xs" name="person" />
            </template>
          </q-input>
        </div>

        <div class="login-field">
          <label for="login-password">{{ t('login.password') }}</label>
          <q-input
            id="login-password"
            v-model="password"
            :placeholder="t('login.passwordPlaceholder')"
            dense
            outlined
            no-error-icon
            :type="isPwd ? 'password' : 'text'"
            lazy-rules
            hide-bottom-space
            :rules="[(val) => !!val || t('login.passwordRequired')]"
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
                size="xs"
              />
            </template>
          </q-input>
        </div>

        <div v-if="enableCaptcha" class="login-field">
          <label for="login-captcha">{{ t('login.captcha') }}</label>
          <q-input
            id="login-captcha"
            v-model="captcha"
            class="login-captcha-field"
            :placeholder="t('login.captchaPlaceholder')"
            dense
            outlined
            no-error-icon
            lazy-rules
            hide-bottom-space
            :rules="[(val) => !!val || t('login.captchaRequired')]"
          >
            <template v-slot:prepend>
              <q-icon size="xs" name="security" />
            </template>
            <template v-slot:append>
              <q-img class="login-captcha-image" :src="captchaImage" @click="fetchCaptcha">
                <q-tooltip>{{ t('login.captchaRefresh') }}</q-tooltip>
              </q-img>
            </template>
          </q-input>
        </div>

        <q-btn
          class="login-submit full-width"
          size="md"
          unelevated
          color="primary"
          type="submit"
          :loading="loading"
        >
          {{ t('login.submit') }}
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
    </q-card-section>
  </q-card>
</template>

<script lang="ts" setup>
import { computed, ref, onMounted } from 'vue'
import { useVModels } from '@vueuse/core'
import { useBasicApi } from 'src/api/services/basic'
import { QForm } from 'quasar'
import LoginFlowVisual from 'src/components/Login/LoginFlowVisual.vue'
import { useConfigureStore } from 'stores/configure'
import { useI18n } from 'vue-i18n'

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
const { t } = useI18n({ useScope: 'global' })

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
  max-width: 380px;
  border-radius: 0;
  background: transparent;
  color: #151a27;
  box-shadow: none;
}

.login-flow-visual {
  width: calc(100% + 48px);
  height: 116px;
  margin: 0 -24px 24px;
  pointer-events: none;
}

.login-brand {
  display: flex;
  align-items: center;
  gap: 12px;
}

.login-brand-logo {
  position: relative;
  isolation: isolate;
  width: 42px;
  height: 42px;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border-radius: 8px;
  color: #ffffff;
  background: #252b38;
}

.login-brand-logo::before,
.login-brand-logo::after {
  position: absolute;
  content: '';
}

.login-brand-logo::before {
  z-index: 0;
  inset: 0;
  background: conic-gradient(#7568dc, #31a8be, #55c991, #efb34b, #d86f62, #7568dc);
}

.login-brand-logo::after {
  z-index: 1;
  inset: 3px;
  border-radius: 6px;
  background: #252b38;
}

.login-brand-logo .q-icon {
  position: relative;
  z-index: 2;
  font-size: 22px;
}

.login-brand-logo img {
  position: relative;
  z-index: 2;
  width: calc(100% - 6px);
  height: calc(100% - 6px);
  border-radius: 6px;
  object-fit: cover;
}

.login-brand-title {
  font-size: 20px;
  font-weight: 800;
  line-height: 1.15;
}

.login-brand-subtitle {
  margin-top: 4px;
  color: #7f899b;
  font-size: 12px;
}

.login-mobile-brand {
  display: none;
  margin-bottom: 22px;
}

.login-body {
  padding: 0;
}

.login-kicker {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 18px;
  color: #6559ee;
  font-size: 12px;
  font-weight: 800;
}

.login-kicker::before {
  content: '';
  width: 22px;
  height: 2px;
  background: #6559ee;
}

.login-welcome h2 {
  margin: 0;
  font-size: 30px;
  font-weight: 700;
  line-height: 1.2;
}

.login-welcome p {
  margin: 5px 0 28px;
  color: #7f899b;
  font-size: 13px;
}

.login-form {
  display: grid;
  gap: 16px;
}

.login-field {
  display: grid;
  gap: 7px;
}

.login-field > label {
  color: #526077;
  font-size: 12px;
  font-weight: 700;
}

.login-form :deep(.q-field__control) {
  height: 48px;
  min-height: 48px;
  border-radius: 6px;
  background: #fbfcfd;
}

.login-form :deep(.q-field__marginal) {
  height: 48px;
  min-height: 48px;
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
  border-color: #d8deea;
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
  min-height: 48px;
  line-height: 22px;
}

.login-form :deep(.q-icon) {
  color: #526077;
}

.login-form :deep(.q-field__label),
.login-form :deep(input::placeholder) {
  color: #64748b;
}

.login-submit {
  height: 48px;
  margin-top: 4px;
  border-radius: 6px;
  font-size: 15px;
  font-weight: 600;
  letter-spacing: 0;
  box-shadow: 0 12px 24px rgba(103, 92, 241, 0.18);
}

.login-message {
  border-radius: 8px;
  background: rgba(244, 67, 54, 0.08);
}

.login-card.q-dark {
  background: transparent;
  color: #f8fafc;
}

.login-card.q-dark .login-brand-subtitle,
.login-card.q-dark .login-welcome p,
.login-card.q-dark .login-field > label {
  color: #a3adbf;
}

.login-card.q-dark .login-form :deep(.q-field__control) {
  background: #242835;
}

.login-card.q-dark .login-form :deep(.q-field--outlined .q-field__control::before) {
  border-color: #3a4050;
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
  .login-mobile-brand {
    display: block;
    margin-bottom: 18px;
    padding-right: 88px;
  }

  .login-flow-visual {
    width: calc(100% + 24px);
    height: 78px;
    margin: 0 -12px 18px;
  }

  .login-welcome h2 {
    font-size: 28px;
  }
}

@media (max-width: 599px) and (min-height: 860px) {
  .login-mobile-brand {
    margin-bottom: 26px;
  }

  .login-flow-visual {
    height: 92px;
    margin-bottom: 28px;
  }

  .login-kicker {
    margin-bottom: 22px;
  }

  .login-welcome p {
    margin-top: 8px;
    margin-bottom: 34px;
  }

  .login-form {
    gap: 20px;
  }
}

@media (min-width: 600px) and (max-width: 1023px) {
  .login-mobile-brand {
    display: block;
  }

  .login-flow-visual {
    width: calc(100% + 32px);
    height: 92px;
    margin: 0 -16px 20px;
  }
}

@media (min-width: 600px) and (max-width: 1023px) and (min-height: 900px) {
  .login-mobile-brand {
    margin-bottom: clamp(28px, 3vh, 40px);
  }

  .login-flow-visual {
    height: clamp(108px, 10vh, 132px);
    margin-bottom: clamp(32px, 3.5vh, 46px);
  }

  .login-kicker {
    margin-bottom: clamp(22px, 2vh, 28px);
  }

  .login-welcome p {
    margin-top: 8px;
    margin-bottom: clamp(34px, 3.5vh, 48px);
  }

  .login-form {
    gap: clamp(20px, 2.2vh, 28px);
  }
}
</style>
