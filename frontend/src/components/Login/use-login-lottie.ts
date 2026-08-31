import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useQuasar } from 'quasar'
import lottie, { type AnimationItem } from 'lottie-web/build/player/lottie_light'

type AnimationFactory = (dark: boolean) => object

export const useLoginLottie = (createAnimation: AnimationFactory, stillFrame = 45) => {
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
      animationData: createAnimation($q.dark.isActive),
      rendererSettings: {
        preserveAspectRatio: 'xMidYMid meet',
        progressiveLoad: true,
      },
    })

    if (reduceMotion) {
      animation.goToAndStop(stillFrame, true)
    }
  }

  onMounted(renderAnimation)
  watch(() => $q.dark.isActive, renderAnimation)

  onBeforeUnmount(() => {
    animation?.destroy()
    animation = null
  })

  return { containerRef }
}
