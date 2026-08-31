<template>
  <section class="login-illustration" :class="{ 'login-illustration--dark': $q.dark.isActive }">
    <header class="visual-brand">
      <div class="visual-brand-logo">
        <img v-if="systemLogo" :src="systemLogo" alt="" />
        <q-icon v-else name="admin_panel_settings" />
      </div>
      <div>
        <div class="visual-brand-name">{{ systemName }}</div>
        <div class="visual-brand-subtitle">{{ systemDescription || '通用低代码底座' }}</div>
      </div>
    </header>

    <div ref="containerRef" class="platform-lottie" aria-hidden="true" />
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useQuasar } from 'quasar'
import { useConfigureStore } from 'stores/configure'
import { createLoginPlatformAnimation } from 'src/components/Login/login-flow-animation'
import { useLoginLottie } from 'src/components/Login/use-login-lottie'

defineOptions({ name: 'LoginIllustration' })

const $q = useQuasar()
const configureStore = useConfigureStore()
const systemName = computed(() => configureStore.getSystemName || 'Sweet Admin')
const systemLogo = computed(() => configureStore.getSystemLogo || '')
const systemDescription = computed(() => configureStore.getSystemDescription || '')
const { containerRef } = useLoginLottie(createLoginPlatformAnimation)
</script>

<style scoped lang="scss">
.login-illustration {
  position: relative;
  width: 100%;
  height: 100%;
  min-height: 640px;
  overflow: hidden;
  background: #edf1f7;
  color: #172033;
  transition:
    background 180ms ease,
    color 180ms ease;
}

.visual-brand {
  position: absolute;
  z-index: 1;
  top: clamp(28px, 4vw, 44px);
  left: clamp(30px, 4vw, 50px);
  display: flex;
  align-items: center;
  gap: 12px;
}

.visual-brand-logo {
  position: relative;
  isolation: isolate;
  width: 42px;
  height: 42px;
  display: grid;
  place-items: center;
  overflow: hidden;
  border-radius: 8px;
  background: #252b38;
  color: #fff;
  box-shadow: 0 8px 22px rgba(40, 52, 78, 0.16);
}

.visual-brand-logo::before,
.visual-brand-logo::after {
  position: absolute;
  content: '';
}

.visual-brand-logo::before {
  z-index: 0;
  inset: 0;
  background: conic-gradient(#7568dc, #31a8be, #55c991, #efb34b, #d86f62, #7568dc);
}

.visual-brand-logo::after {
  z-index: 1;
  inset: 3px;
  border-radius: 6px;
  background: #252b38;
}

.visual-brand-logo .q-icon {
  position: relative;
  z-index: 2;
  font-size: 22px;
}

.visual-brand-logo img {
  position: relative;
  z-index: 2;
  width: calc(100% - 6px);
  height: calc(100% - 6px);
  border-radius: 6px;
  object-fit: cover;
}

.visual-brand-name {
  font-size: 21px;
  font-weight: 800;
  line-height: 1.15;
}

.visual-brand-subtitle {
  margin-top: 4px;
  color: rgba(23, 32, 51, 0.56);
  font-size: 12px;
}

.platform-lottie {
  position: absolute;
  top: 50%;
  left: 50%;
  width: min(82%, 820px);
  height: min(72%, 620px);
  transform: translate(-50%, -47%);
}

.platform-lottie :deep(svg) {
  display: block;
}

.login-illustration--dark {
  background: #171a26;
  color: #f7f8fb;
}

.login-illustration--dark .visual-brand-logo {
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
}

.login-illustration--dark .visual-brand-subtitle {
  color: rgba(247, 248, 251, 0.62);
}
</style>
