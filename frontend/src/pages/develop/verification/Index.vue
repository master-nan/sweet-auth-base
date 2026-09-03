<template>
  <base-content class="q-pa-sm verification-page">
    <div class="verification-workbench">
      <aside class="verification-master">
        <div class="verification-toolbar">
          <q-input
            v-model="keyword"
            outlined
            dense
            clearable
            :placeholder="t('ui.searchForValidationScene')"
          >
            <template #append><q-icon name="search" /></template>
          </q-input>
          <q-select
            v-model="category"
            :options="categoryOptions"
            outlined
            dense
            emit-value
            map-options
            :label="t('ui.sceneClassification')"
          />
        </div>
        <q-scroll-area class="verification-master-scroll">
          <q-list separator>
            <q-item
              v-for="scenario in filteredScenarios"
              :key="scenario.id"
              clickable
              :active="scenario.id === selectedId"
              active-class="verification-item--active"
              @click="selectedId = scenario.id"
            >
              <q-item-section avatar>
                <q-icon :name="scenario.icon" color="primary" />
              </q-item-section>
              <q-item-section>
                <q-item-label>{{ scenario.title }}</q-item-label>
                <q-item-label caption lines="2">{{ scenario.summary }}</q-item-label>
              </q-item-section>
              <q-item-section side>
                <status-chip
                  :label="scenarioChip(scenario).label"
                  :color="scenarioChip(scenario).color"
                  :outline="scenarioChip(scenario).outline"
                />
              </q-item-section>
            </q-item>
          </q-list>
          <div v-if="!filteredScenarios.length" class="verification-empty">
            <q-icon name="search_off" size="40px" />
            <span>{{ t('ui.noMatchingAuthenticationScene') }}</span>
          </div>
        </q-scroll-area>
      </aside>

      <main class="verification-detail">
        <q-banner dense class="verification-guide-note">
          {{ t('ui.theSceneWithThePrepareTheSampleButtonCreatesIndependent') }}
        </q-banner>
        <q-scroll-area v-if="selectedScenario" class="verification-detail-scroll">
          <div class="verification-detail-content">
            <header class="verification-detail-header">
              <div class="verification-detail-title-wrap">
                <q-icon :name="selectedScenario.icon" class="verification-detail-icon" />
                <div>
                  <div class="verification-detail-title">{{ selectedScenario.title }}</div>
                  <div class="verification-detail-summary">{{ selectedScenario.summary }}</div>
                </div>
              </div>
              <q-space />
              <div class="verification-actions">
                <q-btn
                  v-if="selectedScenario.sampleId"
                  flat
                  round
                  color="primary"
                  icon="refresh"
                  :loading="loadingStatuses"
                  @click="loadSampleStatuses"
                >
                  <q-tooltip>{{ t('ui.refreshSampleStatus') }}</q-tooltip>
                </q-btn>
                <q-btn
                  v-if="
                    selectedScenario.sampleId && sampleStatus(selectedScenario)?.state !== 'empty'
                  "
                  outline
                  color="negative"
                  icon="delete_sweep"
                  :label="t('ui.clearSample')"
                  :loading="cleaningId === selectedScenario.sampleId"
                  :disable="sampleStatus(selectedScenario)?.available === false"
                  @click="confirmCleanup(selectedScenario)"
                />
                <q-btn
                  v-if="selectedScenario.sampleId"
                  unelevated
                  color="primary"
                  icon="playlist_add_check"
                  :label="
                    sampleStatus(selectedScenario)?.state === 'ready'
                      ? t('ui.reprepared')
                      : t('ui.prepareASample')
                  "
                  :loading="preparingId === selectedScenario.sampleId"
                  :disable="sampleStatus(selectedScenario)?.available === false"
                  @click="prepareSample(selectedScenario)"
                />
                <q-btn
                  v-if="
                    selectedScenario.id === 'integration-call' &&
                    sampleStatus(selectedScenario)?.state === 'ready'
                  "
                  outline
                  color="primary"
                  icon="play_arrow"
                  :label="t('ui.runConnectivityTests')"
                  :loading="runningIntegrationSample"
                  @click="runIntegrationSample"
                />
                <q-btn
                  v-if="selectedScenario.routeName"
                  unelevated
                  color="primary"
                  icon-right="open_in_new"
                  :label="selectedScenario.actionLabel || t('ui.openRelevantPages')"
                  @click="openScenario(selectedScenario)"
                />
              </div>
            </header>

            <section v-if="selectedScenario.sampleId" class="verification-sample-status">
              <div class="verification-sample-status__main">
                <status-chip
                  :label="scenarioChip(selectedScenario).label"
                  :color="scenarioChip(selectedScenario).color"
                  :outline="scenarioChip(selectedScenario).outline"
                />
                <span>{{
                  sampleStatus(selectedScenario)?.summary || t('ui.readingSampleExampleState')
                }}</span>
              </div>
              <div
                v-if="sampleStatus(selectedScenario)?.details?.length"
                class="verification-sample-facts"
              >
                <div v-for="detail in sampleStatus(selectedScenario)?.details" :key="detail.label">
                  <span>{{ detail.label }}</span>
                  <strong>{{ detail.value }}</strong>
                </div>
              </div>
            </section>

            <section v-if="selectedScenario.sampleFiles?.length" class="verification-sample-files">
              <div>
                <strong>{{ t('ui.testFile') }}</strong>
                <span>{{ t('ui.downloadsAreUploadedDirectlyByTheNextStepAndDo') }}</span>
              </div>
              <div class="verification-sample-files__actions">
                <q-btn
                  v-for="file in selectedScenario.sampleFiles"
                  :key="file.fileName"
                  outline
                  color="primary"
                  icon="download"
                  :label="file.label"
                  @click="downloadSampleFile(file)"
                />
              </div>
            </section>

            <section class="verification-section">
              <h3>{{ t('ui.prepareBeforeYouBegin') }}</h3>
              <q-list dense>
                <q-item v-for="item in selectedScenario.prerequisites" :key="item">
                  <q-item-section avatar
                    ><q-icon name="check_circle_outline" color="primary"
                  /></q-item-section>
                  <q-item-section>{{ item }}</q-item-section>
                </q-item>
              </q-list>
            </section>

            <section class="verification-section">
              <h3>{{ t('ui.operationalSteps') }}</h3>
              <ol class="verification-steps">
                <li v-for="step in selectedScenario.steps" :key="step">{{ step }}</li>
              </ol>
            </section>

            <section class="verification-section">
              <h3>{{ t('ui.expectedResults') }}</h3>
              <q-list dense>
                <q-item v-for="item in selectedScenario.expected" :key="item">
                  <q-item-section avatar
                    ><q-icon name="task_alt" color="positive"
                  /></q-item-section>
                  <q-item-section>{{ item }}</q-item-section>
                </q-item>
              </q-list>
            </section>

            <q-banner v-if="selectedScenario.note" rounded class="verification-note">
              <template #avatar><q-icon name="info_outline" color="primary" /></template>
              {{ selectedScenario.note }}
            </q-banner>
          </div>
        </q-scroll-area>
      </main>
    </div>

    <q-dialog v-model="accountDialog" persistent>
      <q-card class="verification-account-dialog">
        <q-card-section class="row items-start q-gutter-md">
          <q-icon name="password" color="primary" size="36px" />
          <div>
            <div class="text-h6">{{ t('ui.functionalValidationAccount') }}</div>
            <div class="text-caption text-grey-7">
              {{ t('ui.thePasswordIsDisplayedOnlyAfterThisPreparationHasBeen') }}
            </div>
          </div>
        </q-card-section>
        <q-separator />
        <q-card-section class="q-pa-none">
          <q-list separator>
            <q-item v-for="account in preparedAccounts" :key="account.user_name" class="q-py-md">
              <q-item-section>
                <q-item-label class="text-weight-bold">{{ account.user_name }}</q-item-label>
                <q-item-label caption>{{ account.role }}</q-item-label>
                <q-item-label caption>{{ account.expected }}</q-item-label>
              </q-item-section>
              <q-item-section side>
                <div class="verification-password">
                  <code>{{ account.password }}</code>
                  <q-btn
                    flat
                    round
                    dense
                    icon="content_copy"
                    color="primary"
                    @click="copyPassword(account)"
                  >
                    <q-tooltip>{{ t('ui.copyAccountNumberAndPassword') }}</q-tooltip>
                  </q-btn>
                </div>
              </q-item-section>
            </q-item>
          </q-list>
        </q-card-section>
        <q-separator />
        <q-card-actions align="right">
          <q-btn
            flat
            color="primary"
            :label="t('ui.iVeAlreadyWrittenItDown')"
            @click="accountDialog = false"
          />
        </q-card-actions>
      </q-card>
    </q-dialog>
  </base-content>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'develop_verification_page' })

