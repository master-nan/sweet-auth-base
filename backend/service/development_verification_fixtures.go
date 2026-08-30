package service

import (
	"backend/dto/response"
	"backend/enum"
	myerrors "backend/internal/errors"
	"backend/internal/organization/hrsync"
	"backend/internal/security"
	"backend/model"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	verificationIntegrationSystemCode     = "verify_integration_source"
	verificationIntegrationInterfaceCode  = "verify_ping"
	verificationIntegrationCredentialCode = "verify_api_key"
	verificationIntegrationRetryCode      = "verify_integration_retry"

	verificationOrganizationSystemCode = "verify_hr_source"
	verificationOrganizationRetryCode  = "verify_hr_retry"
)

type organizationFixtureDefinition struct {
	interfaceCode string
	interfaceName string
	path          string
	taskCode      string
	taskName      string
	consumerCode  string
}

var organizationFixtureDefinitions = []organizationFixtureDefinition{
	{"verify_hr_legal_entities", "功能验证-HR 法人公司", "/verification-fixtures/hr/legal-entities.json", "verify_hr_legal_entity", "功能验证-法人公司同步", hrsync.ConsumerCodeLegalEntity},
	{"verify_hr_management_companies", "功能验证-HR 管理公司", "/verification-fixtures/hr/management-companies.json", "verify_hr_management_company", "功能验证-管理公司同步", hrsync.ConsumerCodeManagementCompany},
	{"verify_hr_legal_departments", "功能验证-HR 法人部门", "/verification-fixtures/hr/legal-departments.json", "verify_hr_legal_department", "功能验证-法人部门同步", hrsync.ConsumerCodeLegalDepartment},
	{"verify_hr_management_departments", "功能验证-HR 管理部门", "/verification-fixtures/hr/management-departments.json", "verify_hr_management_department", "功能验证-管理部门同步", hrsync.ConsumerCodeManagementDepartment},
	{"verify_hr_positions", "功能验证-HR 岗位", "/verification-fixtures/hr/positions.json", "verify_hr_position", "功能验证-岗位同步", hrsync.ConsumerCodePosition},
	{"verify_hr_employees", "功能验证-HR 员工", "/verification-fixtures/hr/employees.json", "verify_hr_employee", "功能验证-员工同步", hrsync.ConsumerCodeEmployee},
	{"verify_hr_resigned_employees", "功能验证-HR 离职员工", "/verification-fixtures/hr/resigned-employees.json", "verify_hr_resigned_employee", "功能验证-离职员工同步", hrsync.ConsumerCodeResignedEmployee},
}

func (service *DevelopmentVerificationService) integrationFixtureStatus(ctx context.Context) (response.DevelopmentVerificationStatusRes, error) {
	system := service.countModel(ctx, &model.ExternalSystem{}, "system_code = ? AND status = ? AND state = true", verificationIntegrationSystemCode, model.ExternalSystemStatusEnabled)
	definition := service.countModel(ctx, &model.InterfaceDefinition{}, "interface_code = ? AND status = ? AND state = true", verificationIntegrationInterfaceCode, model.InterfaceDefinitionStatusEnabled)
	credential := service.countModel(ctx, &model.Credential{}, "credential_code = ? AND status = ? AND state = true", verificationIntegrationCredentialCode, model.CredentialStatusActive)
	policy := service.countModel(ctx, &model.RetryPolicy{}, "policy_code = ? AND status = ? AND state = true", verificationIntegrationRetryCode, model.RetryPolicyStatusEnabled)
	count := system + definition + credential + policy
	state := verificationState(count == 4, count > 0)
	return response.DevelopmentVerificationStatusRes{
		ScenarioId: verificationScenarioIntegration, State: state, Available: true, ItemCount: count,
		Summary: verificationSummary(state, "外部系统、加密凭证、接口和重试策略已准备", "尚未准备外部接口调用样例"),
		Details: []response.DevelopmentVerificationDetailRes{
			{Label: "样例地址", Value: service.verificationFixtureBaseURL()},
			{Label: "接口", Value: verificationIntegrationInterfaceCode},
			{Label: "执行 Worker", Value: enabledLabel(service.config.Integration.Worker.Enabled)},
		},
	}, nil
}

