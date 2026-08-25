package initialize

import (
	_ "backend/docs"
	"backend/dto/response"
	"backend/middleware"
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func InitRouter(app *App) *gin.Engine {
	router := gin.Default()
	router.Use(middleware.CustomRecovery())
	router.NoRoute(middleware.NoRouteHandler())
	router.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"status": "ok"})
	})
	router.GET("/readyz", readinessHandler(app))
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.
		Use(middleware.CorsHandler(middleware.CorsOptions{
			AllowedOrigins:   app.Config.Security.CORSAllowedOrigins,
			AllowCredentials: app.Config.Security.CORSAllowCredentials,
		})).
		Use(middleware.LogHandler(app.LogService)).
		Use(middleware.ResponseHandler())
	// 总路由
	routerGroup := router.Group("/" + app.Config.Name)
	// API 路由
	apiBaseGroup := routerGroup.Group("/api")
	{
		// API 非验证路由
		apiBaseGroup.POST("/app_token", app.AuthApi.GetAppToken)

		// 单 Token 验证路由
		apiBaseGroup.POST("/send_sms/:mobile/:templateCode", middleware.AuthHMACHandler(app.HmacGenerator, app.ApplicationCache, app.ApplicationService), app.AuthApi.SendSms)
		apiBaseGroup.POST("/sms_code_login", middleware.AuthHMACHandler(app.HmacGenerator, app.ApplicationCache, app.ApplicationService), app.AuthApi.SmsCodeLogin)
		apiBaseGroup.POST("/login", middleware.AuthHMACHandler(app.HmacGenerator, app.ApplicationCache, app.ApplicationService), app.AuthApi.Login)
		apiBaseGroup.GET("/refresh_token", middleware.AuthHMACHandler(app.HmacGenerator, app.ApplicationCache, app.ApplicationService), app.AuthApi.RefreshToken)
		apiBaseGroup.GET("/sso_login", middleware.AuthHMACHandler(app.HmacGenerator, app.ApplicationCache, app.ApplicationService), app.AuthApi.SSOLogin)
		apiBaseGroup.POST("/logout", middleware.AuthHMACHandler(app.HmacGenerator, app.ApplicationCache, app.ApplicationService), app.AuthApi.Logout)
		// 检查短信发送状态
		apiBaseGroup.GET("/check_sms_status/:bizId/:mobile", middleware.AuthHMACHandler(app.HmacGenerator, app.ApplicationCache, app.ApplicationService), app.AuthApi.CheckSmsStatus)
		// 发送钉钉消息
		apiBaseGroup.POST("/dingtalk/send_message", middleware.AuthHMACHandler(app.HmacGenerator, app.ApplicationCache, app.ApplicationService), app.DingTalkApi.SendMessage)
	}
	apiGroup := routerGroup.Group("/api")
	apiGroup.Use(middleware.AuthHMACHandler(app.HmacGenerator, app.ApplicationCache, app.ApplicationService))
	apiGroup.Use(middleware.AuthHandler(app.AuthService))
	{
		// 双 Token 验证路由
		// 系统用户
		apiGroup.GET("/user/me", app.SysUserApi.GetMe)
		apiGroup.POST("/user/password", app.SysUserApi.UpdatePassword)

	}

	// 后台非验证路由
	adminBaseGroup := routerGroup.Group("/admin")
	{
		adminBaseGroup.POST("/login", app.BasicController.Login)
		adminBaseGroup.GET("/captcha", app.BasicController.Captcha)
		adminBaseGroup.GET("/configure", app.BasicController.Configure)
		adminBaseGroup.POST("/logout", app.BasicController.Logout)

	}
	routerGroup.GET("/files/access/preview/:uuid", app.FileAccessController.SignedPreview)
	routerGroup.GET("/files/access/download/:uuid", app.FileAccessController.SignedDownload)
	routerGroup.GET("/files/:uuid", app.FileAccessController.PublicPreview)
	// 后台验证路由
	adminGroup := routerGroup.Group("/admin")
	adminGroup.Use(middleware.AuthHandler(app.AuthService))
	adminGroup.Use(middleware.CasbinHandler(app.Enforcer, middleware.CasbinOptions{
		EnforcePolicyCoverage: app.Config.Security.EnforceCasbinPolicyCoverage,
	}))
	{
		adminGroup.GET("/configure/detail", app.BasicController.ConfigureDetail)
		adminGroup.PUT("/configure/:id", app.BasicController.UpdateConfigure)
		adminGroup.POST("/configure/test-email", app.BasicController.TestConfigureEmail)
		adminGroup.POST("/log/access/query", app.BasicController.QueryAccessLogs)
		adminGroup.GET("/log/access/:id", app.BasicController.GetAccessLogById)

		// 字典
		adminGroup.GET("/dict/id/:id", app.DictController.GetSysDictById)
		adminGroup.GET("/dict/code/:code", app.DictController.GetSysDictByCode)
		adminGroup.GET("/runtime/dict/:code", app.DictController.GetRuntimeDictByCode)
		adminGroup.POST("/dict/query", app.DictController.QuerySysDict)
		adminGroup.POST("/dict", app.DictController.CreateSysDict)
		adminGroup.PUT("/dict/:id", app.DictController.UpdateSysDict)
		adminGroup.DELETE("/dict/:id", app.DictController.DeleteSysDictById)

		// 查询方案中心：Runtime只返回当前Scope可见方案，个人写入继续由Owner约束。
		adminGroup.GET("/runtime/query-scopes/:scope", app.QuerySchemeController.ScopeConfig)
		adminGroup.GET("/runtime/query-schemes/available", app.QuerySchemeController.Available)
		adminGroup.POST("/runtime/query-schemes/:id/resolve", app.QuerySchemeController.Resolve)
		adminGroup.GET("/runtime/notifications/unread-count", app.NotificationController.UnreadCount)
		adminGroup.GET("/runtime/notifications/recent", app.NotificationController.Recent)
		adminGroup.POST("/runtime/notifications/query", app.NotificationController.Query)
		adminGroup.GET("/runtime/notifications/:id", app.NotificationController.Detail)
		adminGroup.POST("/runtime/notifications/:id/read", app.NotificationController.MarkRead)
		adminGroup.POST("/runtime/notifications/read-all", app.NotificationController.MarkAllRead)
		adminGroup.POST("/query-schemes/query", app.QuerySchemeController.Query)
		adminGroup.GET("/query-schemes/:id", app.QuerySchemeController.Detail)
		adminGroup.POST("/query-schemes/personal", app.QuerySchemeController.CreatePersonal)
		adminGroup.PUT("/query-schemes/personal/:id", app.QuerySchemeController.UpdatePersonal)
		adminGroup.DELETE("/query-schemes/personal/:id", app.QuerySchemeController.DeletePersonal)
		adminGroup.PUT("/query-schemes/personal/:id/default", app.QuerySchemeController.SetPersonalDefault)
		adminGroup.POST("/query-schemes/:id/copy-to-personal", app.QuerySchemeController.CopyToPersonal)
		adminGroup.POST("/query-schemes/shared", app.QuerySchemeController.CreateShared)
		adminGroup.PUT("/query-schemes/shared/:id", app.QuerySchemeController.UpdateShared)
		adminGroup.DELETE("/query-schemes/shared/:id", app.QuerySchemeController.DeleteShared)
		adminGroup.PUT("/query-schemes/shared/:id/enabled", app.QuerySchemeController.SetSharedEnabled)

		// 字典项
		adminGroup.GET("/dict/items/:id", app.DictController.GetSysDictItemsByDictId)
		adminGroup.GET("/dict/item/:id", app.DictController.GetSysDictItemById)
		adminGroup.POST("/dict/item", app.DictController.CreateSysDictItem)
		adminGroup.PUT("/dict/item/:id", app.DictController.UpdateSysDictItem)
		adminGroup.DELETE("/dict/item/:id", app.DictController.DeleteSysDictItemById)

		// 数据表
		adminGroup.GET("/table/id/:id", app.TableController.GetTableByID)
		adminGroup.GET("/table/code/:code", app.TableController.GetTableByCode)
		adminGroup.GET("/runtime/table/:code", app.TableController.GetRuntimeTableByCode)
		adminGroup.POST("/runtime/relation-fields/:fieldId/options", app.TableController.QueryRuntimeRelationOptions)
		adminGroup.POST("/table/query", app.TableController.QueryTable)
		adminGroup.POST("/table", app.TableController.CreateTable)
		adminGroup.PUT("/table/:id", app.TableController.UpdateTable)
		adminGroup.DELETE("/table/:id", app.TableController.DeleteTableById)

		// 数据表字段
		adminGroup.GET("/table/fields/:id", app.TableController.GetTableFieldsByTableId)
		adminGroup.GET("/table/field/:id", app.TableController.GetTableFieldById)
		adminGroup.POST("/table/field", app.TableController.CreateTableField)
		adminGroup.PUT("/table/field/:id", app.TableController.UpdateTableField)
		adminGroup.DELETE("/table/field/:id", app.TableController.DeleteTableFieldById)

		adminGroup.GET("/table/init/:code", app.TableController.InitTable)
		adminGroup.POST("/table/sync/:code", app.TableController.SyncTableFields)
		adminGroup.POST("/table/sync/index/:code", app.TableController.SyncTableIndexes)
		adminGroup.POST("/table/publish/:code", app.TableController.PublishTable)
		adminGroup.POST("/table/unpublish/:code", app.TableController.UnpublishTable)

		// 数据表索引
		adminGroup.GET("/table/indexes/:id", app.TableController.GetTableIndexesByTableId)
		adminGroup.GET("/table/index/:id", app.TableController.GetTableIndexById)
		adminGroup.POST("/table/index", app.TableController.CreateTableIndex)
		adminGroup.PUT("/table/index/:id", app.TableController.UpdateTableIndex)
		adminGroup.DELETE("/table/index/:id", app.TableController.DeleteTableIndexById)

		// 数据表关系
		adminGroup.GET("/table/relations/:id", app.TableController.GetTableRelationsByTableId)
		adminGroup.GET("/table/relation/:id", app.TableController.GetTableRelationById)
		adminGroup.POST("/table/relation", app.TableController.CreateTableRelation)
		adminGroup.PUT("/table/relation/:id", app.TableController.UpdateTableRelation)
		adminGroup.DELETE("/table/relation/:id", app.TableController.DeleteTableRelationById)

		// 菜单
		adminGroup.GET("/menu/user/:id", app.MenuController.GetUserMenus)
		adminGroup.GET("/menu/:id", app.MenuController.GetMenuById)
		adminGroup.POST("/menu/query", app.MenuController.QueryMenus)
		adminGroup.POST("/menu", app.MenuController.CreateMenu)
		adminGroup.PUT("/menu/order", app.MenuController.UpdateMenuOrder)
		adminGroup.PUT("/menu/:id", app.MenuController.UpdateMenu)
		adminGroup.DELETE("/menu/:id", app.MenuController.DeleteMenuById)
		adminGroup.GET("/menu/my", app.MenuController.GetMyMenus)

		// 菜单按钮
		adminGroup.GET("/menu/buttons/:menuId", app.MenuController.GetMenuButtons)
		adminGroup.POST("/menu/button", app.MenuController.CreateMenuButton)
		adminGroup.PUT("/menu/button/:id", app.MenuController.UpdateMenuButton)
		adminGroup.DELETE("/menu/button/:id", app.MenuController.DeleteMenuButton)

		// 角色
		adminGroup.GET("/role/menu/:id", app.RoleController.GetRoleMenus)
		adminGroup.GET("/role/menu/buttons/:roleId/:menuId", app.RoleController.GetRoleMenuButtons)
		adminGroup.POST("/role/assign-permissions", app.RoleController.AssignPermissions)
		adminGroup.POST("/role/query", app.RoleController.QueryRole)
		adminGroup.GET("/role/:id", app.RoleController.GetRoleById)
		adminGroup.POST("/role", app.RoleController.CreateRole)
		adminGroup.PUT("/role/:id", app.RoleController.UpdateRole)
		adminGroup.DELETE("/role/:id", app.RoleController.DeleteRoleById)

		// 用户
		adminGroup.GET("/user/me", app.UserController.GetMe)
		adminGroup.POST("/user/query", app.UserController.QuerySysUser)
		adminGroup.GET("/user/:id", app.UserController.GetUserById)
		adminGroup.POST("/user", app.UserController.CreateUser)
		adminGroup.POST("/user/password", app.UserController.UpdatePassword)
		adminGroup.POST("/user/reset_password/:id", app.UserController.ResetPassword)
		adminGroup.POST("/user/unlock_login/:id", app.UserController.UnlockLogin)
		adminGroup.PUT("/user/:id/roles", app.UserController.AssignRoles)
		adminGroup.PUT("/user/:id", app.UserController.UpdateUser)
		adminGroup.DELETE("/user/:id", app.UserController.DeleteUser)

		// 数据权限配置查询和预检
		adminGroup.POST("/data-permission/config/dimension/query", app.DataPermissionConfigController.QueryDimensions)
		adminGroup.POST("/data-permission/config/resource", app.DataPermissionConfigController.CreateResource)
		adminGroup.POST("/data-permission/config/resource/query", app.DataPermissionConfigController.QueryResources)
		adminGroup.GET("/data-permission/config/resource/:id", app.DataPermissionConfigController.GetResource)
		adminGroup.PUT("/data-permission/config/resource/:id", app.DataPermissionConfigController.UpdateResource)
		adminGroup.GET("/data-permission/config/resource/:id/operations", app.DataPermissionConfigController.ListResourceOperations)
		adminGroup.PUT("/data-permission/config/resource/:id/operations", app.DataPermissionConfigController.ReplaceResourceOperations)
		adminGroup.PUT("/data-permission/config/resource/:id/permission", app.DataPermissionConfigController.SetResourcePermission)
		adminGroup.GET("/data-permission/config/resource/:id/ownerships", app.DataPermissionConfigController.ListOwnershipsByResource)
		adminGroup.POST("/data-permission/config/ownership", app.DataPermissionConfigController.CreateOwnership)
		adminGroup.POST("/data-permission/config/ownership/query", app.DataPermissionConfigController.QueryOwnerships)
		adminGroup.GET("/data-permission/config/ownership/:id", app.DataPermissionConfigController.GetOwnership)
		adminGroup.PUT("/data-permission/config/ownership/:id", app.DataPermissionConfigController.UpdateOwnership)
		adminGroup.POST("/data-permission/config/policy", app.DataPermissionConfigController.CreatePolicy)
		adminGroup.POST("/data-permission/config/policy/query", app.DataPermissionConfigController.QueryPolicies)
		adminGroup.GET("/data-permission/config/policy/:id", app.DataPermissionConfigController.GetPolicy)
		adminGroup.PUT("/data-permission/config/policy/:id", app.DataPermissionConfigController.UpdatePolicy)
		adminGroup.PUT("/data-permission/config/policy/:id/rules", app.DataPermissionConfigController.ReplacePolicyRules)
		adminGroup.PUT("/data-permission/config/policy/:id/state", app.DataPermissionConfigController.SetPolicyState)
		adminGroup.POST("/data-permission/config/policy/rule/query", app.DataPermissionConfigController.QueryPolicyRules)
		adminGroup.POST("/data-permission/config/grant", app.DataPermissionConfigController.CreateGrant)
		adminGroup.POST("/data-permission/config/grant/query", app.DataPermissionConfigController.QueryGrants)
		adminGroup.GET("/data-permission/config/grant/:id", app.DataPermissionConfigController.GetGrant)
		adminGroup.PUT("/data-permission/config/grant/:id/state", app.DataPermissionConfigController.SetGrantState)
		adminGroup.GET("/data-permission/config/preflight/resource/:id", app.DataPermissionConfigController.PreflightResource)
		adminGroup.GET("/data-permission/config/preflight/policy/:id", app.DataPermissionConfigController.PreflightPolicy)
		adminGroup.GET("/data-permission/config/preflight/grant/:id", app.DataPermissionConfigController.PreflightGrant)

		// 集成中心 - 外部系统
		adminGroup.POST("/integration/external-system/query", app.ExternalSystemController.Query)
		adminGroup.GET("/integration/external-system/:id", app.ExternalSystemController.Detail)
		adminGroup.POST("/integration/external-system", app.ExternalSystemController.Create)
		adminGroup.PUT("/integration/external-system/:id", app.ExternalSystemController.Update)
		adminGroup.PUT("/integration/external-system/:id/enable", app.ExternalSystemController.Enable)
		adminGroup.PUT("/integration/external-system/:id/disable", app.ExternalSystemController.Disable)

		// 集成中心 - 接口定义
		adminGroup.POST("/integration/interface-definition/query", app.InterfaceDefinitionController.Query)
		adminGroup.GET("/integration/interface-definition/:id", app.InterfaceDefinitionController.Detail)
		adminGroup.POST("/integration/interface-definition", app.InterfaceDefinitionController.Create)
		adminGroup.PUT("/integration/interface-definition/:id", app.InterfaceDefinitionController.Update)
		adminGroup.POST("/integration/interface-definition/:id/versions", app.InterfaceDefinitionController.CreateVersion)
		adminGroup.PUT("/integration/interface-definition/:id/enable", app.InterfaceDefinitionController.Enable)
		adminGroup.PUT("/integration/interface-definition/:id/disable", app.InterfaceDefinitionController.Disable)

		// 集成凭证配置
		adminGroup.POST("/integration/credential/query", app.CredentialController.Query)
		adminGroup.GET("/integration/credential/:id", app.CredentialController.Detail)
		adminGroup.POST("/integration/credential", app.CredentialController.Create)
		adminGroup.PUT("/integration/credential/:id", app.CredentialController.Update)
		adminGroup.POST("/integration/credential/:id/rotate", app.CredentialController.Rotate)
		adminGroup.PUT("/integration/credential/:id/enable", app.CredentialController.Enable)
		adminGroup.PUT("/integration/credential/:id/disable", app.CredentialController.Disable)
		adminGroup.PUT("/integration/credential/:id/revoke", app.CredentialController.Revoke)

		// 集成中心 - 重试策略配置，仅管理版本化策略，不触发运行时重试。
		adminGroup.POST("/integration/retry-policy/query", app.RetryPolicyController.Query)
		adminGroup.GET("/integration/retry-policy/:id", app.RetryPolicyController.Detail)
		adminGroup.POST("/integration/retry-policy", app.RetryPolicyController.Create)
		adminGroup.PUT("/integration/retry-policy/:id", app.RetryPolicyController.Update)
		adminGroup.POST("/integration/retry-policy/:id/versions", app.RetryPolicyController.CreateVersion)
		adminGroup.PUT("/integration/retry-policy/:id/enable", app.RetryPolicyController.Enable)
		adminGroup.PUT("/integration/retry-policy/:id/disable", app.RetryPolicyController.Disable)

		// 集成中心 - 同步任务配置、受控手工运行与批次查询；不暴露批次取消。
		adminGroup.POST("/integration/sync-task/query", app.IntegrationSyncController.QueryTasks)
		adminGroup.GET("/integration/sync-task/consumers", app.IntegrationSyncController.ConsumerMetadata)
		adminGroup.GET("/integration/sync-task/:id", app.IntegrationSyncController.TaskDetail)
		adminGroup.GET("/integration/sync-task/:id/edit", app.IntegrationSyncController.TaskEdit)
		adminGroup.POST("/integration/sync-task", app.IntegrationSyncController.CreateTask)
		adminGroup.PUT("/integration/sync-task/:id", app.IntegrationSyncController.UpdateTask)
		adminGroup.POST("/integration/sync-task/:id/versions", app.IntegrationSyncController.CreateTaskVersion)
		adminGroup.PUT("/integration/sync-task/:id/enable", app.IntegrationSyncController.EnableTask)
		adminGroup.PUT("/integration/sync-task/:id/disable", app.IntegrationSyncController.DisableTask)
		adminGroup.POST("/integration/sync-task/:id/run", app.IntegrationSyncController.RunTask)
		adminGroup.POST("/integration/sync-batch/query", app.IntegrationSyncController.QueryBatches)
		adminGroup.GET("/integration/sync-batch/:id", app.IntegrationSyncController.BatchDetail)

		// 集成执行与调用日志只暴露管理查询、提交和安全取消；状态收敛由 Worker 与 Engine 完成。
		adminGroup.POST("/integration/execution/query", app.IntegrationExecutionController.Query)
		adminGroup.GET("/integration/execution/:id", app.IntegrationExecutionController.Detail)
		adminGroup.POST("/integration/execution", app.IntegrationExecutionController.Create)
		adminGroup.PUT("/integration/execution/:id/cancel", app.IntegrationExecutionController.Cancel)
		adminGroup.POST("/integration/log/query", app.IntegrationExecutionController.QueryLogs)
		adminGroup.GET("/integration/log/:id", app.IntegrationExecutionController.LogDetail)
		adminGroup.GET("/integration/worker/status", func(ctx *gin.Context) {
			resp := response.NewResponse()
			ctx.Set("response", resp)
			if app.IntegrationWorker == nil {
				resp.SetData(map[string]any{
					"enabled":                false,
					"running":                false,
					"worker_id":              "",
					"started_at":             nil,
					"last_poll_at":           nil,
					"last_success_at":        nil,
					"last_error_category":    "",
					"active_execution_count": 0,
					"claimed_total":          0,
					"completed_total":        0,
					"failed_total":           0,
					"recovered_total":        0,
				})
				return
			}
			status := app.IntegrationWorker.Status()
			resp.SetData(map[string]any{
				"enabled":                status.Enabled,
				"running":                status.Running,
				"worker_id":              status.WorkerID,
				"started_at":             status.StartedAt,
				"last_poll_at":           status.LastPollAt,
				"last_success_at":        status.LastSuccessAt,
				"last_error_category":    status.LastErrorCategory,
				"active_execution_count": status.ActiveExecutionCount,
				"claimed_total":          status.ClaimedTotal,
				"completed_total":        status.CompletedTotal,
				"failed_total":           status.FailedTotal,
				"recovered_total":        status.RecoveredTotal,
			})
		})

		// 组织法人主体只读镜像
		adminGroup.POST("/org/legal-entity/query", app.OrgController.QueryLegalEntities)
		adminGroup.GET("/org/legal-entity/:id", app.OrgController.GetLegalEntityDetail)
		adminGroup.POST("/org/legal-entity/tree", app.OrgController.GetLegalEntityTree)
		adminGroup.POST("/org/legal-entity/options", app.OrgController.QueryLegalEntityOptions)

		// 管理架构与组织单元只读镜像
		adminGroup.POST("/org/structure/query", app.OrgController.QueryStructures)
		adminGroup.POST("/org/structure/options", app.OrgController.QueryStructureOptions)
		adminGroup.GET("/org/structure/:id", app.OrgController.GetStructureDetail)
		adminGroup.POST("/org/unit/query", app.OrgController.QueryOrgUnits)
		adminGroup.POST("/org/unit/options", app.OrgController.QueryOrgUnitOptions)
		adminGroup.POST("/org/unit/tree", app.OrgController.GetStructureOrgTree)
		adminGroup.GET("/org/unit/:id", app.OrgController.GetOrgUnitDetail)
		adminGroup.POST("/org/employee/query", app.OrgController.QueryEmployees)
		adminGroup.POST("/org/employee/options", app.OrgController.QueryEmployeeOptions)
		adminGroup.POST("/org/employee/user-options", app.OrgController.QueryEmployeeUserOptions)
		adminGroup.GET("/org/employee/:id", app.OrgController.GetEmployeeDetail)
		adminGroup.POST("/org/employee/:id/bind-user", app.OrgController.BindEmployeeUser)
		adminGroup.POST("/org/employee/:id/unbind-user", app.OrgController.UnbindEmployeeUser)
		adminGroup.GET("/org/employee/:id/assignments/summary", app.OrgController.GetEmployeeCurrentAssignmentSummary)
		adminGroup.POST("/org/position/query", app.OrgController.QueryPositions)
		adminGroup.POST("/org/position/options", app.OrgController.QueryPositionOptions)
		adminGroup.GET("/org/position/:id", app.OrgController.GetPositionDetail)
		adminGroup.POST("/org/assignment/query", app.OrgController.QueryAssignments)
		adminGroup.GET("/org/assignment/:id", app.OrgController.GetAssignmentDetail)
		adminGroup.POST("/org/sync/batch/query", app.OrgController.QuerySyncBatches)
		adminGroup.GET("/org/sync/batch/:id", app.OrgController.GetSyncBatchDetail)
		adminGroup.GET("/org/sync/batch/:id/error", app.OrgController.GetSyncBatchError)
		adminGroup.POST("/org/sync/record/query", app.OrgController.QuerySyncRecords)
		adminGroup.GET("/org/sync/record/:id", app.OrgController.GetSyncRecordDetail)
		adminGroup.GET("/org/sync/record/:id/error", app.OrgController.GetSyncRecordError)

		// 应用
		adminGroup.GET("/application/:id", app.ApplicationController.GetApplicationById)
		adminGroup.POST("/application/query", app.ApplicationController.QueryApplication)
		adminGroup.POST("/application", app.ApplicationController.CreateApplication)
		adminGroup.POST("/application/:id/rotate-secret", app.ApplicationController.RotateApplicationSecret)
		adminGroup.PUT("/application/:id", app.ApplicationController.UpdateApplication)
		adminGroup.DELETE("/application/:id", app.ApplicationController.DeleteApplicationById)

		// 短信
		adminGroup.POST("/sms/template/query", app.SmsController.QuerySmsTemplate)
		adminGroup.GET("/sms/template/:id", app.SmsController.GetSmsTemplateById)
		adminGroup.POST("/sms/template", app.SmsController.CreateSmsTemplate)
		adminGroup.PUT("/sms/template/:id", app.SmsController.UpdateSmsTemplate)
		adminGroup.DELETE("/sms/template/:id", app.SmsController.DeleteSmsTemplateById)

		// 通用低代码
		adminGroup.POST("/generalization/query/code/:code", app.GeneralizationController.QueryByCode)
		adminGroup.GET("/generalization/detail/code/:code/:id", app.GeneralizationController.DetailByCode)
		adminGroup.POST("/generalization/create", app.GeneralizationController.Create)
		adminGroup.PUT("/generalization/update", app.GeneralizationController.Update)
		adminGroup.DELETE("/generalization/delete", app.GeneralizationController.Delete)
		adminGroup.DELETE("/generalization/batch-delete", app.GeneralizationController.BatchDelete)
		adminGroup.POST("/generalization/export", app.GeneralizationController.Export)

		// 报表
		adminGroup.POST("/report/query", app.ReportController.QueryReportDefinitions)
		adminGroup.GET("/report/data-sources", app.ReportController.GetReportDataSources)
		adminGroup.POST("/report/sql-fields", app.ReportController.InferSQLFields)
		adminGroup.GET("/report/:id/versions", app.ReportController.GetReportVersions)
		adminGroup.GET("/report/:id", app.ReportController.GetReportDefinitionById)
		adminGroup.POST("/report", app.ReportController.CreateReportDefinition)
		adminGroup.PUT("/report/:id", app.ReportController.UpdateReportDefinition)
		adminGroup.POST("/report/:id/status", app.ReportController.UpdateReportDefinitionStatus)
		adminGroup.POST("/report/:id/publish", app.ReportController.PublishReport)
		adminGroup.POST("/report/:id/publish-menu", app.ReportController.PublishReportMenu)
		adminGroup.DELETE("/report/:id/publish-menu", app.ReportController.UnpublishReportMenu)
		adminGroup.POST("/report/:id/design-preview", app.ReportController.DesignPreviewReport)
		adminGroup.POST("/report/:id/run", app.ReportController.RunReport)
		adminGroup.POST("/report/:id/export", app.ReportController.ExportReport)
		adminGroup.DELETE("/report/:id", app.ReportController.DeleteReportDefinitionById)
		adminGroup.POST("/report/:id/preview", app.ReportController.PreviewReport)

		// 文件
		adminGroup.POST("/file/upload", app.FileUploadController.Upload)
		adminGroup.GET("/file/:id", app.FileMetadataController.GetFileById)
		adminGroup.DELETE("/file/:id", app.FileMetadataController.DeleteFileById)
		adminGroup.GET("/file/preview-url/:uuid", app.FileAccessController.GetFilePreviewAccessURL)
		adminGroup.GET("/file/download-url/:uuid", app.FileAccessController.GetFileDownloadAccessURL)
		adminGroup.GET("/file/preview/:uuid", app.FileAccessController.Preview)
		adminGroup.GET("/file/download/:uuid", app.FileAccessController.Download)

		// 文件分片上传
		adminGroup.POST("/file/upload/init", app.FileUploadController.InitChunkUpload)
		adminGroup.POST("/file/upload/chunk", app.FileUploadController.UploadChunk)
		adminGroup.POST("/file/upload/merge/:upload_id", app.FileUploadController.MergeChunks)
		adminGroup.GET("/file/upload/progress/:upload_id", app.FileUploadController.GetUploadProgress)

	}
	return router
}