import { computed, onMounted, ref } from 'vue'
import { copyToClipboard, useQuasar } from 'quasar'
import { useRouter } from 'vue-router'
import BaseContent from '@/components/BaseContent/BaseContent.vue'
import StatusChip from '@/components/Display/StatusChip.vue'
import {
  useDevelopmentVerificationApi,
  type VerificationSampleAccount,
  type VerificationSampleScenario,
  type VerificationSampleStatus,
} from '@/api/services/development-verification'
import { useIntegrationApi } from '@/api/services/integration'

const { t } = useI18n({ useScope: 'global' })

type ScenarioStatus = 'ready' | 'configuration' | 'data'

interface VerificationScenario {
  id: string
  title: string
  category: string
  icon: string
  status: ScenarioStatus
  summary: string
  prerequisites: string[]
  steps: string[]
  expected: string[]
  note?: string
  routeName?: string
  actionLabel?: string
  sampleId?: VerificationSampleScenario
  sampleFiles?: VerificationSampleFile[]
}

interface VerificationSampleFile {
  label: string
  fileName: string
  url?: string
  generatedSizeMiB?: number
}

const router = useRouter()
const $q = useQuasar()
const verificationApi = useDevelopmentVerificationApi()
const integrationApi = useIntegrationApi()
const keyword = ref('')
const category = ref('all')
const selectedId = ref('permission-page')
const loadingStatuses = ref(false)
const preparingId = ref<VerificationSampleScenario | null>(null)
const cleaningId = ref<VerificationSampleScenario | null>(null)
const statuses = ref<Partial<Record<VerificationSampleScenario, VerificationSampleStatus>>>({})
const accountDialog = ref(false)
const preparedAccounts = ref<VerificationSampleAccount[]>([])
const runningIntegrationSample = ref(false)

