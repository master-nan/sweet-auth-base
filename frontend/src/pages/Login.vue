<template>
  <div class="login-page" :class="{ 'login-page--dark': $q.dark.isActive }">
    <div v-if="$q.screen.gt.sm" class="login-visual">
      <login-illustration />
    </div>
    <section class="login-side">
      <div class="login-actions">
        <q-btn
          class="login-action-btn"
          dense
          flat
          icon="language"
          aria-label="切换语言"
        >
          <q-tooltip>切换语言</q-tooltip>
          <q-menu anchor="bottom right" self="top right">
            <q-list dense class="login-language-menu">
              <q-item
                v-for="option in localeOptions"
                :key="option.value"
                v-close-popup
                clickable
                @click="setLocale(option.value)"
              >
                <q-item-section>{{ option.label }}</q-item-section>
                <q-item-section side>
                  <q-icon v-if="locale === option.value" name="check" color="primary" size="xs" />
                </q-item-section>
              </q-item>
            </q-list>
          </q-menu>
        </q-btn>
        <dark-mode />
      </div>

      <div class="login-panel-wrap">
        <login-panel
          v-model:username="loginData.user_name"
          v-model:password="loginData.password"
          v-model:captcha="loginData.captcha"
          v-model:captcha_id="loginData.captcha_id"
          v-model:loading="loading"
          @onLoginClick="onLoginClick"
          :message="message"
        />
      </div>
      <footer class="login-footer">
        <span>© 2026 Sweet Admin</span>
        <span class="login-secure" :class="`login-secure--${connectionState}`">
          <i />{{ connectionLabel }}
        </span>
      </footer>
    </section>
  </div>
</template>

<script setup lang="ts">
import type { SignInReq } from 'src/api/services/basic'
import { useBasicApi } from 'src/api/services/basic'
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from 'src/stores/user'
import { useQuasar } from 'quasar'
import { useI18n } from 'vue-i18n'
import { writeUIPreferences, type SupportedLocale } from 'src/utils/ui-preferences'

const $q = useQuasar()

const isLocalConnection = ['localhost', '127.0.0.1', '::1'].includes(window.location.hostname)
const connectionState =
  window.location.protocol === 'https:' ? 'secure' : isLocalConnection ? 'local' : 'warning'
const connectionLabel =
  connectionState === 'secure'
    ? '加密连接'
    : connectionState === 'local'
      ? '本地连接'
      : '未加密连接'

import LoginIllustration from 'src/components/Login/LoginIllustration.vue'
import LoginPanel from 'src/components/Login/LoginPanel.vue'
import DarkMode from 'src/components/Toolbar/DarkMode.vue'
import { useLoadingStore } from 'stores/loading'
import { storeToRefs } from 'pinia'

defineOptions({ name: 'Login' })

const userStore = useUserStore()
const router = useRouter()
const { locale } = useI18n({ useScope: 'global' })
const message = ref<string>('')
const { login } = useBasicApi()

const loadingStore = useLoadingStore()
const { loading } = storeToRefs(loadingStore)
const localeOptions: Array<{ value: SupportedLocale; label: string }> = [
  { value: 'zh-CN', label: '简体中文' },
  { value: 'en-US', label: 'English' },
]

const setLocale = (value: SupportedLocale) => {
  locale.value = value
  writeUIPreferences({ locale: value })
}

// 登录表单只保存用户输入；Token和当前用户状态由User Store接管。
const loginData: SignInReq = reactive({
  user_name: '',
  password: '',
  captcha: '',
  captcha_id: '',
})

const onLoginClick = async () => {
  message.value = ''
  try {
    const response = await login(loginData)
    if (response.success) {
      userStore.setLoginToken(response.data.access_token)
      userStore.setPasswordChangeRequirement(
        response.data.must_change_password,
        response.data.password_change_reason || '',
      )
      await router.push(response.data.must_change_password ? '/change-password' : '/admin/home')
    }
  } catch (error) {
    const response = (error as { response?: { data?: { error_message?: string; message?: string } } })
      .response
    message.value = response?.data?.error_message || response?.data?.message || '登录失败，请重试'
  }
}
</script>

