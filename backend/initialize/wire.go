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
	Config                   *config.Server
	DBs                      map[string]*gorm.DB
	Redis                    *redis.Client
	SF                       *utils.Snowflake
	JwtGenerator             *token.JWTGenerator
	HmacGenerator            *token.HMACGenerator
	Enforcer                 *casbin.Enforcer
	DictController           *controller.DictController
	BasicController          *controller.BasicController
	TableController          *controller.TableController
	MenuController           *controller.MenuController
	RoleController           *controller.RoleController
	UserController           *controller.UserController
	DataPermissionController *controller.DataPermissionController
	ApplicationController    *controller.ApplicationController
	GeneralizationController *controller.GeneralizationController
	ReportController         *controller.ReportController
	SmsController            *controller.SmsController
	FileController           *controller.FileController
	AuthApi                  *api.AuthApi
	SysUserApi               *api.SysUserApi
	DingTalkApi              *api.DingTalkApi
	LogService               *service.LogService
	UserService              *service.SysUserService
	ApplicationService       *service.ApplicationService
	BlackCache               *cache.BlackUserCache
	TokenBlackCache          *cache.TokenBlackCache
	ApplicationCache         *cache.ApplicationCache
}

// Repository providers
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
	impl.NewGeneralizationRepositoryImpl,
	impl.NewReportDefinitionRepositoryImpl,
	impl.NewReportDefinitionVersionRepositoryImpl,
	impl.NewReportExecutionLogRepositoryImpl,
	impl.NewCasbinRuleRepositoryImpl,
	impl.NewSmsLogImpl,
	impl.NewSmsTemplateImpl,
	impl.NewFileRepositoryImpl,
	impl.NewFileChunkRepositoryImpl,
	//impl.NewBasicRepositoryImpl,

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
	wire.Bind(new(repository.GeneralizationRepository), new(*impl.GeneralizationRepositoryImpl)),
	wire.Bind(new(repository.ReportDefinitionRepository), new(*impl.ReportDefinitionRepositoryImpl)),
	wire.Bind(new(repository.ReportDefinitionVersionRepository), new(*impl.ReportDefinitionVersionRepositoryImpl)),
	wire.Bind(new(repository.ReportExecutionLogRepository), new(*impl.ReportExecutionLogRepositoryImpl)),
	wire.Bind(new(repository.CasbinRuleRepository), new(*impl.CasbinRuleRepositoryImpl)),
	wire.Bind(new(repository.SmsLogRepository), new(*impl.SmsLogImpl)),
	wire.Bind(new(repository.SmsTemplateRepository), new(*impl.SmsTemplateImpl)),
	wire.Bind(new(repository.FileRepository), new(*impl.FileRepositoryImpl)),
	wire.Bind(new(repository.FileChunkRepository), new(*impl.FileChunkRepositoryImpl)),
	//wire.Bind(new(repository.BasicRepository), new(*impl.BasicRepositoryImpl[T])),
)

// Cache providers
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

// Service providers
var ServiceProvider = wire.NewSet(
	service.NewLogServer,
	service.NewSysConfigureService,
	service.NewSysDictService,
	service.NewSysRoleService,
	service.NewSysMenuService,
	service.NewSysTableService,
	service.NewSysUserService,
	service.NewGeneralizationService,
	service.NewDataPermissionService,
	service.NewReportService,
	service.NewCasbinRuleService,
	service.NewApplicationService,
	service.NewDingTalkService,
	service.NewSmsService,
	service.NewFileService,
)

// Controller providers
var ControllerProvider = wire.NewSet(
	controller.NewDictController,
	controller.NewTableController,
	controller.NewMenuController,
	controller.NewRoleController,
	controller.NewUserController,
	controller.NewDataPermissionController,
	controller.NewBasicController,
	controller.NewGeneralizationController,
	controller.NewReportController,
	controller.NewApplicationController,
	controller.NewSmsController,
	controller.NewFileController,
)

// Api providers
var ApiProvider = wire.NewSet(
	api.NewAuthApi,
	api.NewSysUserApi,
	api.NewDingTalkApi,
)

func ProvidePrimaryDB(db map[string]*gorm.DB) *database.PrimaryDB {
	return &database.PrimaryDB{DB: db["primary"]}
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
