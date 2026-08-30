package service

import (
	"backend/config"
	"backend/dto/request"
	"backend/dto/response"
	"backend/enum"
	"backend/internal/audit"
	"backend/internal/database"
	myerrors "backend/internal/errors"
	"backend/internal/security"
	"backend/internal/utils"
	"backend/model"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	verificationScenarioDataPermission = "data-permission"
	verificationScenarioTMS            = "tms-company-scope"
	verificationScenarioMetadata       = "metadata-low-code"
	verificationScenarioNotification   = "notification"
	verificationScenarioIntegration    = "integration-call"
	verificationScenarioOrganization   = "organization-sync"
	verificationScenarioFileUpload     = "file-upload"
	verificationScenarioVideoPreview   = "video-preview"

	verificationPermissionTable = "verify_permission_order"
	verificationCategoryTable   = "verify_lowcode_category"
	verificationRecordTable     = "verify_lowcode_record"
	verificationFileTable       = "verify_file_record"
	verificationSource          = "development_verification"
)

var developmentVerificationScenarios = []string{
	verificationScenarioDataPermission,
	verificationScenarioTMS,
	verificationScenarioMetadata,
	verificationScenarioNotification,
	verificationScenarioIntegration,
	verificationScenarioOrganization,
	verificationScenarioFileUpload,
	verificationScenarioVideoPreview,
}

type DevelopmentVerificationService struct {
	db            *gorm.DB
	sf            *utils.Snowflake
	config        *config.Server
	tables        *SysTableService
	publication   *LowCodePublicationService
	users         *SysUserService
	roles         *SysRoleService
	notifications *NotificationService
}

func NewDevelopmentVerificationService(
	primary *database.PrimaryDB,
	sf *utils.Snowflake,
	config *config.Server,
	tables *SysTableService,
	publication *LowCodePublicationService,
	users *SysUserService,
	roles *SysRoleService,
	notifications *NotificationService,
) *DevelopmentVerificationService {
	return &DevelopmentVerificationService{
		db: primary.DB, sf: sf, config: config, tables: tables, publication: publication,
		users: users, roles: roles, notifications: notifications,
	}
}

// Statuses 只检查由本页面负责的样例，不把现有业务数据算进准备结果。
func (service *DevelopmentVerificationService) Statuses(
	ctx context.Context,
) ([]response.DevelopmentVerificationStatusRes, error) {
	result := make([]response.DevelopmentVerificationStatusRes, 0, len(developmentVerificationScenarios))
	for _, scenario := range developmentVerificationScenarios {
		status, err := service.status(ctx, scenario)
		if err != nil {
			return nil, err
		}
		result = append(result, status)
	}
	return result, nil
}

// Prepare 创建可重复使用的开发样例。再次执行会重置专用账号密码，但不会修改非 verify 数据。
func (service *DevelopmentVerificationService) Prepare(
	ctx context.Context,
	scenario string,
) (response.DevelopmentVerificationPrepareRes, error) {
	if err := service.requireMutableEnvironment(); err != nil {
		return response.DevelopmentVerificationPrepareRes{}, err
	}
	var accounts []response.DevelopmentVerificationAccountRes
	var err error
	switch scenario {
	case verificationScenarioDataPermission:
		accounts, err = service.prepareDataPermission(ctx)
	case verificationScenarioTMS:
		accounts, err = service.prepareTMS(ctx)
	case verificationScenarioMetadata:
		err = service.prepareMetadata(ctx)
	case verificationScenarioNotification:
		err = service.prepareNotifications(ctx)
	case verificationScenarioIntegration:
		err = service.prepareIntegrationFixture(ctx)
	case verificationScenarioOrganization:
		err = service.prepareOrganizationFixture(ctx)
	case verificationScenarioFileUpload, verificationScenarioVideoPreview:
		err = service.prepareFileFixture(ctx)
	default:
		return response.DevelopmentVerificationPrepareRes{}, myerrors.NewParameterError("不支持的功能验证场景")
	}
	if err != nil {
		return response.DevelopmentVerificationPrepareRes{}, err
	}
	status, err := service.status(ctx, scenario)
	if err != nil {
		return response.DevelopmentVerificationPrepareRes{}, err
	}
	return response.DevelopmentVerificationPrepareRes{Status: status, Accounts: accounts}, nil
}

// Cleanup 只删除本页面创建且带 verify 标识的数据。
func (service *DevelopmentVerificationService) Cleanup(
	ctx context.Context,
	scenario string,
) (response.DevelopmentVerificationStatusRes, error) {
	if err := service.requireMutableEnvironment(); err != nil {
		return response.DevelopmentVerificationStatusRes{}, err
	}
	var err error
	switch scenario {
	case verificationScenarioDataPermission:
		err = service.cleanupDataPermission(ctx)
	case verificationScenarioTMS:
		err = service.cleanupTMS(ctx)
	case verificationScenarioMetadata:
		err = service.cleanupMetadata(ctx)
	case verificationScenarioNotification:
		err = service.cleanupNotifications(ctx)
	case verificationScenarioIntegration:
		err = service.cleanupIntegrationFixture(ctx)
	case verificationScenarioOrganization:
		err = service.cleanupOrganizationFixture(ctx)
	case verificationScenarioFileUpload, verificationScenarioVideoPreview:
		err = service.cleanupFileFixture(ctx)
	default:
		return response.DevelopmentVerificationStatusRes{}, myerrors.NewParameterError("不支持的功能验证场景")
	}
	if err != nil {
		return response.DevelopmentVerificationStatusRes{}, err
	}
	return service.status(ctx, scenario)
}

func (service *DevelopmentVerificationService) requireMutableEnvironment() error {
	environment := strings.ToLower(strings.TrimSpace(service.config.Environment))
	switch environment {
	case "pro", "prod", "production":
		return myerrors.NewValidationError("生产环境禁止创建或清理功能验证样例")
	default:
		return nil
	}
}

func (service *DevelopmentVerificationService) status(
	ctx context.Context,
	scenario string,
) (response.DevelopmentVerificationStatusRes, error) {
	available := service.requireMutableEnvironment() == nil
	if !available {
		return response.DevelopmentVerificationStatusRes{
			ScenarioId: scenario, State: "unavailable", Available: false,
			Summary: "生产环境不提供样例数据操作",
			Details: []response.DevelopmentVerificationDetailRes{{Label: "环境", Value: service.config.Environment}},
		}, nil
	}
	switch scenario {
	case verificationScenarioDataPermission:
		return service.dataPermissionStatus(ctx)
	case verificationScenarioTMS:
		return service.tmsStatus(ctx)
	case verificationScenarioMetadata:
		return service.metadataStatus(ctx)
	case verificationScenarioNotification:
		return service.notificationStatus(ctx)
	case verificationScenarioIntegration:
		return service.integrationFixtureStatus(ctx)
	case verificationScenarioOrganization:
		return service.organizationFixtureStatus(ctx)
	case verificationScenarioFileUpload:
		return service.fileFixtureStatus(ctx, verificationScenarioFileUpload)
	case verificationScenarioVideoPreview:
		return service.fileFixtureStatus(ctx, verificationScenarioVideoPreview)
	default:
		return response.DevelopmentVerificationStatusRes{}, myerrors.NewParameterError("不支持的功能验证场景")
	}
}

