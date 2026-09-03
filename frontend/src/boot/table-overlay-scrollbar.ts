import { defineBoot } from '#q-app'

type Axis = 'vertical' | 'horizontal'

interface OverlayState {
  host: HTMLElement
  middle: HTMLElement
  vertical: HTMLElement
  verticalThumb: HTMLElement
  horizontal: HTMLElement
  horizontalThumb: HTMLElement
  resizeObserver: ResizeObserver
  frame: number | null
  hideTimer: number | null
  cleanup: Array<() => void>
}

const MIDDLE_SELECTOR = '.q-table__middle'
const INSET = 2
const RAIL_SIZE = 10
const MIN_THUMB_SIZE = 28

function createRail(axis: Axis) {
  const rail = document.createElement('div')
  const thumb = document.createElement('div')
  rail.className = `app-table-overlay-scrollbar app-table-overlay-scrollbar--${axis}`
  rail.setAttribute('aria-hidden', 'true')
  thumb.className = 'app-table-overlay-scrollbar__thumb'
  rail.appendChild(thumb)
  return { rail, thumb }
}

function thumbSize(viewport: number, content: number, track: number) {
  return Math.min(track, Math.max(MIN_THUMB_SIZE, (viewport / content) * track))
}

function updateAxis(
  rail: HTMLElement,
  thumb: HTMLElement,
  viewport: number,
  content: number,
  offset: number,
  track: number,
) {
  const maxScroll = Math.max(0, content - viewport)
  const scrollable = maxScroll > 1 && track > 0
  rail.classList.toggle('app-table-overlay-scrollbar--scrollable', scrollable)
  if (!scrollable) return

  const size = thumbSize(viewport, content, track)
  const travel = Math.max(0, track - size)
  const position = maxScroll === 0 ? 0 : (offset / maxScroll) * travel
  const vertical = rail.classList.contains('app-table-overlay-scrollbar--vertical')
  if (vertical) {
    thumb.style.height = `${size}px`
    thumb.style.transform = `translateY(${position}px)`
  } else {
    thumb.style.width = `${size}px`
    thumb.style.transform = `translateX(${position}px)`
  }
}

function update(state: OverlayState) {
  state.frame = null
  if (!state.host.isConnected || !state.middle.isConnected) return

  const hostRect = state.host.getBoundingClientRect()
  const middleRect = state.middle.getBoundingClientRect()
  const hasVertical = state.middle.scrollHeight - state.middle.clientHeight > 1
  const hasHorizontal = state.middle.scrollWidth - state.middle.clientWidth > 1
  const verticalTrack = Math.max(
    0,
    middleRect.height - INSET * 2 - (hasHorizontal ? RAIL_SIZE : 0),
  )
  const horizontalTrack = Math.max(
    0,
    middleRect.width - INSET * 2 - (hasVertical ? RAIL_SIZE : 0),
  )

  state.vertical.style.top = `${middleRect.top - hostRect.top + INSET}px`
  state.vertical.style.right = `${hostRect.right - middleRect.right + INSET}px`
  state.vertical.style.height = `${verticalTrack}px`
  state.horizontal.style.left = `${middleRect.left - hostRect.left + INSET}px`
  state.horizontal.style.top = `${middleRect.bottom - hostRect.top - RAIL_SIZE - INSET}px`
  state.horizontal.style.width = `${horizontalTrack}px`

  updateAxis(
    state.vertical,
    state.verticalThumb,
    state.middle.clientHeight,
    state.middle.scrollHeight,
    state.middle.scrollTop,
    verticalTrack,
  )
  updateAxis(
    state.horizontal,
    state.horizontalThumb,
    state.middle.clientWidth,
    state.middle.scrollWidth,
    state.middle.scrollLeft,
    horizontalTrack,
  )
}

function scheduleUpdate(state: OverlayState) {
  if (state.frame !== null) return
  state.frame = window.requestAnimationFrame(() => update(state))
}

function scheduleHide(state: OverlayState) {
  if (state.hideTimer !== null) window.clearTimeout(state.hideTimer)
  state.hideTimer = window.setTimeout(() => {
    delete state.host.dataset.appTableOverlayActive
    state.hideTimer = null
  }, 500)
}

function listen(
  state: OverlayState,
  target: EventTarget,
  event: string,
  handler: EventListener,
) {
  target.addEventListener(event, handler)
  state.cleanup.push(() => target.removeEventListener(event, handler))
}

