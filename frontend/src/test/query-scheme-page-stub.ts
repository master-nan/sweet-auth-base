import { ref } from 'vue'
import { vi, type Mock } from 'vitest'

export const createQuerySchemePageStub = (initialize: Mock = vi.fn()) => ({
  runtime: {
    schemes: ref([]),
    currentLabel: ref('查询方案'),
    loading: ref(false),
    error: ref(''),
    scope: { config: ref(null) },
    loadAvailable: vi.fn(),
  },
  showSaveDialog: ref(false),
  saving: ref(false),
  initialize,
  runQueryChange: vi.fn((change: () => void) => change()),
  selectScheme: vi.fn(),
  applyPreset: vi.fn(),
  restoreCurrent: vi.fn(),
  resetDefault: vi.fn(),
  openManager: vi.fn(),
  savePersonal: vi.fn(),
})