func (service *DevelopmentVerificationService) dataPermissionStatus(
	ctx context.Context,
) (response.DevelopmentVerificationStatusRes, error) {
	rows := service.countPhysicalRows(ctx, verificationPermissionTable, "")
	users := service.countUsers(ctx, []string{"verify_permission_east", "verify_permission_all"})
	state := verificationState(rows > 0 && users == 2, rows > 0 || users > 0)
	return response.DevelopmentVerificationStatusRes{
		ScenarioId: verificationScenarioDataPermission, State: state, Available: true, ItemCount: rows,
		Summary: verificationSummary(state, "东西区订单和两个数据范围账号已准备", "尚未准备数据权限样例"),
		Details: []response.DevelopmentVerificationDetailRes{
			{Label: "样例表", Value: verificationPermissionTable},
			{Label: "测试账号", Value: fmt.Sprintf("%d / 2", users)},
			{Label: "预期", Value: "华东账号只见 EAST；全部账号可见全部订单"},
		},
	}, nil
}

func (service *DevelopmentVerificationService) tmsStatus(
	ctx context.Context,
) (response.DevelopmentVerificationStatusRes, error) {
	companies := service.countPhysicalRows(ctx, "tms_company", "company_name LIKE '功能验证-%'")
	vehicles := service.countPhysicalRows(ctx, "tms_vehicle", "driver_name LIKE '功能验证-%'")
	users := service.countUsers(ctx, []string{"verify_tms_east", "verify_tms_multi"})
	state := verificationState(companies >= 3 && vehicles >= 4 && users == 2, companies+vehicles+users > 0)
	return response.DevelopmentVerificationStatusRes{
		ScenarioId: verificationScenarioTMS, State: state, Available: true, ItemCount: vehicles,
		Summary: verificationSummary(state, "三家公司、四辆车和两个公司范围账号已准备", "尚未准备 TMS 公司范围样例"),
		Details: []response.DevelopmentVerificationDetailRes{
			{Label: "公司", Value: strconv.Itoa(companies)},
			{Label: "车辆", Value: strconv.Itoa(vehicles)},
			{Label: "预期", Value: "华东账号只见华东；多公司账号可见华东和华南"},
		},
	}, nil
}

func (service *DevelopmentVerificationService) metadataStatus(
	ctx context.Context,
) (response.DevelopmentVerificationStatusRes, error) {
	records := service.countPhysicalRows(ctx, verificationRecordTable, "")
	var tables int64
	_ = service.db.WithContext(ctx).Model(&model.SysTable{}).
		Where("table_code IN ?", []string{verificationCategoryTable, verificationRecordTable}).Count(&tables).Error
	state := verificationState(records > 0 && tables == 2, records > 0 || tables > 0)
	return response.DevelopmentVerificationStatusRes{
		ScenarioId: verificationScenarioMetadata, State: state, Available: true, ItemCount: records,
		Summary: verificationSummary(state, "包含字典、Relation、日期和布尔字段的低代码样例已发布", "尚未准备元数据样例"),
		Details: []response.DevelopmentVerificationDetailRes{
			{Label: "主表", Value: verificationRecordTable},
			{Label: "关联表", Value: verificationCategoryTable},
			{Label: "样例记录", Value: strconv.Itoa(records)},
		},
	}, nil
}

func (service *DevelopmentVerificationService) notificationStatus(
	ctx context.Context,
) (response.DevelopmentVerificationStatusRes, error) {
	subject, ok := audit.GetAuditSubject(ctx)
	if !ok {
		return response.DevelopmentVerificationStatusRes{}, myerrors.ErrUserNotLogin
	}
	var count int64
	err := service.db.WithContext(ctx).Table("notification_recipient AS recipient").
		Joins("JOIN notification AS notification ON notification.id = recipient.notification_id").
		Where("recipient.user_id = ? AND notification.source_module = ?", subject.UserID, verificationSource).
		Count(&count).Error
	if err != nil {
		return response.DevelopmentVerificationStatusRes{}, myerrors.WrapDatabaseError(err)
	}
	state := verificationState(count >= 3, count > 0)
	return response.DevelopmentVerificationStatusRes{
		ScenarioId: verificationScenarioNotification, State: state, Available: true, ItemCount: int(count),
		Summary: verificationSummary(state, "当前账号的三类通知样例已准备", "尚未给当前账号发送通知样例"),
		Details: []response.DevelopmentVerificationDetailRes{
			{Label: "收件人", Value: subject.UserName},
			{Label: "通知数", Value: strconv.FormatInt(count, 10)},
			{Label: "验证点", Value: "未读数、筛选、详情和全部已读"},
		},
	}, nil
}

func verificationState(ready, partial bool) string {
	if ready {
		return "ready"
	}
	if partial {
		return "partial"
	}
	return "empty"
}

func verificationSummary(state, ready, empty string) string {
	if state == "ready" {
		return ready
	}
	if state == "partial" {
		return "样例只准备了一部分，可以重新准备以补齐"
	}
	return empty
}

func (service *DevelopmentVerificationService) countUsers(ctx context.Context, names []string) int {
	var users []model.SysUser
	if err := service.db.WithContext(ctx).Where("user_name IN ?", names).Find(&users).Error; err != nil {
		return 0
	}
	count := 0
	for _, user := range users {
		if user.Email == user.UserName+"@example.invalid" {
			count++
		}
	}
	return count
}

func (service *DevelopmentVerificationService) countPhysicalRows(ctx context.Context, table, where string) int {
	if !service.db.Migrator().HasTable(table) {
		return 0
	}
	query := service.db.WithContext(ctx).Table(table)
	if where != "" {
		query = query.Where(where)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0
	}
	return int(count)
}

func (service *DevelopmentVerificationService) prepareDataPermission(
	ctx context.Context,
) ([]response.DevelopmentVerificationAccountRes, error) {
	ddl := `CREATE TABLE IF NOT EXISTS verify_permission_order (
		id bigint PRIMARY KEY, gmt_create timestamptz NOT NULL, gmt_create_user bigint NOT NULL,
		gmt_modify timestamptz NOT NULL, gmt_modify_user bigint NOT NULL, gmt_delete timestamptz NULL,
		gmt_delete_user bigint NULL, state boolean NOT NULL DEFAULT true,
		order_no varchar(64) NOT NULL, scope_code varchar(32) NOT NULL, amount numeric(18,2) NOT NULL)`
	table, err := service.ensureMetadataTable(ctx, verificationPermissionTable, "功能验证-数据权限订单", ddl)
	if err != nil {
		return nil, err
	}
	if err = service.configurePermissionFields(ctx, table); err != nil {
		return nil, err
	}
	if err = service.replacePermissionRows(ctx); err != nil {
		return nil, err
	}
	menu, err := service.publishTable(ctx, verificationPermissionTable)
	if err != nil {
		return nil, err
	}
	eastRole, err := service.ensureRole(ctx, "verify_permission_east_role", "功能验证-仅华东订单", menu)
	if err != nil {
		return nil, err
	}
	allRole, err := service.ensureRole(ctx, "verify_permission_all_role", "功能验证-全部订单", menu)
	if err != nil {
		return nil, err
	}
	if err = service.replaceDataPermissionConfig(ctx, permissionFixtureConfig{
		Table: table, FieldCode: "scope_code", ResourceCode: "verify_permission_order",
		DimensionCode: "verify_order_scope", OwnershipCode: "scope",
		Policies: []permissionFixturePolicy{
			{Code: "verify_permission_east_policy", Name: "功能验证-华东订单", RoleID: eastRole.Id, Values: []any{"EAST"}},
			{Code: "verify_permission_all_policy", Name: "功能验证-全部订单", RoleID: allRole.Id, All: true},
		},
	}); err != nil {
		return nil, err
	}
	east, err := service.ensureAccount(ctx, accountFixture{
		UserName: "verify_permission_east", Role: eastRole, Expected: "只能看到 EAST 的两条订单",
	})
	if err != nil {
		return nil, err
	}
	all, err := service.ensureAccount(ctx, accountFixture{
		UserName: "verify_permission_all", Role: allRole, Expected: "可以看到全部三条订单",
	})
	if err != nil {
		return nil, err
	}
	return []response.DevelopmentVerificationAccountRes{east, all}, nil
}

