import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

vi.mock('quasar', () => ({}))

import CredentialFormDialog from './CredentialFormDialog.vue'

describe('credential form secret lifecycle', () => {
  it('clears secret state immediately after dialog closes', async () => {
    const wrapper = mount(CredentialFormDialog, {
      props: { modelValue: true, editData: null, systems: [], loading: false },
      global: {
        stubs: {
          FormDialogShell: { template: '<div><slot /></div>' },
          QForm: { template: '<form><slot /></form>' },
          QSelect: true,
          QInput: true,
          QBanner: true,
          QIcon: true,
        },
      },
    })
    const vm = wrapper.vm as unknown as { form: { secret: Record<string, string> } }
    vm.form.secret = { token: 'must-be-cleared' }
    await wrapper.setProps({ modelValue: false })
    expect(vm.form.secret).toEqual({})
  })
})
