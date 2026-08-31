<template>
  <section class="login-illustration" :class="{ 'login-illustration--dark': $q.dark.isActive }">
    <canvas ref="canvasRef" class="flight-canvas" aria-hidden="true" />

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

    <div class="visual-copy">
      <div class="visual-eyebrow">连接每一条业务路径</div>
      <h1>让每一条业务路径<br />都有清晰的方向</h1>
      <p>统一身份、组织权限、数据管理与系统集成，在稳定底座上连接每一个业务现场。</p>
      <div class="visual-capabilities">
        <span><i />身份与权限</span>
        <span><i />组织与数据</span>
        <span><i />集成与审计</span>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useQuasar } from 'quasar'
import { useConfigureStore } from 'stores/configure'

defineOptions({ name: 'LoginIllustration' })

interface FlightStar {
  x: number
  y: number
  z: number
  speed: number
  colorIndex: number
}

const lightPalette = ['#687188', '#7568dc', '#288da2', '#c9614d']
const darkPalette = ['#ffffff', '#9b8aff', '#5fc2d8', '#e27662']
const routeColors = ['#7568dc', '#288da2', '#c9614d']

const $q = useQuasar()
const configureStore = useConfigureStore()
const canvasRef = ref<HTMLCanvasElement | null>(null)
const systemName = computed(() => configureStore.getSystemName || 'Sweet Admin')
const systemLogo = computed(() => configureStore.getSystemLogo || '')
const systemDescription = computed(() => configureStore.getSystemDescription || '')

let context: CanvasRenderingContext2D | null = null
let resizeObserver: ResizeObserver | null = null
let animationFrame = 0
let width = 0
let height = 0
let stars: FlightStar[] = []
let lastTime = 0
let reducedMotion = false

const resetStar = (star: FlightStar, initial = false) => {
  star.x = Math.random() * 2 - 1
  star.y = Math.random() * 2 - 1
  star.z = initial ? Math.random() : 1
  star.speed = 0.00011 + Math.random() * 0.00018
  star.colorIndex = Math.floor(Math.random() * lightPalette.length)
}

const createStar = (): FlightStar => {
  const star: FlightStar = { x: 0, y: 0, z: 1, speed: 0, colorIndex: 0 }
  resetStar(star, true)
  return star
}

const resizeCanvas = () => {
  const canvas = canvasRef.value
  if (!canvas) return

  const ratio = Math.min(window.devicePixelRatio || 1, 2)
  const rect = canvas.getBoundingClientRect()
  width = rect.width
  height = rect.height
  canvas.width = Math.round(width * ratio)
  canvas.height = Math.round(height * ratio)
  context = canvas.getContext('2d')
  context?.setTransform(ratio, 0, 0, ratio, 0, 0)
  const count = Math.max(120, Math.round((width * height) / 5700))
  stars = Array.from({ length: count }, createStar)
}

const projectStar = (star: FlightStar, zOffset = 0) => {
  const z = Math.max(0.025, star.z + zOffset)
  const spread = Math.min(width, height) * 0.72
  return {
    x: width * 0.62 + (star.x / z) * spread,
    y: height * 0.39 + (star.y / z) * spread,
  }
}

const drawStar = (star: FlightStar, delta: number, dark: boolean) => {
  if (!context) return

  const previous = projectStar(star, delta * star.speed * 1.8)
  star.z -= delta * star.speed
  if (star.z < 0.035) resetStar(star)
  const current = projectStar(star)
  if (
    current.x < -80 ||
    current.x > width + 80 ||
    current.y < -80 ||
    current.y > height + 80
  ) {
    resetStar(star)
    return
  }

  const palette = dark ? darkPalette : lightPalette
  const alpha = dark
    ? Math.min(0.82, Math.max(0.08, (1 - star.z) * 0.9))
    : Math.min(0.76, Math.max(0.14, (1 - star.z) * 0.84))

  context.beginPath()
  context.moveTo(previous.x, previous.y)
  context.lineTo(current.x, current.y)
  context.strokeStyle = `${palette[star.colorIndex]}${Math.round(alpha * 255)
    .toString(16)
    .padStart(2, '0')}`
  context.lineWidth = Math.max(0.5, (1 - star.z) * 2.1)
  context.stroke()
}

const curvePoint = (points: number[], progress: number) => {
  const inverse = 1 - progress
  return {
    x:
      inverse ** 3 * (points[0] ?? 0) +
      3 * inverse ** 2 * progress * (points[2] ?? 0) +
      3 * inverse * progress ** 2 * (points[4] ?? 0) +
      progress ** 3 * (points[6] ?? 0),
    y:
      inverse ** 3 * (points[1] ?? 0) +
      3 * inverse ** 2 * progress * (points[3] ?? 0) +
      3 * inverse * progress ** 2 * (points[5] ?? 0) +
      progress ** 3 * (points[7] ?? 0),
  }
}