func (service *DevelopmentVerificationService) replacePermissionRows(ctx context.Context) error {
	if err := service.db.WithContext(ctx).Exec("DELETE FROM verify_permission_order").Error; err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	for _, item := range []struct {
		order  string
		scope  string
		amount string
	}{{"VERIFY-EAST-001", "EAST", "1200.00"}, {"VERIFY-EAST-002", "EAST", "2600.00"}, {"VERIFY-WEST-001", "WEST", "1800.00"}} {
		id, err := service.nextID()
		if err != nil {
			return err
		}
		err = service.db.WithContext(ctx).Exec(
			`INSERT INTO verify_permission_order
			(id,gmt_create,gmt_create_user,gmt_modify,gmt_modify_user,state,order_no,scope_code,amount)
			VALUES (?,CURRENT_TIMESTAMP,0,CURRENT_TIMESTAMP,0,true,?,?,?)`, id, item.order, item.scope, item.amount,
		).Error
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
	}
	return nil
}

func (service *DevelopmentVerificationService) prepareTMS(
	ctx context.Context,
) ([]response.DevelopmentVerificationAccountRes, error) {
	companyTable, err := service.requireMetadataTable(ctx, "tms_company", "请先在数据管理初始化 tms_company 元数据")
	if err != nil {
		return nil, err
	}
	vehicleTable, err := service.requireMetadataTable(ctx, "tms_vehicle", "请先在数据管理初始化 tms_vehicle 元数据")
	if err != nil {
		return nil, err
	}
	_ = companyTable
	if err = service.replaceTMSRows(ctx); err != nil {
		return nil, err
	}
	menu, err := service.publishedMenu(ctx, "tms_vehicle")
	if err != nil {
		return nil, myerrors.NewValidationError("请先在数据管理发布 tms_vehicle 低代码页面")
	}
	eastRole, err := service.ensureRole(ctx, "verify_tms_east_role", "功能验证-TMS华东公司", menu)
	if err != nil {
		return nil, err
	}
	multiRole, err := service.ensureRole(ctx, "verify_tms_multi_role", "功能验证-TMS多公司", menu)
	if err != nil {
		return nil, err
	}
	companyIDs, err := service.verificationCompanyIDs(ctx)
	if err != nil {
		return nil, err
	}
	if err = service.replaceDataPermissionConfig(ctx, permissionFixtureConfig{
		Table: vehicleTable, FieldCode: "company_id", ResourceCode: "verify_tms_vehicle",
		DimensionCode: "verify_tms_company", OwnershipCode: "company",
		Policies: []permissionFixturePolicy{
			{Code: "verify_tms_east_policy", Name: "功能验证-TMS华东", RoleID: eastRole.Id, Values: []any{companyIDs["EAST"]}},
			{Code: "verify_tms_multi_policy", Name: "功能验证-TMS多公司", RoleID: multiRole.Id, Values: []any{companyIDs["EAST"], companyIDs["SOUTH"]}},
		},
	}); err != nil {
		return nil, err
	}
	east, err := service.ensureAccount(ctx, accountFixture{
		UserName: "verify_tms_east", Role: eastRole, Expected: "只看到功能验证-华东公司的两辆车",
	})
	if err != nil {
		return nil, err
	}
	multi, err := service.ensureAccount(ctx, accountFixture{
		UserName: "verify_tms_multi", Role: multiRole, Expected: "看到华东和华南共三辆车，不包含华西",
	})
	if err != nil {
		return nil, err
	}
	return []response.DevelopmentVerificationAccountRes{east, multi}, nil
}

func (service *DevelopmentVerificationService) replaceTMSRows(ctx context.Context) error {
	if !service.db.Migrator().HasTable("tms_company") || !service.db.Migrator().HasTable("tms_vehicle") {
		return myerrors.NewValidationError("缺少 tms_company 或 tms_vehicle 样例表")
	}
	companyIDs, _ := service.verificationCompanyIDs(ctx)
	if len(companyIDs) > 0 {
		ids := make([]int, 0, len(companyIDs))
		for _, id := range companyIDs {
			ids = append(ids, id)
		}
		if err := service.db.WithContext(ctx).Exec("DELETE FROM tms_vehicle WHERE company_id IN ?", ids).Error; err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if err := service.db.WithContext(ctx).Exec("DELETE FROM tms_company WHERE id IN ?", ids).Error; err != nil {
			return myerrors.WrapDatabaseError(err)
		}
	}
	created := make(map[string]int, 3)
	for _, item := range []struct{ code, name string }{
		{"EAST", "功能验证-华东公司"}, {"SOUTH", "功能验证-华南公司"}, {"WEST", "功能验证-华西公司"},
	} {
		id, err := service.nextID()
		if err != nil {
			return err
		}
		created[item.code] = id
		if err = service.db.WithContext(ctx).Exec(
			`INSERT INTO tms_company
			(id,gmt_create,gmt_create_user,gmt_modify,gmt_modify_user,state,company_name,parent_id)
			VALUES (?,CURRENT_TIMESTAMP,0,CURRENT_TIMESTAMP,0,true,?,NULL)`, id, item.name,
		).Error; err != nil {
			return myerrors.WrapDatabaseError(err)
		}
	}
	vehicles := []struct {
		plate, company, driver string
	}{{"VERIFY-E-001", "EAST", "功能验证-华东司机甲"}, {"VERIFY-E-002", "EAST", "功能验证-华东司机乙"}, {"VERIFY-S-001", "SOUTH", "功能验证-华南司机"}, {"VERIFY-W-001", "WEST", "功能验证-华西司机"}}
	for _, item := range vehicles {
		id, err := service.nextID()
		if err != nil {
			return err
		}
		if err = service.db.WithContext(ctx).Exec(
			`INSERT INTO tms_vehicle
			(id,gmt_create,gmt_create_user,gmt_modify,gmt_modify_user,state,plate_no,company_id,driver_name)
			VALUES (?,CURRENT_TIMESTAMP,0,CURRENT_TIMESTAMP,0,true,?,?,?)`, id, item.plate, created[item.company], item.driver,
		).Error; err != nil {
			return myerrors.WrapDatabaseError(err)
		}
	}
	return nil
}