func (service *DevelopmentVerificationService) organizationFixtureStatus(ctx context.Context) (response.DevelopmentVerificationStatusRes, error) {
	system := service.countModel(ctx, &model.ExternalSystem{}, "system_code = ? AND status = ? AND state = true", verificationOrganizationSystemCode, model.ExternalSystemStatusEnabled)
	interfaces := service.countModel(ctx, &model.InterfaceDefinition{}, "interface_code LIKE ? AND status = ? AND state = true", "verify_hr_%", model.InterfaceDefinitionStatusEnabled)
	tasks := service.countModel(ctx, &model.IntegrationSyncTask{}, "task_code LIKE ? AND status = ? AND state = true", "verify_hr_%", model.IntegrationSyncTaskStatusEnabled)
	ready := system == 1 && interfaces == len(organizationFixtureDefinitions) && tasks == len(organizationFixtureDefinitions) && service.config.Integration.OrganizationHR.Enabled
	state := verificationState(ready, system+interfaces+tasks > 0)
	return response.DevelopmentVerificationStatusRes{
		ScenarioId: verificationScenarioOrganization, State: state, Available: true, ItemCount: tasks,
		Summary: verificationSummary(state, "法人、管理组织、岗位和人员的真实同步任务已准备", "尚未准备组织同步样例配置"),
		Details: []response.DevelopmentVerificationDetailRes{
			{Label: "接口 / 任务", Value: fmt.Sprintf("%d / %d", interfaces, tasks)},
			{Label: "HR Consumer", Value: enabledLabel(service.config.Integration.OrganizationHR.Enabled)},
			{Label: "调度 / Worker", Value: enabledLabel(service.config.Integration.SyncRunner.Enabled) + " / " + enabledLabel(service.config.Integration.Worker.Enabled)},
		},
	}, nil
}

func (service *DevelopmentVerificationService) fileFixtureStatus(ctx context.Context, scenario string) (response.DevelopmentVerificationStatusRes, error) {
	records := service.countPhysicalRows(ctx, verificationFileTable, "")
	var tables int64
	_ = service.db.WithContext(ctx).Model(&model.SysTable{}).Where("table_code = ? AND state = true", verificationFileTable).Count(&tables).Error
	state := verificationState(records > 0 && tables == 1, records > 0 || tables > 0)
	return response.DevelopmentVerificationStatusRes{
		ScenarioId: scenario, State: state, Available: true, ItemCount: records,
		Summary: verificationSummary(state, "普通文件与 MP4 分片上传字段的低代码页面已发布", "尚未准备文件与视频样例页面"),
		Details: []response.DevelopmentVerificationDetailRes{
			{Label: "样例表", Value: verificationFileTable},
			{Label: "普通文件阈值", Value: "1 MiB"},
			{Label: "视频分片阈值", Value: "0.1 MiB"},
		},
	}, nil
}

func (service *DevelopmentVerificationService) countModel(ctx context.Context, value any, query string, args ...any) int {
	var count int64
	if err := service.db.WithContext(ctx).Model(value).Where(query, args...).Count(&count).Error; err != nil {
		return 0
	}
	return int(count)
}

func enabledLabel(enabled bool) string {
	if enabled {
		return "已启用"
	}
	return "未启用"
}

func (service *DevelopmentVerificationService) verificationFixtureBaseURL() string {
	if service == nil || service.config == nil {
		return "未配置"
	}
	value := strings.TrimRight(strings.TrimSpace(service.config.Integration.VerificationFixtureBaseURL), "/")
	if value == "" {
		return "未配置"
	}
	return value
}

func (service *DevelopmentVerificationService) requireVerificationFixtureBaseURL() (string, error) {
	value := service.verificationFixtureBaseURL()
	if value == "未配置" {
		return "", myerrors.NewValidationError("当前环境未配置 integration.verification_fixture_base_url；默认 Docker 环境已内置")
	}
	return value, nil
}

