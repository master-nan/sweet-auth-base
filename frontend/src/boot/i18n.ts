import { defineBoot } from '#q-app/wrappers'

import type messages from 'src/i18n'
import { i18n } from 'src/i18n/runtime/instance'
import { applyQuasarLanguage } from 'src/i18n/runtime/quasar'
import { readUIPreferences } from 'src/utils/ui-preferences'

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

export default defineBoot(({ app }) => {
  const preferences = readUIPreferences()
  i18n.global.locale.value = preferences.locale
  applyQuasarLanguage(preferences.locale)

  app.use(i18n)
})
