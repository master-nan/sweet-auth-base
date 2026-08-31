<template>
  <div ref="containerRef" class="login-flow-lottie" aria-hidden="true" />
</template>

<script lang="ts" setup>
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useQuasar } from 'quasar'
import lottie, { type AnimationItem } from 'lottie-web/build/player/lottie_light'
import { createLoginFlowAnimation } from 'src/components/Login/login-flow-animation'

defineOptions({ name: 'LoginFlowVisual' })

const $q = useQuasar()
const containerRef = ref<HTMLElement | null>(null)
let animation: AnimationItem | null = null

const renderAnimation = () => {
  if (containerRef.value == null) return

  animation?.destroy()
  const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  animation = lottie.loadAnimation({
    container: containerRef.value,
    renderer: 'svg',
    loop: !reduceMotion,
    autoplay: !reduceMotion,
    animationData: createLoginFlowAnimation($q.dark.isActive),
    rendererSettings: {
      preserveAspectRatio: 'xMidYMid meet',
      progressiveLoad: true,
    },
  })

  if (reduceMotion) {
    animation.goToAndStop(45, true)
  }
}

onMounted(renderAnimation)
watch(() => $q.dark.isActive, renderAnimation)

onBeforeUnmount(() => {
  animation?.destroy()
  animation = null
})
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