func (service *DevelopmentVerificationService) prepareIntegrationFixture(ctx context.Context) error {
	baseURL, err := service.requireVerificationFixtureBaseURL()
	if err != nil {
		return err
	}
	system, err := service.ensureVerificationExternalSystem(ctx, verificationIntegrationSystemCode, "功能验证-外部接口源", model.ExternalSystemTypeOther, baseURL)
	if err != nil {
		return err
	}
	policy, err := service.ensureVerificationRetryPolicy(ctx, verificationIntegrationRetryCode, "功能验证-接口重试策略")
	if err != nil {
		return err
	}
	credential, err := service.ensureVerificationCredential(ctx, system, verificationIntegrationCredentialCode)
	if err != nil {
		return err
	}
	_, err = service.ensureVerificationInterface(ctx, system, verificationIntegrationInterfaceCode, "功能验证-连通性接口", "/verification-fixtures/integration/ping.json", &credential.Id, &policy.Id, false)
	return err
}

func (service *DevelopmentVerificationService) prepareOrganizationFixture(ctx context.Context) error {
	if !service.config.Integration.OrganizationHR.Enabled {
		return myerrors.NewValidationError("当前服务未启用 integration.organization_hr，无法注册组织同步 Consumer")
	}
	baseURL, err := service.requireVerificationFixtureBaseURL()
	if err != nil {
		return err
	}
	system, err := service.ensureVerificationExternalSystem(ctx, verificationOrganizationSystemCode, "功能验证-HR 数据源", model.ExternalSystemTypeHR, baseURL)
	if err != nil {
		return err
	}
	policy, err := service.ensureVerificationRetryPolicy(ctx, verificationOrganizationRetryCode, "功能验证-HR 重试策略")
	if err != nil {
		return err
	}
	for _, fixture := range organizationFixtureDefinitions {
		definition, ensureErr := service.ensureVerificationInterface(ctx, system, fixture.interfaceCode, fixture.interfaceName, fixture.path, nil, &policy.Id, true)
		if ensureErr != nil {
			return ensureErr
		}
		if ensureErr = service.ensureVerificationSyncTask(ctx, system, definition, fixture); ensureErr != nil {
			return ensureErr
		}
	}
	return nil
}

func (service *DevelopmentVerificationService) prepareFileFixture(ctx context.Context) error {
	ddl := `CREATE TABLE IF NOT EXISTS verify_file_record (
		id bigint PRIMARY KEY, gmt_create timestamptz NOT NULL, gmt_create_user bigint NOT NULL,
		gmt_modify timestamptz NOT NULL, gmt_modify_user bigint NOT NULL, gmt_delete timestamptz NULL,
		gmt_delete_user bigint NULL, state boolean NOT NULL DEFAULT true,
		record_name varchar(128) NOT NULL, attachment jsonb NULL, video jsonb NULL, notes text NULL)`
	table, err := service.ensureMetadataTable(ctx, verificationFileTable, "功能验证-文件与视频", ddl)
	if err != nil {
		return err
	}
	if err = service.configureFields(ctx, table, map[string]map[string]any{
		"record_name": {"field_name": "记录名称", "is_quick_search": true, "is_advanced_search": true, "is_sort": true, "is_list_show": true, "is_insert_show": true, "is_update_show": true, "sequence": 9},
		"attachment":  {"field_name": "普通文件", "input_type": enum.FilePickerInputType, "binding": `{"multiple":false,"accept":".txt,.pdf,.zip","maxSize":50,"chunkThreshold":1,"concurrency":4}`, "is_quick_search": false, "is_advanced_search": false, "is_list_show": true, "is_insert_show": true, "is_update_show": true, "sequence": 10},
		"video":       {"field_name": "MP4 视频", "input_type": enum.FilePickerInputType, "binding": `{"multiple":false,"accept":"video/mp4,.mp4","maxSize":50,"chunkThreshold":0.1,"concurrency":4}`, "is_quick_search": false, "is_advanced_search": false, "is_list_show": true, "is_insert_show": true, "is_update_show": true, "sequence": 11},
		"notes":       {"field_name": "验证说明", "input_type": enum.TextareaInputType, "is_list_show": true, "is_insert_show": true, "is_update_show": true, "sequence": 12},
	}); err != nil {
		return err
	}
	if err = service.db.WithContext(ctx).Exec("DELETE FROM " + verificationFileTable).Error; err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	id, err := service.nextID()
	if err != nil {
		return err
	}
	if err = service.db.WithContext(ctx).Exec(`INSERT INTO verify_file_record
		(id,gmt_create,gmt_create_user,gmt_modify,gmt_modify_user,state,record_name,notes)
		VALUES (?,CURRENT_TIMESTAMP,0,CURRENT_TIMESTAMP,0,true,?,?)`, id, "功能验证-上传测试", "编辑本记录并上传页面提供的测试文件").Error; err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	_, err = service.publishTable(ctx, verificationFileTable)
	return err
}

