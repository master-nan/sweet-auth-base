import { defineBoot } from '#q-app/wrappers'
import { createI18n } from 'vue-i18n'

import messages from 'src/i18n'
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
  const i18n = createI18n<{ message: MessageSchema }, MessageLanguages>({
    locale: preferences.locale,
    legacy: false,
    messages,
  })

  app.use(i18n)
})