const categoryOptions = [
  {
    get label() {
      return t('ui.allScenes')
    },
    value: 'all',
  },
  {
    get label() {
      return t('ui.authorityAndSecurity')
    },
    value: 'permission',
  },
  {
    get label() {
      return t('ui.dataAndLowCode')
    },
    value: 'data',
  },
  {
    get label() {
      return t('ui.organizationAndIntegration')
    },
    value: 'integration',
  },
  {
    get label() {
      return t('ui.documentationAndContent')
    },
    value: 'file',
  },
  {
    get label() {
      return t('ui.queryAndMessage')
    },
    value: 'query',
  },
  {
    get label() {
      return t('ui.report')
    },
    value: 'report',
  },
]

const statusLabels: Record<ScenarioStatus, { label: string; color: string }> = {
  ready: {
    get label() {
      return t('ui.directlyVerifiable')
    },
    color: 'positive',
  },
  configuration: {
    get label() {
      return t('ui.requiresConfiguration')
    },
    color: 'warning',
  },
  data: {
    get label() {
      return t('ui.sampleDataRequired')
    },
    color: 'info',
  },
}

const sampleStateLabels = {
  empty: {
    get label() {
      return t('ui.notPrepared')
    },
    color: 'grey-7',
    outline: true,
  },
  partial: {
    get label() {
      return t('ui.needToBeCompleted')
    },
    color: 'warning',
    outline: true,
  },
  ready: {
    get label() {
      return t('ui.sampleIsReady')
    },
    color: 'positive',
    outline: false,
  },
  unavailable: {
    get label() {
      return t('ui.currentEnvironmentNotAvailable')
    },
    color: 'grey-7',
    outline: true,
  },
} as const