func (service *DevelopmentVerificationService) ensureVerificationExternalSystem(ctx context.Context, code, name, systemType, baseURL string) (model.ExternalSystem, error) {
	var value model.ExternalSystem
	err := service.db.WithContext(ctx).Unscoped().Where("system_code = ?", code).First(&value).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		id, nextErr := service.nextID()
		if nextErr != nil {
			return model.ExternalSystem{}, nextErr
		}
		value = model.ExternalSystem{Basic: model.Basic{Id: id, State: true}, SystemCode: code, Revision: 1}
	} else if err != nil {
		return model.ExternalSystem{}, myerrors.WrapDatabaseError(err)
	}
	value.Name, value.SystemType, value.BaseURL = name, systemType, baseURL
	value.OwnerIdentifier, value.OwnerName = "development_verification", "功能验证"
	value.Status, value.Description, value.State = model.ExternalSystemStatusEnabled, "功能验证专用，可安全清理", true
	value.Revision, value.GmtDelete, value.DeleteUser = 1, gorm.DeletedAt{}, nil
	if err = service.db.WithContext(ctx).Unscoped().Save(&value).Error; err != nil {
		return model.ExternalSystem{}, myerrors.WrapDatabaseError(err)
	}
	return value, nil
}

func (service *DevelopmentVerificationService) ensureVerificationRetryPolicy(ctx context.Context, code, name string) (model.RetryPolicy, error) {
	var value model.RetryPolicy
	err := service.db.WithContext(ctx).Unscoped().Where("policy_code = ? AND version = 1", code).First(&value).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		id, nextErr := service.nextID()
		if nextErr != nil {
			return model.RetryPolicy{}, nextErr
		}
		value = model.RetryPolicy{Basic: model.Basic{Id: id, State: true}, PolicyCode: code, Version: 1}
	} else if err != nil {
		return model.RetryPolicy{}, myerrors.WrapDatabaseError(err)
	}
	value.PolicyName, value.Description, value.Status = name, "功能验证专用，可安全清理", model.RetryPolicyStatusEnabled
	value.MaxAttempts, value.InitialDelayMs, value.MaxDelayMs = 2, 1000, 2000
	value.BackoffType, value.BackoffMultiplier = model.RetryBackoffTypeFixed, 1
	value.JitterType, value.JitterRatio, value.RetryWindowMs = model.RetryJitterTypeFull, 1, 60000
	value.RetryableErrorCategories = datatypes.JSON([]byte(`["network","timeout","remote"]`))
	value.RetryableHTTPStatuses = datatypes.JSON([]byte(`[429,502,503,504]`))
	value.RespectRetryAfter, value.Revision, value.State = true, 1, true
	value.GmtDelete, value.DeleteUser = gorm.DeletedAt{}, nil
	err = service.db.WithContext(ctx).Unscoped().Save(&value).Error
	if err != nil {
		return model.RetryPolicy{}, myerrors.WrapDatabaseError(err)
	}
	return value, nil
}