func (service *DevelopmentVerificationService) verificationCompanyIDs(ctx context.Context) (map[string]int, error) {
	result := map[string]int{}
	if !service.db.Migrator().HasTable("tms_company") {
		return result, nil
	}
	var rows []struct {
		Id   int
		Name string `gorm:"column:company_name"`
	}
	if err := service.db.WithContext(ctx).Table("tms_company").Select("id, company_name").
		Where("company_name LIKE ?", "功能验证-%").Scan(&rows).Error; err != nil {
		return nil, myerrors.WrapDatabaseError(err)
	}
	for _, row := range rows {
		switch row.Name {
		case "功能验证-华东公司":
			result["EAST"] = row.Id
		case "功能验证-华南公司":
			result["SOUTH"] = row.Id
		case "功能验证-华西公司":
			result["WEST"] = row.Id
		}
	}
	return result, nil
}

func (service *DevelopmentVerificationService) prepareMetadata(ctx context.Context) error {
	categoryDDL := `CREATE TABLE IF NOT EXISTS verify_lowcode_category (
		id bigint PRIMARY KEY, gmt_create timestamptz NOT NULL, gmt_create_user bigint NOT NULL,
		gmt_modify timestamptz NOT NULL, gmt_modify_user bigint NOT NULL, gmt_delete timestamptz NULL,
		gmt_delete_user bigint NULL, state boolean NOT NULL DEFAULT true, category_name varchar(64) NOT NULL)`
	category, err := service.ensureMetadataTable(ctx, verificationCategoryTable, "功能验证-低代码分类", categoryDDL)
	if err != nil {
		return err
	}
	if err = service.configureCategoryFields(ctx, category); err != nil {
		return err
	}
	recordDDL := `CREATE TABLE IF NOT EXISTS verify_lowcode_record (
		id bigint PRIMARY KEY, gmt_create timestamptz NOT NULL, gmt_create_user bigint NOT NULL,
		gmt_modify timestamptz NOT NULL, gmt_modify_user bigint NOT NULL, gmt_delete timestamptz NULL,
		gmt_delete_user bigint NULL, state boolean NOT NULL DEFAULT true, record_name varchar(128) NOT NULL,
		quantity bigint NOT NULL, occurred_on date NOT NULL, enabled boolean NOT NULL,
		status_code varchar(32) NOT NULL, category_id bigint NOT NULL)`
	record, err := service.ensureMetadataTable(ctx, verificationRecordTable, "功能验证-低代码记录", recordDDL)
	if err != nil {
		return err
	}
	if err = service.ensureVerificationDict(ctx); err != nil {
		return err
	}
	if err = service.configureRecordFields(ctx, record); err != nil {
		return err
	}
	if err = service.replaceMetadataRows(ctx); err != nil {
		return err
	}
	_, err = service.publishTable(ctx, verificationRecordTable)
	return err
}

func (service *DevelopmentVerificationService) replaceMetadataRows(ctx context.Context) error {
	if err := service.db.WithContext(ctx).Exec("DELETE FROM verify_lowcode_record").Error; err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	if err := service.db.WithContext(ctx).Exec("DELETE FROM verify_lowcode_category").Error; err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	categories := make([]int, 0, 2)
	for _, name := range []string{"功能验证-运输", "功能验证-仓储"} {
		id, err := service.nextID()
		if err != nil {
			return err
		}
		categories = append(categories, id)
		if err = service.db.WithContext(ctx).Exec(
			`INSERT INTO verify_lowcode_category
			(id,gmt_create,gmt_create_user,gmt_modify,gmt_modify_user,state,category_name)
			VALUES (?,CURRENT_TIMESTAMP,0,CURRENT_TIMESTAMP,0,true,?)`, id, name,
		).Error; err != nil {
			return myerrors.WrapDatabaseError(err)
		}
	}
	for index, item := range []struct {
		name, date, status string
		quantity           int
		enabled            bool
	}{{"功能验证-文本和字典", "2026-08-01", "active", 12, true}, {"功能验证-日期和关联", "2026-08-15", "paused", 7, false}} {
		id, err := service.nextID()
		if err != nil {
			return err
		}
		if err = service.db.WithContext(ctx).Exec(
			`INSERT INTO verify_lowcode_record
			(id,gmt_create,gmt_create_user,gmt_modify,gmt_modify_user,state,record_name,quantity,occurred_on,enabled,status_code,category_id)
			VALUES (?,CURRENT_TIMESTAMP,0,CURRENT_TIMESTAMP,0,true,?,?,?,?,?,?)`,
			id, item.name, item.quantity, item.date, item.enabled, item.status, categories[index],
		).Error; err != nil {
			return myerrors.WrapDatabaseError(err)
		}
	}
	return nil
}

func (service *DevelopmentVerificationService) prepareNotifications(ctx context.Context) error {
	subject, ok := audit.GetAuditSubject(ctx)
	if !ok {
		return myerrors.ErrUserNotLogin
	}
	commands := []NotificationCommand{
		{Category: model.NotificationCategorySystem, Level: model.NotificationLevelInfo, Title: "功能验证：系统通知", Content: "用于检查未读数、消息详情和已读状态。", SourceId: "system"},
		{Category: model.NotificationCategoryBusiness, Level: model.NotificationLevelSuccess, Title: "功能验证：业务完成", Content: "用于检查分类筛选和成功级别样式。", SourceId: "business"},
		{Category: model.NotificationCategoryReminder, Level: model.NotificationLevelWarning, Title: "功能验证：待处理提醒", Content: "用于检查提醒分类、批量已读和用户隔离。", SourceId: "reminder"},
	}
	for index := range commands {
		commands[index].Recipients = []int{subject.UserID}
		commands[index].SourceModule = verificationSource
		commands[index].SourceType = "sample"
		commands[index].DedupKey = fmt.Sprintf("%s-%d-%s", verificationSource, subject.UserID, commands[index].SourceId)
		if _, err := service.notifications.Send(ctx, commands[index]); err != nil {
			return err
		}
	}
	return nil
}

type permissionFixtureConfig struct {
	Table         model.SysTable
	FieldCode     string
	ResourceCode  string
	DimensionCode string
	OwnershipCode string
	Policies      []permissionFixturePolicy
}

type permissionFixturePolicy struct {
	Code   string
	Name   string
	RoleID int
	Values []any
	All    bool
}

