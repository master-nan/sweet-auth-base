<template>
  <div class="change-password-page">
    <section class="change-password-panel">
      <div>
        <div class="text-h5 text-weight-medium">修改密码</div>
        <div class="text-body2 text-grey-7 q-mt-sm">{{ reasonText }}</div>
      </div>

      <q-form class="q-gutter-md q-mt-lg" @submit="submit">
        <q-input
          v-model="password"
          outlined
          dense
          label="新密码"
          :type="showPassword ? 'text' : 'password'"
          :rules="passwordRules"
          autocomplete="new-password"
        >
          <template #append>
            <q-btn
              flat
              dense
              round
              :icon="showPassword ? 'visibility_off' : 'visibility'"
              @click="showPassword = !showPassword"
            />
          </template>
        </q-input>

        <q-input
          v-model="confirmPassword"
          outlined
          dense
          label="确认密码"
          :type="showPassword ? 'text' : 'password'"
          :rules="confirmRules"
          autocomplete="new-password"
        />

        <div class="text-caption text-grey-7">{{ policyText }}</div>

        <div class="row items-center justify-end q-gutter-sm">
          <q-btn flat color="grey-8" label="退出登录" @click="logout" />
          <q-btn color="primary" label="保存" type="submit" :loading="loading" />
        </div>
      </q-form>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { Notify } from 'quasar'
import { useSysUserApi } from 'src/api/services/sys-user'
import { useConfigureStore } from 'src/stores/configure'
import { useUserStore } from 'src/stores/user'
import { useLoadingStore } from 'src/stores/loading'
import { buildPasswordRules, passwordPolicyDescription } from 'src/utils/passwordPolicy'

defineOptions({ name: 'ChangePassword' })

const userStore = useUserStore()
const configureStore = useConfigureStore()
const loadingStore = useLoadingStore()
const { loading } = storeToRefs(loadingStore)
const sysUserApi = useSysUserApi()

const password = ref('')
const confirmPassword = ref('')
const showPassword = ref(false)

const passwordRules = computed(() => buildPasswordRules(configureStore.$state))
const confirmRules = computed(() => [
  (val: string | null | undefined) => (val ?? '') === password.value || '两次输入的密码不一致',
])
const policyText = computed(() => passwordPolicyDescription(configureStore.$state))
const reasonText = computed(() => {
  if (userStore.password_change_reason === 'expired') {
    return '当前密码已过期，请设置新密码后继续使用。'
  }
  return '当前账号需要先修改密码。'
})

onMounted(async () => {
  await configureStore.fetchConfigure()
})

const submit = async () => {
  const result = await sysUserApi.updatePassword(password.value)
  if (result.success) {
    userStore.setPasswordChangeRequirement(false)
    Notify.create({ type: 'positive', position: 'top-right', message: '密码已修改，请重新登录' })
    userStore.setLogout()
  }
}

const logout = async () => {
  await userStore.logout()
}
</script>

<style scoped>
.change-password-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: #f6f8fb;
}

.change-password-panel {
  width: min(420px, 100%);
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 28px;
  box-shadow: 0 12px 32px rgba(15, 23, 42, 0.08);
}
</style>