func (service *DevelopmentVerificationService) ensureVerificationCredential(ctx context.Context, system model.ExternalSystem, code string) (model.Credential, error) {
	protector, err := security.NewCredentialSecretProtector(service.config)
	if err != nil {
		return model.Credential{}, myerrors.WrapSystemError(err)
	}
	secret, _ := json.Marshal(map[string]string{"api_key": "sweet-admin-verification-key"})
	envelope, err := protector.Seal(secret)
	if err != nil {
		return model.Credential{}, myerrors.WrapSystemError(err)
	}
	var value model.Credential
	err = service.db.WithContext(ctx).Unscoped().Where("external_system_id = ? AND credential_code = ?", system.Id, code).First(&value).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		id, nextErr := service.nextID()
		if nextErr != nil {
			return model.Credential{}, nextErr
		}
		value = model.Credential{Basic: model.Basic{Id: id, State: true}, ExternalSystemID: system.Id, CredentialCode: code}
	} else if err != nil {
		return model.Credential{}, myerrors.WrapDatabaseError(err)
	}
	value.Name, value.CredentialType, value.Status = "功能验证-API Key", model.CredentialTypeAPIKey, model.CredentialStatusActive
	value.SecretStorageRef, value.SecretCiphertext, value.SecretNonce, value.SecretFingerprint = envelope.StorageRef, envelope.Ciphertext, envelope.Nonce, envelope.Fingerprint
	value.Version, value.Description, value.Revision, value.State = 1, "明文不落库的功能验证凭证", 1, true
	value.GmtDelete, value.DeleteUser = gorm.DeletedAt{}, nil
	err = service.db.WithContext(ctx).Unscoped().Save(&value).Error
	if err != nil {
		return model.Credential{}, myerrors.WrapDatabaseError(err)
	}
	return value, nil
}

func (service *DevelopmentVerificationService) ensureVerificationInterface(ctx context.Context, system model.ExternalSystem, code, name, path string, credentialID, retryPolicyID *int, withWindow bool) (model.InterfaceDefinition, error) {
	contract := datatypes.JSON([]byte(`{"version":1,"parameters":[]}`))
	if withWindow {
		contract = datatypes.JSON([]byte(`{"version":1,"parameters":[{"code":"updated_from","location":"query","data_type":"string","required":true,"max_length":64},{"code":"updated_to","location":"query","data_type":"string","required":true,"max_length":64}]}`))
	}
	protocol := model.InterfaceProtocolHTTPS
	if strings.HasPrefix(strings.ToLower(system.BaseURL), "http://") {
		protocol = model.InterfaceProtocolHTTP
	}
	var value model.InterfaceDefinition
	err := service.db.WithContext(ctx).Unscoped().Where("external_system_id = ? AND interface_code = ? AND version = 1", system.Id, code).First(&value).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		id, nextErr := service.nextID()
		if nextErr != nil {
			return model.InterfaceDefinition{}, nextErr
		}
		value = model.InterfaceDefinition{Basic: model.Basic{Id: id, State: true}, ExternalSystemID: system.Id, InterfaceCode: code, Version: 1}
	} else if err != nil {
		return model.InterfaceDefinition{}, myerrors.WrapDatabaseError(err)
	}
	value.Name, value.Protocol, value.HTTPMethod, value.RelativePath = name, protocol, model.InterfaceMethodGET, path
	value.InputContract, value.CredentialID, value.TimeoutSeconds, value.ResponseLimit = contract, credentialID, 30, 1<<20
	value.RetryPolicyID, value.IdempotencyMode = retryPolicyID, model.InterfaceIdempotencyModeSafeMethod
	value.Status, value.Description, value.Revision, value.State = model.InterfaceDefinitionStatusEnabled, "功能验证专用，可安全清理", 1, true
	value.GmtDelete, value.DeleteUser = gorm.DeletedAt{}, nil
	err = service.db.WithContext(ctx).Unscoped().Save(&value).Error
	if err != nil {
		return model.InterfaceDefinition{}, myerrors.WrapDatabaseError(err)
	}
	return value, nil
}

