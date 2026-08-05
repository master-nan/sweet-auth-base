import { defineBoot } from '#q-app/wrappers'

type ScrollAxis = 'vertical' | 'horizontal'

interface TableScrollbarState {
  host: HTMLElement
  middle: HTMLElement
  verticalRail: HTMLElement
  verticalThumb: HTMLElement
  horizontalRail: HTMLElement
  horizontalThumb: HTMLElement
  resizeObserver: ResizeObserver
  frameId: number | null
  hideTimer: number | null
  verticalTrackLength: number
  horizontalTrackLength: number
  cleanup: Array<() => void>
}

const TABLE_MIDDLE_SELECTOR = '.q-table__middle'
const MIN_THUMB_SIZE = 28
const RAIL_INSET = 2
const RAIL_SIZE = 10

function getThumbSize(viewportSize: number, contentSize: number, trackLength: number) {
  return Math.min(trackLength, Math.max(MIN_THUMB_SIZE, (viewportSize / contentSize) * trackLength))
}

function createScrollbar(axis: ScrollAxis) {
  const rail = document.createElement('div')
  const thumb = document.createElement('div')
  rail.className = `app-table-scrollbar app-table-scrollbar--${axis}`
  rail.setAttribute('aria-hidden', 'true')
  thumb.className = 'app-table-scrollbar__thumb'
  rail.appendChild(thumb)
  return { rail, thumb }
}

function listen(
  state: TableScrollbarState,
  target: EventTarget,
  event: string,
  handler: EventListener,
) {
  target.addEventListener(event, handler)
  state.cleanup.push(() => target.removeEventListener(event, handler))
}

function setActive(state: TableScrollbarState, active: boolean) {
  if (state.hideTimer !== null) {
    window.clearTimeout(state.hideTimer)
    state.hideTimer = null
  }
  state.host.classList.toggle('app-table-scrollbar-active', active)
}

function scheduleHide(state: TableScrollbarState) {
  if (state.hideTimer !== null) window.clearTimeout(state.hideTimer)
  state.hideTimer = window.setTimeout(() => {
    const remainsActive =
      state.middle.matches(':hover, :focus-within') ||
      state.verticalRail.matches(':hover') ||
      state.horizontalRail.matches(':hover')
    if (remainsActive) {
      state.hideTimer = null
      return
    }
    state.host.classList.remove('app-table-scrollbar-active')
    state.hideTimer = null
  }, 180)
}

function updateAxis(
  rail: HTMLElement,
  thumb: HTMLElement,
  viewportSize: number,
  contentSize: number,
  scrollOffset: number,
  trackLength: number,
) {
  const maxScroll = Math.max(0, contentSize - viewportSize)
  const scrollable = maxScroll > 1 && trackLength > 0
  rail.classList.toggle('app-table-scrollbar--scrollable', scrollable)
  if (!scrollable) return

  const thumbSize = getThumbSize(viewportSize, contentSize, trackLength)
  const thumbTravel = Math.max(0, trackLength - thumbSize)
  const thumbOffset = maxScroll === 0 ? 0 : (scrollOffset / maxScroll) * thumbTravel
  const vertical = rail.classList.contains('app-table-scrollbar--vertical')

  if (vertical) {
    thumb.style.height = `${thumbSize}px`
    thumb.style.transform = `translateY(${thumbOffset}px)`
  } else {
    thumb.style.width = `${thumbSize}px`
    thumb.style.transform = `translateX(${thumbOffset}px)`
  }
}

function updateScrollbar(state: TableScrollbarState) {
  state.frameId = null
  if (!state.middle.isConnected || !state.host.isConnected) return

  const hostRect = state.host.getBoundingClientRect()
  const middleRect = state.middle.getBoundingClientRect()
  const hasVertical = state.middle.scrollHeight - state.middle.clientHeight > 1
  const hasHorizontal = state.middle.scrollWidth - state.middle.clientWidth > 1
  const top = Math.max(0, middleRect.top - hostRect.top + RAIL_INSET)
  const left = Math.max(0, middleRect.left - hostRect.left + RAIL_INSET)
  const verticalLength = Math.max(
    0,
    middleRect.height - RAIL_INSET * 2 - (hasHorizontal ? RAIL_SIZE : 0),
  )
  const horizontalLength = Math.max(
    0,
    middleRect.width - RAIL_INSET * 2 - (hasVertical ? RAIL_SIZE : 0),
  )

  state.verticalTrackLength = verticalLength
  state.horizontalTrackLength = horizontalLength
  state.verticalRail.style.top = `${top}px`
  state.verticalRail.style.right = `${Math.max(0, hostRect.right - middleRect.right)}px`
  state.verticalRail.style.height = `${verticalLength}px`
  state.horizontalRail.style.left = `${left}px`
  state.horizontalRail.style.top = `${Math.max(
    0,
    middleRect.bottom - hostRect.top - RAIL_SIZE,
  )}px`
  state.horizontalRail.style.width = `${horizontalLength}px`

  updateAxis(
    state.verticalRail,
    state.verticalThumb,
    state.middle.clientHeight,
    state.middle.scrollHeight,
    state.middle.scrollTop,
    verticalLength,
  )
  updateAxis(
    state.horizontalRail,
    state.horizontalThumb,
    state.middle.clientWidth,
    state.middle.scrollWidth,
    state.middle.scrollLeft,
    horizontalLength,
  )
}

function scheduleUpdate(state: TableScrollbarState) {
  if (state.frameId !== null) return
  state.frameId = window.requestAnimationFrame(() => updateScrollbar(state))
}