function bindDrag(state: OverlayState, axis: Axis, thumb: HTMLElement) {
  listen(state, thumb, 'pointerdown', ((event: PointerEvent) => {
    event.preventDefault()
    event.stopPropagation()
    state.host.dataset.appTableOverlayActive = 'true'

    const vertical = axis === 'vertical'
    const startPointer = vertical ? event.clientY : event.clientX
    const startScroll = vertical ? state.middle.scrollTop : state.middle.scrollLeft
    const viewport = vertical ? state.middle.clientHeight : state.middle.clientWidth
    const content = vertical ? state.middle.scrollHeight : state.middle.scrollWidth
    const track = vertical ? state.vertical.clientHeight : state.horizontal.clientWidth
    const travel = Math.max(1, track - thumbSize(viewport, content, track))
    const maxScroll = Math.max(0, content - viewport)

    const move = (moveEvent: PointerEvent) => {
      const pointer = vertical ? moveEvent.clientY : moveEvent.clientX
      const next = startScroll + ((pointer - startPointer) / travel) * maxScroll
      if (vertical) state.middle.scrollTop = next
      else state.middle.scrollLeft = next
    }
    const stop = () => {
      window.removeEventListener('pointermove', move)
      window.removeEventListener('pointerup', stop)
      window.removeEventListener('pointercancel', stop)
      scheduleHide(state)
    }

    window.addEventListener('pointermove', move)
    window.addEventListener('pointerup', stop)
    window.addEventListener('pointercancel', stop)
  }) as EventListener)
}

function initialize(middle: HTMLElement, states: Set<OverlayState>) {
  if (middle.dataset.appTableOverlay === 'ready') return
  if (middle.closest('.q-scrollarea')) return

  const overflow = window.getComputedStyle(middle)
  const hasNativeScrollContainer = [overflow.overflowX, overflow.overflowY].some((value) =>
    ['auto', 'scroll', 'overlay'].includes(value),
  )
  if (!hasNativeScrollContainer) return

  const host = middle.closest<HTMLElement>('.q-table__container')
  if (!host) return

  const vertical = createRail('vertical')
  const horizontal = createRail('horizontal')
  const state: OverlayState = {
    host,
    middle,
    vertical: vertical.rail,
    verticalThumb: vertical.thumb,
    horizontal: horizontal.rail,
    horizontalThumb: horizontal.thumb,
    resizeObserver: undefined as unknown as ResizeObserver,
    frame: null,
    hideTimer: null,
    cleanup: [],
  }

  middle.dataset.appTableOverlay = 'ready'
  host.dataset.appTableOverlayHost = 'ready'
  host.append(vertical.rail, horizontal.rail)
  state.resizeObserver = new ResizeObserver(() => scheduleUpdate(state))
  state.resizeObserver.observe(host)
  state.resizeObserver.observe(middle)
  if (middle.firstElementChild instanceof HTMLElement) {
    state.resizeObserver.observe(middle.firstElementChild)
  }

  listen(state, middle, 'scroll', () => {
    host.dataset.appTableOverlayActive = 'true'
    scheduleUpdate(state)
    scheduleHide(state)
  })
  bindDrag(state, 'vertical', vertical.thumb)
  bindDrag(state, 'horizontal', horizontal.thumb)
  states.add(state)
  scheduleUpdate(state)
}

function destroy(state: OverlayState, states: Set<OverlayState>) {
  if (state.frame !== null) window.cancelAnimationFrame(state.frame)
  if (state.hideTimer !== null) window.clearTimeout(state.hideTimer)
  state.resizeObserver.disconnect()
  state.cleanup.forEach((cleanup) => cleanup())
  state.vertical.remove()
  state.horizontal.remove()
  delete state.host.dataset.appTableOverlayHost
  delete state.host.dataset.appTableOverlayActive
  delete state.middle.dataset.appTableOverlay
  states.delete(state)
}

export default defineBoot(() => {
  if (typeof document === 'undefined') return
  const states = new Set<OverlayState>()
  const scan = (root: ParentNode) => {
    if (root instanceof HTMLElement && root.matches(MIDDLE_SELECTOR)) initialize(root, states)
    root.querySelectorAll<HTMLElement>(MIDDLE_SELECTOR).forEach((middle) => initialize(middle, states))
  }

  scan(document)
  const observer = new MutationObserver((mutations) => {
    mutations.forEach((mutation) => {
      mutation.addedNodes.forEach((node) => {
        if (node instanceof HTMLElement) scan(node)
      })
    })
    states.forEach((state) => {
      if (!state.host.isConnected || !state.middle.isConnected) destroy(state, states)
      else scheduleUpdate(state)
    })
  })
  observer.observe(document.body, { childList: true, subtree: true })
})