func (service *DevelopmentVerificationService) replaceDataPermissionConfig(
	ctx context.Context,
	fixture permissionFixtureConfig,
) error {
	if err := service.deletePermissionConfig(ctx, fixture.ResourceCode, fixture.DimensionCode, policyCodes(fixture.Policies)); err != nil {
		return err
	}
	field, err := service.metadataField(ctx, fixture.Table.Id, fixture.FieldCode)
	if err != nil {
		return err
	}
	dimensionID, err := service.nextID()
	if err != nil {
		return err
	}
	valueType := model.DataDimensionValueTypeString
	if field.FieldType == enum.BigIntFieldType || field.FieldType == enum.IntFieldType {
		valueType = model.DataDimensionValueTypeBigint
	}
	dimension := model.DataDimensionDefinition{
		Basic: model.Basic{Id: dimensionID, State: true}, Code: fixture.DimensionCode,
		Name: "功能验证-" + fixture.OwnershipCode, Category: model.DataDimensionCategoryBusiness,
		ValueType: valueType, ProviderCode: "specified_values", Description: "功能验证页面创建，可安全清理",
	}
	if err = service.db.WithContext(ctx).Create(&dimension).Error; err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	resourceID, err := service.nextID()
	if err != nil {
		return err
	}
	resource := model.DataResource{
		Basic: model.Basic{Id: resourceID, State: true}, ResourceCode: fixture.ResourceCode,
		Name: "功能验证-" + fixture.Table.TableName, ResourceType: model.DataResourceTypeLowCodeTable,
		TableId: &fixture.Table.Id, AdapterCode: "metadata_filter", PermissionEnabled: true,
		Description: "功能验证页面创建，可安全清理",
	}
	if err = service.db.WithContext(ctx).Create(&resource).Error; err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	ownershipID, err := service.nextID()
	if err != nil {
		return err
	}
	ownership := model.DataOwnershipField{
		Basic: model.Basic{Id: ownershipID, State: true}, ResourceId: resource.Id,
		OwnershipCode: fixture.OwnershipCode, DimensionId: dimension.Id,
		BindingType: model.DataOwnershipBindingTypeMetadataField, TableFieldId: &field.Id,
		ValueType: valueType, Description: "功能验证字段归属",
	}
	if err = service.db.WithContext(ctx).Create(&ownership).Error; err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	for _, operation := range []string{model.DataPermissionOperationQuery, model.DataPermissionOperationDetail} {
		operationID, nextErr := service.nextID()
		if nextErr != nil {
			return nextErr
		}
		value := model.DataResourceOperation{
			Basic: model.Basic{Id: operationID, State: true}, ResourceId: resource.Id,
			Operation: operation, PermissionEnabled: true, Description: "功能验证只读操作",
		}
		if err = service.db.WithContext(ctx).Create(&value).Error; err != nil {
			return myerrors.WrapDatabaseError(err)
		}
	}
	for _, policyFixture := range fixture.Policies {
		policyID, nextErr := service.nextID()
		if nextErr != nil {
			return nextErr
		}
		policyType := model.DataPolicyTypeRuleSet
		if policyFixture.All {
			policyType = model.DataPolicyTypeAll
		}
		policy := model.DataPolicy{
			Basic: model.Basic{Id: policyID, State: true}, Code: policyFixture.Code,
			Name: policyFixture.Name, PolicyType: policyType, Description: "功能验证页面创建，可安全清理",
		}
		if err = service.db.WithContext(ctx).Create(&policy).Error; err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if !policyFixture.All {
			ruleID, nextErr := service.nextID()
			if nextErr != nil {
				return nextErr
			}
			encoded, _ := json.Marshal(policyFixture.Values)
			rule := model.DataPolicyRule{
				Basic: model.Basic{Id: ruleID, State: true}, PolicyId: policy.Id, Sequence: 1,
				DimensionId: dimension.Id, OwnershipCode: fixture.OwnershipCode,
				ScopeSource: model.DataPolicyScopeSourceSpecifiedValues, Relation: model.DataPolicyRelationExact,
				Operator: model.DataPolicyOperatorIn, SpecifiedValues: datatypes.JSON(encoded),
			}
			if err = service.db.WithContext(ctx).Create(&rule).Error; err != nil {
				return myerrors.WrapDatabaseError(err)
			}
		}
		for _, operation := range []string{model.DataPermissionOperationQuery, model.DataPermissionOperationDetail} {
			grantID, nextErr := service.nextID()
			if nextErr != nil {
				return nextErr
			}
			grant := model.DataGrant{
				Basic: model.Basic{Id: grantID, State: true}, SubjectType: model.DataGrantSubjectTypeRole,
				SubjectId: policyFixture.RoleID, ResourceId: resource.Id, Operation: operation,
				PolicyId: policy.Id, Description: "功能验证角色授权",
			}
			if err = service.db.WithContext(ctx).Create(&grant).Error; err != nil {
				return myerrors.WrapDatabaseError(err)
			}
		}
	}
	return nil
}

func policyCodes(policies []permissionFixturePolicy) []string {
	result := make([]string, 0, len(policies))
	for _, policy := range policies {
		result = append(result, policy.Code)
	}
	return result
}

type accountFixture struct {
	UserName string
	Role     model.SysRole
	Expected string
}

func (service *DevelopmentVerificationService) ensureAccount(
	ctx context.Context,
	fixture accountFixture,
) (response.DevelopmentVerificationAccountRes, error) {
	var configure model.SysConfigure
	_ = service.db.WithContext(ctx).First(&configure).Error
	password, err := security.GeneratePasswordByConfigure(configure)
	if err != nil {
		return response.DevelopmentVerificationAccountRes{}, err
	}
	var user model.SysUser
	err = service.db.WithContext(ctx).Where("user_name = ?", fixture.UserName).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		id, nextErr := service.nextID()
		if nextErr != nil {
			return response.DevelopmentVerificationAccountRes{}, nextErr
		}
		user = model.SysUser{
			Basic: model.Basic{Id: id, State: true}, UserName: fixture.UserName,
			Email: fixture.UserName + "@example.invalid", Language: "zh-CN",
		}
	} else if err != nil {
		return response.DevelopmentVerificationAccountRes{}, myerrors.WrapDatabaseError(err)
	} else if user.Email != fixture.UserName+"@example.invalid" {
		return response.DevelopmentVerificationAccountRes{}, myerrors.NewValidationError("已存在同名非样例账号，不能由功能验证页面接管")
	}
	now := model.Now()
	customNow := model.CustomTime(now)
	user.State = true
	user.Password = utils.Encryption(password, strconv.Itoa(user.Id)+service.config.Conf.Salt)
	user.PasswordChangedAt = &customNow
	user.IsReset = false
	user.AccessTokens = ""
	if err = service.db.WithContext(ctx).Save(&user).Error; err != nil {
		return response.DevelopmentVerificationAccountRes{}, myerrors.WrapDatabaseError(err)
	}
	if err = service.users.AssignRoles(ctx, user.Id, []int{fixture.Role.Id}); err != nil {
		return response.DevelopmentVerificationAccountRes{}, err
	}
	if err = service.ensureEmployee(ctx, user); err != nil {
		return response.DevelopmentVerificationAccountRes{}, err
	}
	service.users.RefreshCache(user.Id)
	return response.DevelopmentVerificationAccountRes{
		UserName: user.UserName, Password: password, Role: fixture.Role.Memo, Expected: fixture.Expected,
	}, nil
}

func (service *DevelopmentVerificationService) ensureEmployee(ctx context.Context, user model.SysUser) error {
	var employee model.OrgEmployee
	err := service.db.WithContext(ctx).Where("source_system_code = ? AND source_id = ?", verificationSource, user.UserName).
		First(&employee).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		id, nextErr := service.nextID()
		if nextErr != nil {
			return nextErr
		}
		employee = model.OrgEmployee{
			Basic: model.Basic{Id: id, State: true}, SourceSystemCode: verificationSource,
			SourceId: user.UserName, SourceCode: strings.ToUpper(user.UserName),
			EmployeeNo: strings.ToUpper(user.UserName), Name: "功能验证-" + user.UserName,
			Email: user.Email, EmploymentStatus: "active", UserId: &user.Id,
			SourceVersion: "1", SyncStatus: "success", LocalNote: "功能验证页面创建，可安全清理",
		}
		if err = service.db.WithContext(ctx).Create(&employee).Error; err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		return nil
	}
	if err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	return service.db.WithContext(ctx).Model(&employee).Updates(map[string]any{
		"state": true, "employment_status": "active", "user_id": user.Id, "source_deleted": false,
	}).Error
}