const scenarios: VerificationScenario[] = [
  {
    id: 'permission-page',
    get title() {
      return t('ui.menuPagesAndButtonPermissions')
    },
    category: 'permission',
    icon: 'admin_panel_settings',
    status: 'ready',
    get summary() {
      return t('ui.verifyesPageAndBusinessButtonBoundariesUsingAdministratorsReadOnly')
    },
    get prerequisites() {
      return [
        t('ui.prepareThreeTestAccountsAndAssignAdministratorReadOnlyAnd'),
        t('ui.itIsNotKnownThatTheAdministratorWillExecuteThe'),
      ]
    },
    get steps() {
      return [
        t('ui.administratorLoginToConfirmThatYouCanSeeTheUser'),
        t('ui.readOnlyLoginToConfirmThatThePageIsOpen'),
        t('ui.unauthorizedUserLoginToConfirmThatTheLeftMenuDoes'),
      ]
    },
    get expected() {
      return [
        t('ui.pageVisibilityCorrespondsToTheRoleMenu'),
        t('ui.buttonIsConsistentWithThePermissionOfTheMenuButton'),
        t('ui.directCallsToUnauthorizedApiWillBeRejected'),
      ]
    },
    routeName: 'system_role',
    get actionLabel() {
      return t('ui.openRoleManagement')
    },
  },
  {
    id: 'data-permission',
    sampleId: 'data-permission',
    get title() {
      return t('ui.dataPermissionRange')
    },
    category: 'permission',
    icon: 'rule',
    status: 'data',
    get summary() {
      return t('ui.configureDifferentDataRangesForTheSameBusinessPageComparing')
    },
    get prerequisites() {
      return [
        t('ui.clickPrepareASampleToRecordTwoTemporaryAccountsIn'),
        t('ui.reEntryOpensTheFunctionalAuthenticationOrderPageFromDevelopment'),
      ]
    },
    get steps() {
      return [
        t('ui.loginWithTheEastChinaAccountNumberOpenTheFunctional'),
        t('ui.afterExitLogInWithAllThePurchaseOrdersAccounts'),
        t('ui.opensTheOrderDetailsSeparatelyConfirmingThatTheListAnd'),
        t('ui.goBackToTheAdministratorAccountAndSeeHowThe'),
      ]
    },
    get expected() {
      return [
        t('ui.theAdministratorSeesTheCompleteData'),
        t('ui.limitedUsersOnlySeeDataAllowedByTheRules'),
        t('ui.detailsEditingAndRemovalOfInterfacesAreEquallyNotSubject'),
      ]
    },
    get note() {
      return t('ui.dataPermissionTestingDoesNotDependOnTheOrganizationS')
    },
    routeName: 'lowcode_verify_permission_order',
    get actionLabel() {
      return t('ui.openSampleOrder')
    },
  },
  {
    id: 'tms-company-scope',
    sampleId: 'tms-company-scope',
    get title() {
      return t('ui.tmsVehicleCompany')
    },
    category: 'permission',
    icon: 'local_shipping',
    status: 'data',
    get summary() {
      return t('ui.theListOfAuthenticationVehiclesIsByDefaultSubjectTo')
    },
    get prerequisites() {
      return [
        t('ui.ensureThatTmsCompanyAndTmsVehicleAreInitializedAnd'),
        t('ui.clickPrepareASampleToRecordTwoTemporaryAccounts'),
      ]
    },
    get steps() {
      return [
        t('ui.theTmsVehiclePageWasEnteredUsingTheAccountNumber'),
        t('ui.theSamePageWasEnteredUsingAMultiCompanyAccount'),
        t('ui.theCompanySelectedWahesyInItsSearchTermsAndConfirmed'),
        t('ui.goBackToTheAdministratorAccountToSeeTheVerify'),
      ]
    },
    get expected() {
      return [
        t('ui.singleCompanyUsersDefaultOnlyToReturnToCompanyA'),
        t('ui.multiCorporateUsersCanReturnToTheAAndB'),
        t('ui.companyCWillNotReturnEvenIfItIsBrought'),
      ]
    },
    get note() {
      return t('ui.theQueryConditionShowsTheCurrentAuthorizedCompanyButThe')
    },
    routeName: 'lowcode_tms_vehicle',
    get actionLabel() {
      return t('ui.openTmsVehicle')
    },
  },
  {
    id: 'metadata-low-code',
    sampleId: 'metadata-low-code',
    get title() {
      return t('ui.metadataAndLowCodePages')
    },
    category: 'data',
    icon: 'dynamic_form',
    status: 'data',
    get summary() {
      return t('ui.verifyHowFieldConfigurationAffectsTheListQueryAddEdit')
    },
    get prerequisites() {
      return [
        t('ui.clickPrepareASampleTheSystemCreatesTwoVerivyTables'),
        t('ui.theSamplePageNeedsToBeReLoggedInOnce'),
      ]
    },
    get steps() {
      return [
        t('ui.opensTheFunctionalValidationLowCodeRecordToObserveThe'),
        t('ui.addANewRecordToCheckTheAvailabilityOfDate'),
        t('ui.editsAndOpensDetailsToConfirmThatTheDictionaryAnd'),
        t('ui.viewTheVerifyLowcodeRecordFieldConfigurationInTheData'),
      ]
    },
    get expected() {
      return [
        t('ui.onlyFieldsAllowedByMetadataAreShownOnThePage'),
        t('ui.dictionaryAndRelationDisplayBusinessNamesInsteadOfCodeOr'),
        t('ui.fieldVerificationIsConsistentWithTheBackEndTypeOf'),
      ]
    },
    routeName: 'lowcode_verify_lowcode_record',
    get actionLabel() {
      return t('ui.openLowCodeSample')
    },
  },
  {
    id: 'organization-sync',
    sampleId: 'organization-sync',
    get title() {
      return t('ui.organizationPeopleAndJobsSynchronized')
    },
    category: 'integration',
    icon: 'account_tree',
    status: 'configuration',
    get summary() {
      return t('ui.validationOfCorporateStructuresManagementStructuresPostsAndPersonnelThrough')
    },
    get prerequisites() {
      return [
        t('ui.clickingOnThePrepareSampleTheSystemCreatesIndependentVerify'),
        t('ui.defaultDockerEnvironmentWillEnableIntegrationWorkerSyncRunnerAnd'),
      ]
    },
    get steps() {
      return [
        t('ui.theTaskOfSynchronizationIsOpenedAndIsPerformedManually'),
        t('ui.waitsForEachTaskToBeCompletedInTheSimultaneous'),
        t('ui.theCorporateStructureAndTheManagementStructureAreReplacedIn'),
        t('ui.repeatAllTasksAndConfirmThatTheSameSourcekeyDoes'),
      ]
    },
    get expected() {
      return [
        t('ui.theCorporateAndRegulatoryStructuresAreSeparatelyAccessible'),
        t('ui.theFactThatTheSameOrganizationDoesNotCreateDuplicate'),
        t('ui.synchronizationOfBatchingAndExecutionDetailsCanTraceEveryReal'),
      ]
    },
    get note() {
      return t('ui.theCurrentHrConsumerContractCoversCorporateCompaniesManagementCompanies')
    },
    routeName: 'organization_sync_batch',
    get actionLabel() {
      return t('ui.openSyncBatch')
    },
  },
  {
    id: 'integration-call',
    sampleId: 'integration-call',
    get title() {
      return t('ui.externalInterfaceCall')
    },
    category: 'integration',
    icon: 'hub',
    status: 'configuration',
    get summary() {
      return t('ui.validateTheSystemEncryptionVouchersInterfacesRetestingStrategiesAndExecution')
    },
    get prerequisites() {
      return [
        t('ui.clickOnPrepareASampleTheSystemCreatesAVerivy'),
        t('ui.theSampleOnlyConnectsToTheStaticJsonAddressIn'),
      ]
    },
    get steps() {
      return [
        t('ui.opensTheInterfaceDefinitionSearchForVerifyPingCheckMethod'),
        t('ui.clickTheManualExecutionButtonOfTheInterfaceAndView'),
        t('ui.enterAttemptToCallLogToCheckHttpStatusTime'),
        t('ui.aNonExistentPathIsTemporarilyFilledOutAfterThe'),
      ]
    },
    get expected() {
      return [
        t('ui.theDocumentBodyDoesNotAppearInTheListDetails'),
        t('ui.theDetailsOfTheImplementationCanBeRolledAndPresented'),
        t('ui.businessFailuresAndTechnicalReTestsAreNotConfused'),
      ]
    },
    routeName: 'integration_execution',
    get actionLabel() {
      return t('ui.openExecutionRecord')
    },
  },
  {
    id: 'file-upload',
    sampleId: 'file-upload',
    get title() {
      return t('ui.uploadDownloadAndDeleteFiles')
    },
    category: 'file',
    icon: 'upload_file',
    status: 'data',
    get summary() {
      return t('ui.validatesNormalUploadAccessAndDeletionLifeCycleByLow')
    },
    get prerequisites() {
      return [t('ui.clickingOnThePrepareASampleWillReleaseAFunctional')]
    },
    get steps() {
      return [
        t('ui.downloadsTheSmallFilesProvidedOnThePageOpensThe'),
        t('ui.selectTheFileInTheNormalFileFieldSaveAnd'),
        t('ui.previewsOrDownloadsTheFileFromTheDetailedPageAfter'),
        t('ui.copyTheFileAddressAndExitLoginAndAccessIt'),
        t('ui.deletesOrReplacesTheFileConfirmingThatThePageReferences'),
      ]
    },
    get expected() {
      return [
        t('ui.permissionedUsersCanPreviewOrDownloadAsTheyWant'),
        t('ui.unauthorizedUsersCannotAccessOnFileIdAlone'),
        t('ui.theOldReferenceDoesNotAppearInTheRecordAfter'),
      ]
    },
    sampleFiles: [
      {
        get label() {
          return t('ui.downloadSmallFile')
        },
        fileName: 'sweet-admin-file-upload-sample.txt',
        url: '/verification-fixtures/files/sample-small.txt',
      },
      {
        get label() {
          return t('ui.generate6MibFiles')
        },
        fileName: 'sweet-admin-chunk-upload-sample.txt',
        generatedSizeMiB: 6,
      },
    ],
    routeName: 'lowcode_verify_file_record',
    get actionLabel() {
      return t('ui.openFileSample')
    },
  },
  {
    id: 'video-preview',
    sampleId: 'video-preview',
    get title() {
      return t('ui.videoAndBigFilePreview')
    },
    category: 'file',
    icon: 'movie',
    status: 'data',
    get summary() {
      return t('ui.validateVideoRangeRequestsPreviewPermissionsAndUploadsLargeDocument')
    },
    get prerequisites() {
      return [
        t('ui.clickOnThePrepareASampleWhichWillReleaseA'),
        t('ui.downloadsTheUnsensitiveContentMp4FromThePageTheFile'),
      ]
    },
    get steps() {
      return [
        t('ui.downloadsMp4OpensTheSamplePageAndEditsTheFunction'),
        t('ui.uploadTheDocumentToMp4VideoFieldsToObserveThe'),
        t('ui.opensTheVideoPreviewFromTheDetailsPageAndDrags'),
        t('ui.updatesThePageAndPreviewsItAgainToConfirmThat'),
        t('ui.reSelectFilesAndInterruptUploadsToConfirmThatIncomplete'),
      ]
    },
    get expected() {
      return [
        t('ui.theFractionsAreMergedOnlyAfterAllUploadsHaveBeen'),
        t('ui.rengeRespondedNormalWhenTheVideoWasDragged'),
        t('ui.previewCannotBeInterchangeableWithDownload'),
      ]
    },
    sampleFiles: [
      {
        get label() {
          return t('ui.downloadMp4Video')
        },
        fileName: 'sweet-admin-video-preview-sample.mp4',
        url: '/verification-fixtures/files/sample-video.mp4',
      },
    ],
    routeName: 'lowcode_verify_file_record',
    get actionLabel() {
      return t('ui.openVideoSample')
    },
  },
  {
    id: 'query-center',
    get title() {
      return t('ui.keywordsAndAdvancedQueries')
    },
    category: 'query',
    icon: 'manage_search',
    status: 'ready',
    get summary() {
      return t('ui.useTheUnifiedQueryProtocolForVerifyingKeywordsAdvancedConditions')
    },
    get prerequisites() {
      return [t('ui.selectAStandardListPageForWhichDataAreAvailable')]
    },
    get steps() {
      return [
        t('ui.enterTheKeywordAndQueryAndRecordTheResults'),
        t('ui.opensAnAdvancedQuerySetsASetOfConditionsWith'),
        t('ui.retainTheAdvancedConditionBeforeSubmittingKeywordQueries'),
        t('ui.repeatsTheQueryAfterYouHaveChangedTheSortingAnd'),
      ]
    },
    get expected() {
      return [
        t('ui.keywordsAreCombinedWithAdvancedConditions'),
        t('ui.advancedConditionsWillNotBeEmptiedByKeywordQueries'),
        t('ui.theConditionPreviewShowsTheTrueLogicByLevel'),
      ]
    },
    routeName: 'system_application',
    get actionLabel() {
      return t('ui.openStandardList')
    },
  },
  {
    id: 'query-scheme',
    get title() {
      return t('ui.queryScheme')
    },
    category: 'query',
    icon: 'bookmark',
    status: 'ready',
    get summary() {
      return t('ui.verifyThePreservationApplicationAndPermissionOfTheDefaultPrograms')
    },
    get prerequisites() {
      return [t('ui.theCurrentAccountNumberCanAccessAtLeastOnePage')]
    },
    get steps() {
      return [
        t('ui.setKeywordsAdvancedConditionsRankingAndDynamicBindingToSave'),
        t('ui.modifysTheCurrentConditionChecksTheDirtyStatusAndSaves'),
        t('ui.viewDetailsDefaultStatusAndVisibleScopeOfTheProgram'),
        t('ui.switchAccountsToVerifyPersonalIsolationAndRoleProgrammeVisibility'),
      ]
    },
    get expected() {
      return [
        t('ui.programSavesTheFullQueryStatus'),
        t('ui.individualProgrammesWillNotBeSeenByOtherUsers'),
        t('ui.unpermittedTargetPageCannotBeBypassedByTheUseProgram'),
      ]
    },
    routeName: 'query_scheme_manager',
    get actionLabel() {
      return t('ui.manageQueryPrograms')
    },
  },
  {
    id: 'notification',
    sampleId: 'notification',
    get title() {
      return t('ui.messageNotification')
    },
    category: 'query',
    icon: 'notifications',
    status: 'data',
    get summary() {
      return t('ui.verifyUnreadRecentNewsReadStatusUserIsolationAndSafe')
    },
    get prerequisites() {
      return [t('ui.clickOnThePrepareASampleWhichSendsThreeTypes')]
    },
    get steps() {
      return [
        t('ui.watchIfHeaderSUnreadingNumbersAndRecentNotificationsAre'),
        t('ui.markAReadItemAndExecuteAllRead'),
        t('ui.opensTheMessageCentreToVerifyAllUnreadAndRead'),
        t('ui.clickOnThePermissionsAndUnauthorisedActionMessage'),
      ]
    },
    get expected() {
      return [
        t('ui.usersCanOnlySeeTheirOwnNotifications'),
        t('ui.readsTheOperationChargonEtcAndDoesNotReadThe'),
        t('ui.noPermissionToActionDoesNotSkipOrDivulgeTarget'),
      ]
    },
    routeName: 'notification_center',
    get actionLabel() {
      return t('ui.openTheMessageCenter')
    },
  },
  {
    id: 'report-runtime',
    get title() {
      return t('ui.reportReleaseAndRunning')
    },
    category: 'report',
    icon: 'analytics',
    status: 'ready',
    get summary() {
      return t('ui.verifysTheDesignPublicationOperationExportAndExecutionLogFor')
    },
    get prerequisites() {
      return [
        t('ui.theSystemHasBeenInitializedAndPublishedAsAnAccess'),
        t('ui.severalQueriesOrSavesAreMadeInTheSystemTo'),
      ]
    },
    get steps() {
      return [
        t('ui.opensTheAccessLogOverviewAtTheReportCentreAnd'),
        t('ui.validatesTheRunResultsTheCsvExportAndTheExecution'),
        t('ui.opensThisDefinitionInTheReportManagementContrastsTheTable'),
        t('ui.copyAStatementToModifyAndPublishToConfirmThat'),
      ]
    },
    get expected() {
      return [
        t('ui.designVersionIsIsolatedFromThePublishedRunOffVersion'),
        t('ui.exportContentIsConsistentWithTheResultsOfYourOperation'),
        t('ui.unauthorizedUsersCannotRunOrReadTheReportData'),
      ]
    },
    routeName: 'report_center',
    get actionLabel() {
      return t('ui.openReportCenter')
    },
  },
]