function bindThumbDrag(state: TableScrollbarState, axis: ScrollAxis, thumb: HTMLElement) {
  listen(state, thumb, 'pointerdown', ((event: PointerEvent) => {
    event.preventDefault()
    event.stopPropagation()
    setActive(state, true)
    thumb.setPointerCapture(event.pointerId)

    const startPosition = axis === 'vertical' ? event.clientY : event.clientX
    const startScroll = axis === 'vertical' ? state.middle.scrollTop : state.middle.scrollLeft
    const viewportSize =
      axis === 'vertical' ? state.middle.clientHeight : state.middle.clientWidth
    const contentSize = axis === 'vertical' ? state.middle.scrollHeight : state.middle.scrollWidth
    const trackLength =
      axis === 'vertical' ? state.verticalTrackLength : state.horizontalTrackLength
    const thumbSize = getThumbSize(viewportSize, contentSize, trackLength)
    const thumbTravel = Math.max(1, trackLength - thumbSize)
    const maxScroll = Math.max(0, contentSize - viewportSize)

    const move = (moveEvent: PointerEvent) => {
      const position = axis === 'vertical' ? moveEvent.clientY : moveEvent.clientX
      const nextScroll = startScroll + ((position - startPosition) / thumbTravel) * maxScroll
      if (axis === 'vertical') state.middle.scrollTop = nextScroll
      else state.middle.scrollLeft = nextScroll
    }
    const stop = (stopEvent: PointerEvent) => {
      if (thumb.hasPointerCapture(stopEvent.pointerId)) {
        thumb.releasePointerCapture(stopEvent.pointerId)
      }
      thumb.removeEventListener('pointermove', move)
      thumb.removeEventListener('pointerup', stop)
      thumb.removeEventListener('pointercancel', stop)
      scheduleHide(state)
    }

    thumb.addEventListener('pointermove', move)
    thumb.addEventListener('pointerup', stop)
    thumb.addEventListener('pointercancel', stop)
  }) as EventListener)
}

function initializeTableScrollbar(middle: HTMLElement, states: Set<TableScrollbarState>) {
  if (middle.dataset.appTableScrollbar === 'ready') return
  const host = middle.closest<HTMLElement>('.q-table__container')
  if (!host) return

  const vertical = createScrollbar('vertical')
  const horizontal = createScrollbar('horizontal')
  host.classList.add('app-table-scrollbar-host')
  middle.dataset.appTableScrollbar = 'ready'
  host.append(vertical.rail, horizontal.rail)

  const state: TableScrollbarState = {
    host,
    middle,
    verticalRail: vertical.rail,
    verticalThumb: vertical.thumb,
    horizontalRail: horizontal.rail,
    horizontalThumb: horizontal.thumb,
    resizeObserver: undefined as unknown as ResizeObserver,
    frameId: null,
    hideTimer: null,
    verticalTrackLength: 0,
    horizontalTrackLength: 0,
    cleanup: [],
  }

  state.resizeObserver = new ResizeObserver(() => scheduleUpdate(state))
  state.resizeObserver.observe(host)
  state.resizeObserver.observe(middle)
  if (middle.firstElementChild instanceof HTMLElement) {
    state.resizeObserver.observe(middle.firstElementChild)
  }

  listen(state, middle, 'scroll', () => {
    setActive(state, true)
    scheduleUpdate(state)
    scheduleHide(state)
  })
  listen(state, middle, 'pointerenter', () => setActive(state, true))
  listen(state, middle, 'pointerleave', () => scheduleHide(state))
  listen(state, middle, 'focusin', () => setActive(state, true))
  listen(state, middle, 'focusout', () => scheduleHide(state))
  for (const rail of [vertical.rail, horizontal.rail]) {
    listen(state, rail, 'pointerenter', () => setActive(state, true))
    listen(state, rail, 'pointerleave', () => scheduleHide(state))
  }
  bindThumbDrag(state, 'vertical', vertical.thumb)
  bindThumbDrag(state, 'horizontal', horizontal.thumb)

  states.add(state)
  scheduleUpdate(state)
}

function destroyTableScrollbar(state: TableScrollbarState, states: Set<TableScrollbarState>) {
  if (state.frameId !== null) window.cancelAnimationFrame(state.frameId)
  if (state.hideTimer !== null) window.clearTimeout(state.hideTimer)
  state.resizeObserver.disconnect()
  state.cleanup.forEach((cleanup) => cleanup())
  state.verticalRail.remove()
  state.horizontalRail.remove()
  state.host.classList.remove('app-table-scrollbar-host', 'app-table-scrollbar-active')
  delete state.middle.dataset.appTableScrollbar
  states.delete(state)
}

export default defineBoot(() => {
  if (typeof document === 'undefined') return

  const states = new Set<TableScrollbarState>()
  const scan = (root: ParentNode) => {
    if (root instanceof HTMLElement && root.matches(TABLE_MIDDLE_SELECTOR)) {
      initializeTableScrollbar(root, states)
    }
    root
      .querySelectorAll<HTMLElement>(TABLE_MIDDLE_SELECTOR)
      .forEach((middle) => initializeTableScrollbar(middle, states))
  }

  scan(document)
  const observer = new MutationObserver((mutations) => {
    for (const mutation of mutations) {
      mutation.addedNodes.forEach((node) => {
        if (node instanceof HTMLElement) scan(node)
      })
    }
    states.forEach((state) => {
      if (!state.middle.isConnected || !state.host.isConnected) {
        destroyTableScrollbar(state, states)
      } else {
        scheduleUpdate(state)
      }
    })
  })
  observer.observe(document.body, { childList: true, subtree: true })
})