func (service *DevelopmentVerificationService) ensureRole(
	ctx context.Context,
	name, memo string,
	menu model.SysMenu,
) (model.SysRole, error) {
	var role model.SysRole
	err := service.db.WithContext(ctx).Where("name = ?", name).First(&role).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		id, nextErr := service.nextID()
		if nextErr != nil {
			return model.SysRole{}, nextErr
		}
		role = model.SysRole{Basic: model.Basic{Id: id, State: true}, Name: name, Memo: memo}
		if err = service.db.WithContext(ctx).Create(&role).Error; err != nil {
			return model.SysRole{}, myerrors.WrapDatabaseError(err)
		}
	} else if err != nil {
		return model.SysRole{}, myerrors.WrapDatabaseError(err)
	} else if !strings.HasPrefix(role.Memo, "功能验证-") {
		return model.SysRole{}, myerrors.NewValidationError("已存在同名非样例角色，不能由功能验证页面接管")
	} else if err = service.db.WithContext(ctx).Model(&role).Updates(map[string]any{"memo": memo, "state": true}).Error; err != nil {
		return model.SysRole{}, myerrors.WrapDatabaseError(err)
	}
	menuIDs, err := service.menuAncestors(ctx, menu)
	if err != nil {
		return model.SysRole{}, err
	}
	var buttons []model.SysMenuButton
	if err = service.db.WithContext(ctx).Where("menu_id = ? AND state = true AND is_disabled = false", menu.Id).
		Where("code IN ?", []string{menu.TableCode + "_query", menu.TableCode + "_detail"}).Find(&buttons).Error; err != nil {
		return model.SysRole{}, myerrors.WrapDatabaseError(err)
	}
	buttonIDs := make([]int, 0, len(buttons))
	for _, button := range buttons {
		buttonIDs = append(buttonIDs, button.Id)
	}
	if err = service.roles.AssignPermissions(ctx, request.RoleAssignPermissionsReq{
		RoleId: role.Id, MenuIds: menuIDs, ButtonIds: buttonIDs,
	}); err != nil {
		return model.SysRole{}, err
	}
	return role, nil
}

func (service *DevelopmentVerificationService) menuAncestors(ctx context.Context, menu model.SysMenu) ([]int, error) {
	result := []int{menu.Id}
	seen := map[int]bool{menu.Id: true}
	parentID := menu.Pid
	for parentID > 0 && !seen[parentID] {
		seen[parentID] = true
		var parent model.SysMenu
		if err := service.db.WithContext(ctx).First(&parent, parentID).Error; err != nil {
			return nil, myerrors.WrapDatabaseError(err)
		}
		result = append(result, parent.Id)
		parentID = parent.Pid
	}
	return result, nil
}

func (service *DevelopmentVerificationService) ensureMetadataTable(
	ctx context.Context,
	code, name, ddl string,
) (model.SysTable, error) {
	if err := service.db.WithContext(ctx).Exec(ddl).Error; err != nil {
		return model.SysTable{}, myerrors.WrapDatabaseError(err)
	}
	var table model.SysTable
	err := service.db.WithContext(ctx).Where("table_code = ?", code).First(&table).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err = service.tables.InitTable(ctx, code); err != nil {
			return model.SysTable{}, err
		}
		if err = service.db.WithContext(ctx).Where("table_code = ?", code).First(&table).Error; err != nil {
			return model.SysTable{}, myerrors.WrapDatabaseError(err)
		}
	} else if err != nil {
		return model.SysTable{}, myerrors.WrapDatabaseError(err)
	}
	if err = service.db.WithContext(ctx).Model(&model.SysTable{}).Where("id = ?", table.Id).
		Updates(map[string]any{"table_name": name, "state": true}).Error; err != nil {
		return model.SysTable{}, myerrors.WrapDatabaseError(err)
	}
	service.tables.DeleteCache(table.Id)
	table.TableName = name
	return table, nil
}

func (service *DevelopmentVerificationService) requireMetadataTable(
	ctx context.Context,
	code, message string,
) (model.SysTable, error) {
	if !service.db.Migrator().HasTable(code) {
		return model.SysTable{}, myerrors.NewValidationError(message)
	}
	var table model.SysTable
	if err := service.db.WithContext(ctx).Where("table_code = ?", code).First(&table).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.SysTable{}, myerrors.NewValidationError(message)
		}
		return model.SysTable{}, myerrors.WrapDatabaseError(err)
	}
	return table, nil
}

func (service *DevelopmentVerificationService) metadataField(
	ctx context.Context,
	tableID int,
	code string,
) (model.SysTableField, error) {
	var field model.SysTableField
	if err := service.db.WithContext(ctx).Where("table_id = ? AND field_code = ?", tableID, code).First(&field).Error; err != nil {
		return model.SysTableField{}, myerrors.NewValidationError("功能验证字段元数据不存在：" + code)
	}
	return field, nil
}

func (service *DevelopmentVerificationService) configurePermissionFields(ctx context.Context, table model.SysTable) error {
	return service.configureFields(ctx, table, map[string]map[string]any{
		"order_no":   {"field_name": "订单号", "is_quick_search": true, "is_advanced_search": true, "is_sort": true, "is_list_show": true, "is_insert_show": true, "is_update_show": true, "sequence": 9},
		"scope_code": {"field_name": "数据范围", "is_advanced_search": true, "is_sort": true, "is_list_show": true, "is_insert_show": true, "is_update_show": true, "sequence": 10},
		"amount":     {"field_name": "订单金额", "logical_type": enum.LogicalTypeMoney, "display_format": enum.DisplayFormatMoney, "is_advanced_search": true, "is_sort": true, "is_list_show": true, "is_insert_show": true, "is_update_show": true, "sequence": 11},
	})
}

func (service *DevelopmentVerificationService) configureCategoryFields(ctx context.Context, table model.SysTable) error {
	return service.configureFields(ctx, table, map[string]map[string]any{
		"category_name": {"field_name": "分类名称", "is_quick_search": true, "is_advanced_search": true, "is_sort": true, "is_list_show": true, "is_insert_show": true, "is_update_show": true, "sequence": 9},
	})
}

