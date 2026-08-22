<template>
  <div
    class="login-page flex justify-center items-center full-height"
    :class="{ 'login-page--dark': $q.dark.isActive }"
  >
    <corner-bottom :start-color="'#1976D2'" :end-color="'#00ACC1'" class="wave fit" />
    <div class="login-visual col-6 flex justify-center items-center" v-show="$q.screen.gt.sm">
      <login-illustration />
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
  </div>
</template>

<script setup lang="ts">
import type { SignInReq } from 'src/api/services/basic'
import { useBasicApi } from 'src/api/services/basic'
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from 'src/stores/user'
import { useQuasar } from 'quasar'

const $q = useQuasar()

import CornerBottom from 'src/components/Login/CornerBottom.vue'
import LoginIllustration from 'src/components/Login/LoginIllustration.vue'
import LoginPanel from 'src/components/Login/LoginPanel.vue'
import { useLoadingStore } from 'stores/loading'
import { storeToRefs } from 'pinia'

defineOptions({ name: 'Login' })

const userStore = useUserStore()
const router = useRouter()
const message = ref<string>('')
const { login } = useBasicApi()

const loadingStore = useLoadingStore()
const { loading } = storeToRefs(loadingStore)

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
  position: relative;
  overflow: hidden;
  display: grid !important;
  grid-template-columns: minmax(420px, 1fr) minmax(360px, 520px);
  gap: clamp(32px, 6vw, 88px);
  padding: 48px clamp(32px, 7vw, 96px);
  background:
    linear-gradient(120deg, rgba(25, 118, 210, 0.08) 0 32%, transparent 32% 100%),
    linear-gradient(135deg, #eef6f2 0%, #f8fafc 48%, #fff8ed 100%);
}

.wave {
  position: absolute;
  left: 0;
  bottom: 0;
  z-index: -1;
  opacity: 0.55;
}

.login-visual {
  min-width: 420px;
  justify-self: end;
}

.login-panel-wrap {
  width: min(500px, calc(100vw - 32px));
  display: flex;
  justify-content: center;
  align-items: center;
}

.login-page--dark {
  background:
    linear-gradient(120deg, rgba(38, 166, 154, 0.12) 0 32%, transparent 32% 100%),
    linear-gradient(135deg, #0f172a 0%, #111827 58%, #1f2937 100%);
}

@media (max-width: 1023px) {
  .login-page {
    display: flex !important;
    padding: 32px 16px;
  }
}
</style>