func readinessHandler(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		checkCtx, cancel := context.WithTimeout(ctx.Request.Context(), 2*time.Second)
		defer cancel()

		components := gin.H{}
		ready := true

		if len(app.DBs) == 0 {
			components["db"] = dependencyStatus{OK: false, Error: "database clients unavailable"}
			ready = false
		}
		for name, db := range app.DBs {
			status := dependencyStatus{OK: true}
			if db == nil {
				status.OK = false
				status.Error = "database client unavailable"
			} else if sqlDB, err := db.DB(); err != nil {
				status.OK = false
				status.Error = "database handle unavailable"
			} else if err := sqlDB.PingContext(checkCtx); err != nil {
				status.OK = false
				status.Error = "database ping failed"
			}
			if !status.OK {
				ready = false
			}
			components["db_"+name] = status
		}

		redisStatus := dependencyStatus{OK: true}
		if app.Redis == nil {
			redisStatus.OK = false
			redisStatus.Error = "redis client unavailable"
		} else if err := app.Redis.Ping(checkCtx).Err(); err != nil {
			redisStatus.OK = false
			redisStatus.Error = "redis ping failed"
		}
		if !redisStatus.OK {
			ready = false
		}
		components["redis"] = redisStatus

		statusCode := http.StatusOK
		if !ready {
			statusCode = http.StatusServiceUnavailable
		}
		ctx.JSON(statusCode, gin.H{
			"status":     map[bool]string{true: "ready", false: "not_ready"}[ready],
			"components": components,
		})
	}
}

type dependencyStatus struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}