<style scoped>
.login-page {
  width: 100%;
  min-height: 100vh;
  display: grid;
  grid-template-columns: minmax(600px, 1.4fr) minmax(430px, 0.76fr);
  overflow: hidden;
  background: #ffffff;
}

.login-visual {
  min-width: 0;
  height: 100vh;
  overflow: hidden;
}

.login-side {
  position: relative;
  min-width: 0;
  min-height: 100vh;
  display: grid;
  grid-template-rows: 1fr auto;
  padding: clamp(30px, 4vw, 54px) clamp(34px, 5vw, 72px) 24px;
  border-left: 1px solid rgba(23, 32, 51, 0.08);
  background: #ffffff;
  color: #172033;
}

.login-panel-wrap {
  width: 100%;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: clamp(28px, 7vh, 76px);
}

.login-actions {
  position: absolute;
  z-index: 3;
  top: clamp(22px, 3vw, 32px);
  right: clamp(24px, 4vw, 44px);
  display: flex;
  align-items: center;
  gap: 8px;
}

.login-action-btn,
.login-actions :deep(.dark-mode-btn) {
  width: 36px;
  min-width: 36px;
  height: 36px;
  min-height: 36px;
  padding: 0;
  border: 1px solid #d8deea;
  border-radius: 50%;
  background: #fbfcfd;
  color: #151a27;
}

.login-action-btn:hover,
.login-actions :deep(.dark-mode-btn:hover) {
  color: var(--q-primary);
  background: #f3f5fa;
}

.login-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding-top: 22px;
  border-top: 1px solid rgba(23, 32, 51, 0.08);
  color: rgba(23, 32, 51, 0.48);
  font-size: 11px;
}

.login-secure {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  white-space: nowrap;
}

.login-secure i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #55c991;
  box-shadow: 0 0 10px rgba(85, 201, 145, 0.66);
}

.login-secure--local i {
  background: #31a8be;
  box-shadow: 0 0 10px rgba(49, 168, 190, 0.58);
}

.login-secure--warning i {
  background: #efb34b;
  box-shadow: 0 0 10px rgba(239, 179, 75, 0.56);
}

.login-page--dark {
  background: #171a26;
}

.login-page--dark .login-side {
  border-left-color: rgba(255, 255, 255, 0.08);
  background: #1d202c;
  color: #f7f8fb;
}

.login-page--dark .login-action-btn,
.login-page--dark .login-actions :deep(.dark-mode-btn) {
  border-color: #3a4050;
  background: #242835;
  color: #f2f4f8;
}

.login-page--dark .login-action-btn:hover,
.login-page--dark .login-actions :deep(.dark-mode-btn:hover) {
  color: #a99ff8;
  background: #2b3040;
}

.login-page--dark .login-footer {
  border-top-color: rgba(255, 255, 255, 0.08);
  color: rgba(247, 248, 251, 0.44);
}

@media (max-width: 1023px) {
  .login-page {
    display: block;
  }

  .login-side {
    width: 100%;
    padding: 28px clamp(20px, 7vw, 64px) 22px;
    border-left: 0;
  }

  .login-panel-wrap {
    padding-top: 0;
  }

  .login-actions {
    top: 28px;
    right: clamp(20px, 7vw, 64px);
  }

  .login-footer {
    width: min(100%, 440px);
    margin: 0 auto;
  }
}

@media (max-width: 599px) {
  .login-side {
    padding: 22px 22px 18px;
  }

  .login-actions {
    top: 22px;
    right: 22px;
  }

  .login-footer {
    align-items: flex-start;
    font-size: 10px;
  }
}
</style>