func (service *DevelopmentVerificationService) configureRecordFields(ctx context.Context, table model.SysTable) error {
	dictCode := "verify_record_status"
	linkage := `{"linkage":{"enabled":true,"mode":"relation","tableCode":"verify_lowcode_category","labelKey":"category_name","valueKey":"id","pageSize":200}}`
	return service.configureFields(ctx, table, map[string]map[string]any{
		"record_name": {"field_name": "记录名称", "is_quick_search": true, "is_advanced_search": true, "is_sort": true, "is_list_show": true, "is_insert_show": true, "is_update_show": true, "sequence": 9},
		"quantity":    {"field_name": "数量", "logical_type": enum.LogicalTypeInteger, "display_format": enum.DisplayFormatInteger, "input_type": enum.InputNumberInputType, "is_advanced_search": true, "is_sort": true, "is_list_show": true, "is_insert_show": true, "is_update_show": true, "sequence": 10},
		"occurred_on": {"field_name": "发生日期", "logical_type": enum.LogicalTypeDate, "display_format": enum.DisplayFormatDate, "input_type": enum.DatePickerInputType, "is_advanced_search": true, "is_sort": true, "is_list_show": true, "is_insert_show": true, "is_update_show": true, "sequence": 11},
		"enabled":     {"field_name": "是否启用", "logical_type": enum.LogicalTypeBoolean, "input_type": enum.BooleanInputType, "is_advanced_search": true, "is_list_show": true, "is_insert_show": true, "is_update_show": true, "sequence": 12},
		"status_code": {"field_name": "处理状态", "logical_type": enum.LogicalTypeEnum, "display_format": enum.DisplayFormatDictionary, "input_type": enum.SelectInputType, "dict_code": &dictCode, "is_advanced_search": true, "is_list_show": true, "is_insert_show": true, "is_update_show": true, "sequence": 13},
		"category_id": {"field_name": "所属分类", "logical_type": enum.LogicalTypeRelation, "display_format": enum.DisplayFormatRelation, "input_type": enum.SelectInputType, "linkage_config": &linkage, "is_advanced_search": true, "is_list_show": true, "is_insert_show": true, "is_update_show": true, "sequence": 14},
	})
}

func (service *DevelopmentVerificationService) configureFields(
	ctx context.Context,
	table model.SysTable,
	fields map[string]map[string]any,
) error {
	for code, updates := range fields {
		if err := service.db.WithContext(ctx).Model(&model.SysTableField{}).
			Where("table_id = ? AND field_code = ?", table.Id, code).Updates(updates).Error; err != nil {
			return myerrors.WrapDatabaseError(err)
		}
	}
	if err := service.db.WithContext(ctx).Model(&model.SysTableField{}).
		Where("table_id = ? AND field_code IN ?", table.Id, []string{"id", "gmt_create_user", "gmt_modify_user", "gmt_delete", "gmt_delete_user"}).
		Updates(map[string]any{"is_insert_show": false, "is_update_show": false, "is_list_show": false}).Error; err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	service.tables.DeleteCache(table.Id)
	return nil
}

func (service *DevelopmentVerificationService) ensureVerificationDict(ctx context.Context) error {
	const code = "verify_record_status"
	var dict model.SysDict
	err := service.db.WithContext(ctx).Where("dict_code = ?", code).First(&dict).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		id, nextErr := service.nextID()
		if nextErr != nil {
			return nextErr
		}
		dict = model.SysDict{Basic: model.Basic{Id: id, State: true}, DictName: "功能验证-处理状态", DictCode: code}
		if err = service.db.WithContext(ctx).Create(&dict).Error; err != nil {
			return myerrors.WrapDatabaseError(err)
		}
	} else if err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	if err = service.db.WithContext(ctx).Exec("DELETE FROM sys_dict_item WHERE dict_id = ?", dict.Id).Error; err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	for _, item := range []struct{ name, code, value string }{{"处理中", "verify_status_active", "active"}, {"已暂停", "verify_status_paused", "paused"}} {
		id, nextErr := service.nextID()
		if nextErr != nil {
			return nextErr
		}
		value := model.SysDictItem{Basic: model.Basic{Id: id, State: true}, DictId: dict.Id, ItemName: item.name, ItemCode: item.code, ItemValue: item.value}
		if err = service.db.WithContext(ctx).Create(&value).Error; err != nil {
			return myerrors.WrapDatabaseError(err)
		}
	}
	return nil
}

func (service *DevelopmentVerificationService) publishTable(ctx context.Context, code string) (model.SysMenu, error) {
	if err := service.publication.PublishTableAsMenu(ctx, code, 0); err != nil {
		return model.SysMenu{}, err
	}
	return service.publishedMenu(ctx, code)
}

func (service *DevelopmentVerificationService) publishedMenu(ctx context.Context, code string) (model.SysMenu, error) {
	var menu model.SysMenu
	if err := service.db.WithContext(ctx).Where("table_code = ? AND is_hidden = false AND state = true", code).
		Order("id ASC").First(&menu).Error; err != nil {
		return model.SysMenu{}, err
	}
	return menu, nil
}

func (service *DevelopmentVerificationService) nextID() (int, error) {
	id, err := service.sf.GenerateUniqueID()
	if err != nil {
		return 0, myerrors.WrapSystemError(err)
	}
	return int(id), nil
}

func (service *DevelopmentVerificationService) cleanupDataPermission(ctx context.Context) error {
	if err := service.deletePermissionConfig(ctx, "verify_permission_order", "verify_order_scope", []string{"verify_permission_east_policy", "verify_permission_all_policy"}); err != nil {
		return err
	}
	if err := service.cleanupAccounts(ctx, []string{"verify_permission_east", "verify_permission_all"}, []string{"verify_permission_east_role", "verify_permission_all_role"}); err != nil {
		return err
	}
	return service.dropOwnedMetadataTable(ctx, verificationPermissionTable)
}

func (service *DevelopmentVerificationService) cleanupTMS(ctx context.Context) error {
	if err := service.deletePermissionConfig(ctx, "verify_tms_vehicle", "verify_tms_company", []string{"verify_tms_east_policy", "verify_tms_multi_policy"}); err != nil {
		return err
	}
	if err := service.cleanupAccounts(ctx, []string{"verify_tms_east", "verify_tms_multi"}, []string{"verify_tms_east_role", "verify_tms_multi_role"}); err != nil {
		return err
	}
	companyIDs, err := service.verificationCompanyIDs(ctx)
	if err != nil {
		return err
	}
	ids := make([]int, 0, len(companyIDs))
	for _, id := range companyIDs {
		ids = append(ids, id)
	}
	if len(ids) > 0 {
		if err = service.db.WithContext(ctx).Exec("DELETE FROM tms_vehicle WHERE company_id IN ?", ids).Error; err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if err = service.db.WithContext(ctx).Exec("DELETE FROM tms_company WHERE id IN ?", ids).Error; err != nil {
			return myerrors.WrapDatabaseError(err)
		}
	}
	return nil
}

func (service *DevelopmentVerificationService) cleanupMetadata(ctx context.Context) error {
	if err := service.dropOwnedMetadataTable(ctx, verificationRecordTable); err != nil {
		return err
	}
	if err := service.dropOwnedMetadataTable(ctx, verificationCategoryTable); err != nil {
		return err
	}
	var dict model.SysDict
	if err := service.db.WithContext(ctx).Unscoped().Where("dict_code = ?", "verify_record_status").First(&dict).Error; err == nil {
		if err = service.db.WithContext(ctx).Exec("DELETE FROM sys_dict_item WHERE dict_id = ?", dict.Id).Error; err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if err = service.db.WithContext(ctx).Exec("DELETE FROM sys_dict WHERE id = ?", dict.Id).Error; err != nil {
			return myerrors.WrapDatabaseError(err)
		}
	}
	return nil
}

