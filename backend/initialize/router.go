package initialize

import (
	_ "backend/docs"
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
	//store := cookie.NewStore([]byte(app.Config.Session.Secret))
	router.
		Use(middleware.CorsHandler(middleware.CorsOptions{
			AllowedOrigins:   app.Config.Security.CORSAllowedOrigins,
			AllowCredentials: app.Config.Security.CORSAllowCredentials,
		})).
		Use(middleware.LogHandler(app.LogService)).
		Use(middleware.ResponseHandler())
	//Use(sessions.Sessions("backend-session", store))
	//总路由
	routerGroup := router.Group("/" + app.Config.Name)
	// api路由
	apiBaseGroup := routerGroup.Group("/api")
	{
		// api非验证路由
		apiBaseGroup.POST("/app_token", app.AuthApi.GetAppToken)

		// 单个token验证路由
		apiBaseGroup.POST("/send_sms/:mobile/:templateCode", middleware.AuthHMACHandler(app.HmacGenerator, app.ApplicationCache), app.AuthApi.SendSms)
		apiBaseGroup.POST("/sms_code_login", middleware.AuthHMACHandler(app.HmacGenerator, app.ApplicationCache), app.AuthApi.SmsCodeLogin)
		apiBaseGroup.POST("/login", middleware.AuthHMACHandler(app.HmacGenerator, app.ApplicationCache), app.AuthApi.Login)
		apiBaseGroup.GET("/refresh_token", middleware.AuthHMACHandler(app.HmacGenerator, app.ApplicationCache), app.AuthApi.RefreshToken)
		apiBaseGroup.GET("/sso_login", middleware.AuthHMACHandler(app.HmacGenerator, app.ApplicationCache), app.AuthApi.SSOLogin)
		// 检查短信发送状态
		apiBaseGroup.GET("/check_sms_status/:bizId/:mobile", middleware.AuthHMACHandler(app.HmacGenerator, app.ApplicationCache), app.AuthApi.CheckSmsStatus)
		// 发送钉钉消息
		apiBaseGroup.POST("/dingtalk/send_message", middleware.AuthHMACHandler(app.HmacGenerator, app.ApplicationCache), app.DingTalkApi.SendMessage)
	}
	apiGroup := routerGroup.Group("/api")
	apiGroup.Use(middleware.AuthHMACHandler(app.HmacGenerator, app.ApplicationCache))
	apiGroup.Use(middleware.AuthHandler(app.Config, app.JwtGenerator, app.UserService, app.TokenBlackCache))
	{
		// 双token验证路由
		// auth
		apiGroup.POST("/logout", app.AuthApi.Logout)
		// sys_user
		apiGroup.GET("/user/me", app.SysUserApi.GetMe)
		apiGroup.POST("/user/password", app.SysUserApi.UpdatePassword)

	}

	//后台非验证路由
	adminBaseGroup := routerGroup.Group("/admin")
	{
		adminBaseGroup.POST("/login", app.BasicController.Login)
		adminBaseGroup.GET("/captcha", app.BasicController.Captcha)
		adminBaseGroup.GET("/configure", app.BasicController.Configure)

		//adminBaseGroup.GET("/test", app.BasicController.Test)

	}
	routerGroup.GET("/files/access/preview/:uuid", app.FileController.SignedPreview)
	routerGroup.GET("/files/access/download/:uuid", app.FileController.SignedDownload)
	routerGroup.GET("/files/:uuid", app.FileController.PublicPreview)
	//后台验证路由
	adminGroup := routerGroup.Group("/admin")
	adminGroup.Use(middleware.AuthHandler(app.Config, app.JwtGenerator, app.UserService, app.TokenBlackCache))
	adminGroup.Use(middleware.CasbinHandler(app.Enforcer, middleware.CasbinOptions{
		EnforcePolicyCoverage: app.Config.Security.EnforceCasbinPolicyCoverage,
	}))
	{
		// 退出
		adminGroup.POST("/logout", app.BasicController.Logout)
		adminGroup.GET("/configure/detail", app.BasicController.ConfigureDetail)
		adminGroup.PUT("/configure/:id", app.BasicController.UpdateConfigure)
		adminGroup.POST("/configure/test-email", app.BasicController.TestConfigureEmail)
		adminGroup.POST("/log/access/query", app.BasicController.QueryAccessLogs)
		adminGroup.GET("/log/access/:id", app.BasicController.GetAccessLogById)

		// dict
		adminGroup.GET("/dict/id/:id", app.DictController.GetSysDictById)
		adminGroup.GET("/dict/code/:code", app.DictController.GetSysDictByCode)
		adminGroup.POST("/dict/query", app.DictController.QuerySysDict)
		adminGroup.POST("/dict", app.DictController.CreateSysDict)
		adminGroup.PUT("/dict/:id", app.DictController.UpdateSysDict)
		adminGroup.DELETE("/dict/:id", app.DictController.DeleteSysDictById)

		// dict_item
		adminGroup.GET("/dict/items/:id", app.DictController.GetSysDictItemsByDictId)
		adminGroup.GET("/dict/item/:id", app.DictController.GetSysDictItemById)
		adminGroup.POST("/dict/item", app.DictController.CreateSysDictItem)
		adminGroup.PUT("/dict/item/:id", app.DictController.UpdateSysDictItem)
		adminGroup.DELETE("/dict/item/:id", app.DictController.DeleteSysDictItemById)

		// table
		adminGroup.GET("/table/id/:id", app.TableController.GetTableByID)
		adminGroup.GET("/table/code/:code", app.TableController.GetTableByCode)
		adminGroup.POST("/table/query", app.TableController.QueryTable)
		adminGroup.POST("/table", app.TableController.CreateTable)
		adminGroup.PUT("/table/:id", app.TableController.UpdateTable)
		adminGroup.DELETE("/table/:id", app.TableController.DeleteTableById)

		// table_field
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

		// table_index
		adminGroup.GET("/table/indexes/:id", app.TableController.GetTableIndexesByTableId)
		adminGroup.GET("/table/index/:id", app.TableController.GetTableIndexById)
		adminGroup.POST("/table/index", app.TableController.CreateTableIndex)
		adminGroup.PUT("/table/index/:id", app.TableController.UpdateTableIndex)
		adminGroup.DELETE("/table/index/:id", app.TableController.DeleteTableIndexById)

		// table_relation
		adminGroup.GET("/table/relations/:id", app.TableController.GetTableRelationsByTableId)
		adminGroup.GET("/table/relation/:id", app.TableController.GetTableRelationById)
		adminGroup.POST("/table/relation", app.TableController.CreateTableRelation)
		adminGroup.PUT("/table/relation/:id", app.TableController.UpdateTableRelation)
		adminGroup.DELETE("/table/relation/:id", app.TableController.DeleteTableRelationById)

		// menu
		adminGroup.GET("/menu/user/:id", app.MenuController.GetUserMenus)
		adminGroup.GET("/menu/:id", app.MenuController.GetMenuById)
		adminGroup.POST("/menu/query", app.MenuController.QueryMenus)
		adminGroup.POST("/menu", app.MenuController.CreateMenu)
		adminGroup.PUT("/menu/order", app.MenuController.UpdateMenuOrder)
		adminGroup.POST("/menu/refresh-cache", app.MenuController.RefreshMenuCache)
		adminGroup.PUT("/menu/:id", app.MenuController.UpdateMenu)
		adminGroup.DELETE("/menu/:id", app.MenuController.DeleteMenuById)
		adminGroup.GET("/menu/my", app.MenuController.GetMyMenus)

		// menu button
		adminGroup.GET("/menu/buttons/:menuId", app.MenuController.GetMenuButtons)
		adminGroup.POST("/menu/button", app.MenuController.CreateMenuButton)
		adminGroup.PUT("/menu/button/:id", app.MenuController.UpdateMenuButton)
		adminGroup.DELETE("/menu/button/:id", app.MenuController.DeleteMenuButton)

		// role
		adminGroup.GET("/role/menu/:id", app.RoleController.GetRoleMenus)
		adminGroup.GET("/role/menu/buttons/:roleId/:menuId", app.RoleController.GetRoleMenuButtons)
		adminGroup.GET("/role/:id/data-permissions", app.DataPermissionController.GetRoleDataScopes)
		adminGroup.PUT("/role/:id/data-permissions", app.DataPermissionController.SaveRoleDataScopes)
		adminGroup.POST("/role/assign-permissions", app.RoleController.AssignPermissions)
		adminGroup.POST("/role/query", app.RoleController.QueryRole)
		adminGroup.GET("/role/:id", app.RoleController.GetRoleById)
		adminGroup.POST("/role", app.RoleController.CreateRole)
		adminGroup.PUT("/role/:id", app.RoleController.UpdateRole)
		adminGroup.DELETE("/role/:id", app.RoleController.DeleteRoleById)

		// user
		adminGroup.GET("/user/me", app.UserController.GetMe)
		adminGroup.POST("/user/query", app.UserController.QuerySysUser)
		adminGroup.GET("/user/:id", app.UserController.GetUserById)
		adminGroup.POST("/user", app.UserController.CreateUser)
		adminGroup.POST("/user/password", app.UserController.UpdatePassword)
		adminGroup.POST("/user/reset_password/:id", app.UserController.ResetPassword)
		adminGroup.POST("/user/unlock_login/:id", app.UserController.UnlockLogin)
		adminGroup.GET("/user/:id/data-permissions", app.DataPermissionController.GetUserOverrides)
		adminGroup.PUT("/user/:id/data-permissions", app.DataPermissionController.SaveUserOverrides)
		adminGroup.GET("/user/:id/dimension-values", app.DataPermissionController.GetUserDimensionValues)
		adminGroup.PUT("/user/:id/dimension-values", app.DataPermissionController.SaveUserDimensionValues)
		adminGroup.PUT("/user/:id/roles", app.UserController.AssignRoles)
		adminGroup.PUT("/user/:id", app.UserController.UpdateUser)
		adminGroup.DELETE("/user/:id", app.UserController.DeleteUser)

		// data permission
		adminGroup.POST("/data-permission/dimension/query", app.DataPermissionController.QueryDimensions)
		adminGroup.GET("/data-permission/dimension/:id", app.DataPermissionController.GetDimensionById)
		adminGroup.POST("/data-permission/dimension", app.DataPermissionController.CreateDimension)
		adminGroup.PUT("/data-permission/dimension/:id", app.DataPermissionController.UpdateDimension)
		adminGroup.DELETE("/data-permission/dimension/:id", app.DataPermissionController.DeleteDimension)
		adminGroup.GET("/data-permission/dimension-options/:code", app.DataPermissionController.GetDimensionOptions)
		adminGroup.GET("/data-permission/bindings/menu/:menuId", app.DataPermissionController.GetMenuBindings)
		adminGroup.PUT("/data-permission/bindings/menu/:menuId", app.DataPermissionController.SaveMenuBindings)
		adminGroup.GET("/data-permission/debug", app.DataPermissionController.DebugDataScope)

		// application
		adminGroup.GET("/application/:id", app.ApplicationController.GetApplicationById)
		adminGroup.POST("/application/query", app.ApplicationController.QueryApplication)
		adminGroup.POST("/application", app.ApplicationController.CreateApplication)
		adminGroup.POST("/application/:id/rotate-secret", app.ApplicationController.RotateApplicationSecret)
		adminGroup.PUT("/application/:id", app.ApplicationController.UpdateApplication)
		adminGroup.DELETE("/application/:id", app.ApplicationController.DeleteApplicationById)

		// sms
		adminGroup.POST("/sms/template/query", app.SmsController.QuerySmsTemplate)
		adminGroup.GET("/sms/template/:id", app.SmsController.GetSmsTemplateById)
		adminGroup.POST("/sms/template", app.SmsController.CreateSmsTemplate)
		adminGroup.PUT("/sms/template/:id", app.SmsController.UpdateSmsTemplate)
		adminGroup.DELETE("/sms/template/:id", app.SmsController.DeleteSmsTemplateById)

		// generalization
		adminGroup.POST("/generalization/query/:id", app.GeneralizationController.Query)
		adminGroup.POST("/generalization/query/code/:code", app.GeneralizationController.QueryByCode)
		adminGroup.GET("/generalization/detail/code/:code/:id", app.GeneralizationController.DetailByCode)
		adminGroup.POST("/generalization/create", app.GeneralizationController.Create)
		adminGroup.PUT("/generalization/update", app.GeneralizationController.Update)
		adminGroup.DELETE("/generalization/delete", app.GeneralizationController.Delete)
		adminGroup.DELETE("/generalization/batch-delete", app.GeneralizationController.BatchDelete)
		adminGroup.POST("/generalization/export", app.GeneralizationController.Export)

		// file
		adminGroup.POST("/file/upload", app.FileController.Upload)
		adminGroup.GET("/file/:id", app.FileController.GetFileById)
		adminGroup.DELETE("/file/:id", app.FileController.DeleteFileById)
		adminGroup.GET("/file/preview-url/:uuid", app.FileController.GetFilePreviewAccessURL)
		adminGroup.GET("/file/download-url/:uuid", app.FileController.GetFileDownloadAccessURL)
		adminGroup.GET("/file/preview/:uuid", app.FileController.Preview)
		adminGroup.GET("/file/download/:uuid", app.FileController.Download)

		// file chunk upload
		adminGroup.POST("/file/upload/init", app.FileController.InitChunkUpload)
		adminGroup.POST("/file/upload/chunk", app.FileController.UploadChunk)
		adminGroup.POST("/file/upload/merge/:upload_id", app.FileController.MergeChunks)
		adminGroup.GET("/file/upload/progress/:upload_id", app.FileController.GetUploadProgress)

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
