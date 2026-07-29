import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import FormDialogShell from './FormDialogShell.vue'

const SlotStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', slots.default?.())
  },
})

describe('FormDialogShell', () => {
  it('keeps navigation, scrollable body and footer actions in the common shell', () => {
    const wrapper = mount(FormDialogShell, {
      props: {
        modelValue: true,
        embedded: true,
        readonly: true,
        title: '人员详情',
      },
      slots: {
        navigation: '<nav data-testid="navigation">章节</nav>',
        default: '<div data-testid="body">详情内容</div>',
        'footer-actions': '<button data-testid="footer-action">业务操作</button>',
      },
      global: {
        stubs: {
          QCardSection: SlotStub,
          QSpace: true,
          QIcon: true,
          QSpinnerDots: true,
          QBtn: SlotStub,
        },
      },
    })

    expect(wrapper.find('.form-dialog-shell__header').exists()).toBe(true)
    expect(wrapper.find('.form-dialog-shell__navigation').text()).toContain('章节')
    expect(wrapper.find('.form-dialog-shell__main').text()).toContain('详情内容')
    expect(wrapper.find('.form-dialog-shell__footer').text()).toContain('业务操作')
  })
})