func (service *DevelopmentVerificationService) cleanupNotifications(ctx context.Context) error {
	subject, ok := audit.GetAuditSubject(ctx)
	if !ok {
		return myerrors.ErrUserNotLogin
	}
	if err := service.db.WithContext(ctx).Exec(`DELETE FROM notification_recipient AS recipient
		USING notification AS notification
		WHERE recipient.notification_id = notification.id AND recipient.user_id = ?
		AND notification.source_module = ?`, subject.UserID, verificationSource).Error; err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	if err := service.db.WithContext(ctx).Exec(`DELETE FROM notification AS notification
		WHERE notification.source_module = ? AND NOT EXISTS
		(SELECT 1 FROM notification_recipient AS recipient WHERE recipient.notification_id = notification.id)`, verificationSource).Error; err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	return nil
}

func (service *DevelopmentVerificationService) deletePermissionConfig(
	ctx context.Context,
	resourceCode, dimensionCode string,
	policyCodeList []string,
) error {
	if !strings.HasPrefix(resourceCode, "verify_") ||
		(dimensionCode != "" && !strings.HasPrefix(dimensionCode, "verify_")) {
		return myerrors.NewValidationError("拒绝删除非功能验证数据权限配置")
	}
	for _, code := range policyCodeList {
		if !strings.HasPrefix(code, "verify_") {
			return myerrors.NewValidationError("拒绝删除非功能验证数据权限配置")
		}
	}
	tx := service.db.WithContext(ctx)
	var resources []model.DataResource
	if err := tx.Unscoped().Where("resource_code = ?", resourceCode).Find(&resources).Error; err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	for _, resource := range resources {
		if err := tx.Exec("DELETE FROM sys_data_grant WHERE resource_id = ?", resource.Id).Error; err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if err := tx.Exec("DELETE FROM sys_data_resource_operation WHERE resource_id = ?", resource.Id).Error; err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if err := tx.Exec("DELETE FROM sys_data_ownership_field WHERE resource_id = ?", resource.Id).Error; err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if err := tx.Exec("DELETE FROM sys_data_resource WHERE id = ?", resource.Id).Error; err != nil {
			return myerrors.WrapDatabaseError(err)
		}
	}
	var policies []model.DataPolicy
	if len(policyCodeList) > 0 {
		if err := tx.Unscoped().Where("code IN ?", policyCodeList).Find(&policies).Error; err != nil {
			return myerrors.WrapDatabaseError(err)
		}
	}
	for _, policy := range policies {
		if err := tx.Exec("DELETE FROM sys_data_grant WHERE policy_id = ?", policy.Id).Error; err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if err := tx.Exec("DELETE FROM sys_data_policy_rule WHERE policy_id = ?", policy.Id).Error; err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if err := tx.Exec("DELETE FROM sys_data_policy WHERE id = ?", policy.Id).Error; err != nil {
			return myerrors.WrapDatabaseError(err)
		}
	}
	if dimensionCode != "" {
		if err := tx.Exec("DELETE FROM sys_data_dimension_definition WHERE code = ?", dimensionCode).Error; err != nil {
			return myerrors.WrapDatabaseError(err)
		}
	}
	return nil
}

func (service *DevelopmentVerificationService) cleanupAccounts(
	ctx context.Context,
	userNames, roleNames []string,
) error {
	tx := service.db.WithContext(ctx)
	var users []model.SysUser
	if err := tx.Unscoped().Where("user_name IN ?", userNames).Find(&users).Error; err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	for _, user := range users {
		if user.Email != user.UserName+"@example.invalid" {
			continue
		}
		service.users.DeleteCache(user.Id)
		if err := tx.Exec("DELETE FROM org_employee WHERE source_system_code = ? AND user_id = ?", verificationSource, user.Id).Error; err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if err := tx.Where("user_id = ?", user.Id).Delete(&model.SysUserRole{}).Error; err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if err := tx.Exec("DELETE FROM sys_user WHERE id = ?", user.Id).Error; err != nil {
			return myerrors.WrapDatabaseError(err)
		}
	}
	var roles []model.SysRole
	if err := tx.Unscoped().Where("name IN ?", roleNames).Find(&roles).Error; err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	for _, role := range roles {
		if !strings.HasPrefix(role.Memo, "功能验证-") {
			continue
		}
		if !role.GmtDelete.Valid {
			if err := service.roles.AssignPermissions(ctx, request.RoleAssignPermissionsReq{RoleId: role.Id}); err != nil {
				return err
			}
		}
		if err := tx.Exec("DELETE FROM sys_role WHERE id = ?", role.Id).Error; err != nil {
			return myerrors.WrapDatabaseError(err)
		}
	}
	return nil
}

func (service *DevelopmentVerificationService) dropOwnedMetadataTable(ctx context.Context, code string) error {
	if !strings.HasPrefix(code, "verify_") {
		return myerrors.NewValidationError("拒绝删除非功能验证表")
	}
	var table model.SysTable
	err := service.db.WithContext(ctx).Unscoped().Where("table_code = ?", code).First(&table).Error
	if err == nil {
		_ = service.publication.UnpublishTableMenu(ctx, code)
		var menus []model.SysMenu
		if queryErr := service.db.WithContext(ctx).Unscoped().Where("table_code = ?", code).Find(&menus).Error; queryErr != nil {
			return myerrors.WrapDatabaseError(queryErr)
		}
		for _, menu := range menus {
			if deleteErr := service.db.WithContext(ctx).Where("menu_id = ?", menu.Id).Delete(&model.SysRoleMenuButton{}).Error; deleteErr != nil {
				return myerrors.WrapDatabaseError(deleteErr)
			}
			if deleteErr := service.db.WithContext(ctx).Where("menu_id = ?", menu.Id).Delete(&model.SysRoleMenu{}).Error; deleteErr != nil {
				return myerrors.WrapDatabaseError(deleteErr)
			}
			if deleteErr := service.db.WithContext(ctx).Exec("DELETE FROM sys_menu_button WHERE menu_id = ?", menu.Id).Error; deleteErr != nil {
				return myerrors.WrapDatabaseError(deleteErr)
			}
			if deleteErr := service.db.WithContext(ctx).Exec("DELETE FROM sys_menu WHERE id = ?", menu.Id).Error; deleteErr != nil {
				return myerrors.WrapDatabaseError(deleteErr)
			}
		}
		if deleteErr := service.db.WithContext(ctx).Exec("DELETE FROM sys_table_index_field WHERE index_id IN (SELECT id FROM sys_table_index WHERE table_id = ?)", table.Id).Error; deleteErr != nil {
			return myerrors.WrapDatabaseError(deleteErr)
		}
		for _, metadataTable := range []string{"sys_table_relation", "sys_table_index", "sys_table_field"} {
			if deleteErr := service.db.WithContext(ctx).Exec("DELETE FROM "+metadataTable+" WHERE table_id = ?", table.Id).Error; deleteErr != nil {
				return myerrors.WrapDatabaseError(deleteErr)
			}
		}
		if deleteErr := service.db.WithContext(ctx).Exec("DELETE FROM sys_table WHERE id = ?", table.Id).Error; deleteErr != nil {
			return myerrors.WrapDatabaseError(deleteErr)
		}
		service.tables.DeleteCache(table.Id)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return myerrors.WrapDatabaseError(err)
	}
	if err = service.db.WithContext(ctx).Exec("DROP TABLE IF EXISTS " + code + " CASCADE").Error; err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	return nil
}
