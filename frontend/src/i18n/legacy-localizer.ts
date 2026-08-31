import { onBeforeUnmount, onMounted, watch, type Ref } from 'vue'
import { legacyEnglishMessages } from './legacy-en'

type LocaleValue = 'en-US' | 'zh-CN' | string

interface RenderState {
  source: string
  rendered: string
}

const localizedAttributes = ['aria-label', 'placeholder', 'title'] as const
const ignoredElements = new Set(['CODE', 'PRE', 'SCRIPT', 'STYLE'])
const textStates = new WeakMap<Text, RenderState>()
const attributeStates = new WeakMap<Element, Map<string, RenderState>>()

const translateExactText = (value: string) => legacyEnglishMessages[value] || value
const legacyEnglishPatterns: Array<[RegExp, string]> = [
  [/^共\s*(.+?)\s*条通知$/, '$1 notifications'],
  [/^共\s*(.+?)\s*条$/, '$1 items'],
  [/^共\s*(.+?)\s*行$/, '$1 rows'],
  [/^当前版本：V(.+)$/, 'Current version: V$1'],
  [/^(.+?)\s*个数据集$/, '$1 datasets'],
  [/^(.+?)\s*个查询参数$/, '$1 query parameters'],
  [/^(.+?)\s*个字段$/, '$1 fields'],
  [/^编码：(.+)$/, 'Code: $1'],
  [/^权限表：(.+)$/, 'Permission table: $1'],
  [/^数据来源：(.+)$/, 'Data source: $1'],
  [/^(.+?)\s*菜单$/, '$1 menus'],
  [/^(.+?)\s*按钮$/, '$1 buttons'],
  [/^(.+?)\s*接口权限$/, '$1 API permissions'],
  [/^(.+?)\s*已选$/, '$1 selected'],
  [/^临时密码：(.+)$/, 'Temporary password: $1'],
  [/^当前表：(.+)$/, 'Current table: $1'],
  [/^当前单元格：(.+)$/, 'Current cell: $1'],
  [/^数据集：(.+)$/, 'Dataset: $1'],
  [/^已绑定字段：(.+)$/, 'Bound fields: $1'],
  [/^预览版本\s*V(.+)$/, 'Preview version V$1'],
  [/^主数据集\s*(.+)$/, 'Primary dataset $1'],
  [/^线上版本\s*V(.+)$/, 'Published version V$1'],
  [/^(.+?)\s*毫秒$/, '$1 ms'],
  [/^活动\s*(.+)$/, 'Active $1'],
  [/^已完成\s*(.+)$/, 'Completed $1'],
  [/^最近轮询\s*(.+)$/, 'Last poll $1'],
  [/^表达式组\s*(.+)$/, 'Expression group $1'],
  [/^嵌套组\s*(.+)$/, 'Nested group $1'],
]

const translatePatternText = (value: string) => {
  for (const [pattern, replacement] of legacyEnglishPatterns) {
    if (pattern.test(value)) return value.replace(pattern, replacement)
  }
  return value
}

export const translateLegacyText = (value: string, locale: LocaleValue) => {
  if (locale !== 'en-US' || !value.trim()) return value

  const leading = value.match(/^\s*/)?.[0] || ''
  const trailing = value.match(/\s*$/)?.[0] || ''
  const source = value.slice(leading.length, value.length - trailing.length || undefined)
  const exactTranslation = translateExactText(source)
  const translated = exactTranslation === source ? translatePatternText(source) : exactTranslation
  return translated === source ? value : `${leading}${translated}${trailing}`
}

const nextState = (current: string, previous?: RenderState): RenderState => {
  // Keep the original Vue-rendered copy so changing back to Chinese is lossless.
  if (!previous || (current !== previous.source && current !== previous.rendered)) {
    return { source: current, rendered: current }
  }
  return previous
}

const isLocalizationIgnored = (element: Element | null) =>
  Boolean(element?.closest('[data-i18n-ignore]'))

const localizeTextNode = (node: Text, locale: LocaleValue) => {
  if (
    (node.parentElement && ignoredElements.has(node.parentElement.tagName)) ||
    isLocalizationIgnored(node.parentElement)
  )
    return

  const state = nextState(node.data, textStates.get(node))
  const rendered = locale === 'en-US' ? translateLegacyText(state.source, locale) : state.source
  state.rendered = rendered
  textStates.set(node, state)
  if (node.data !== rendered) node.data = rendered
}

const localizeAttribute = (element: Element, name: string, locale: LocaleValue) => {
  if (isLocalizationIgnored(element)) return
  const current = element.getAttribute(name)
  if (current === null) return

  const states = attributeStates.get(element) || new Map<string, RenderState>()
  const state = nextState(current, states.get(name))
  const rendered = locale === 'en-US' ? translateLegacyText(state.source, locale) : state.source
  state.rendered = rendered
  states.set(name, state)
  attributeStates.set(element, states)
  if (current !== rendered) element.setAttribute(name, rendered)
}

const localizeElement = (element: Element, locale: LocaleValue) => {
  if (ignoredElements.has(element.tagName)) return
  localizedAttributes.forEach((name) => localizeAttribute(element, name, locale))
}

const localizeTree = (root: Node, locale: LocaleValue) => {
  if (root.nodeType === Node.TEXT_NODE) {
    localizeTextNode(root as Text, locale)
    return
  }
  if (root.nodeType !== Node.ELEMENT_NODE && root.nodeType !== Node.DOCUMENT_NODE) return

  if (root.nodeType === Node.ELEMENT_NODE) localizeElement(root as Element, locale)
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_ELEMENT | NodeFilter.SHOW_TEXT)
  let node = walker.nextNode()
  while (node) {
    if (node.nodeType === Node.TEXT_NODE) localizeTextNode(node as Text, locale)
    else localizeElement(node as Element, locale)
    node = walker.nextNode()
  }
}

export const useLegacyUiLocalizer = (locale: Ref<string>) => {
  let observer: MutationObserver | undefined

  const localizeDocument = () => {
    if (document.body) localizeTree(document.body, locale.value)
  }

  const stopLocaleWatch = watch(locale, localizeDocument, { flush: 'post' })

  onMounted(() => {
    localizeDocument()
    // Quasar teleports menus, dialogs, and notifications under body after page render.
    observer = new MutationObserver((mutations) => {
      mutations.forEach((mutation) => {
        if (mutation.type === 'characterData') {
          localizeTextNode(mutation.target as Text, locale.value)
          return
        }
        if (mutation.type === 'attributes') {
          localizeAttribute(mutation.target as Element, mutation.attributeName || '', locale.value)
          return
        }
        mutation.addedNodes.forEach((node) => localizeTree(node, locale.value))
      })
    })
    observer.observe(document.body, {
      attributes: true,
      attributeFilter: [...localizedAttributes],
      characterData: true,
      childList: true,
      subtree: true,
    })
  })

  onBeforeUnmount(() => {
    observer?.disconnect()
    stopLocaleWatch()
  })
}