const drawRoute = (points: number[], color: string, phase: number, time: number, dark: boolean) => {
  if (!context) return

  context.beginPath()
  context.moveTo(points[0] ?? 0, points[1] ?? 0)
  context.bezierCurveTo(
    points[2] ?? 0,
    points[3] ?? 0,
    points[4] ?? 0,
    points[5] ?? 0,
    points[6] ?? 0,
    points[7] ?? 0,
  )
  context.strokeStyle = `${color}${dark ? '64' : '82'}`
  context.lineWidth = 1.2
  context.stroke()

  const progress = ((time * 0.000045 + phase) % 1 + 1) % 1
  const point = curvePoint(points, progress)
  const glow = context.createRadialGradient(point.x, point.y, 0, point.x, point.y, 18)
  glow.addColorStop(0, `${color}f2`)
  glow.addColorStop(0.24, `${color}8a`)
  glow.addColorStop(1, `${color}00`)
  context.fillStyle = glow
  context.beginPath()
  context.arc(point.x, point.y, 18, 0, Math.PI * 2)
  context.fill()
  context.fillStyle = dark ? '#ffffff' : '#303746'
  context.beginPath()
  context.arc(point.x, point.y, 2.2, 0, Math.PI * 2)
  context.fill()
}

const drawFrame = (time: number) => {
  if (!context) return

  const dark = $q.dark.isActive
  const delta = Math.min(42, time - lastTime || 16)
  lastTime = time
  context.fillStyle = dark ? '#171a26' : '#edf1f7'
  context.fillRect(0, 0, width, height)

  const horizon = context.createRadialGradient(
    width * 0.66,
    height * 0.38,
    0,
    width * 0.66,
    height * 0.38,
    width * 0.7,
  )
  horizon.addColorStop(0, dark ? 'rgba(107, 88, 230, 0.16)' : 'rgba(107, 88, 230, 0.12)')
  horizon.addColorStop(0.48, dark ? 'rgba(55, 132, 142, 0.07)' : 'rgba(55, 132, 142, 0.08)')
  horizon.addColorStop(1, dark ? 'rgba(23, 26, 38, 0)' : 'rgba(237, 241, 247, 0)')
  context.fillStyle = horizon
  context.fillRect(0, 0, width, height)

  const motionDelta = reducedMotion ? delta * 0.12 : delta
  stars.forEach((star) => drawStar(star, motionDelta, dark))
  const animationTime = reducedMotion ? 0 : time

  drawRoute(
    [
      width * -0.1,
      height * 0.58,
      width * 0.28,
      height * 0.04,
      width * 0.72,
      height * 0.72,
      width * 1.08,
      height * 0.26,
    ],
    routeColors[0] ?? '#7568dc',
    0,
    animationTime,
    dark,
  )
  drawRoute(
    [
      width * -0.08,
      height * 0.76,
      width * 0.34,
      height * 0.96,
      width * 0.56,
      height * 0.02,
      width * 1.06,
      height * 0.48,
    ],
    routeColors[1] ?? '#288da2',
    0.38,
    animationTime,
    dark,
  )
  drawRoute(
    [
      width * 0.04,
      height * 0.18,
      width * 0.42,
      height * 0.58,
      width * 0.74,
      height * 0.06,
      width * 1.08,
      height * 0.7,
    ],
    routeColors[2] ?? '#c9614d',
    0.72,
    animationTime,
    dark,
  )

  animationFrame = window.requestAnimationFrame(drawFrame)
}

watch(
  () => $q.dark.isActive,
  () => stars.forEach((star) => resetStar(star, true)),
)

onMounted(async () => {
  reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  await nextTick()
  resizeCanvas()
  if (canvasRef.value) {
    resizeObserver = new ResizeObserver(resizeCanvas)
    resizeObserver.observe(canvasRef.value)
  }
  animationFrame = window.requestAnimationFrame(drawFrame)
})

onBeforeUnmount(() => {
  resizeObserver?.disconnect()
  window.cancelAnimationFrame(animationFrame)
})
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

.flight-canvas {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
}

.visual-brand,
.visual-copy {
  position: absolute;
  z-index: 1;
}

.visual-brand {
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

.visual-copy {
  left: clamp(30px, 4vw, 50px);
  bottom: clamp(38px, 5vw, 62px);
  width: min(610px, calc(100% - 80px));
}

.visual-eyebrow {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
  color: rgba(23, 32, 51, 0.62);
  font-size: 11px;
  font-weight: 800;
  text-transform: uppercase;
}

.visual-eyebrow::before {
  content: '';
  width: 34px;
  height: 1px;
  background: #7568dc;
}

.visual-copy h1 {
  margin: 0;
  font-size: clamp(34px, 3.3vw, 46px);
  line-height: 1.16;
  letter-spacing: 0;
}

.visual-copy p {
  max-width: 560px;
  margin: 13px 0 0;
  color: rgba(23, 32, 51, 0.62);
  font-size: 14px;
  line-height: 1.75;
}

.visual-capabilities {
  display: flex;
  gap: 22px;
  margin-top: 22px;
  color: rgba(23, 32, 51, 0.62);
  font-size: 11px;
}

.visual-capabilities span {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.visual-capabilities i {
  width: 16px;
  height: 2px;
  background: #7568dc;
}

.visual-capabilities span:nth-child(2) i {
  background: #288da2;
}

.visual-capabilities span:nth-child(3) i {
  background: #c9614d;
}

.login-illustration--dark {
  background: #171a26;
  color: #f7f8fb;
}

.login-illustration--dark .visual-brand-logo {
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
}

.login-illustration--dark .visual-brand-subtitle,
.login-illustration--dark .visual-eyebrow,
.login-illustration--dark .visual-copy p,
.login-illustration--dark .visual-capabilities {
  color: rgba(247, 248, 251, 0.62);
}

@media (prefers-reduced-motion: reduce) {
  .flight-canvas {
    opacity: 0.88;
  }
}
</style>