func (service *DevelopmentVerificationService) ensureVerificationSyncTask(ctx context.Context, system model.ExternalSystem, definition model.InterfaceDefinition, fixture organizationFixtureDefinition) error {
	initial := model.Now().UTC().Add(-time.Hour)
	plan := datatypes.JSON([]byte(`{"version":1,"static_input":{},"window_start_binding":{"location":"query","code":"updated_from","format":"rfc3339"},"window_end_binding":{"location":"query","code":"updated_to","format":"rfc3339"}}`))
	var value model.IntegrationSyncTask
	err := service.db.WithContext(ctx).Unscoped().Where("task_code = ? AND version = 1", fixture.taskCode).First(&value).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		id, nextErr := service.nextID()
		if nextErr != nil {
			return nextErr
		}
		value = model.IntegrationSyncTask{Basic: model.Basic{Id: id, State: true}, TaskCode: fixture.taskCode, Version: 1}
	} else if err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	value.TaskName, value.Description, value.Status = fixture.taskName, "功能验证静态来源，按页面步骤手工运行", model.IntegrationSyncTaskStatusEnabled
	value.ExternalSystemID, value.InterfaceDefinitionID = system.Id, definition.Id
	value.ConsumerCode, value.ConsumerVersion = fixture.consumerCode, hrsync.ConsumerVersionV1
	value.ScheduleType, value.CronExpression, value.Timezone = model.IntegrationSyncScheduleNone, "", "Asia/Shanghai"
	value.CheckpointMode, value.InitialCheckpointAt, value.CheckpointAt = model.IntegrationSyncCheckpointTimestamp, &initial, &initial
	value.LookbackSeconds, value.WindowSliceSeconds, value.InputPlan = 0, 3600, plan
	value.Revision, value.State = 1, true
	value.GmtDelete, value.DeleteUser = gorm.DeletedAt{}, nil
	err = service.db.WithContext(ctx).Unscoped().Save(&value).Error
	if err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	return nil
}

func (service *DevelopmentVerificationService) cleanupIntegrationFixture(ctx context.Context) error {
	return service.cleanupVerificationIntegrationConfiguration(ctx, verificationIntegrationSystemCode, verificationIntegrationRetryCode)
}

func (service *DevelopmentVerificationService) cleanupOrganizationFixture(ctx context.Context) error {
	if err := service.cleanupVerificationOrganizationData(ctx); err != nil {
		return err
	}
	return service.cleanupVerificationIntegrationConfiguration(ctx, verificationOrganizationSystemCode, verificationOrganizationRetryCode)
}

func (service *DevelopmentVerificationService) cleanupFileFixture(ctx context.Context) error {
	return service.dropOwnedMetadataTable(ctx, verificationFileTable)
}

