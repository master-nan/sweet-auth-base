import { instance } from '@/boot/axios'
import { localLoadingRequestConfig } from '@/api/request-config'
import type { ResponseData } from '@/types/global'

export type VerificationSampleScenario =
  | 'data-permission'
  | 'tms-company-scope'
  | 'metadata-low-code'
  | 'organization-sync'
  | 'integration-call'
  | 'file-upload'
  | 'video-preview'
  | 'notification'

export type VerificationSampleState = 'empty' | 'partial' | 'ready' | 'unavailable'

export interface VerificationSampleDetail {
  label: string
  value: string
}

export interface VerificationSampleStatus {
  scenario_id: VerificationSampleScenario
  state: VerificationSampleState
  available: boolean
  item_count: number
  summary: string
  details: VerificationSampleDetail[]
}

export interface VerificationSampleAccount {
  user_name: string
  password: string
  role: string
  expected: string
}

export interface VerificationSamplePrepareResult {
  status: VerificationSampleStatus
  accounts: VerificationSampleAccount[]
}

export const useDevelopmentVerificationApi = () => ({
  statuses: () =>
    instance
      .get<
        ResponseData<VerificationSampleStatus[]>
      >('/admin/development/verification/samples', localLoadingRequestConfig)
      .then((response) => response.data),
  prepare: (scenario: VerificationSampleScenario) =>
    instance
      .post<
        ResponseData<VerificationSamplePrepareResult>
      >(`/admin/development/verification/samples/${scenario}/prepare`, undefined, localLoadingRequestConfig)
      .then((response) => response.data),
  cleanup: (scenario: VerificationSampleScenario) =>
    instance
      .delete<
        ResponseData<VerificationSampleStatus>
      >(`/admin/development/verification/samples/${scenario}`, localLoadingRequestConfig)
      .then((response) => response.data),
})
