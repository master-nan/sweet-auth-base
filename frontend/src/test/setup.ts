import { config } from '@vue/test-utils'
import { beforeEach } from 'vitest'
import { createI18n } from 'vue-i18n'

import messages from 'src/i18n'

export const testI18n = createI18n({
  legacy: false,
  locale: 'zh-CN',
  messages,
})

config.global.plugins = [testI18n]

beforeEach(() => {
  testI18n.global.locale.value = 'zh-CN'
})