func (service *DevelopmentVerificationService) cleanupVerificationIntegrationConfiguration(ctx context.Context, systemCode, retryCode string) error {
	if !strings.HasPrefix(systemCode, "verify_") || !strings.HasPrefix(retryCode, "verify_") {
		return myerrors.NewValidationError("拒绝清理非功能验证集成配置")
	}
	return service.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var system model.ExternalSystem
		if err := tx.Unscoped().Where("system_code = ?", systemCode).First(&system).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Unscoped().Where("policy_code = ?", retryCode).Delete(&model.RetryPolicy{}).Error
		} else if err != nil {
			return err
		}
		var executionIDs, batchIDs []int
		if err := tx.Unscoped().Model(&model.IntegrationExecution{}).Where("external_system_id = ?", system.Id).Pluck("id", &executionIDs).Error; err != nil {
			return err
		}
		if len(executionIDs) > 0 {
			if err := tx.Exec("DELETE FROM org_sync_record WHERE execution_id IN ?", executionIDs).Error; err != nil {
				return err
			}
			if err := tx.Exec("DELETE FROM org_sync_batch WHERE execution_id IN ?", executionIDs).Error; err != nil {
				return err
			}
			if err := tx.Exec("DELETE FROM integration_log WHERE execution_id IN ?", executionIDs).Error; err != nil {
				return err
			}
		}
		if err := tx.Exec("DELETE FROM integration_execution WHERE external_system_id = ?", system.Id).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Model(&model.IntegrationSyncBatch{}).Where("system_code = ?", systemCode).Pluck("id", &batchIDs).Error; err != nil {
			return err
		}
		if len(batchIDs) > 0 {
			if err := tx.Exec("DELETE FROM integration_sync_batch WHERE id IN ?", batchIDs).Error; err != nil {
				return err
			}
		}
		for _, query := range []struct {
			model any
			where string
			args  []any
		}{
			{&model.IntegrationSyncTask{}, "external_system_id = ?", []any{system.Id}},
			{&model.InterfaceDefinition{}, "external_system_id = ?", []any{system.Id}},
			{&model.Credential{}, "external_system_id = ?", []any{system.Id}},
			{&model.RetryPolicy{}, "policy_code = ?", []any{retryCode}},
			{&model.ExternalSystem{}, "id = ?", []any{system.Id}},
		} {
			if err := tx.Unscoped().Where(query.where, query.args...).Delete(query.model).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (service *DevelopmentVerificationService) cleanupVerificationOrganizationData(ctx context.Context) error {
	return service.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var employeeIDs, positionIDs, unitIDs, nodeIDs, entityIDs []int
		if err := tx.Model(&model.OrgEmployee{}).Where(
			"source_system_code = ? AND source_id IN ?",
			hrsync.OrganizationHRSourceSystemCode,
			[]string{"verify-employee-001"},
		).Pluck("id", &employeeIDs).Error; err != nil {
			return err
		}
		if len(employeeIDs) > 0 {
			if err := tx.Exec("DELETE FROM org_assignment WHERE employee_id IN ?", employeeIDs).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&model.OrgPosition{}).Where(
			"source_system_code = ? AND source_id IN ?",
			hrsync.OrganizationHRSourceSystemCode,
			[]string{"verify-position-engineer"},
		).Pluck("id", &positionIDs).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.OrgUnit{}).Where(
			"source_system_code = ? AND source_id IN ?",
			hrsync.OrganizationHRSourceSystemCode,
			[]string{
				"management_company:verify-management-hq",
				"legal_unit:verify-legal-dept-rd",
				"management_unit:verify-management-dept-rd",
			},
		).Pluck("id", &unitIDs).Error; err != nil {
			return err
		}
		if len(unitIDs) > 0 {
			if err := tx.Model(&model.OrgStructureNode{}).Where("org_unit_id IN ?", unitIDs).Pluck("id", &nodeIDs).Error; err != nil {
				return err
			}
			if len(nodeIDs) > 0 {
				if err := tx.Model(&model.OrgStructureNode{}).Where("parent_node_id IN ?", nodeIDs).Update("parent_node_id", nil).Error; err != nil {
					return err
				}
				if err := tx.Exec("DELETE FROM org_structure_node WHERE id IN ?", nodeIDs).Error; err != nil {
					return err
				}
			}
		}
		if err := tx.Model(&model.OrgLegalEntity{}).Where(
			"source_system_code = ? AND source_id IN ?",
			hrsync.OrganizationHRSourceSystemCode,
			[]string{"verify-legal-hq"},
		).Pluck("id", &entityIDs).Error; err != nil {
			return err
		}
		for _, deletion := range []struct {
			table string
			ids   []int
		}{
			{"org_employee", employeeIDs},
			{"org_position", positionIDs},
			{"org_unit", unitIDs},
			{"org_legal_entity", entityIDs},
		} {
			if len(deletion.ids) > 0 {
				if err := tx.Exec("DELETE FROM "+deletion.table+" WHERE id IN ?", deletion.ids).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}