const filteredScenarios = computed(() => {
  const needle = keyword.value.trim().toLowerCase()
  return scenarios.filter((scenario) => {
    if (category.value !== 'all' && scenario.category !== category.value) return false
    if (!needle) return true
    return `${scenario.title} ${scenario.summary}`.toLowerCase().includes(needle)
  })
})

const selectedScenario = computed(() => {
  const visible = filteredScenarios.value
  return visible.find((scenario) => scenario.id === selectedId.value) || visible[0] || null
})

const sampleStatus = (scenario: VerificationScenario) => {
  if (!scenario.sampleId) return undefined
  return statuses.value[scenario.sampleId]
}

const scenarioChip = (scenario: VerificationScenario) => {
  if (scenario.sampleId) {
    const status = sampleStatus(scenario)
    if (!status)
      return {
        get label() {
          return t('ui.reading')
        },
        color: 'grey-7',
        outline: true,
      }
    return sampleStateLabels[status.state]
  }
  const status = statusLabels[scenario.status]
  return { ...status, outline: scenario.status !== 'ready' }
}

const loadSampleStatuses = async () => {
  loadingStatuses.value = true
  try {
    const response = await verificationApi.statuses()
    const next: Partial<Record<VerificationSampleScenario, VerificationSampleStatus>> = {}
    for (const status of response.data || []) next[status.scenario_id] = status
    statuses.value = next
  } finally {
    loadingStatuses.value = false
  }
}

