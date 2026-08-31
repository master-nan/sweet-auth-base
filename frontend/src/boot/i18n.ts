import { defineBoot } from '#q-app/wrappers'
import { createI18n } from 'vue-i18n'
import { Lang } from 'quasar'
import enUS from 'quasar/lang/en-US'
import zhCN from 'quasar/lang/zh-CN'

import messages from 'src/i18n'
import { readUIPreferences, type SupportedLocale } from 'src/utils/ui-preferences'

export type MessageLanguages = keyof typeof messages
// 以当前中文资源定义消息Schema，其他语言必须保持相同键结构。
export type MessageSchema = (typeof messages)['zh-CN']

/* eslint-disable @typescript-eslint/no-empty-object-type */
declare module 'vue-i18n' {
  export interface DefineLocaleMessage extends MessageSchema {}
  export interface DefineDateTimeFormat {}
  export interface DefineNumberFormat {}
}
/* eslint-enable @typescript-eslint/no-empty-object-type */

export const i18n = createI18n({
  locale: 'zh-CN',
  legacy: false,
  messages,
})

export const translate = i18n.global.t as (
  key: string,
  named?: Record<string, unknown>,
) => string

export const applyQuasarLanguage = (locale: SupportedLocale) => {
  Lang.set(locale === 'en-US' ? enUS : zhCN)
  document.documentElement.lang = locale
}

export default defineBoot(({ app }) => {
  const preferences = readUIPreferences()
  i18n.global.locale.value = preferences.locale
  applyQuasarLanguage(preferences.locale)

  app.use(i18n)
})
