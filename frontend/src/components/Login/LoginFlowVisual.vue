<template>
  <div ref="containerRef" class="login-flow-lottie" aria-hidden="true" />
</template>

<script lang="ts" setup>
import { createLoginFlowAnimation } from '@/components/Login/login-flow-animation'
import { useLoginLottie } from '@/components/Login/use-login-lottie'

defineOptions({ name: 'LoginFlowVisual' })

const { containerRef } = useLoginLottie(createLoginFlowAnimation)
</script>

<style scoped lang="scss">
.login-flow-lottie {
  width: 100%;
  height: 100%;
  overflow: hidden;
}

.login-flow-lottie :deep(svg) {
  display: block;
}
</style>
