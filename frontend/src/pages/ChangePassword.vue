<template>
  <div class="change-password-page">
    <section class="change-password-panel">
      <div>
        <div class="text-h5 text-weight-medium">{{ t('ui.changePassword') }}</div>
        <div class="text-body2 text-grey-7 q-mt-sm">{{ reasonText }}</div>
      </div>

      <q-form class="q-gutter-md q-mt-lg" @submit="submit">
        <q-input
          v-model="password"
          outlined
          dense
          :label="t('ui.newPassword')"
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
          :label="t('ui.confirmPassword')"
          :type="showPassword ? 'text' : 'password'"
          :rules="confirmRules"
          autocomplete="new-password"
        />

        <div class="text-caption text-grey-7">{{ policyText }}</div>

        <div class="row items-center justify-end q-gutter-sm">
          <q-btn flat color="grey-8" :label="t('ui.exitLogin')" @click="logout" />
          <q-btn color="primary" :label="t('ui.save')" type="submit" :loading="loading" />
        </div>
      </q-form>
    </section>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { computed, onMounted, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { Notify } from 'quasar'
import { useSysUserApi } from '@/api/services/sys-user'
import { useConfigureStore } from '@/stores/configure'
import { useUserStore } from '@/stores/user'
import { useLoadingStore } from '@/stores/loading'
import { buildPasswordRules, passwordPolicyDescription } from '@/utils/passwordPolicy'

const { t } = useI18n({ useScope: 'global' })

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
  (val: string | null | undefined) =>
    (val ?? '') === password.value || t('ui.thePasswordsDoNotMatch'),
])
const policyText = computed(() => passwordPolicyDescription(configureStore.$state))
const reasonText = computed(() => {
  if (userStore.password_change_reason === 'expired') {
    return t('ui.theCurrentPasswordIsExpiredAndPleaseSetANew')
  }
  return t('ui.theCurrentAccountNumberRequiresAPasswordChangeFirst')
})

onMounted(async () => {
  await configureStore.fetchConfigure()
})

const submit = async () => {
  const result = await sysUserApi.updatePassword(password.value)
  if (result.success) {
    userStore.setPasswordChangeRequirement(false)
    Notify.create({
      type: 'positive',
      position: 'top-right',
      get message() {
        return t('ui.passwordModifiedPleaseReEnter')
      },
    })
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
