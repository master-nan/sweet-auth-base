import { defineComponent, h } from 'vue'
import { shallowMount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('src/stores/user', () => ({
  useUserStore: () => ({ getUserName: 'admin', setLogout: vi.fn() }),
}))
vi.mock('src/stores/app', () => ({ useAppStore: () => ({ reloadPage: vi.fn() }) }))
vi.mock('src/components/Toolbar/DarkMode.vue', () => ({ default: { template: '<div />' } }))
vi.mock('src/components/Toolbar/LangSelector.vue', () => ({ default: { template: '<div />' } }))
vi.mock('src/components/Notification/NotificationPopover.vue', () => ({
  default: { name: 'NotificationPopover', template: '<div data-notification-popover />' },
}))
vi.mock('quasar', () => ({
  useQuasar: () => ({
    fullscreen: { isActive: false, toggle: vi.fn() },
    screen: { gt: { sm: true } },
  }),
}))

import ToolbarItem from './ToolbarItem.vue'

const QBtnStub = defineComponent({
  name: 'QBtn',
  props: { icon: String },
  emits: ['click'],
  setup(props, { emit, slots }) {
    return () =>
      h('button', { 'data-icon': props.icon, onClick: () => emit('click') }, slots.default?.())
  },
})

describe('ToolbarItem settings entry', () => {
  it('emits the header settings action with an accessible button', async () => {
    const wrapper = shallowMount(ToolbarItem, {
      global: { stubs: { QBtn: QBtnStub } },
    })
    const button = wrapper.find('[data-icon="settings"]')
    expect(button.exists()).toBe(true)
    await button.trigger('click')
    expect(wrapper.emitted('open-settings')).toHaveLength(1)
  })

  it('hosts the notification entry without owning notification state', () => {
    const wrapper = shallowMount(ToolbarItem, {
      global: { stubs: { QBtn: QBtnStub } },
    })
    expect(wrapper.findComponent({ name: 'NotificationPopover' }).exists()).toBe(true)
  })
})
