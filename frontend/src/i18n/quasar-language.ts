import { Lang } from 'quasar'
import enUS from 'quasar/lang/en-US'
import zhCN from 'quasar/lang/zh-CN'
import type { SupportedLocale } from 'src/utils/ui-preferences'

export const applyQuasarLanguage = (locale: SupportedLocale) => {
  Lang.set(locale === 'en-US' ? enUS : zhCN)
  document.documentElement.lang = locale
}