const prepareSample = async (scenario: VerificationScenario) => {
  if (!scenario.sampleId) return
  preparingId.value = scenario.sampleId
  try {
    const response = await verificationApi.prepare(scenario.sampleId)
    if (!response.data) return
    statuses.value = { ...statuses.value, [scenario.sampleId]: response.data.status }
    preparedAccounts.value = response.data.accounts || []
    accountDialog.value = preparedAccounts.value.length > 0
    $q.notify({
      type: 'positive',
      get message() {
        return t('ui.samplePrepared', { value1: scenario.title })
      },
    })
  } finally {
    preparingId.value = null
  }
}

const cleanupSample = async (scenario: VerificationScenario) => {
  if (!scenario.sampleId) return
  cleaningId.value = scenario.sampleId
  try {
    const response = await verificationApi.cleanup(scenario.sampleId)
    if (response.data) statuses.value = { ...statuses.value, [scenario.sampleId]: response.data }
    $q.notify({
      type: 'positive',
      get message() {
        return t('ui.sampleCleared', { value1: scenario.title })
      },
    })
  } finally {
    cleaningId.value = null
  }
}

const confirmCleanup = (scenario: VerificationScenario) => {
  $q.dialog({
    get title() {
      return t('ui.cleanupFunctionValidationSample')
    },
    get message() {
      return t('ui.onlyTheVerifyConfigurationDataAndFunctionalValidationAccountCreated')
    },
    cancel: true,
    persistent: true,
  }).onOk(() => {
    void cleanupSample(scenario)
  })
}

