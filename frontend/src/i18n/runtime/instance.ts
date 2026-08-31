import { createI18n } from 'vue-i18n'
import messages from 'src/i18n'

export const i18n = createI18n({
  locale: 'zh-CN',
  legacy: false,
  messages,
})

export const translate = i18n.global.t as (key: string, named?: Record<string, unknown>) => string
