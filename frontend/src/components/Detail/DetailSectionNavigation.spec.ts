import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import DetailSectionNavigation from './DetailSectionNavigation.vue'

const IconStub = defineComponent({
  props: {
    name: String,
  },
  setup(props) {
    return () => h('i', props.name)
  },
})

describe('DetailSectionNavigation', () => {
  it('renders section copy and emits the selected stable key', async () => {
    const wrapper = mount(DetailSectionNavigation, {
      props: {
        modelValue: 'basic',
        items: [
          { key: 'basic', label: '基本资料', caption: '人员身份与联系方式' },
          { key: 'assignments', label: '任职记录', caption: '当前和历史任职', count: 2 },
        ],
      },
      global: {
        stubs: {
          QIcon: IconStub,
        },
      },
    })

    expect(wrapper.text()).toContain('人员身份与联系方式')
    expect(wrapper.text()).toContain('2')
    expect(wrapper.find('.is-active').text()).toContain('基本资料')

    await wrapper.findAll('button')[1]?.trigger('click')
    expect(wrapper.emitted('update:modelValue')).toEqual([['assignments']])
  })
})