const copyPassword = async (account: VerificationSampleAccount) => {
  await copyToClipboard(
    t('ui.accountNumberPassword', { value1: account.user_name, value2: account.password }),
  )
  $q.notify({
    type: 'positive',
    get message() {
      return t('ui.accountNumberAndPasswordCopied', { value1: account.user_name })
    },
  })
}

const downloadSampleFile = async (file: VerificationSampleFile) => {
  try {
    let blob: Blob
    if (file.generatedSizeMiB) {
      const bytes = Math.max(1, Math.round(file.generatedSizeMiB * 1024 * 1024))
      const marker = 'Sweet Admin chunk upload verification sample.\n'
      blob = new Blob([marker.repeat(Math.ceil(bytes / marker.length)).slice(0, bytes)], {
        type: 'text/plain;charset=utf-8',
      })
    } else if (file.url) {
      const response = await fetch(file.url)
      if (!response.ok) throw new Error(`HTTP ${response.status}`)
      blob = await response.blob()
    } else {
      throw new Error(t('ui.sampleFileAddressNotConfigured'))
    }
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = file.fileName
    document.body.appendChild(link)
    link.click()
    link.remove()
    URL.revokeObjectURL(url)
  } catch {
    $q.notify({
      type: 'negative',
      get message() {
        return t('ui.theDownloadOfTheSampleFileFailedPleaseConfirmThat')
      },
    })
  }
}

const runIntegrationSample = async () => {
  runningIntegrationSample.value = true
  try {
    const systems = await integrationApi.queryExternalSystems({
      page: 1,
      num: 100,
      expressions: [],
    })
    const system = systems.data?.find((item) => item.system_code === 'verify_integration_source')
    if (!system) throw new Error(t('ui.noExternalSystemFoundForFunctionalValidation'))

    const definitions = await integrationApi.queryInterfaceDefinitions({
      page: 1,
      num: 100,
      expressions: [],
      external_system_id: system.id,
    })
    const definition = definitions.data?.find((item) => item.interface_code === 'verify_ping')
    if (!definition) throw new Error(t('ui.noFunctionalAuthenticationInterfaceFound'))

    const result = await integrationApi.createExecution({
      external_system_id: system.id,
      interface_definition_id: definition.id,
      trigger_source: 'manual',
      idempotency_scope: 'development_verification',
      idempotency_key: `verify-ping-${Date.now()}`,
      input: { path_params: {}, query_params: {}, headers: {} },
    })
    if (!result.success || !result.data) throw new Error(t('ui.failedToCreateExecutionRecord'))
    $q.notify({
      type: 'positive',
      get message() {
        return t('ui.createdConnectivityTest', { value1: result.data.execution_no })
      },
    })
    await router.push({ name: 'integration_execution_detail_page', params: { id: result.data.id } })
  } catch (error) {
    $q.notify({
      type: 'negative',
      get message() {
        return error instanceof Error ? error.message : t('ui.runConnectivityTestFailed')
      },
    })
  } finally {
    runningIntegrationSample.value = false
  }
}

