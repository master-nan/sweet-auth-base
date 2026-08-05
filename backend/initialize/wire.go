//go:build wireinject
// +build wireinject

/**
 * @Author: Nan
 * @Date: 2024/6/1 下午9:54
 */
package initialize

import (
	"backend/api"
	"backend/config"
	"backend/controller"
	"backend/internal/cache"
	"backend/internal/database"
	"backend/internal/datapermission"
	"backend/internal/security"
	"backend/internal/storage"
	"backend/internal/token"
	"backend/internal/utils"
	"backend/repository"
	"backend/repository/impl"
	"backend/service"

	"github.com/casbin/casbin/v2"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type App struct {
	Config                         *config.Server
	DBs                            map[string]*gorm.DB
	Redis                          *redis.Client
	SF                             *utils.Snowflake
	JwtGenerator                   *token.JWTGenerator
	HmacGenerator                  *token.HMACGenerator
	Enforcer                       *casbin.SyncedEnforcer
	DictController                 *controller.DictController
	BasicController                *controller.BasicController
	TableController                *controller.TableController
	MenuController                 *controller.MenuController
	RoleController                 *controller.RoleController
	UserController                 *controller.UserController
	DataPermissionConfigController *controller.DataPermissionConfigController
	ExternalSystemController       *controller.ExternalSystemController
	InterfaceDefinitionController  *controller.InterfaceDefinitionController
	CredentialController           *controller.CredentialController
	ApplicationController          *controller.ApplicationController
	GeneralizationController       *controller.GeneralizationController
	ReportController               *controller.ReportController
	OrgController                  *controller.OrgController
	SmsController                  *controller.SmsController
	FileController                 *controller.FileController
	AuthApi                        *api.AuthApi
	SysUserApi                     *api.SysUserApi
	DingTalkApi                    *api.DingTalkApi
	LogService                     *service.LogService
	UserService                    *service.SysUserService
	ApplicationService             *service.ApplicationService
	SubjectContextBuilder          *service.SubjectContextBuilder
	DimensionProviderRuntime       *service.DimensionProviderRuntime
	DataPermissionResolver         datapermission.Resolver
	BlackCache                     *cache.BlackUserCache
	TokenBlackCache                *cache.TokenBlackCache
	ApplicationCache               *cache.ApplicationCache
}

// Repository 提供者
var RepositoryProvider = wire.NewSet(

	impl.NewAccessLogRepositoryImpl,
	impl.NewLoginLogRepositoryImpl,
	impl.NewSysConfigureRepositoryImpl,
	impl.NewSysDictRepositoryImpl,
	impl.NewSysDictItemRepositoryImpl,
	impl.NewSysTableIndexFieldRepositoryImpl,
	impl.NewSysTableIndexRepositoryImpl,
	impl.NewSysTableRelationRepositoryImpl,
	impl.NewSysTableFieldRepositoryImpl,
	impl.NewSysTableRepositoryImpl,
	impl.NewSysUserRepositoryImpl,
	impl.NewSysMenuRepositoryImpl,
	impl.NewSysMenuButtonRepositoryImpl,
	impl.NewSysMenuButtonTemplateRepositoryImpl,
	impl.NewSysRoleRepositoryImpl,
	impl.NewSysRoleMenuButtonRepositoryImpl,
	impl.NewSysRoleMenuRepositoryImpl,
	impl.NewSysUserRoleRepositoryImpl,
	impl.NewApplicationRepositoryImpl,
	impl.NewExternalSystemRepositoryImpl,
	impl.NewInterfaceDefinitionRepositoryImpl,
	impl.NewCredentialRepositoryImpl,
	impl.NewIntegrationExecutionRepositoryImpl,
	impl.NewIntegrationLogRepositoryImpl,
	impl.NewGeneralizationRepositoryImpl,
	impl.NewReportDefinitionRepositoryImpl,
	impl.NewReportDefinitionVersionRepositoryImpl,
	impl.NewReportExecutionLogRepositoryImpl,
	impl.NewDataDimensionDefinitionRepositoryImpl,
	impl.NewDataResourceRepositoryImpl,
	impl.NewDataResourceOperationRepositoryImpl,
	impl.NewDataOwnershipFieldRepositoryImpl,
	impl.NewDataPolicyRepositoryImpl,
	impl.NewDataPolicyRuleRepositoryImpl,
	impl.NewDataGrantRepositoryImpl,
	impl.NewDataPermissionMetadataReaderImpl,
	impl.NewOrgLegalEntityRepositoryImpl,
	impl.NewOrgUnitRepositoryImpl,
	impl.NewOrgStructureRepositoryImpl,
	impl.NewOrgStructureNodeRepositoryImpl,
	impl.NewOrgEmployeeRepositoryImpl,
	impl.NewOrgPositionRepositoryImpl,
	impl.NewOrgAssignmentRepositoryImpl,
	impl.NewOrgSyncBatchRepositoryImpl,
	impl.NewOrgSyncRecordRepositoryImpl,
	impl.NewCasbinRuleRepositoryImpl,
	impl.NewSmsLogImpl,
	impl.NewSmsTemplateImpl,
	impl.NewFileRepositoryImpl,
	impl.NewFileChunkRepositoryImpl,
	wire.Bind(new(repository.AccessLogRepository), new(*impl.AccessLogRepositoryImpl)),
	wire.Bind(new(repository.LoginLogRepository), new(*impl.LoginLogRepositoryImpl)),
	wire.Bind(new(repository.SysConfigureRepository), new(*impl.SysConfigureRepositoryImpl)),
	wire.Bind(new(repository.SysDictRepository), new(*impl.SysDictRepositoryImpl)),
	wire.Bind(new(repository.SysDictItemRepository), new(*impl.SysDictItemRepositoryImpl)),
	wire.Bind(new(repository.SysTableIndexFieldRepository), new(*impl.SysTableIndexFieldRepositoryImpl)),
	wire.Bind(new(repository.SysTableIndexRepository), new(*impl.SysTableIndexRepositoryImpl)),
	wire.Bind(new(repository.SysTableRelationRepository), new(*impl.SysTableRelationRepositoryImpl)),
	wire.Bind(new(repository.SysTableFieldRepository), new(*impl.SysTableFieldRepositoryImpl)),
	wire.Bind(new(repository.SysTableRepository), new(*impl.SysTableRepositoryImpl)),
	wire.Bind(new(repository.SysUserRepository), new(*impl.SysUserRepositoryImpl)),
	wire.Bind(new(repository.SysMenuRepository), new(*impl.SysMenuRepositoryImpl)),
	wire.Bind(new(repository.SysMenuButtonRepository), new(*impl.SysMenuButtonRepositoryImpl)),
	wire.Bind(new(repository.SysMenuButtonTemplateRepository), new(*impl.SysMenuButtonTemplateRepositoryImpl)),
	wire.Bind(new(repository.SysRoleRepository), new(*impl.SysRoleRepositoryImpl)),
	wire.Bind(new(repository.SysRoleMenuButtonRepository), new(*impl.SysRoleMenuButtonRepositoryImpl)),
	wire.Bind(new(repository.SysRoleMenuRepository), new(*impl.SysRoleMenuRepositoryImpl)),
	wire.Bind(new(repository.SysUserRoleRepository), new(*impl.SysUserRoleRepositoryImpl)),
	wire.Bind(new(repository.ApplicationRepository), new(*impl.ApplicationRepositoryImpl)),
	wire.Bind(new(repository.ExternalSystemRepository), new(*impl.ExternalSystemRepositoryImpl)),
	wire.Bind(new(repository.InterfaceDefinitionRepository), new(*impl.InterfaceDefinitionRepositoryImpl)),
	wire.Bind(new(repository.CredentialRepository), new(*impl.CredentialRepositoryImpl)),
	wire.Bind(new(repository.IntegrationExecutionRepository), new(*impl.IntegrationExecutionRepositoryImpl)),
	wire.Bind(new(repository.IntegrationLogRepository), new(*impl.IntegrationLogRepositoryImpl)),
	wire.Bind(new(repository.GeneralizationRepository), new(*impl.GeneralizationRepositoryImpl)),
	wire.Bind(new(repository.ReportDefinitionRepository), new(*impl.ReportDefinitionRepositoryImpl)),
	wire.Bind(new(repository.ReportDefinitionVersionRepository), new(*impl.ReportDefinitionVersionRepositoryImpl)),
	wire.Bind(new(repository.ReportExecutionLogRepository), new(*impl.ReportExecutionLogRepositoryImpl)),
	wire.Bind(new(repository.DataDimensionDefinitionRepository), new(*impl.DataDimensionDefinitionRepositoryImpl)),
	wire.Bind(new(repository.DataResourceRepository), new(*impl.DataResourceRepositoryImpl)),
	wire.Bind(new(repository.DataResourceOperationRepository), new(*impl.DataResourceOperationRepositoryImpl)),
	wire.Bind(new(repository.DataOwnershipFieldRepository), new(*impl.DataOwnershipFieldRepositoryImpl)),
	wire.Bind(new(repository.DataPolicyRepository), new(*impl.DataPolicyRepositoryImpl)),
	wire.Bind(new(repository.DataPolicyRuleRepository), new(*impl.DataPolicyRuleRepositoryImpl)),
	wire.Bind(new(repository.DataGrantRepository), new(*impl.DataGrantRepositoryImpl)),
	wire.Bind(new(datapermission.MetadataFieldReader), new(*impl.DataPermissionMetadataReaderImpl)),
	wire.Bind(new(repository.OrgLegalEntityRepository), new(*impl.OrgLegalEntityRepositoryImpl)),
	wire.Bind(new(repository.OrgUnitRepository), new(*impl.OrgUnitRepositoryImpl)),
	wire.Bind(new(repository.OrgStructureRepository), new(*impl.OrgStructureRepositoryImpl)),
	wire.Bind(new(repository.OrgStructureNodeRepository), new(*impl.OrgStructureNodeRepositoryImpl)),
	wire.Bind(new(repository.OrgEmployeeRepository), new(*impl.OrgEmployeeRepositoryImpl)),
	wire.Bind(new(repository.OrgPositionRepository), new(*impl.OrgPositionRepositoryImpl)),
	wire.Bind(new(repository.OrgAssignmentRepository), new(*impl.OrgAssignmentRepositoryImpl)),
	wire.Bind(new(repository.OrgSyncBatchRepository), new(*impl.OrgSyncBatchRepositoryImpl)),
	wire.Bind(new(repository.OrgSyncRecordRepository), new(*impl.OrgSyncRecordRepositoryImpl)),
	wire.Bind(new(repository.CasbinRuleRepository), new(*impl.CasbinRuleRepositoryImpl)),
	wire.Bind(new(repository.SmsLogRepository), new(*impl.SmsLogImpl)),
	wire.Bind(new(repository.SmsTemplateRepository), new(*impl.SmsTemplateImpl)),
	wire.Bind(new(repository.FileRepository), new(*impl.FileRepositoryImpl)),
	wire.Bind(new(repository.FileChunkRepository), new(*impl.FileChunkRepositoryImpl)),
)

// Cache 提供者
var CacheProvider = wire.NewSet(
	cache.NewSysConfigureCache,
	cache.NewSysUserRoleCache,
	cache.NewSysUserCache,
	cache.NewSysMenuButtonCache,
	cache.NewSysDictCache,
	cache.NewSysMenuCache,
	cache.NewSysRoleCache,
	cache.NewSysRoleMenuButtonCache,
	cache.NewSysRoleMenuCache,
	cache.NewSysTableCache,
	cache.NewSysTableFieldCache,
	cache.NewGeneralizationCache,
	cache.NewBlackCache,
	cache.NewTokenBlackCache,
	cache.NewLoginAttemptCache,
	cache.NewApplicationCache,
	cache.NewDingTalkCache,
	cache.NewSmsTemplateCache,
	cache.NewSmsLogCache,
	cache.NewSendCodeCache,
	cache.NewDingTalkUserIDCache,
)

// Service 提供者
var ServiceProvider = wire.NewSet(
	service.NewLogServer,
	wire.Bind(new(service.TransactionalAuditWriter), new(*service.LogService)),
	wire.Bind(new(service.StandardContextAuditWriter), new(*service.LogService)),
	service.NewSysConfigureService,
	service.NewSysDictService,
	service.NewSysRoleService,
	service.NewSysMenuService,
	service.NewSysTableService,
	service.NewSysUserService,
	service.NewGeneralizationServiceWithDataPermission,
	service.NewDataResourceConfigService,
	service.NewDataOwnershipConfigService,
	service.NewDataPolicyConfigService,
	service.NewDataGrantConfigService,
	service.NewDataPermissionConfigPreflightService,
	service.NewSubjectContextBuilder,
	service.NewDimensionProviderRuntime,
	service.NewDataPermissionPolicyResolver,
	datapermission.NewMetadataFieldAdapter,
	service.NewLowCodeDataPermissionRuntime,
	wire.Bind(new(service.DimensionProvider), new(*service.DimensionProviderRuntime)),
	wire.Bind(new(datapermission.Resolver), new(*service.DataPermissionPolicyResolver)),
	ProvideOwnershipFieldRegistry,
	wire.Bind(
		new(datapermission.OwnershipFieldBindingValidator),
		new(*datapermission.OwnershipFieldRegistry),
	),
	wire.Bind(
		new(datapermission.OwnershipFieldOperationValidator),
		new(*datapermission.OwnershipFieldRegistry),
	),
	service.NewReportService,
	service.NewOrgService,
	wire.Bind(new(service.OrgPermissionProvider), new(*service.OrgService)),
	service.NewCasbinRuleService,
	service.NewApplicationService,
	service.NewExternalSystemService,
	service.NewInterfaceDefinitionService,
	security.NewCredentialSecretProtector,
	service.NewCredentialService,
	service.NewIntegrationExecutionService,
	service.NewDingTalkService,
	service.NewSmsService,
	service.NewFileService,
)

// Controller 提供者
var ControllerProvider = wire.NewSet(
	controller.NewDictController,
	controller.NewTableController,
	controller.NewMenuController,
	controller.NewRoleController,
	controller.NewUserController,
	controller.NewDataPermissionConfigController,
	controller.NewExternalSystemController,
	controller.NewInterfaceDefinitionController,
	controller.NewCredentialController,
	controller.NewBasicController,
	controller.NewGeneralizationController,
	controller.NewReportController,
	controller.NewOrgController,
	controller.NewApplicationController,
	controller.NewSmsController,
	controller.NewFileController,
)

// API 提供者
var ApiProvider = wire.NewSet(
	api.NewAuthApi,
	api.NewSysUserApi,
	api.NewDingTalkApi,
)

func ProvidePrimaryDB(db map[string]*gorm.DB) *database.PrimaryDB {
	return &database.PrimaryDB{DB: db["primary"]}
}

// ProvideOwnershipFieldRegistry 创建进程内注册表。
// 经过审查的业务模块可在启用其数据资源时添加声明。
func ProvideOwnershipFieldRegistry() (*datapermission.OwnershipFieldRegistry, error) {
	return datapermission.NewOwnershipFieldRegistry()
}

// ProvideJWTToken 提供 JWT 生成器
func ProvideJWTToken() token.JWTToken {
	return token.JWTToken{Generator: token.NewJWTGenerator()}
}

// ProvideHMACToken 提供 HMAC 生成器
func ProvideHMACToken() token.HMACToken {
	return token.HMACToken{Generator: token.NewHMACGenerator()}
}

var Providers = wire.NewSet(
	InitLogger,
	LoadConfig,
	InitDB,
	ProvidePrimaryDB,

	InitRedis,
	InitCasbin,
	wire.Bind(new(repository.CasbinPolicyEnforcer), new(*casbin.SyncedEnforcer)),
	InitSnowflake,
	InitValidators,

	cache.NewRedisUtil,
	wire.Bind(new(cache.Cacher), new(*cache.RedisUtil)),
	ProvideJWTToken,  // 提供 JWTToken 实现
	ProvideHMACToken, // 提供 HMACToken 实现
	token.NewJWTGenerator,
	token.NewHMACGenerator,

	storage.NewStorage,

	RepositoryProvider,
	CacheProvider,
	ServiceProvider,
	ControllerProvider,
	ApiProvider,

	wire.Struct(new(App), "*"),
)

func InitializeApp() (*App, error) {
	wire.Build(Providers)
	return nil, nil
}