const openScenario = async (scenario: VerificationScenario) => {
  if (!scenario.routeName || !router.hasRoute(scenario.routeName)) {
    const prepared = scenario.sampleId && sampleStatus(scenario)?.state === 'ready'
    $q.notify({
      type: 'warning',
      get message() {
        return prepared
          ? t('ui.theSamplePageHasJustBeenCreatedPleaseReEnter')
          : t('ui.noRelevantPageToOpenForTheCurrentAccount')
      },
    })
    return
  }
  await router.push({ name: scenario.routeName })
}

onMounted(loadSampleStatuses)
</script>

<style scoped lang="scss">
.verification-page {
  overflow: hidden;
}

.verification-workbench {
  display: grid;
  grid-template-columns: minmax(320px, 390px) minmax(0, 1fr);
  height: 100%;
  min-height: 0;
  overflow: hidden;
  border: 1px solid var(--app-border);
  background: var(--app-surface);
}

.verification-master,
.verification-detail {
  min-width: 0;
  min-height: 0;
}

.verification-detail {
  display: flex;
  flex-direction: column;
}

.verification-guide-note {
  flex: 0 0 auto;
  border-bottom: 1px solid var(--app-border);
  background: var(--app-surface-muted);
  color: var(--app-text-secondary);
}

.verification-master {
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--app-border);
}

.verification-toolbar {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 132px;
  gap: 8px;
  padding: 12px;
  border-bottom: 1px solid var(--app-border);
}

.verification-master-scroll,
.verification-detail-scroll {
  flex: 1;
  height: 100%;
  min-height: 0;
}

.verification-item--active {
  color: var(--q-primary);
  background: var(--app-primary-soft);
  box-shadow: inset 3px 0 0 var(--q-primary);
}

.verification-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 48px 16px;
  color: var(--app-text-muted);
}

.verification-detail-content {
  max-width: 1100px;
  margin: 0 auto;
  padding: 20px 24px 40px;
}

.verification-detail-header {
  display: flex;
  align-items: center;
  gap: 16px;
  padding-bottom: 18px;
  border-bottom: 1px solid var(--app-border);
}

.verification-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  flex-wrap: wrap;
}

.verification-detail-title-wrap {
  display: flex;
  align-items: center;
  gap: 14px;
  min-width: 0;
}

.verification-detail-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: 8px;
  color: #fff;
  background: var(--q-primary);
  font-size: 26px;
}

.verification-detail-title {
  color: var(--app-text-strong);
  font-size: 22px;
  font-weight: 700;
}

.verification-detail-summary {
  margin-top: 4px;
  color: var(--app-text-muted);
  line-height: 1.5;
}

.verification-sample-status {
  padding: 16px;
  margin-top: 16px;
  border: 1px solid var(--app-border);
  border-radius: 6px;
  background: var(--app-surface-muted);
}

.verification-sample-status__main {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--app-text-strong);
}

.verification-sample-files {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 16px;
  margin-top: 12px;
  border: 1px solid var(--app-border);
  border-radius: 6px;
}

.verification-sample-files > div:first-child {
  display: grid;
  gap: 4px;
}

.verification-sample-files span {
  color: var(--app-text-muted);
  font-size: 12px;
}

.verification-sample-files__actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.verification-sample-facts {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  margin-top: 14px;
}

.verification-sample-facts > div {
  display: grid;
  gap: 4px;
  min-width: 0;
}

.verification-sample-facts span {
  color: var(--app-text-muted);
  font-size: 12px;
}

.verification-sample-facts strong {
  overflow-wrap: anywhere;
  color: var(--app-text-strong);
  font-weight: 600;
}

.verification-section {
  padding: 20px 0;
  border-bottom: 1px solid var(--app-border);
}

.verification-section h3 {
  margin: 0 0 12px;
  color: var(--app-text-strong);
  font-size: 17px;
}

.verification-steps {
  display: grid;
  gap: 12px;
  margin: 0;
  padding-left: 28px;
  color: var(--app-text-strong);
  line-height: 1.65;
}

.verification-note {
  margin-top: 20px;
  border: 1px solid var(--app-primary-border);
  color: var(--app-text-strong);
  background: var(--app-primary-soft);
}

.verification-account-dialog {
  width: min(720px, calc(100vw - 32px));
  max-width: 720px;
}

.verification-password {
  display: flex;
  align-items: center;
  gap: 4px;
}

.verification-password code {
  padding: 5px 8px;
  border: 1px solid var(--app-border);
  border-radius: 4px;
  color: var(--app-text-strong);
  background: var(--app-surface-muted);
}

@media (max-width: 900px) {
  .verification-workbench {
    grid-template-columns: minmax(280px, 36%) minmax(0, 1fr);
  }

  .verification-toolbar {
    grid-template-columns: 1fr;
  }

  .verification-detail-header {
    align-items: flex-start;
    flex-wrap: wrap;
  }

  .verification-actions {
    width: 100%;
    justify-content: flex-start;
  }

  .verification-sample-facts {
    grid-template-columns: 1fr;
  }

  .verification-sample-files {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
