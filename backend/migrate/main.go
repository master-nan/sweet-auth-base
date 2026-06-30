package main

import (
	"backend/config"
	"backend/enum"
	"backend/initialize"
	"backend/internal/cache"
	"backend/internal/security"
	"backend/internal/utils"
	"backend/model"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

const lowCodeCrudButtonTemplateScene = "lowcode_crud"

func main() {
	command := migrationCommand(os.Args)
	cfg, err := initialize.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	db, err := openPrimaryDB(cfg)
	if err != nil {
		log.Fatal(err)
	}

	switch command {
	case "migrate":
		if err := migrateSchema(db); err != nil {
			log.Fatal(err)
		}
		if err := backfillSysMenuPageBinding(db); err != nil {
			log.Fatal(err)
		}
		log.Println("schema migration completed")
	case "seed":
		sf, err := initialize.InitSnowflake(cfg)
		if err != nil {
			log.Fatal(err)
		}
		if err := seedBaseData(db, cfg, sf); err != nil {
			log.Fatal(err)
		}
		if err := seedSystemTableMetadata(db, sf); err != nil {
			log.Fatal(err)
		}
		if err := seedSystemTableRelations(db, sf); err != nil {
			log.Fatal(err)
		}
		if err := flushMigrationCaches(cfg); err != nil {
			log.Fatal(err)
		}
		log.Println("base seed completed")
	default:
		log.Fatalf("unknown migrate command %q, use migrate or seed", command)
	}
}

func migrationCommand(args []string) string {
	if len(args) < 2 || strings.TrimSpace(args[1]) == "" {
		return "migrate"
	}
	switch strings.TrimSpace(args[1]) {
	case "schema":
		return "migrate"
	default:
		return strings.TrimSpace(args[1])
	}
}

func openPrimaryDB(cfg *config.Server) (*gorm.DB, error) {
	dbCfg := cfg.DBS.Primary
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Shanghai", dbCfg.Host, dbCfg.Port, dbCfg.User, dbCfg.Password, dbCfg.Name)
	dbLogger := logger.New(log.Default(), logger.Config{
		SlowThreshold:             time.Second,
		Colorful:                  false,
		IgnoreRecordNotFoundError: true,
		LogLevel:                  logger.Info,
	})
	var lastErr error
	for i := 0; i < 30; i++ {
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				TablePrefix:   dbCfg.Prefix,
				SingularTable: true,
			},
			DisableForeignKeyConstraintWhenMigrating: true,
			Logger:                                   dbLogger,
			NowFunc:                                  model.Now,
		})
		if err == nil {
			sqlDB, dbErr := db.DB()
			if dbErr != nil {
				return nil, dbErr
			}
			sqlDB.SetMaxIdleConns(10)
			sqlDB.SetMaxOpenConns(100)
			sqlDB.SetConnMaxLifetime(time.Hour)
			return db, nil
		}
		lastErr = err
		time.Sleep(2 * time.Second)
	}
	return nil, lastErr
}

func migrateSchema(db *gorm.DB) error {
	err := db.AutoMigrate(
		&model.SysConfigure{},
		&model.SysTable{},
		&model.SysTableField{},
		&model.SysTableRelation{},
		&model.SysTableIndex{},
		&model.SysTableIndexField{},
		&model.SysDict{},
		&model.SysDictItem{},
		&model.AccessLog{},
		&model.LoginLog{},
		&model.SysUser{},
		&model.SysUserRole{},
		&model.SysMenu{},
		&model.SysMenuButton{},
		&model.SysMenuButtonTemplate{},
		&model.SysRole{},
		&model.SysRoleMenu{},
		&model.SysRoleMenuButton{},
		&model.SysDataDimension{},
		&model.SysDataScopeBinding{},
		&model.SysRoleDataScope{},
		&model.SysUserDataScopeOverride{},
		&model.SysUserDimensionValue{},
		&model.Application{},
		&model.SmsTemplate{},
		&model.SmsLog{},
		&model.File{},
		&model.FileChunk{},
		&model.CasbinRule{},
	)
	if err != nil {
		return err
	}
	return ensureDataPermissionIndexes(db)
}

func ensureDataPermissionIndexes(db *gorm.DB) error {
	if db.Dialector.Name() != "postgres" {
		return nil
	}
	table := db.NamingStrategy.TableName("SysUserDimensionValue")
	if err := db.Exec(`DROP INDEX IF EXISTS uni_user_dimension_value`).Error; err != nil {
		return err
	}
	sql := fmt.Sprintf(
		`CREATE UNIQUE INDEX IF NOT EXISTS uni_user_dimension_value_active ON "%s" ("user_id", "dimension_code") WHERE "gmt_delete" IS NULL`,
		table,
	)
	return db.Exec(sql).Error
}

func backfillSysMenuPageBinding(db *gorm.DB) error {
	var menus []model.SysMenu
	if err := db.Find(&menus).Error; err != nil {
		return err
	}
	for _, item := range menus {
		updates := make(map[string]interface{})
		pageType := inferMenuPageType(item)
		if item.PageType != pageType {
			updates["page_type"] = pageType
		}
		if strings.TrimSpace(item.TableCode) == "" {
			if tableCode := singleTableCode(item.Option); tableCode != "" {
				updates["table_code"] = tableCode
			}
		}
		if len(updates) == 0 {
			continue
		}
		if err := db.Model(&model.SysMenu{}).Where("id = ?", item.Id).Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}

func inferMenuPageType(menu model.SysMenu) enum.SysMenuPageType {
	if menu.Component == "src/components/Layout/Layout.vue" {
		return enum.MenuPageTypeDirectory
	}
	if menu.Component == "pages/develop/generalization/Index.vue" &&
		(strings.TrimSpace(menu.Option) != "" || strings.TrimSpace(menu.TableCode) != "") {
		return enum.MenuPageTypeLowCode
	}
	if menu.PageType != "" {
		return menu.PageType
	}
	return enum.MenuPageTypeFixed
}

func singleTableCode(option string) string {
	items := strings.FieldsFunc(strings.TrimSpace(option), func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\t' || r == ' '
	})
	if len(items) != 1 {
		return ""
	}
	return strings.TrimSpace(items[0])
}

func flushMigrationCaches(cfg *config.Server) error {
	if cfg == nil || strings.TrimSpace(cfg.Redis.Host) == "" || cfg.Redis.Port <= 0 {
		return nil
	}
	redisCfg := cfg.Redis
	client := redis.NewClient(&redis.Options{
		Addr:            fmt.Sprintf("%s:%d", redisCfg.Host, redisCfg.Port),
		Password:        redisCfg.Password,
		DB:              redisCfg.DB,
		PoolSize:        redisCfg.PoolSize,
		MinIdleConns:    redisCfg.MinIdleConns,
		ConnMaxIdleTime: time.Duration(redisCfg.ConnMaxIdleTime) * time.Second,
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("connect redis for migration cache flush: %w", err)
	}
	for _, prefix := range migrationCachePrefixes() {
		if err := deleteRedisKeysByPrefix(ctx, client, prefix); err != nil {
			return err
		}
	}
	return nil
}

func migrationCachePrefixes() []string {
	return []string{
		cache.ApplicationCacheKey,
		cache.ConfigureCacheKey,
		cache.DictCacheKey,
		cache.GeneralizationCacheKey,
		cache.MenuButtonCacheKey,
		cache.MenuCacheKey,
		cache.RoleCacheKey,
		cache.RoleMenuButtonCacheKey,
		cache.RoleMenuCacheKey,
		cache.TableCacheKey,
		cache.TableFieldCacheKey,
		cache.UserCacheKey,
		cache.UserRoleCacheKey,
	}
}

func deleteRedisKeysByPrefix(ctx context.Context, client *redis.Client, prefix string) error {
	var cursor uint64
	pattern := prefix + "*"
	for {
		keys, nextCursor, err := client.Scan(ctx, cursor, pattern, 200).Result()
		if err != nil {
			return fmt.Errorf("scan redis cache prefix %q: %w", prefix, err)
		}
		if len(keys) > 0 {
			if err := client.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("delete redis cache prefix %q: %w", prefix, err)
			}
		}
		if nextCursor == 0 {
			return nil
		}
		cursor = nextCursor
	}
}

func newMigrationID(sf *utils.Snowflake) (int, error) {
	id, err := sf.GenerateUniqueID()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func seedBaseData(db *gorm.DB, cfg *config.Server, sf *utils.Snowflake) error {
	if err := seedConfigure(db); err != nil {
		return err
	}
	if err := seedDicts(db, sf); err != nil {
		return err
	}
	if err := seedApplication(db); err != nil {
		return err
	}
	if err := seedAdminUser(db, cfg); err != nil {
		return err
	}
	if err := seedMenusAndRole(db, sf); err != nil {
		return err
	}
	if err := seedLowCodeMenuButtonTemplates(db, sf); err != nil {
		return err
	}
	return repairSeededMenuButtonDefaults(db)
}

func repairSeededMenuButtonDefaults(db *gorm.DB) error {
	if err := db.Model(&model.SysMenuButton{}).
		Where("display_mode = '' OR display_mode IS NULL").
		Update("display_mode", enum.ButtonDisplayAuto).Error; err != nil {
		return err
	}
	return db.Model(&model.SysMenuButtonTemplate{}).
		Where("display_mode = '' OR display_mode IS NULL").
		Update("display_mode", enum.ButtonDisplayAuto).Error
}

func seedConfigure(db *gorm.DB) error {
	cfg := model.SysConfigure{
		Basic:               model.Basic{Id: 1, State: true},
		EnableCaptcha:       false,
		PasswordLength:      8,
		PasswordComplexity:  2,
		PasswordExpireTime:  90,
		PasswordErrorCount:  5,
		PasswordLockMinutes: 15,
		PasswordPolicy:      "medium",
		SystemName:          "Sweet Admin",
		SystemVersion:       "0.1",
		SystemDescription:   "通用低代码底座",
		EnableEmail:         false,
		SmtpPort:            465,
	}
	var existing model.SysConfigure
	err := db.First(&existing).Error
	if err == nil {
		updates := map[string]interface{}{}
		if existing.PasswordLength <= 0 {
			updates["password_length"] = cfg.PasswordLength
		}
		if existing.PasswordComplexity <= 0 {
			updates["password_complexity"] = cfg.PasswordComplexity
		}
		if existing.PasswordExpireTime <= 0 {
			updates["password_expire_time"] = cfg.PasswordExpireTime
		}
		if existing.PasswordErrorCount <= 0 {
			updates["password_error_count"] = cfg.PasswordErrorCount
		}
		if existing.PasswordLockMinutes <= 0 {
			updates["password_lock_minutes"] = cfg.PasswordLockMinutes
		}
		if strings.TrimSpace(existing.PasswordPolicy) == "" {
			updates["password_policy"] = cfg.PasswordPolicy
		}
		if strings.TrimSpace(existing.SystemName) == "" {
			updates["system_name"] = cfg.SystemName
		}
		if strings.TrimSpace(existing.SystemVersion) == "" {
			updates["system_version"] = cfg.SystemVersion
		}
		if strings.TrimSpace(existing.SystemDescription) == "" {
			updates["system_description"] = cfg.SystemDescription
		}
		if existing.SmtpPort <= 0 {
			updates["smtp_port"] = cfg.SmtpPort
		}
		if len(updates) == 0 {
			return nil
		}
		updates["gmt_modify"] = model.Now()
		return db.Model(&model.SysConfigure{}).Where("id = ?", existing.Id).Updates(updates).Error
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}
	return db.Create(&cfg).Error
}

func seedDicts(db *gorm.DB, sf *utils.Snowflake) error {
	dicts := []systemDictSeed{
		{
			name: "是否",
			code: "whether",
			items: []systemDictItemSeed{
				{name: "是", code: "whether_yes", value: "true"},
				{name: "否", code: "whether_no", value: "false"},
			},
		},
		{
			name: "表类型",
			code: "sys_table_type",
			items: []systemDictItemSeed{
				{name: "系统表", code: "sys_table_type_system", value: "1"},
				{name: "视图", code: "sys_table_type_view", value: "2"},
			},
		},
		{
			name: "字段类型",
			code: "sys_table_field_type",
			items: []systemDictItemSeed{
				{name: "大数字", code: "sys_table_field_type_bigint", value: "1"},
				{name: "浮点", code: "sys_table_field_type_float", value: "2"},
				{name: "字符串", code: "sys_table_field_type_varchar", value: "3"},
				{name: "文本", code: "sys_table_field_type_text", value: "4"},
				{name: "布尔", code: "sys_table_field_type_boolean", value: "5"},
				{name: "日期", code: "sys_table_field_type_date", value: "6"},
				{name: "日期时间", code: "sys_table_field_type_datetime", value: "7"},
				{name: "时间", code: "sys_table_field_type_time", value: "8"},
				{name: "微型整数", code: "sys_table_field_type_tinyint", value: "9"},
				{name: "JSON", code: "sys_table_field_type_json", value: "10"},
				{name: "数字", code: "sys_table_field_type_int", value: "11"},
			},
		},
		{
			name: "输入类型",
			code: "sys_table_field_input_type",
			items: []systemDictItemSeed{
				{name: "输入框", code: "sys_table_field_input_type_input", value: "1"},
				{name: "数字输入", code: "sys_table_field_input_type_number", value: "2"},
				{name: "多行文本", code: "sys_table_field_input_type_textarea", value: "3"},
				{name: "下拉选择", code: "sys_table_field_input_type_select", value: "4"},
				{name: "日期选择", code: "sys_table_field_input_type_date", value: "5"},
				{name: "日期时间", code: "sys_table_field_input_type_datetime", value: "6"},
				{name: "时间选择", code: "sys_table_field_input_type_time", value: "7"},
				{name: "年份选择", code: "sys_table_field_input_type_year", value: "8"},
				{name: "年月选择", code: "sys_table_field_input_type_year_month", value: "9"},
				{name: "文件选择", code: "sys_table_field_input_type_file", value: "10"},
				{name: "布尔开关", code: "sys_table_field_input_type_boolean", value: "11"},
				{name: "JSON编辑器", code: "sys_table_field_input_type_json", value: "12"},
				{name: "数组输入", code: "sys_table_field_input_type_array", value: "13"},
				{name: "键值对编辑", code: "sys_table_field_input_type_key_value", value: "14"},
				{name: "级联选择", code: "sys_table_field_input_type_cascader", value: "15"},
				{name: "富文本编辑器", code: "sys_table_field_input_type_rich_text", value: "16"},
			},
		},
		{
			name: "菜单按钮位置",
			code: "sys_menu_button_position",
			items: []systemDictItemSeed{
				{name: "行按钮", code: "sys_menu_button_position_line", value: "1"},
				{name: "表格顶部", code: "sys_menu_button_position_top", value: "2"},
				{name: "表格底部", code: "sys_menu_button_position_bottom", value: "3"},
				{name: "表单顶部", code: "sys_menu_button_position_form_top", value: "4"},
				{name: "表单底部", code: "sys_menu_button_position_form_bottom", value: "5"},
				{name: "详情顶部", code: "sys_menu_button_position_detail_top", value: "6"},
				{name: "详情底部", code: "sys_menu_button_position_detail_bottom", value: "7"},
			},
		},
		{
			name: "菜单按钮展示方式",
			code: "sys_menu_button_display_mode",
			items: []systemDictItemSeed{
				{name: "自动", code: "sys_menu_button_display_mode_auto", value: string(enum.ButtonDisplayAuto)},
				{name: "仅图标", code: "sys_menu_button_display_mode_icon", value: string(enum.ButtonDisplayIcon)},
				{name: "仅文字", code: "sys_menu_button_display_mode_text", value: string(enum.ButtonDisplayText)},
				{name: "图标文字", code: "sys_menu_button_display_mode_icon_text", value: string(enum.ButtonDisplayIconText)},
			},
		},
		{
			name:  "菜单按钮事件动作",
			code:  "sys_menu_button_event_action",
			items: menuButtonEventActionDictItems(),
		},
		{
			name: "主子表展示模式",
			code: "sys_master_detail_mode",
			items: []systemDictItemSeed{
				{name: "自动", code: "sys_master_detail_mode_auto", value: string(enum.MasterDetailAuto)},
				{name: "摘要主表", code: "sys_master_detail_mode_summary", value: string(enum.MasterDetailSummary)},
				{name: "主表表格", code: "sys_master_detail_mode_table", value: string(enum.MasterDetailTable)},
				{name: "上下主子表", code: "sys_master_detail_mode_stacked", value: string(enum.MasterDetailStacked)},
			},
		},
		{
			name: "表单打开方式",
			code: "sys_form_open_mode",
			items: []systemDictItemSeed{
				{name: "自动", code: "sys_form_open_mode_auto", value: string(enum.FormOpenAuto)},
				{name: "弹框", code: "sys_form_open_mode_dialog", value: string(enum.FormOpenDialog)},
				{name: "页签", code: "sys_form_open_mode_page", value: string(enum.FormOpenPage)},
			},
		},
		{
			name: "详情打开方式",
			code: "sys_detail_open_mode",
			items: []systemDictItemSeed{
				{name: "自动", code: "sys_detail_open_mode_auto", value: string(enum.DetailOpenAuto)},
				{name: "弹框", code: "sys_detail_open_mode_dialog", value: string(enum.DetailOpenDialog)},
				{name: "页签", code: "sys_detail_open_mode_page", value: string(enum.DetailOpenPage)},
			},
		},
		{
			name: "HTTP方法",
			code: "http_method",
			items: []systemDictItemSeed{
				{name: "GET", code: "http_method_get", value: "GET"},
				{name: "POST", code: "http_method_post", value: "POST"},
				{name: "PUT", code: "http_method_put", value: "PUT"},
				{name: "DELETE", code: "http_method_delete", value: "DELETE"},
				{name: "PATCH", code: "http_method_patch", value: "PATCH"},
			},
		},
		{
			name: "表关系类型",
			code: "sys_table_relation_type",
			items: []systemDictItemSeed{
				{name: "一对一", code: "sys_table_relation_type_one_to_one", value: "1"},
				{name: "一对多", code: "sys_table_relation_type_one_to_many", value: "2"},
				{name: "多对一", code: "sys_table_relation_type_many_to_one", value: "3"},
				{name: "多对多", code: "sys_table_relation_type_many_to_many", value: "4"},
			},
		},
		{
			name: "字段类别",
			code: "sys_table_field_category",
			items: []systemDictItemSeed{
				{name: "普通字段", code: "sys_table_field_category_normal", value: "normal_field"},
				{name: "虚拟列", code: "sys_table_field_category_virtual", value: "virtual_field"},
				{name: "计算字段", code: "sys_table_field_category_calculated", value: "calculated_field"},
			},
		},
	}
	for _, dict := range dicts {
		if err := seedSystemDict(db, sf, dict); err != nil {
			return err
		}
	}
	return nil
}

func menuButtonEventActionDictItems() []systemDictItemSeed {
	return []systemDictItemSeed{
		{name: "查询", code: "sys_menu_button_event_action_query", value: string(enum.ButtonActionQuery)},
		{name: "页面元数据", code: "sys_menu_button_event_action_metadata", value: string(enum.ButtonActionMetadata)},
		{name: "详情", code: "sys_menu_button_event_action_detail", value: string(enum.ButtonActionDetail)},
		{name: "新增", code: "sys_menu_button_event_action_create", value: string(enum.ButtonActionCreate)},
		{name: "新增子级", code: "sys_menu_button_event_action_create_child", value: string(enum.ButtonActionCreateChild)},
		{name: "编辑", code: "sys_menu_button_event_action_update", value: string(enum.ButtonActionUpdate)},
		{name: "删除", code: "sys_menu_button_event_action_delete", value: string(enum.ButtonActionDelete)},
		{name: "刷新", code: "sys_menu_button_event_action_refresh", value: string(enum.ButtonActionRefresh)},
		{name: "批量删除", code: "sys_menu_button_event_action_batch_delete", value: string(enum.ButtonActionBatchDelete)},
		{name: "复制", code: "sys_menu_button_event_action_copy", value: string(enum.ButtonActionCopy)},
		{name: "复制记录", code: "sys_menu_button_event_action_duplicate", value: string(enum.ButtonActionDuplicate)},
		{name: "导出", code: "sys_menu_button_event_action_export", value: string(enum.ButtonActionExport)},
		{name: "页面跳转", code: "sys_menu_button_event_action_navigate", value: string(enum.ButtonActionNavigate)},
		{name: "自定义", code: "sys_menu_button_event_action_custom", value: string(enum.ButtonActionCustom)},
		{name: "保存", code: "sys_menu_button_event_action_save", value: string(enum.ButtonActionSave)},
		{name: "排序", code: "sys_menu_button_event_action_order", value: string(enum.ButtonActionOrder)},
		{name: "刷新缓存", code: "sys_menu_button_event_action_refresh_cache", value: string(enum.ButtonActionRefreshCache)},
		{name: "测试邮件", code: "sys_menu_button_event_action_test_email", value: string(enum.ButtonActionTestEmail)},
		{name: "新增按钮", code: "sys_menu_button_event_action_create_button", value: string(enum.ButtonActionCreateButton)},
		{name: "编辑按钮", code: "sys_menu_button_event_action_update_button", value: string(enum.ButtonActionUpdateButton)},
		{name: "删除按钮", code: "sys_menu_button_event_action_delete_button", value: string(enum.ButtonActionDeleteButton)},
		{name: "按钮查询", code: "sys_menu_button_event_action_query_button", value: string(enum.ButtonActionQueryButton)},
		{name: "按钮元数据", code: "sys_menu_button_event_action_button_metadata", value: string(enum.ButtonActionButtonMetadata)},
		{name: "新增字典项", code: "sys_menu_button_event_action_create_item", value: string(enum.ButtonActionCreateItem)},
		{name: "编辑字典项", code: "sys_menu_button_event_action_update_item", value: string(enum.ButtonActionUpdateItem)},
		{name: "删除字典项", code: "sys_menu_button_event_action_delete_item", value: string(enum.ButtonActionDeleteItem)},
		{name: "字典项查询", code: "sys_menu_button_event_action_query_item", value: string(enum.ButtonActionQueryItem)},
		{name: "字典项详情", code: "sys_menu_button_event_action_detail_item", value: string(enum.ButtonActionDetailItem)},
		{name: "字典项元数据", code: "sys_menu_button_event_action_item_metadata", value: string(enum.ButtonActionItemMetadata)},
		{name: "分配权限", code: "sys_menu_button_event_action_assign_permission", value: string(enum.ButtonActionAssignPermission)},
		{name: "分配数据权限", code: "sys_menu_button_event_action_assign_data_permission", value: string(enum.ButtonActionAssignData)},
		{name: "用户菜单查询", code: "sys_menu_button_event_action_query_user_menu", value: string(enum.ButtonActionQueryUserMenu)},
		{name: "数据权限查询", code: "sys_menu_button_event_action_query_data_permission", value: string(enum.ButtonActionQueryDataPerm)},
		{name: "授权菜单查询", code: "sys_menu_button_event_action_query_permission_menu", value: string(enum.ButtonActionQueryPermMenu)},
		{name: "重置密码", code: "sys_menu_button_event_action_reset_password", value: string(enum.ButtonActionResetPassword)},
		{name: "解除锁定", code: "sys_menu_button_event_action_unlock_login", value: string(enum.ButtonActionUnlockLogin)},
		{name: "轮换密钥", code: "sys_menu_button_event_action_rotate_secret", value: string(enum.ButtonActionRotateSecret)},
		{name: "发布", code: "sys_menu_button_event_action_publish", value: string(enum.ButtonActionPublish)},
		{name: "取消发布", code: "sys_menu_button_event_action_unpublish", value: string(enum.ButtonActionUnpublish)},
		{name: "初始化元数据", code: "sys_menu_button_event_action_init_meta", value: string(enum.ButtonActionInitMeta)},
		{name: "同步字段", code: "sys_menu_button_event_action_sync_fields", value: string(enum.ButtonActionSyncFields)},
		{name: "同步索引", code: "sys_menu_button_event_action_sync_index", value: string(enum.ButtonActionSyncIndex)},
		{name: "字段管理", code: "sys_menu_button_event_action_field_manager", value: string(enum.ButtonActionFieldManager)},
		{name: "新增字段", code: "sys_menu_button_event_action_create_field", value: string(enum.ButtonActionCreateField)},
		{name: "编辑字段", code: "sys_menu_button_event_action_update_field", value: string(enum.ButtonActionUpdateField)},
		{name: "删除字段", code: "sys_menu_button_event_action_delete_field", value: string(enum.ButtonActionDeleteField)},
		{name: "字段列表", code: "sys_menu_button_event_action_query_field", value: string(enum.ButtonActionQueryField)},
		{name: "字段详情", code: "sys_menu_button_event_action_detail_field", value: string(enum.ButtonActionDetailField)},
		{name: "新增索引", code: "sys_menu_button_event_action_create_index", value: string(enum.ButtonActionCreateIndex)},
		{name: "编辑索引", code: "sys_menu_button_event_action_update_index", value: string(enum.ButtonActionUpdateIndex)},
		{name: "删除索引", code: "sys_menu_button_event_action_delete_index", value: string(enum.ButtonActionDeleteIndex)},
		{name: "索引列表", code: "sys_menu_button_event_action_query_index", value: string(enum.ButtonActionQueryIndex)},
		{name: "索引详情", code: "sys_menu_button_event_action_detail_index", value: string(enum.ButtonActionDetailIndex)},
		{name: "新增关系", code: "sys_menu_button_event_action_create_relation", value: string(enum.ButtonActionCreateRelation)},
		{name: "编辑关系", code: "sys_menu_button_event_action_update_relation", value: string(enum.ButtonActionUpdateRelation)},
		{name: "删除关系", code: "sys_menu_button_event_action_delete_relation", value: string(enum.ButtonActionDeleteRelation)},
		{name: "关系列表", code: "sys_menu_button_event_action_query_relation", value: string(enum.ButtonActionQueryRelation)},
		{name: "关系详情", code: "sys_menu_button_event_action_detail_relation", value: string(enum.ButtonActionDetailRelation)},
	}
}

type systemDictSeed struct {
	name  string
	code  string
	items []systemDictItemSeed
}

type systemDictItemSeed struct {
	name  string
	code  string
	value string
}

func seedSystemDict(db *gorm.DB, sf *utils.Snowflake, seed systemDictSeed) error {
	var dict model.SysDict
	err := db.Where("dict_code = ?", seed.code).First(&dict).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		id, err := newMigrationID(sf)
		if err != nil {
			return err
		}
		dict = model.SysDict{
			Basic:    model.Basic{Id: id, State: true},
			DictName: seed.name,
			DictCode: seed.code,
		}
		if err := db.Create(&dict).Error; err != nil {
			return err
		}
	}
	for _, item := range seed.items {
		var existing model.SysDictItem
		err := db.Where("item_code = ?", item.code).First(&existing).Error
		if err == nil {
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}
		id, err := newMigrationID(sf)
		if err != nil {
			return err
		}
		existing = model.SysDictItem{
			Basic:     model.Basic{Id: id, State: true},
			DictId:    dict.Id,
			ItemName:  item.name,
			ItemCode:  item.code,
			ItemValue: item.value,
		}
		if err := db.Create(&existing).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedApplication(db *gorm.DB) error {
	var app model.Application
	err := db.Where("app_key = ?", "sweet-admin").First(&app).Error
	if err == nil {
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}
	app = model.Application{
		Basic:      model.Basic{Id: 1, State: true},
		Name:       "Default Admin App",
		AppKey:     "sweet-admin",
		AppSecret:  "sweet-admin-secret",
		Expiration: 7200,
		Remark:     "本地开发默认应用",
	}
	return db.Create(&app).Error
}

func seedAdminUser(db *gorm.DB, cfg *config.Server) error {
	now := model.CustomTime(model.Now())
	rawPassword, err := bootstrapAdminPassword()
	if err != nil {
		return err
	}
	password := utils.Encryption(rawPassword, "1"+cfg.Conf.Salt)
	var user model.SysUser
	err = db.Where("user_name = ?", "admin").First(&user).Error
	if err == nil {
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}
	user = model.SysUser{
		Basic:             model.Basic{Id: 1, State: true},
		UserName:          "admin",
		Password:          password,
		Email:             "admin@example.com",
		PhoneNumber:       "13800000000",
		PasswordChangedAt: &now,
		Language:          "zh-CN",
		IsReset:           false,
	}
	return db.Create(&user).Error
}

func bootstrapAdminPassword() (string, error) {
	password := strings.TrimSpace(os.Getenv("APP_BOOTSTRAP_ADMIN_PASSWORD"))
	if password == "" {
		password = "admin123"
	}
	if requiresSecureBootstrap() && insecureBootstrapAdminPassword(password) {
		return "", fmt.Errorf("insecure bootstrap admin password: set APP_BOOTSTRAP_ADMIN_PASSWORD to a non-default value with at least 12 characters")
	}
	return password, nil
}

func requiresSecureBootstrap() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))) {
	case "pro", "prod", "production":
		return true
	}
	return parseBoolEnv(os.Getenv("APP_REQUIRE_SECURE_CONFIG"))
}

func parseBoolEnv(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func insecureBootstrapAdminPassword(password string) bool {
	normalized := strings.ToLower(strings.TrimSpace(password))
	if len(normalized) < 12 {
		return true
	}
	defaults := []string{"admin", "admin123", "password", "123456", "change-me"}
	for _, item := range defaults {
		if normalized == item || strings.Contains(normalized, item) {
			return true
		}
	}
	return false
}

func seedMenusAndRole(db *gorm.DB, sf *utils.Snowflake) error {
	menus := []model.SysMenu{
		menu(100, 0, "home", "home", "pages/dashboard/Dashboard.vue", "router.home", "home", 1),
		directoryMenu(menu(200, 0, "system", "system", "src/components/Layout/Layout.vue", "router.system.default", "settings", 2)),
		menuWithTable(menu(201, 200, "system_application", "application", "pages/system/application/Index.vue", "router.system.application", "apps", 1), "application"),
		menuWithTable(menu(202, 200, "system_sms", "sms", "pages/system/sms/Index.vue", "router.system.sms", "sms", 2), "sms_template"),
		menuWithTable(menu(203, 200, "system_menu", "menu", "pages/system/menu/Index.vue", "router.system.menu", "menu", 3), "sys_menu"),
		menuWithTable(menu(204, 200, "system_role", "role", "pages/system/role/Index.vue", "router.system.role", "admin_panel_settings", 4), "sys_role"),
		menuWithTable(menu(205, 200, "system_user", "user", "pages/system/user/Index.vue", "router.system.user", "person", 5), "sys_user"),
		menu(207, 200, "system_data_permission", "data-permission", "pages/system/data-permission/Index.vue", "数据权限", "rule", 6),
		menuWithTable(menu(206, 200, "system_audit", "audit", "pages/system/audit/Index.vue", "router.system.audit", "manage_search", 7), "access_log"),
		directoryMenu(menu(300, 0, "develop", "develop", "src/components/Layout/Layout.vue", "router.develop.default", "developer_mode", 3)),
		menu(301, 300, "develop_configure", "configure", "pages/develop/configure/Index.vue", "router.develop.configure", "tune", 1),
		menu(302, 300, "develop_generalization", "generalization/:table_code", "pages/develop/generalization/Index.vue", "router.develop.generalization", "dynamic_form", 2),
		menuWithOption(menu(303, 300, "develop_database", "database", "pages/develop/database/Index.vue", "router.develop.database", "storage", 3), "sys_table,sys_table_field,sys_table_index,sys_table_relation"),
		menuWithOption(menu(304, 300, "develop_dictionary", "dictionary", "pages/develop/dictionary/Index.vue", "router.develop.dictionary", "menu_book", 4), "sys_dict,sys_dict_item"),
	}
	menuByName := make(map[string]model.SysMenu, len(menus))
	for _, item := range menus {
		seeded, err := seedMenu(db, sf, item)
		if err != nil {
			return err
		}
		menuByName[seeded.Name] = seeded
	}
	role, err := seedRole(db, sf)
	if err != nil {
		return err
	}
	var admin model.SysUser
	if err := db.Where("user_name = ?", "admin").First(&admin).Error; err != nil {
		return err
	}
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.SysUserRole{UserId: admin.Id, RoleId: role.Id}).Error; err != nil {
		return err
	}
	for _, item := range menuByName {
		if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.SysRoleMenu{RoleId: role.Id, MenuId: item.Id}).Error; err != nil {
			return err
		}
	}
	if err := seedBuiltinMenuButtons(db, sf, role.Id, role.Name, menuByName); err != nil {
		return err
	}
	if err := seedSuperAdminRoutePolicies(db, role.Name); err != nil {
		return err
	}
	return nil
}

func seedBuiltinMenuButtons(db *gorm.DB, sf *utils.Snowflake, roleID int, roleName string, menuByName map[string]model.SysMenu) error {
	configureMenu, ok := menuByName["develop_configure"]
	if !ok {
		return fmt.Errorf("develop_configure menu missing after seed")
	}
	if err := seedConfigureMenuButtons(db, sf, roleID, roleName, configureMenu.Id); err != nil {
		return err
	}
	applicationMenu, ok := menuByName["system_application"]
	if !ok {
		return fmt.Errorf("system_application menu missing after seed")
	}
	if err := seedApplicationMenuButtons(db, sf, roleID, roleName, applicationMenu.Id); err != nil {
		return err
	}
	smsMenu, ok := menuByName["system_sms"]
	if !ok {
		return fmt.Errorf("system_sms menu missing after seed")
	}
	if err := seedSmsMenuButtons(db, sf, roleID, roleName, smsMenu.Id); err != nil {
		return err
	}
	menuMenu, ok := menuByName["system_menu"]
	if !ok {
		return fmt.Errorf("system_menu menu missing after seed")
	}
	if err := seedSystemMenuButtons(db, sf, roleID, roleName, menuMenu.Id); err != nil {
		return err
	}
	roleMenu, ok := menuByName["system_role"]
	if !ok {
		return fmt.Errorf("system_role menu missing after seed")
	}
	if err := seedRoleMenuButtons(db, sf, roleID, roleName, roleMenu.Id); err != nil {
		return err
	}
	userMenu, ok := menuByName["system_user"]
	if !ok {
		return fmt.Errorf("system_user menu missing after seed")
	}
	if err := seedUserMenuButtons(db, sf, roleID, roleName, userMenu.Id); err != nil {
		return err
	}
	dataPermissionMenu, ok := menuByName["system_data_permission"]
	if !ok {
		return fmt.Errorf("system_data_permission menu missing after seed")
	}
	if err := seedDataPermissionMenuButtons(db, sf, roleID, roleName, dataPermissionMenu.Id); err != nil {
		return err
	}
	developDatabaseMenu, ok := menuByName["develop_database"]
	if !ok {
		return fmt.Errorf("develop_database menu missing after seed")
	}
	if err := seedDatabaseMenuButtons(db, sf, roleID, roleName, developDatabaseMenu.Id); err != nil {
		return err
	}
	dictionaryMenu, ok := menuByName["develop_dictionary"]
	if !ok {
		return fmt.Errorf("develop_dictionary menu missing after seed")
	}
	if err := seedDictionaryMenuButtons(db, sf, roleID, roleName, dictionaryMenu.Id); err != nil {
		return err
	}
	auditMenu, ok := menuByName["system_audit"]
	if !ok {
		return fmt.Errorf("system_audit menu missing after seed")
	}
	if err := seedAuditMenuButtons(db, sf, roleID, roleName, auditMenu.Id); err != nil {
		return err
	}
	return nil
}

func seedMenu(db *gorm.DB, sf *utils.Snowflake, item model.SysMenu) (model.SysMenu, error) {
	var existing model.SysMenu
	err := db.Where("name = ?", item.Name).First(&existing).Error
	if err == nil {
		updates := map[string]interface{}{
			"pid":        item.Pid,
			"path":       item.Path,
			"component":  item.Component,
			"title":      item.Title,
			"is_hidden":  item.IsHidden,
			"sequence":   item.Sequence,
			"option":     item.Option,
			"page_type":  item.PageType,
			"table_code": item.TableCode,
			"icon":       item.Icon,
			"redirect":   item.Redirect,
			"is_unfold":  item.IsUnfold,
			"gmt_modify": model.Now(),
		}
		if err := db.Model(&model.SysMenu{}).Where("id = ?", existing.Id).Updates(updates).Error; err != nil {
			return model.SysMenu{}, err
		}
		item.Id = existing.Id
		return item, nil
	}
	if err != gorm.ErrRecordNotFound {
		return model.SysMenu{}, err
	}
	id, err := seedPrimaryId(db, &model.SysMenu{}, item.Id, sf)
	if err != nil {
		return model.SysMenu{}, err
	}
	item.Id = id
	if err := db.Create(&item).Error; err != nil {
		return model.SysMenu{}, err
	}
	return item, nil
}

func seedRole(db *gorm.DB, sf *utils.Snowflake) (model.SysRole, error) {
	var role model.SysRole
	err := db.Where("name = ?", "super_admin").First(&role).Error
	if err == nil {
		if err := db.Model(&model.SysRole{}).Where("id = ?", role.Id).Updates(map[string]interface{}{
			"memo":       "超级管理员",
			"state":      true,
			"gmt_modify": model.Now(),
		}).Error; err != nil {
			return model.SysRole{}, err
		}
		return role, nil
	}
	if err != gorm.ErrRecordNotFound {
		return model.SysRole{}, err
	}
	id, err := seedPrimaryId(db, &model.SysRole{}, 1, sf)
	if err != nil {
		return model.SysRole{}, err
	}
	role = model.SysRole{
		Basic: model.Basic{Id: id, State: true},
		Name:  "super_admin",
		Memo:  "超级管理员",
	}
	if err := db.Create(&role).Error; err != nil {
		return model.SysRole{}, err
	}
	return role, nil
}

func seedPrimaryId(db *gorm.DB, modelValue interface{}, desired int, sf *utils.Snowflake) (int, error) {
	var count int64
	if err := db.Model(modelValue).Where("id = ?", desired).Count(&count).Error; err != nil {
		return 0, err
	}
	if count == 0 {
		return desired, nil
	}
	return newMigrationID(sf)
}

func migrationTimestamps(createAt, modifyAt model.CustomTime) (model.CustomTime, model.CustomTime) {
	now := model.CustomTime(model.Now())
	if createAt.IsZero() {
		createAt = now
	}
	if modifyAt.IsZero() {
		modifyAt = now
	}
	return createAt, modifyAt
}

func migrationMenuButtonCreateMap(button model.SysMenuButton) map[string]interface{} {
	gmtCreate, gmtModify := migrationTimestamps(button.GmtCreate, button.GmtModify)
	button = normalizeMigrationMenuButton(button)
	return map[string]interface{}{
		"id":            button.Id,
		"gmt_create":    gmtCreate,
		"create_user":   button.CreateUser,
		"create_name":   button.CreateName,
		"gmt_modify":    gmtModify,
		"modify_user":   button.ModifyUser,
		"modify_name":   button.ModifyName,
		"state":         true,
		"menu_id":       button.MenuId,
		"name":          button.Name,
		"code":          button.Code,
		"memo":          button.Memo,
		"position":      button.Position,
		"event_type":    button.EventType,
		"event_action":  button.EventAction,
		"icon":          button.Icon,
		"color":         button.Color,
		"display_mode":  button.DisplayMode,
		"sequence":      button.Sequence,
		"path":          button.Path,
		"method":        button.Method,
		"params_schema": button.ParamsSchema,
		"confirm_text":  button.ConfirmText,
		"disable_when":  button.DisableWhen,
		"is_button":     button.IsButton,
		"is_hidden":     button.IsHidden,
		"is_disabled":   button.IsDisabled,
		"before_hooks":  button.BeforeHooks,
		"after_hooks":   button.AfterHooks,
	}
}

func migrationMenuButtonTemplateCreateMap(template model.SysMenuButtonTemplate) map[string]interface{} {
	gmtCreate, gmtModify := migrationTimestamps(template.GmtCreate, template.GmtModify)
	template = normalizeMigrationMenuButtonTemplate(template)
	return map[string]interface{}{
		"id":            template.Id,
		"gmt_create":    gmtCreate,
		"create_user":   template.CreateUser,
		"create_name":   template.CreateName,
		"gmt_modify":    gmtModify,
		"modify_user":   template.ModifyUser,
		"modify_name":   template.ModifyName,
		"state":         true,
		"scene":         template.Scene,
		"name":          template.Name,
		"code_suffix":   template.CodeSuffix,
		"memo":          template.Memo,
		"position":      template.Position,
		"event_type":    template.EventType,
		"event_action":  template.EventAction,
		"icon":          template.Icon,
		"color":         template.Color,
		"display_mode":  template.DisplayMode,
		"sequence":      template.Sequence,
		"path":          template.Path,
		"method":        template.Method,
		"params_schema": template.ParamsSchema,
		"confirm_text":  template.ConfirmText,
		"disable_when":  template.DisableWhen,
		"is_button":     template.IsButton,
		"is_disabled":   template.IsDisabled,
		"before_hooks":  template.BeforeHooks,
		"after_hooks":   template.AfterHooks,
	}
}

func migrationTableFieldCreateMap(field model.SysTableField) map[string]interface{} {
	gmtCreate, gmtModify := migrationTimestamps(field.GmtCreate, field.GmtModify)
	return map[string]interface{}{
		"id":                   field.Id,
		"gmt_create":           gmtCreate,
		"create_user":          field.CreateUser,
		"create_name":          field.CreateName,
		"gmt_modify":           gmtModify,
		"modify_user":          field.ModifyUser,
		"modify_name":          field.ModifyName,
		"state":                true,
		"table_id":             field.TableId,
		"field_name":           field.FieldName,
		"field_code":           field.FieldCode,
		"field_type":           field.FieldType,
		"field_length":         field.FieldLength,
		"field_decimal_length": field.FieldDecimalLength,
		"input_type":           field.InputType,
		"form_span":            field.FormSpan,
		"detail_span":          field.DetailSpan,
		"default_value":        field.DefaultValue,
		"dict_code":            field.DictCode,
		"is_primary_key":       field.IsPrimaryKey,
		"is_index":             field.IsIndex,
		"is_quick_search":      field.IsQuickSearch,
		"is_advanced_search":   field.IsAdvancedSearch,
		"is_sort":              field.IsSort,
		"is_null":              field.IsNull,
		"is_list_show":         field.IsListShow,
		"is_insert_show":       field.IsInsertShow,
		"is_update_show":       field.IsUpdateShow,
		"sequence":             field.Sequence,
		"original_field_id":    field.OriginalFieldId,
		"binding":              field.Binding,
		"field_category":       field.FieldCategory,
		"expression":           field.Expression,
		"tag":                  field.Tag,
		"linkage_config":       field.LinkageConfig,
	}
}

func seedConfigureMenuButtons(db *gorm.DB, sf *utils.Snowflake, roleID int, roleName string, menuID int) error {
	buttons := []model.SysMenuButton{
		apiPermissionWithAPI(489, menuID, "配置详情", "develop_configure_detail", enum.Top, "detail", "visibility", "primary", 90, "/admin/configure/detail", "GET"),
		apiPermissionWithAPI(491, menuID, "测试邮件", "develop_configure_test_email", enum.Top, "test_email", "outgoing_mail", "primary", 91, "/admin/configure/test-email", "POST"),
		menuButtonWithAPI(414, menuID, "保存", "develop_configure_save", enum.Top, "save", "save", "primary", 1, "/admin/configure/:id", "PUT"),
	}
	return seedMenuButtons(db, sf, roleID, roleName, buttons)
}

func seedApplicationMenuButtons(db *gorm.DB, sf *utils.Snowflake, roleID int, roleName string, menuID int) error {
	buttons := []model.SysMenuButton{
		apiPermissionWithAPI(415, menuID, "列表查询", "system_application_query", enum.Top, "query", "search", "primary", 90, "/admin/application/query", "POST"),
		apiPermissionWithAPI(416, menuID, "页面元数据", "system_application_metadata", enum.Top, "metadata", "data_object", "primary", 91, "/admin/table/code/:code", "GET"),
		apiPermissionWithAPI(417, menuID, "详情", "system_application_detail", enum.Line, "detail", "visibility", "primary", 92, "/admin/application/:id", "GET"),
		menuButtonWithAPI(418, menuID, "新增", "system_application_create", enum.Top, "create", "add", "primary", 1, "/admin/application", "POST"),
		menuButtonWithAPI(419, menuID, "编辑", "system_application_update", enum.Line, "update", "edit", "primary", 1, "/admin/application/:id", "PUT"),
		menuButtonWithAPI(490, menuID, "轮换密钥", "system_application_rotate_secret", enum.Line, "rotate_secret", "vpn_key", "warning", 2, "/admin/application/:id/rotate-secret", "POST"),
		menuButtonWithAPI(420, menuID, "删除", "system_application_delete", enum.Line, "delete", "delete", "negative", 3, "/admin/application/:id", "DELETE"),
	}
	return seedMenuButtons(db, sf, roleID, roleName, buttons)
}

func seedSmsMenuButtons(db *gorm.DB, sf *utils.Snowflake, roleID int, roleName string, menuID int) error {
	buttons := []model.SysMenuButton{
		apiPermissionWithAPI(421, menuID, "列表查询", "system_sms_query", enum.Top, "query", "search", "primary", 90, "/admin/sms/template/query", "POST"),
		apiPermissionWithAPI(422, menuID, "页面元数据", "system_sms_metadata", enum.Top, "metadata", "data_object", "primary", 91, "/admin/table/code/:code", "GET"),
		apiPermissionWithAPI(423, menuID, "详情", "system_sms_detail", enum.Line, "detail", "visibility", "primary", 92, "/admin/sms/template/:id", "GET"),
		menuButtonWithAPI(424, menuID, "新增", "system_sms_create", enum.Top, "create", "add", "primary", 1, "/admin/sms/template", "POST"),
		menuButtonWithAPI(425, menuID, "编辑", "system_sms_update", enum.Line, "update", "edit", "primary", 1, "/admin/sms/template/:id", "PUT"),
		menuButtonWithAPI(426, menuID, "删除", "system_sms_delete", enum.Line, "delete", "delete", "negative", 2, "/admin/sms/template/:id", "DELETE"),
	}
	return seedMenuButtons(db, sf, roleID, roleName, buttons)
}

func seedSystemMenuButtons(db *gorm.DB, sf *utils.Snowflake, roleID int, roleName string, menuID int) error {
	buttons := []model.SysMenuButton{
		apiPermissionWithAPI(427, menuID, "列表查询", "system_menu_query", enum.Top, "query", "search", "primary", 90, "/admin/menu/query", "POST"),
		apiPermissionWithAPI(428, menuID, "菜单元数据", "system_menu_metadata", enum.Top, "metadata", "data_object", "primary", 91, "/admin/table/code/:code", "GET"),
		apiPermissionWithAPI(429, menuID, "按钮元数据", "system_menu_button_metadata", enum.Top, "button_metadata", "data_object", "primary", 92, "/admin/table/code/:code", "GET"),
		apiPermissionWithAPI(430, menuID, "详情", "system_menu_detail", enum.Line, "detail", "visibility", "primary", 93, "/admin/menu/:id", "GET"),
		apiPermissionWithAPI(431, menuID, "按钮查询", "system_menu_button_query", enum.Line, "query_button", "search", "primary", 94, "/admin/menu/buttons/:menuId", "GET"),
		menuButtonWithAPI(432, menuID, "新增", "system_menu_create", enum.Top, "create", "add", "primary", 1, "/admin/menu", "POST"),
		menuButtonWithAPI(433, menuID, "新增子菜单", "system_menu_create_child", enum.Line, "create_child", "subdirectory_arrow_right", "primary", 1, "/admin/menu", "POST"),
		menuButtonWithAPI(434, menuID, "复制", "system_menu_duplicate", enum.Line, "duplicate", "content_copy", "primary", 2, "/admin/menu", "POST"),
		menuButtonWithAPI(435, menuID, "编辑", "system_menu_update", enum.Line, "update", "edit", "primary", 3, "/admin/menu/:id", "PUT"),
		menuButtonWithAPI(436, menuID, "删除", "system_menu_delete", enum.Line, "delete", "delete", "negative", 4, "/admin/menu/:id", "DELETE"),
		menuButtonWithAPI(437, menuID, "新增按钮", "system_menu_button_create", enum.Top, "create_button", "add_circle", "primary", 2, "/admin/menu/button", "POST"),
		menuButtonWithAPI(438, menuID, "编辑按钮", "system_menu_button_update", enum.Line, "update_button", "edit", "primary", 5, "/admin/menu/button/:id", "PUT"),
		menuButtonWithAPI(439, menuID, "删除按钮", "system_menu_button_delete", enum.Line, "delete_button", "delete", "negative", 6, "/admin/menu/button/:id", "DELETE"),
		apiPermissionWithAPI(487, menuID, "排序", "system_menu_order", enum.Top, "order", "sort", "primary", 95, "/admin/menu/order", "PUT"),
		apiPermissionWithAPI(488, menuID, "刷新缓存", "system_menu_refresh_cache", enum.Top, "refresh_cache", "refresh", "primary", 96, "/admin/menu/refresh-cache", "POST"),
	}
	return seedMenuButtons(db, sf, roleID, roleName, buttons)
}

func seedRoleMenuButtons(db *gorm.DB, sf *utils.Snowflake, roleID int, roleName string, menuID int) error {
	buttons := []model.SysMenuButton{
		apiPermissionWithAPI(440, menuID, "列表查询", "system_role_query", enum.Top, "query", "search", "primary", 90, "/admin/role/query", "POST"),
		apiPermissionWithAPI(441, menuID, "页面元数据", "system_role_metadata", enum.Top, "metadata", "data_object", "primary", 91, "/admin/table/code/:code", "GET"),
		apiPermissionWithAPI(442, menuID, "详情", "system_role_detail", enum.Line, "detail", "visibility", "primary", 92, "/admin/role/:id", "GET"),
		apiPermissionWithAPI(443, menuID, "授权菜单查询", "system_role_permission_menu_query", enum.Line, "query_permission_menu", "account_tree", "primary", 93, "/admin/menu/query", "POST"),
		menuButtonWithAPI(444, menuID, "新增", "system_role_create", enum.Top, "create", "add", "primary", 1, "/admin/role", "POST"),
		menuButtonWithAPI(445, menuID, "编辑", "system_role_update", enum.Line, "update", "edit", "primary", 1, "/admin/role/:id", "PUT"),
		menuButtonWithAPI(446, menuID, "删除", "system_role_delete", enum.Line, "delete", "delete", "negative", 2, "/admin/role/:id", "DELETE"),
		menuButtonWithAPI(447, menuID, "分配权限", "system_role_assign_permission", enum.Line, "assign_permission", "admin_panel_settings", "primary", 3, "/admin/role/assign-permissions", "POST"),
		apiPermissionWithAPI(612, menuID, "角色数据权限查询", "system_role_data_permission_query", enum.Line, "query_data_permission", "rule", "primary", 94, "/admin/role/:id/data-permissions", "GET"),
		apiPermissionWithAPI(613, menuID, "角色数据权限保存", "system_role_data_permission_save", enum.Line, "save", "save", "primary", 95, "/admin/role/:id/data-permissions", "PUT"),
	}
	return seedMenuButtons(db, sf, roleID, roleName, buttons)
}

func seedDatabaseMenuButtons(db *gorm.DB, sf *utils.Snowflake, roleID int, roleName string, menuID int) error {
	buttons := []model.SysMenuButton{
		menuButtonWithAPI(401, menuID, "新增", "develop_database_create", enum.Top, "create", "add", "primary", 1, "/admin/table", "POST"),
		menuButtonWithAPI(402, menuID, "初始化元数据", "develop_database_init_meta", enum.Top, "init_meta", "data_object", "primary", 2, "/admin/table/init/:code", "GET"),
		menuButtonWithAPI(403, menuID, "发布", "develop_database_publish", enum.Line, "publish", "rocket_launch", "primary", 1, "/admin/table/publish/:code", "POST"),
		menuButtonWithAPI(408, menuID, "下线", "develop_database_unpublish", enum.Line, "unpublish", "visibility_off", "warning", 2, "/admin/table/unpublish/:code", "POST"),
		menuButton(404, menuID, "字段", "develop_database_field_manager", enum.Line, "field_manager", "view_column", "primary", 3),
		menuButtonWithAPI(405, menuID, "同步字段", "develop_database_sync_fields", enum.Line, "sync_fields", "sync", "primary", 4, "/admin/table/sync/:code", "POST"),
		menuButtonWithAPI(406, menuID, "编辑", "develop_database_update", enum.Line, "update", "edit", "primary", 5, "/admin/table/:id", "PUT"),
		menuButtonWithAPI(407, menuID, "删除", "develop_database_delete", enum.Line, "delete", "delete", "negative", 6, "/admin/table/:id", "DELETE"),
		apiPermissionWithAPI(468, menuID, "列表查询", "develop_database_query", enum.Top, "query", "search", "primary", 90, "/admin/table/query", "POST"),
		apiPermissionWithAPI(469, menuID, "表详情", "develop_database_detail", enum.Line, "detail", "visibility", "primary", 91, "/admin/table/id/:id", "GET"),
		apiPermissionWithAPI(470, menuID, "表元数据", "develop_database_metadata", enum.Top, "metadata", "data_object", "primary", 92, "/admin/table/code/:code", "GET"),
		apiPermissionWithAPI(471, menuID, "字段列表", "develop_database_field_query", enum.Line, "query_field", "view_column", "primary", 93, "/admin/table/fields/:id", "GET"),
		apiPermissionWithAPI(472, menuID, "字段详情", "develop_database_field_detail", enum.Line, "detail_field", "visibility", "primary", 94, "/admin/table/field/:id", "GET"),
		apiPermissionWithAPI(473, menuID, "新增字段", "develop_database_field_create", enum.Line, "create_field", "add", "primary", 95, "/admin/table/field", "POST"),
		apiPermissionWithAPI(474, menuID, "编辑字段", "develop_database_field_update", enum.Line, "update_field", "edit", "primary", 96, "/admin/table/field/:id", "PUT"),
		apiPermissionWithAPI(475, menuID, "删除字段", "develop_database_field_delete", enum.Line, "delete_field", "delete", "negative", 97, "/admin/table/field/:id", "DELETE"),
		apiPermissionWithAPI(476, menuID, "索引列表", "develop_database_index_query", enum.Line, "query_index", "toc", "primary", 98, "/admin/table/indexes/:id", "GET"),
		apiPermissionWithAPI(477, menuID, "索引详情", "develop_database_index_detail", enum.Line, "detail_index", "visibility", "primary", 99, "/admin/table/index/:id", "GET"),
		apiPermissionWithAPI(478, menuID, "新增索引", "develop_database_index_create", enum.Line, "create_index", "add", "primary", 100, "/admin/table/index", "POST"),
		apiPermissionWithAPI(479, menuID, "编辑索引", "develop_database_index_update", enum.Line, "update_index", "edit", "primary", 101, "/admin/table/index/:id", "PUT"),
		apiPermissionWithAPI(480, menuID, "删除索引", "develop_database_index_delete", enum.Line, "delete_index", "delete", "negative", 102, "/admin/table/index/:id", "DELETE"),
		apiPermissionWithAPI(481, menuID, "同步索引", "develop_database_index_sync", enum.Line, "sync_index", "sync", "primary", 103, "/admin/table/sync/index/:code", "POST"),
		apiPermissionWithAPI(482, menuID, "关系列表", "develop_database_relation_query", enum.Line, "query_relation", "device_hub", "primary", 104, "/admin/table/relations/:id", "GET"),
		apiPermissionWithAPI(483, menuID, "关系详情", "develop_database_relation_detail", enum.Line, "detail_relation", "visibility", "primary", 105, "/admin/table/relation/:id", "GET"),
		apiPermissionWithAPI(484, menuID, "新增关系", "develop_database_relation_create", enum.Line, "create_relation", "add", "primary", 106, "/admin/table/relation", "POST"),
		apiPermissionWithAPI(485, menuID, "编辑关系", "develop_database_relation_update", enum.Line, "update_relation", "edit", "primary", 107, "/admin/table/relation/:id", "PUT"),
		apiPermissionWithAPI(486, menuID, "删除关系", "develop_database_relation_delete", enum.Line, "delete_relation", "delete", "negative", 108, "/admin/table/relation/:id", "DELETE"),
	}
	return seedMenuButtons(db, sf, roleID, roleName, buttons)
}

func seedAuditMenuButtons(db *gorm.DB, sf *utils.Snowflake, roleID int, roleName string, menuID int) error {
	buttons := []model.SysMenuButton{
		menuButtonWithAPI(409, menuID, "查询", "system_audit_query", enum.Top, "query", "search", "primary", 1, "/admin/log/access/query", "POST"),
		menuButtonWithAPI(410, menuID, "详情", "system_audit_detail", enum.Line, "detail", "visibility", "primary", 1, "/admin/log/access/:id", "GET"),
		menuButton(493, menuID, "刷新详情", "system_audit_detail_refresh", enum.DetailTop, "refresh", "refresh", "primary", 1),
	}
	for _, button := range buttons {
		if err := seedMenuButton(db, sf, roleID, roleName, button); err != nil {
			return err
		}
	}
	return nil
}

func seedUserMenuButtons(db *gorm.DB, sf *utils.Snowflake, roleID int, roleName string, menuID int) error {
	buttons := []model.SysMenuButton{
		apiPermissionWithAPI(448, menuID, "列表查询", "system_user_query", enum.Top, "query", "search", "primary", 90, "/admin/user/query", "POST"),
		apiPermissionWithAPI(449, menuID, "页面元数据", "system_user_metadata", enum.Top, "metadata", "data_object", "primary", 91, "/admin/table/code/:code", "GET"),
		apiPermissionWithAPI(450, menuID, "详情", "system_user_detail", enum.Line, "detail", "visibility", "primary", 92, "/admin/user/:id", "GET"),
		menuButtonWithAPI(451, menuID, "新增", "system_user_create", enum.Top, "create", "add", "primary", 1, "/admin/user", "POST"),
		menuButtonWithAPI(452, menuID, "编辑", "system_user_update", enum.Line, "update", "edit", "primary", 1, "/admin/user/:id", "PUT"),
		menuButtonWithAPI(453, menuID, "删除", "system_user_delete", enum.Line, "delete", "delete", "negative", 2, "/admin/user/:id", "DELETE"),
		menuButtonWithAPI(454, menuID, "重置密码", "system_user_reset_password", enum.Line, "reset_password", "lock_reset", "warning", 3, "/admin/user/reset_password/:id", "POST"),
		menuButtonWithAPI(492, menuID, "解除锁定", "system_user_unlock_login", enum.Line, "unlock_login", "lock_open", "warning", 4, "/admin/user/unlock_login/:id", "POST"),
		menuButtonWithAPI(615, menuID, "分配角色", "system_user_assign_role", enum.Line, "assign_role", "supervisor_account", "primary", 5, "/admin/user/:id/roles", "PUT"),
		menuButtonWithAPI(411, menuID, "数据权限", "system_user_data_permission", enum.Line, "assign_data_permission", "shield", "primary", 6, "/admin/user/:id/data-permissions", "PUT"),
		menuButtonWithAPI(412, menuID, "用户菜单", "system_user_menu_query", enum.Line, "query_user_menu", "account_tree", "primary", 98, "/admin/menu/user/:id", "GET"),
		apiPermissionWithAPI(413, menuID, "数据权限查询", "system_user_data_permission_query", enum.Line, "query_data_permission", "search", "primary", 99, "/admin/user/:id/data-permissions", "GET"),
		apiPermissionWithAPI(616, menuID, "角色选项", "system_user_role_options", enum.Line, "query_role_options", "groups", "primary", 100, "/admin/role/query", "POST"),
		apiPermissionWithAPI(617, menuID, "用户归属查询", "system_user_dimension_value_query", enum.Line, "query_data_permission", "person_search", "primary", 101, "/admin/user/:id/dimension-values", "GET"),
		apiPermissionWithAPI(618, menuID, "用户归属保存", "system_user_dimension_value_save", enum.Line, "save", "badge", "primary", 102, "/admin/user/:id/dimension-values", "PUT"),
	}
	for i := range buttons {
		switch buttons[i].Code {
		case "system_user_menu_query", "system_user_data_permission_query", "system_user_role_options", "system_user_dimension_value_query", "system_user_dimension_value_save":
			buttons[i].IsButton = false
			buttons[i].IsHidden = false
		}
	}
	return seedMenuButtons(db, sf, roleID, roleName, buttons)
}

func seedDataPermissionMenuButtons(db *gorm.DB, sf *utils.Snowflake, roleID int, roleName string, menuID int) error {
	buttons := []model.SysMenuButton{
		apiPermissionWithAPI(600, menuID, "维度列表", "system_data_permission_dimension_query", enum.Top, "query", "search", "primary", 90, "/admin/data-permission/dimension/query", "POST"),
		apiPermissionWithAPI(601, menuID, "维度详情", "system_data_permission_dimension_detail", enum.Line, "detail", "visibility", "primary", 91, "/admin/data-permission/dimension/:id", "GET"),
		menuButtonWithAPI(602, menuID, "新增维度", "system_data_permission_dimension_create", enum.Top, "create", "add", "primary", 1, "/admin/data-permission/dimension", "POST"),
		menuButtonWithAPI(603, menuID, "编辑维度", "system_data_permission_dimension_update", enum.Line, "update", "edit", "primary", 2, "/admin/data-permission/dimension/:id", "PUT"),
		menuButtonWithAPI(604, menuID, "删除维度", "system_data_permission_dimension_delete", enum.Line, "delete", "delete", "negative", 3, "/admin/data-permission/dimension/:id", "DELETE"),
		apiPermissionWithAPI(605, menuID, "维度选项", "system_data_permission_dimension_options", enum.Line, "query", "list", "primary", 92, "/admin/data-permission/dimension-options/:code", "GET"),
		apiPermissionWithAPI(606, menuID, "菜单绑定查询", "system_data_permission_binding_query", enum.Line, "query", "account_tree", "primary", 93, "/admin/data-permission/bindings/menu/:menuId", "GET"),
		apiPermissionWithAPI(607, menuID, "菜单绑定保存", "system_data_permission_binding_save", enum.Line, "save", "save", "primary", 94, "/admin/data-permission/bindings/menu/:menuId", "PUT"),
		apiPermissionWithAPI(608, menuID, "角色数据权限查询", "system_data_permission_role_query", enum.Line, "query", "admin_panel_settings", "primary", 95, "/admin/role/:id/data-permissions", "GET"),
		apiPermissionWithAPI(609, menuID, "角色数据权限保存", "system_data_permission_role_save", enum.Line, "save", "save", "primary", 96, "/admin/role/:id/data-permissions", "PUT"),
		apiPermissionWithAPI(610, menuID, "用户覆盖查询", "system_data_permission_user_query", enum.Line, "query", "person", "primary", 97, "/admin/user/:id/data-permissions", "GET"),
		apiPermissionWithAPI(611, menuID, "用户覆盖保存", "system_data_permission_user_save", enum.Line, "save", "save", "primary", 98, "/admin/user/:id/data-permissions", "PUT"),
		apiPermissionWithAPI(619, menuID, "用户归属查询", "system_data_permission_user_dimension_query", enum.Line, "query", "person_search", "primary", 99, "/admin/user/:id/dimension-values", "GET"),
		apiPermissionWithAPI(620, menuID, "用户归属保存", "system_data_permission_user_dimension_save", enum.Line, "save", "badge", "primary", 100, "/admin/user/:id/dimension-values", "PUT"),
		apiPermissionWithAPI(614, menuID, "权限排查", "system_data_permission_debug", enum.Line, "debug", "manage_search", "primary", 101, "/admin/data-permission/debug", "GET"),
	}
	return seedMenuButtons(db, sf, roleID, roleName, buttons)
}

func seedDictionaryMenuButtons(db *gorm.DB, sf *utils.Snowflake, roleID int, roleName string, menuID int) error {
	buttons := []model.SysMenuButton{
		apiPermissionWithAPI(456, menuID, "字典查询", "develop_dictionary_query", enum.Top, "query", "search", "primary", 90, "/admin/dict/query", "POST"),
		apiPermissionWithAPI(457, menuID, "字典元数据", "develop_dictionary_metadata", enum.Top, "metadata", "data_object", "primary", 91, "/admin/table/code/:code", "GET"),
		apiPermissionWithAPI(458, menuID, "字典项元数据", "develop_dictionary_item_metadata", enum.Top, "item_metadata", "data_object", "primary", 92, "/admin/table/code/:code", "GET"),
		apiPermissionWithAPI(459, menuID, "字典详情", "develop_dictionary_detail", enum.Line, "detail", "visibility", "primary", 93, "/admin/dict/id/:id", "GET"),
		apiPermissionWithAPI(460, menuID, "字典项查询", "develop_dictionary_item_query", enum.Line, "query_item", "search", "primary", 94, "/admin/dict/items/:id", "GET"),
		apiPermissionWithAPI(461, menuID, "字典项详情", "develop_dictionary_item_detail", enum.Line, "detail_item", "visibility", "primary", 95, "/admin/dict/item/:id", "GET"),
		menuButtonWithAPI(462, menuID, "新增", "develop_dictionary_create", enum.Top, "create", "add", "primary", 1, "/admin/dict", "POST"),
		menuButtonWithAPI(463, menuID, "编辑", "develop_dictionary_update", enum.Line, "update", "edit", "primary", 1, "/admin/dict/:id", "PUT"),
		menuButtonWithAPI(464, menuID, "删除", "develop_dictionary_delete", enum.Line, "delete", "delete", "negative", 2, "/admin/dict/:id", "DELETE"),
		menuButtonWithAPI(465, menuID, "新增字典项", "develop_dictionary_item_create", enum.Top, "create_item", "add", "primary", 2, "/admin/dict/item", "POST"),
		menuButtonWithAPI(466, menuID, "编辑字典项", "develop_dictionary_item_update", enum.Line, "update_item", "edit", "primary", 3, "/admin/dict/item/:id", "PUT"),
		menuButtonWithAPI(467, menuID, "删除字典项", "develop_dictionary_item_delete", enum.Line, "delete_item", "delete", "negative", 4, "/admin/dict/item/:id", "DELETE"),
	}
	return seedMenuButtons(db, sf, roleID, roleName, buttons)
}

func seedMenuButtons(db *gorm.DB, sf *utils.Snowflake, roleID int, roleName string, buttons []model.SysMenuButton) error {
	for _, button := range buttons {
		if err := seedMenuButton(db, sf, roleID, roleName, button); err != nil {
			return err
		}
	}
	return nil
}

func seedMenuButton(db *gorm.DB, sf *utils.Snowflake, roleID int, roleName string, button model.SysMenuButton) error {
	button = normalizeMigrationMenuButton(button)
	var existing model.SysMenuButton
	err := db.Where("menu_id = ? AND code = ?", button.MenuId, button.Code).First(&existing).Error
	if err == nil {
		button.Id = existing.Id
		if err := db.Model(&model.SysMenuButton{}).Where("id = ?", existing.Id).Updates(map[string]interface{}{
			"name":          button.Name,
			"memo":          button.Memo,
			"position":      button.Position,
			"event_type":    button.EventType,
			"event_action":  button.EventAction,
			"icon":          button.Icon,
			"color":         button.Color,
			"display_mode":  button.DisplayMode,
			"sequence":      button.Sequence,
			"path":          button.Path,
			"method":        strings.ToUpper(button.Method),
			"params_schema": button.ParamsSchema,
			"confirm_text":  button.ConfirmText,
			"disable_when":  button.DisableWhen,
			"is_button":     button.IsButton,
			"is_hidden":     button.IsHidden,
			"is_disabled":   button.IsDisabled,
			"before_hooks":  button.BeforeHooks,
			"after_hooks":   button.AfterHooks,
			"state":         true,
			"gmt_modify":    model.Now(),
		}).Error; err != nil {
			return err
		}
	} else {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		id, err := seedPrimaryId(db, &model.SysMenuButton{}, button.Id, sf)
		if err != nil {
			return err
		}
		button.Id = id
		button.Method = strings.ToUpper(button.Method)
		if err := db.Model(&model.SysMenuButton{}).Create(migrationMenuButtonCreateMap(button)).Error; err != nil {
			return err
		}
	}
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.SysRoleMenuButton{RoleId: roleID, MenuId: button.MenuId, ButtonId: button.Id}).Error; err != nil {
		return err
	}
	if button.Path != "" && button.Method != "" && roleName != "" {
		if err := seedCasbinPolicy(db, roleName, button.Path, button.Method); err != nil {
			return err
		}
	}
	return nil
}

func seedLowCodeMenuButtonTemplates(db *gorm.DB, sf *utils.Snowflake) error {
	templates := []model.SysMenuButtonTemplate{
		apiPermissionTemplate(600, "列表查询", "_query", enum.Top, "query", "search", "primary", 0, "/admin/generalization/query/code/:code", "POST"),
		buttonTemplate(601, "新增", "_create", enum.Top, "create", "add", "primary", 1),
		buttonTemplate(602, "刷新", "_refresh", enum.Top, "refresh", "refresh", "primary", 2),
		buttonTemplate(603, "详情", "_detail", enum.Line, "detail", "visibility", "primary", 0),
		buttonTemplate(604, "编辑", "_update", enum.Line, "update", "edit", "primary", 1),
		buttonTemplate(605, "删除", "_delete", enum.Line, "delete", "delete", "negative", 2),
		buttonTemplate(606, "批量删除", "_batch_delete", enum.Top, "batch_delete", "delete_sweep", "negative", 3),
		buttonTemplate(607, "导出", "_export", enum.Top, "export", "download", "primary", 4),
	}
	templates[1].Path = "/admin/generalization/create"
	templates[1].Method = "POST"
	templates[1].AfterHooks = `["refresh"]`
	templates[4].Path = "/admin/generalization/update"
	templates[4].Method = "PUT"
	templates[5].Path = "/admin/generalization/delete"
	templates[5].Method = "DELETE"
	templates[5].ConfirmText = "确定要删除该数据吗？"
	templates[6].Path = "/admin/generalization/batch-delete"
	templates[6].Method = "DELETE"
	templates[6].ConfirmText = "确定要批量删除选中的数据吗？"
	templates[7].Path = "/admin/generalization/export"
	templates[7].Method = "POST"
	for _, template := range templates {
		if err := seedLowCodeMenuButtonTemplate(db, sf, template); err != nil {
			return err
		}
	}
	return disableLowCodeFileButtonTemplates(db)
}

func disableLowCodeFileButtonTemplates(db *gorm.DB) error {
	return db.Model(&model.SysMenuButtonTemplate{}).
		Where("scene = ?", lowCodeCrudButtonTemplateScene).
		Where("code_suffix IN ?", []string{
			"_file_upload",
			"_file_detail",
			"_file_preview_url",
			"_file_download_url",
			"_file_upload_init",
			"_file_upload_chunk",
			"_file_upload_merge",
			"_file_upload_progress",
		}).
		Update("state", false).Error
}

func seedLowCodeMenuButtonTemplate(db *gorm.DB, sf *utils.Snowflake, template model.SysMenuButtonTemplate) error {
	template = normalizeMigrationMenuButtonTemplate(template)
	var existing model.SysMenuButtonTemplate
	err := db.Where("scene = ? AND code_suffix = ?", template.Scene, template.CodeSuffix).First(&existing).Error
	if err == nil {
		return db.Model(&model.SysMenuButtonTemplate{}).Where("id = ?", existing.Id).Updates(map[string]interface{}{
			"name":          template.Name,
			"memo":          template.Memo,
			"position":      template.Position,
			"event_type":    template.EventType,
			"event_action":  template.EventAction,
			"icon":          template.Icon,
			"color":         template.Color,
			"display_mode":  template.DisplayMode,
			"sequence":      template.Sequence,
			"path":          template.Path,
			"method":        strings.ToUpper(template.Method),
			"params_schema": template.ParamsSchema,
			"confirm_text":  template.ConfirmText,
			"disable_when":  template.DisableWhen,
			"is_button":     template.IsButton,
			"is_disabled":   template.IsDisabled,
			"before_hooks":  template.BeforeHooks,
			"after_hooks":   template.AfterHooks,
			"state":         true,
			"gmt_modify":    model.Now(),
		}).Error
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}
	id, err := seedPrimaryId(db, &model.SysMenuButtonTemplate{}, template.Id, sf)
	if err != nil {
		return err
	}
	template.Id = id
	template.Method = strings.ToUpper(template.Method)
	return db.Model(&model.SysMenuButtonTemplate{}).Create(migrationMenuButtonTemplateCreateMap(template)).Error
}

func buttonTemplate(id int, name, codeSuffix string, position enum.SysMenuButtonPosition, action, icon, color string, sequence uint8) model.SysMenuButtonTemplate {
	return model.SysMenuButtonTemplate{
		Basic:       model.Basic{Id: id, State: true},
		Scene:       lowCodeCrudButtonTemplateScene,
		Name:        name,
		CodeSuffix:  codeSuffix,
		Position:    position,
		EventAction: action,
		Icon:        icon,
		Color:       color,
		DisplayMode: enum.ButtonDisplayAuto,
		Sequence:    sequence,
		IsButton:    true,
	}
}

func apiPermissionTemplate(id int, name, codeSuffix string, position enum.SysMenuButtonPosition, action, icon, color string, sequence uint8, path, method string) model.SysMenuButtonTemplate {
	template := buttonTemplate(id, name, codeSuffix, position, action, icon, color, sequence)
	template.Path = path
	template.Method = strings.ToUpper(method)
	template.IsButton = false
	return template
}

func seedCasbinPolicy(db *gorm.DB, roleName, path, method string) error {
	method = strings.ToUpper(method)
	var count int64
	if err := db.Model(&model.CasbinRule{}).
		Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?", "p", roleName, path, method).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return db.Create(&model.CasbinRule{PType: "p", V0: roleName, V1: path, V2: method}).Error
}

type routePolicy struct {
	Path   string
	Method string
}

func seedSuperAdminRoutePolicies(db *gorm.DB, roleName string) error {
	policies := []routePolicy{
		{"/admin/configure/detail", "GET"},
		{"/admin/configure/:id", "PUT"},
		{"/admin/configure/test-email", "POST"},
		{"/admin/log/access/query", "POST"},
		{"/admin/log/access/:id", "GET"},
		{"/admin/dict/id/:id", "GET"},
		{"/admin/dict/code/:code", "GET"},
		{"/admin/dict/query", "POST"},
		{"/admin/dict", "POST"},
		{"/admin/dict/:id", "PUT"},
		{"/admin/dict/:id", "DELETE"},
		{"/admin/dict/items/:id", "GET"},
		{"/admin/dict/item/:id", "GET"},
		{"/admin/dict/item", "POST"},
		{"/admin/dict/item/:id", "PUT"},
		{"/admin/dict/item/:id", "DELETE"},
		{"/admin/table/id/:id", "GET"},
		{"/admin/table/code/:code", "GET"},
		{"/admin/table/query", "POST"},
		{"/admin/table", "POST"},
		{"/admin/table/:id", "PUT"},
		{"/admin/table/:id", "DELETE"},
		{"/admin/table/fields/:id", "GET"},
		{"/admin/table/field/:id", "GET"},
		{"/admin/table/field", "POST"},
		{"/admin/table/field/:id", "PUT"},
		{"/admin/table/field/:id", "DELETE"},
		{"/admin/table/init/:code", "GET"},
		{"/admin/table/sync/:code", "POST"},
		{"/admin/table/sync/index/:code", "POST"},
		{"/admin/table/publish/:code", "POST"},
		{"/admin/table/unpublish/:code", "POST"},
		{"/admin/table/indexes/:id", "GET"},
		{"/admin/table/index/:id", "GET"},
		{"/admin/table/index", "POST"},
		{"/admin/table/index/:id", "PUT"},
		{"/admin/table/index/:id", "DELETE"},
		{"/admin/table/relations/:id", "GET"},
		{"/admin/table/relation/:id", "GET"},
		{"/admin/table/relation", "POST"},
		{"/admin/table/relation/:id", "PUT"},
		{"/admin/table/relation/:id", "DELETE"},
		{"/admin/menu/user/:id", "GET"},
		{"/admin/menu/:id", "GET"},
		{"/admin/menu/query", "POST"},
		{"/admin/menu", "POST"},
		{"/admin/menu/:id", "PUT"},
		{"/admin/menu/order", "PUT"},
		{"/admin/menu/refresh-cache", "POST"},
		{"/admin/menu/:id", "DELETE"},
		{"/admin/menu/buttons/:menuId", "GET"},
		{"/admin/menu/button", "POST"},
		{"/admin/menu/button/:id", "PUT"},
		{"/admin/menu/button/:id", "DELETE"},
		{"/admin/role/menu/:id", "GET"},
		{"/admin/role/menu/buttons/:roleId/:menuId", "GET"},
		{"/admin/role/:id/data-permissions", "GET"},
		{"/admin/role/:id/data-permissions", "PUT"},
		{"/admin/role/assign-permissions", "POST"},
		{"/admin/role/query", "POST"},
		{"/admin/role/:id", "GET"},
		{"/admin/role", "POST"},
		{"/admin/role/:id", "PUT"},
		{"/admin/role/:id", "DELETE"},
		{"/admin/user/query", "POST"},
		{"/admin/user/:id", "GET"},
		{"/admin/user", "POST"},
		{"/admin/user/password", "POST"},
		{"/admin/user/reset_password/:id", "POST"},
		{"/admin/user/unlock_login/:id", "POST"},
		{"/admin/user/:id/data-permissions", "GET"},
		{"/admin/user/:id/data-permissions", "PUT"},
		{"/admin/user/:id/dimension-values", "GET"},
		{"/admin/user/:id/dimension-values", "PUT"},
		{"/admin/user/:id", "PUT"},
		{"/admin/user/:id", "DELETE"},
		{"/admin/data-permission/dimension/query", "POST"},
		{"/admin/data-permission/dimension/:id", "GET"},
		{"/admin/data-permission/dimension", "POST"},
		{"/admin/data-permission/dimension/:id", "PUT"},
		{"/admin/data-permission/dimension/:id", "DELETE"},
		{"/admin/data-permission/dimension-options/:code", "GET"},
		{"/admin/data-permission/bindings/menu/:menuId", "GET"},
		{"/admin/data-permission/bindings/menu/:menuId", "PUT"},
		{"/admin/data-permission/debug", "GET"},
		{"/admin/application/:id", "GET"},
		{"/admin/application/query", "POST"},
		{"/admin/application", "POST"},
		{"/admin/application/:id/rotate-secret", "POST"},
		{"/admin/application/:id", "PUT"},
		{"/admin/application/:id", "DELETE"},
		{"/admin/sms/template/query", "POST"},
		{"/admin/sms/template/:id", "GET"},
		{"/admin/sms/template", "POST"},
		{"/admin/sms/template/:id", "PUT"},
		{"/admin/sms/template/:id", "DELETE"},
		{"/admin/generalization/query/:id", "POST"},
		{"/admin/generalization/query/code/:code", "POST"},
		{"/admin/generalization/detail/code/:code/:id", "GET"},
		{"/admin/generalization/create", "POST"},
		{"/admin/generalization/update", "PUT"},
		{"/admin/generalization/delete", "DELETE"},
		{"/admin/generalization/batch-delete", "DELETE"},
		{"/admin/generalization/export", "POST"},
		{"/admin/file/upload", "POST"},
		{"/admin/file/:id", "GET"},
		{"/admin/file/:id", "DELETE"},
		{"/admin/file/preview-url/:uuid", "GET"},
		{"/admin/file/download-url/:uuid", "GET"},
		{"/admin/file/preview/:uuid", "GET"},
		{"/admin/file/download/:uuid", "GET"},
		{"/admin/file/upload/init", "POST"},
		{"/admin/file/upload/chunk", "POST"},
		{"/admin/file/upload/merge/:upload_id", "POST"},
		{"/admin/file/upload/progress/:upload_id", "GET"},
	}
	for _, policy := range policies {
		if err := seedCasbinPolicy(db, roleName, policy.Path, policy.Method); err != nil {
			return err
		}
	}
	return nil
}

func menuButton(id, menuID int, name, code string, position enum.SysMenuButtonPosition, action, icon, color string, sequence uint8) model.SysMenuButton {
	return model.SysMenuButton{
		Basic:       model.Basic{Id: id, State: true},
		MenuId:      menuID,
		Name:        name,
		Code:        code,
		Position:    position,
		EventAction: action,
		Icon:        icon,
		Color:       color,
		DisplayMode: enum.ButtonDisplayAuto,
		Sequence:    sequence,
		IsButton:    true,
	}
}

func normalizeMigrationMenuButton(button model.SysMenuButton) model.SysMenuButton {
	displayMode, ok := enum.NormalizeSysMenuButtonDisplayMode(string(button.DisplayMode))
	if !ok {
		displayMode = enum.ButtonDisplayAuto
	}
	button.DisplayMode = displayMode
	return button
}

func normalizeMigrationMenuButtonTemplate(template model.SysMenuButtonTemplate) model.SysMenuButtonTemplate {
	displayMode, ok := enum.NormalizeSysMenuButtonDisplayMode(string(template.DisplayMode))
	if !ok {
		displayMode = enum.ButtonDisplayAuto
	}
	template.DisplayMode = displayMode
	return template
}

func menuButtonWithAPI(id, menuID int, name, code string, position enum.SysMenuButtonPosition, action, icon, color string, sequence uint8, path, method string) model.SysMenuButton {
	button := menuButton(id, menuID, name, code, position, action, icon, color, sequence)
	button.Path = path
	button.Method = strings.ToUpper(method)
	return button
}

func apiPermissionWithAPI(id, menuID int, name, code string, position enum.SysMenuButtonPosition, action, icon, color string, sequence uint8, path, method string) model.SysMenuButton {
	button := menuButtonWithAPI(id, menuID, name, code, position, action, icon, color, sequence, path, method)
	button.IsButton = false
	button.IsHidden = false
	return button
}

func menu(id, pid int, name, path, component, title, icon string, sequence uint8) model.SysMenu {
	return model.SysMenu{
		Basic:     model.Basic{Id: id, State: true},
		Pid:       pid,
		Name:      name,
		Path:      path,
		Component: component,
		Title:     title,
		IsHidden:  false,
		Sequence:  sequence,
		PageType:  enum.MenuPageTypeFixed,
		Icon:      &icon,
		IsUnfold:  false,
	}
}

func directoryMenu(menu model.SysMenu) model.SysMenu {
	menu.PageType = enum.MenuPageTypeDirectory
	return menu
}

func menuWithTable(menu model.SysMenu, tableCode string) model.SysMenu {
	menu.TableCode = tableCode
	menu.PageType = enum.MenuPageTypeFixed
	return menu
}

func menuWithOption(menu model.SysMenu, option string) model.SysMenu {
	menu.Option = option
	return menu
}

type systemTableMetadataSeed struct {
	code string
	name string
}

func seedSystemTableMetadata(db *gorm.DB, sf *utils.Snowflake) error {
	for _, seed := range systemTableMetadataSeeds() {
		if !db.Migrator().HasTable(seed.code) {
			continue
		}
		table, err := seedSystemTable(db, sf, seed)
		if err != nil {
			return err
		}
		if err := seedSystemTableFields(db, sf, table); err != nil {
			return err
		}
		if err := seedSystemTableIndexes(db, sf, table); err != nil {
			return err
		}
	}
	return nil
}

func systemTableMetadataSeeds() []systemTableMetadataSeed {
	return []systemTableMetadataSeed{
		{code: "application", name: "应用"},
		{code: "sms_template", name: "短信模板"},
		{code: "sms_log", name: "短信日志"},
		{code: "file", name: "文件"},
		{code: "file_chunk", name: "文件分片"},
		{code: "access_log", name: "访问日志"},
		{code: "login_log", name: "登录日志"},
		{code: "sys_configure", name: "系统配置"},
		{code: "sys_dict", name: "字典"},
		{code: "sys_dict_item", name: "字典项"},
		{code: "sys_menu", name: "菜单"},
		{code: "sys_menu_button", name: "菜单按钮"},
		{code: "sys_menu_button_template", name: "菜单按钮模板"},
		{code: "sys_role", name: "角色"},
		{code: "sys_role_menu", name: "角色菜单"},
		{code: "sys_role_menu_button", name: "角色按钮"},
		{code: "sys_user", name: "用户"},
		{code: "sys_user_role", name: "用户角色"},
		{code: "sys_table", name: "数据表"},
		{code: "sys_table_field", name: "数据字段"},
		{code: "sys_table_index", name: "数据索引"},
		{code: "sys_table_index_field", name: "数据索引字段"},
		{code: "sys_table_relation", name: "数据关系"},
		{code: "sys_data_dimension", name: "数据权限维度"},
		{code: "sys_data_scope_binding", name: "数据权限绑定"},
		{code: "sys_role_data_scope", name: "角色数据权限"},
		{code: "sys_user_data_scope_override", name: "用户数据权限覆盖"},
		{code: "sys_user_dimension_value", name: "用户维度归属"},
		{code: "casbin_rule", name: "接口权限规则"},
	}
}

func seedSystemTable(db *gorm.DB, sf *utils.Snowflake, seed systemTableMetadataSeed) (model.SysTable, error) {
	var table model.SysTable
	err := db.Unscoped().Where("table_code = ?", seed.code).First(&table).Error
	if err == nil {
		updates := map[string]interface{}{
			"table_name":  seed.name,
			"table_type":  enum.System,
			"state":       true,
			"gmt_delete":  nil,
			"delete_user": nil,
			"delete_name": nil,
		}
		if err := db.Unscoped().Model(&model.SysTable{}).Where("id = ?", table.Id).Updates(updates).Error; err != nil {
			return model.SysTable{}, err
		}
		table.TableName = seed.name
		table.TableType = enum.System
		table.State = true
		table.GmtDelete.Valid = false
		return table, nil
	}
	if err != gorm.ErrRecordNotFound {
		return model.SysTable{}, err
	}
	id, err := newMigrationID(sf)
	if err != nil {
		return model.SysTable{}, err
	}
	table = model.SysTable{
		Basic: model.Basic{
			Id:    id,
			State: true,
		},
		TableName: seed.name,
		TableCode: seed.code,
		TableType: enum.System,
	}
	if err := db.Create(&table).Error; err != nil {
		return model.SysTable{}, err
	}
	return table, nil
}

func seedSystemTableFields(db *gorm.DB, sf *utils.Snowflake, table model.SysTable) error {
	columns, err := db.Migrator().ColumnTypes(table.TableCode)
	if err != nil {
		return err
	}
	for index, column := range columns {
		field := systemColumnToTableField(table.TableCode, column, index+1)
		if err := seedSystemTableField(db, sf, table, field); err != nil {
			return err
		}
	}
	return nil
}

func seedSystemTableField(db *gorm.DB, sf *utils.Snowflake, table model.SysTable, field model.SysTableField) error {
	var existing model.SysTableField
	err := db.Unscoped().Where("table_id = ? AND field_code = ?", table.Id, field.FieldCode).First(&existing).Error
	if err == nil {
		updates := map[string]interface{}{
			"field_type":           field.FieldType,
			"field_length":         field.FieldLength,
			"field_decimal_length": field.FieldDecimalLength,
			"input_type":           field.InputType,
			"is_primary_key":       field.IsPrimaryKey,
			"is_index":             field.IsIndex,
			"is_quick_search":      field.IsQuickSearch,
			"is_advanced_search":   field.IsAdvancedSearch,
			"is_sort":              field.IsSort,
			"is_null":              field.IsNull,
			"is_list_show":         field.IsListShow,
			"is_insert_show":       field.IsInsertShow,
			"is_update_show":       field.IsUpdateShow,
			"sequence":             field.Sequence,
			"binding":              field.Binding,
			"field_category":       field.FieldCategory,
			"state":                true,
			"gmt_delete":           nil,
			"delete_user":          nil,
			"delete_name":          nil,
		}
		if shouldUpdateSystemFieldName(existing.FieldName, table.TableCode, field.FieldCode) {
			updates["field_name"] = field.FieldName
		}
		if field.DictCode != nil {
			updates["dict_code"] = *field.DictCode
		}
		if field.DefaultValue != nil {
			updates["default_value"] = *field.DefaultValue
		}
		return db.Unscoped().Model(&model.SysTableField{}).Where("id = ?", existing.Id).Updates(updates).Error
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}
	id, err := newMigrationID(sf)
	if err != nil {
		return err
	}
	field.Id = id
	field.TableId = table.Id
	return db.Model(&model.SysTableField{}).Create(migrationTableFieldCreateMap(field)).Error
}

func systemColumnToTableField(tableCode string, column gorm.ColumnType, sequence int) model.SysTableField {
	length, hasLength := column.Length()
	precision, scale, hasDecimal := column.DecimalSize()
	nullable, hasNullable := column.Nullable()
	if !hasNullable {
		nullable = true
	}
	field := model.SysTableField{
		Basic:              model.Basic{State: true},
		FieldName:          systemFieldDisplayName(tableCode, column.Name()),
		FieldCode:          column.Name(),
		FieldType:          systemFieldType(column.DatabaseTypeName()),
		InputType:          enum.InputType,
		IsPrimaryKey:       column.Name() == "id",
		IsIndex:            false,
		IsQuickSearch:      false,
		IsAdvancedSearch:   false,
		IsSort:             true,
		IsNull:             nullable,
		IsListShow:         true,
		IsInsertShow:       !security.IsManagedMetadataField(column.Name()),
		IsUpdateShow:       !security.IsManagedMetadataField(column.Name()),
		Sequence:           uint8(sequence),
		FieldCategory:      enum.NormalField,
		FieldDecimalLength: 0,
	}
	if !nullable {
		field.Binding = "required"
	}
	if hasLength && length > 0 && field.FieldType == enum.VarcharFieldType {
		field.FieldLength = int(length)
	}
	if hasDecimal && field.FieldType == enum.FloatFieldType {
		field.FieldLength = int(precision)
		field.FieldDecimalLength = int(scale)
	}
	if defaultValue, ok := column.DefaultValue(); ok {
		field.DefaultValue = utils.StringPtr(normalizeSystemColumnDefault(field.FieldType, defaultValue))
	}
	switch field.FieldType {
	case enum.IntFieldType, enum.BigIntFieldType, enum.TinyintFieldType, enum.FloatFieldType:
		field.InputType = enum.InputNumberInputType
	case enum.TextFieldType:
		field.InputType = enum.TextareaInputType
	case enum.BooleanFieldType:
		field.InputType = enum.BooleanInputType
	case enum.DateFieldType:
		field.InputType = enum.DatePickerInputType
	case enum.DatetimeFieldType:
		field.InputType = enum.DatetimePickerInputType
	case enum.TimeFieldType:
		field.InputType = enum.TimePickerInputType
	case enum.JsonFieldType:
		field.InputType = enum.JsonInputType
	}
	if systemSearchableField(field) {
		field.IsQuickSearch = true
		field.IsAdvancedSearch = true
	}
	if dictCode := systemMetadataDictCode(tableCode, field.FieldCode, field.FieldType); dictCode != "" {
		field.DictCode = utils.StringPtr(dictCode)
		field.InputType = enum.SelectInputType
	}
	if defaultValue := systemMetadataDefaultValue(tableCode, field.FieldCode, field.FieldType); defaultValue != "" {
		field.DefaultValue = utils.StringPtr(defaultValue)
	}
	applyMigrationSensitiveFieldDefaults(&field)
	applyMigrationManagedFieldDefaults(&field)
	return field
}

func normalizeSystemColumnDefault(fieldType enum.SysTableFieldType, defaultValue string) string {
	value := strings.TrimSpace(defaultValue)
	value = strings.TrimSuffix(value, "::character varying")
	value = strings.TrimSuffix(value, "::text")
	value = strings.TrimSuffix(value, "::bpchar")
	value = strings.TrimSuffix(value, "::smallint")
	value = strings.TrimSuffix(value, "::integer")
	value = strings.TrimSuffix(value, "::bigint")
	value = strings.Trim(value, "'")
	if fieldType == enum.BooleanFieldType {
		switch strings.ToLower(value) {
		case "true", "t", "1":
			return "true"
		case "false", "f", "0":
			return "false"
		}
	}
	return value
}

func systemMetadataDefaultValue(tableCode, fieldCode string, fieldType enum.SysTableFieldType) string {
	switch fieldCode {
	case "field_category":
		if tableCode == "sys_table_field" {
			return string(enum.NormalField)
		}
	case "relation_type":
		if tableCode == "sys_table_relation" {
			return strconv.Itoa(int(enum.OneToMany))
		}
	}
	if fieldType != enum.BooleanFieldType {
		return ""
	}
	if fieldCode == "state" || fieldCode == "is_button" {
		return "true"
	}
	return "false"
}

func systemFieldType(databaseType string) enum.SysTableFieldType {
	normalized := strings.ToLower(strings.TrimSpace(databaseType))
	switch {
	case normalized == "bigint" || normalized == "int8":
		return enum.BigIntFieldType
	case normalized == "integer" || normalized == "int" || normalized == "int4":
		return enum.IntFieldType
	case normalized == "smallint" || normalized == "int2" || normalized == "tinyint":
		return enum.TinyintFieldType
	case normalized == "numeric" || normalized == "decimal" || normalized == "double precision" || normalized == "float" || normalized == "float4" || normalized == "float8" || normalized == "real":
		return enum.FloatFieldType
	case normalized == "boolean" || normalized == "bool":
		return enum.BooleanFieldType
	case normalized == "date":
		return enum.DateFieldType
	case strings.HasPrefix(normalized, "timestamp") || normalized == "datetime" || normalized == "timestamptz":
		return enum.DatetimeFieldType
	case strings.HasPrefix(normalized, "time"):
		return enum.TimeFieldType
	case normalized == "text":
		return enum.TextFieldType
	case normalized == "json" || normalized == "jsonb":
		return enum.JsonFieldType
	default:
		return enum.VarcharFieldType
	}
}

func systemSearchableField(field model.SysTableField) bool {
	if security.IsSensitiveFieldName(field.FieldCode) || security.IsManagedMetadataField(field.FieldCode) {
		return false
	}
	switch field.FieldType {
	case enum.VarcharFieldType, enum.TextFieldType:
		return true
	default:
		return false
	}
}

func systemMetadataDictCode(tableCode, fieldCode string, fieldType enum.SysTableFieldType) string {
	switch fieldCode {
	case "master_detail_mode":
		if tableCode == "sys_table" {
			return "sys_master_detail_mode"
		}
	case "form_open_mode":
		if tableCode == "sys_table" {
			return "sys_form_open_mode"
		}
	case "detail_open_mode":
		if tableCode == "sys_table" {
			return "sys_detail_open_mode"
		}
	case "table_type":
		return "sys_table_type"
	case "field_type":
		return "sys_table_field_type"
	case "input_type":
		return "sys_table_field_input_type"
	case "field_category":
		return "sys_table_field_category"
	case "relation_type":
		return "sys_table_relation_type"
	case "position":
		if tableCode == "sys_menu_button" || tableCode == "sys_menu_button_template" {
			return "sys_menu_button_position"
		}
	case "display_mode":
		if tableCode == "sys_menu_button" || tableCode == "sys_menu_button_template" {
			return "sys_menu_button_display_mode"
		}
	case "event_action":
		if tableCode == "sys_menu_button" || tableCode == "sys_menu_button_template" {
			return "sys_menu_button_event_action"
		}
	case "method", "http_method":
		return "http_method"
	case "state", "success", "is_hidden", "is_button", "is_disabled", "is_unfold", "required":
		return "whether"
	}
	if fieldType == enum.BooleanFieldType || strings.HasPrefix(fieldCode, "is_") {
		return "whether"
	}
	return ""
}

func applyMigrationSensitiveFieldDefaults(field *model.SysTableField) {
	if !security.IsSensitiveFieldName(field.FieldCode) {
		return
	}
	field.IsListShow = false
	field.IsInsertShow = false
	field.IsUpdateShow = false
	field.IsQuickSearch = false
	field.IsAdvancedSearch = false
}

func applyMigrationManagedFieldDefaults(field *model.SysTableField) {
	if !security.IsManagedMetadataField(field.FieldCode) {
		return
	}
	field.IsListShow = false
	field.IsInsertShow = false
	field.IsUpdateShow = false
	field.IsQuickSearch = false
	field.IsAdvancedSearch = false
}

func shouldUpdateSystemFieldName(existingName, tableCode, fieldCode string) bool {
	existing := strings.TrimSpace(existingName)
	if existing == "" || existing == fieldCode || existing == strings.ReplaceAll(fieldCode, "_", " ") {
		return true
	}
	if tableNames, ok := systemTableFieldDisplayNameMap[tableCode]; ok {
		_, known := tableNames[fieldCode]
		if known {
			return true
		}
	}
	_, known := systemFieldDisplayNameMap[fieldCode]
	return known
}

var systemTableFieldDisplayNameMap = map[string]map[string]string{
	"access_log": {
		"method":        "操作",
		"locality":      "地域",
		"body":          "请求数据",
		"query":         "查询参数",
		"response":      "响应数据",
		"action":        "业务动作",
		"resource_type": "资源类型",
		"resource_code": "资源编码",
		"resource_id":   "资源ID",
		"status_code":   "HTTP状态码",
	},
	"application": {
		"name":        "应用名称",
		"app_key":     "应用Key",
		"app_secret":  "应用Secret",
		"expiration":  "过期时间",
		"ding_key":    "钉钉Key",
		"ding_secret": "钉钉Secret",
		"ding_app_id": "钉钉AppID",
	},
	"sms_template": {
		"sign_name":       "短信签名",
		"template_code":   "模板编号",
		"template_name":   "模板名称",
		"template_params": "模板参数",
	},
	"sys_menu": {
		"pid":        "父菜单ID",
		"name":       "路由",
		"path":       "路径",
		"component":  "路由主体",
		"title":      "显示标题",
		"table_code": "绑定表编码",
		"option":     "扩展配置",
		"redirect":   "重定向地址",
	},
	"sys_menu_button": {
		"name":          "按钮名称",
		"code":          "按钮编码",
		"path":          "接口路径",
		"method":        "请求方法",
		"params_schema": "参数Schema",
		"confirm_text":  "确认提示",
		"disable_when":  "禁用条件",
		"is_hidden":     "是否隐藏",
		"is_disabled":   "是否禁用",
	},
	"sys_menu_button_template": {
		"name":          "模板名称",
		"code_suffix":   "编码后缀",
		"path":          "接口路径",
		"method":        "请求方法",
		"params_schema": "参数Schema",
		"confirm_text":  "确认提示",
		"disable_when":  "禁用条件",
		"is_disabled":   "是否禁用",
	},
	"sys_table": {
		"table_code": "数据库中表名",
		"parent_id":  "父节点ID",
		"sql":        "视图定义SQL",
	},
	"sys_table_field": {
		"table_id":             "表ID",
		"field_name":           "列名",
		"field_code":           "表字段名",
		"field_decimal_length": "小数位数",
		"dict_code":            "所用字典",
		"original_field_id":    "原字段ID",
		"binding":              "验证器",
	},
}

var systemFieldDisplayNameMap = map[string]string{
	"id":                    "ID",
	"gmt_create":            "创建时间",
	"create_user":           "创建人ID",
	"create_name":           "创建人",
	"gmt_modify":            "修改时间",
	"modify_user":           "修改人ID",
	"modify_name":           "修改人",
	"gmt_delete":            "删除时间",
	"delete_user":           "删除人ID",
	"delete_name":           "删除人",
	"state":                 "状态",
	"user_name":             "用户名",
	"name":                  "名称",
	"memo":                  "备注",
	"remark":                "备注",
	"email":                 "邮箱",
	"phone_number":          "手机号",
	"password":              "密码",
	"language":              "语言",
	"access_tokens":         "用户最近5次Token",
	"is_reset":              "是否重置密码",
	"gmt_last_login":        "最后登录时间",
	"password_changed_at":   "密码最后修改时间",
	"table_name":            "表名",
	"table_code":            "表编码",
	"table_type":            "表类型",
	"master_detail_mode":    "主子表展示模式",
	"form_open_mode":        "表单打开方式",
	"detail_open_mode":      "详情打开方式",
	"field_name":            "字段名",
	"field_code":            "字段编码",
	"field_type":            "字段类型",
	"field_length":          "字段长度",
	"field_decimal_length":  "小数位数",
	"input_type":            "输入类型",
	"form_span":             "表单列宽",
	"detail_span":           "详情列宽",
	"default_value":         "默认值",
	"dict_code":             "字典编码",
	"is_primary_key":        "是否主键",
	"is_index":              "是否索引",
	"is_quick_search":       "是否快捷搜索",
	"is_advanced_search":    "是否高级搜索",
	"is_sort":               "是否可排序",
	"is_null":               "是否可空",
	"is_list_show":          "是否列表显示",
	"is_insert_show":        "是否新增显示",
	"is_update_show":        "是否更新显示",
	"original_field_id":     "原字段ID",
	"binding":               "验证器",
	"field_category":        "字段类别",
	"expression":            "计算字段表达式",
	"tag":                   "标签",
	"linkage_config":        "联动配置",
	"dict_name":             "字典名称",
	"dict_id":               "字典ID",
	"item_name":             "字典项名称",
	"item_code":             "字典项编码",
	"item_value":            "字典项值",
	"menu_id":               "菜单ID",
	"role_id":               "角色ID",
	"button_id":             "按钮ID",
	"user_id":               "用户ID",
	"table_id":              "表ID",
	"app_key":               "应用Key",
	"app_secret":            "应用Secret",
	"application_id":        "应用ID",
	"application_name":      "应用名称",
	"expiration":            "过期时间",
	"ding_key":              "钉钉Key",
	"ding_secret":           "钉钉Secret",
	"ding_app_id":           "钉钉AppID",
	"method":                "请求方法",
	"ip":                    "IP",
	"url":                   "路径",
	"body":                  "请求数据",
	"query":                 "查询参数",
	"response":              "响应数据",
	"locality":              "地域",
	"action":                "业务动作",
	"resource_type":         "资源类型",
	"resource_code":         "资源编码",
	"resource_id":           "资源ID",
	"status_code":           "HTTP状态码",
	"success":               "是否成功",
	"duration_ms":           "耗时毫秒",
	"pid":                   "父菜单ID",
	"path":                  "路径",
	"component":             "路由主体",
	"title":                 "显示标题",
	"is_hidden":             "是否隐藏",
	"sequence":              "排序",
	"page_type":             "页面类型",
	"option":                "扩展配置",
	"icon":                  "图标",
	"redirect":              "重定向地址",
	"is_unfold":             "默认展开",
	"code":                  "编码",
	"position":              "位置",
	"event_type":            "事件类型",
	"event_action":          "事件动作",
	"color":                 "颜色",
	"display_mode":          "展示方式",
	"params_schema":         "参数Schema",
	"confirm_text":          "确认提示",
	"disable_when":          "禁用条件",
	"is_button":             "是否页面按钮",
	"is_disabled":           "是否禁用",
	"before_hooks":          "前置钩子JSON",
	"after_hooks":           "后置钩子JSON",
	"scene":                 "场景",
	"code_suffix":           "编码后缀",
	"index_id":              "索引ID",
	"index_name":            "索引名称",
	"is_unique":             "是否唯一",
	"field_id":              "字段ID",
	"related_table_id":      "关联表ID",
	"reference_key":         "主表字段",
	"foreign_key":           "关联字段",
	"on_delete":             "删除策略",
	"on_update":             "更新策略",
	"relation_type":         "关系类型",
	"many_table_code":       "中间表编码",
	"enable_captcha":        "启用验证码",
	"password_length":       "密码长度",
	"password_complexity":   "密码复杂度",
	"password_expire_time":  "密码过期天数",
	"password_error_count":  "密码错误次数",
	"password_lock_minutes": "密码锁定分钟数",
	"password_policy":       "密码策略",
	"system_name":           "系统名称",
	"system_version":        "系统版本",
	"system_logo":           "系统Logo",
	"system_description":    "系统描述",
	"enable_email":          "启用邮件",
	"smtp_server":           "SMTP服务器",
	"smtp_port":             "SMTP端口",
	"sender_email":          "发件人邮箱",
	"sender_password":       "发件人密码",
	"sign_name":             "短信签名",
	"template_code":         "模板编号",
	"template_name":         "模板名称",
	"template_params":       "模板参数",
	"content":               "内容",
	"mobile":                "手机号",
	"biz_id":                "业务ID",
	"result":                "结果",
	"status":                "状态",
	"file_name":             "文件名",
	"file_ext":              "文件扩展名",
	"file_size":             "文件大小",
	"file_type":             "文件类型",
	"file_md5":              "文件MD5",
	"file_uuid":             "文件UUID",
	"file_path":             "文件路径",
	"file_url":              "文件URL",
	"storage_type":          "存储类型",
	"upload_id":             "上传ID",
	"chunk_index":           "分片序号",
	"chunk_size":            "分片大小",
	"chunk_count":           "分片总数",
	"chunk_md5":             "分片MD5",
	"chunk_path":            "分片路径",
	"uploaded":              "是否上传完成",
	"merged":                "是否合并完成",
	"dimension_code":        "维度编码",
	"value_type":            "值类型",
	"source_type":           "来源类型",
	"source_code":           "来源编码",
	"label_field":           "展示字段",
	"value_field":           "值字段",
	"parent_field":          "父级字段",
	"match_type":            "匹配方式",
	"required":              "是否必配授权",
	"actions":               "生效动作",
	"strategy":              "范围策略",
	"scope_values":          "范围值",
	"override_mode":         "覆盖模式",
	"expire_at":             "过期时间",
	"ptype":                 "策略类型",
	"v0":                    "主体",
	"v1":                    "资源",
	"v2":                    "动作",
	"v3":                    "扩展1",
	"v4":                    "扩展2",
	"v5":                    "扩展3",
}

func systemFieldDisplayName(tableCode, fieldCode string) string {
	if tableNames, ok := systemTableFieldDisplayNameMap[tableCode]; ok {
		if name, ok := tableNames[fieldCode]; ok {
			return name
		}
	}
	if name, ok := systemFieldDisplayNameMap[fieldCode]; ok {
		return name
	}
	return strings.ReplaceAll(fieldCode, "_", " ")
}

func seedSystemTableIndexes(db *gorm.DB, sf *utils.Snowflake, table model.SysTable) error {
	indexes, err := db.Migrator().GetIndexes(table.TableCode)
	if err != nil {
		return err
	}
	if len(indexes) == 0 {
		return nil
	}
	var fields []model.SysTableField
	if err := db.Unscoped().Where("table_id = ?", table.Id).Find(&fields).Error; err != nil {
		return err
	}
	fieldIDs := make(map[string]int, len(fields))
	for _, field := range fields {
		fieldIDs[field.FieldCode] = field.Id
	}
	for _, index := range indexes {
		if primary, ok := index.PrimaryKey(); ok && primary {
			continue
		}
		if strings.TrimSpace(index.Name()) == "" {
			continue
		}
		indexID, err := seedSystemTableIndex(db, sf, table.Id, index.Name(), index)
		if err != nil {
			return err
		}
		for _, column := range index.Columns() {
			fieldID, ok := fieldIDs[column]
			if !ok {
				continue
			}
			if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.SysTableIndexField{
				IndexId: indexID,
				FieldId: fieldID,
			}).Error; err != nil {
				return err
			}
			if err := db.Model(&model.SysTableField{}).Where("id = ?", fieldID).Update("is_index", true).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func seedSystemTableIndex(db *gorm.DB, sf *utils.Snowflake, tableID int, indexName string, index gorm.Index) (int, error) {
	unique, ok := index.Unique()
	if !ok {
		unique = false
	}
	var existing model.SysTableIndex
	err := db.Unscoped().Where("table_id = ? AND index_name = ?", tableID, indexName).First(&existing).Error
	if err == nil {
		updates := map[string]interface{}{
			"is_unique":   unique,
			"state":       true,
			"gmt_delete":  nil,
			"delete_user": nil,
			"delete_name": nil,
		}
		if err := db.Unscoped().Model(&model.SysTableIndex{}).Where("id = ?", existing.Id).Updates(updates).Error; err != nil {
			return 0, err
		}
		return existing.Id, nil
	}
	if err != gorm.ErrRecordNotFound {
		return 0, err
	}
	id, err := newMigrationID(sf)
	if err != nil {
		return 0, err
	}
	if err := db.Create(&model.SysTableIndex{
		Basic: model.Basic{
			Id:    id,
			State: true,
		},
		TableId:   tableID,
		IndexName: indexName,
		IsUnique:  unique,
	}).Error; err != nil {
		return 0, err
	}
	return id, nil
}

type systemTableRelationSeed struct {
	parentCode   string
	childCode    string
	referenceKey string
	foreignKey   string
	relationType enum.SysTableRelationType
}

func seedSystemTableRelations(db *gorm.DB, sf *utils.Snowflake) error {
	seeds := []systemTableRelationSeed{
		{
			parentCode:   "sys_dict",
			childCode:    "sys_dict_item",
			referenceKey: "id",
			foreignKey:   "dict_id",
			relationType: enum.OneToMany,
		},
		{
			parentCode:   "sys_table",
			childCode:    "sys_table_field",
			referenceKey: "id",
			foreignKey:   "table_id",
			relationType: enum.OneToMany,
		},
		{
			parentCode:   "sys_menu",
			childCode:    "sys_menu_button",
			referenceKey: "id",
			foreignKey:   "menu_id",
			relationType: enum.OneToMany,
		},
	}
	for _, seed := range seeds {
		if err := seedSystemTableRelation(db, sf, seed); err != nil {
			return err
		}
	}
	return nil
}

func seedSystemTableRelation(db *gorm.DB, sf *utils.Snowflake, seed systemTableRelationSeed) error {
	var parent model.SysTable
	if err := db.Unscoped().Where("table_code = ?", seed.parentCode).First(&parent).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}
	var child model.SysTable
	if err := db.Unscoped().Where("table_code = ?", seed.childCode).First(&child).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	var existing model.SysTableRelation
	err := db.Unscoped().
		Where("table_id = ? AND related_table_id = ? AND reference_key = ? AND foreign_key = ?", parent.Id, child.Id, seed.referenceKey, seed.foreignKey).
		First(&existing).Error
	if err == nil {
		updates := map[string]interface{}{
			"relation_type": seed.relationType,
			"state":         true,
		}
		if existing.GmtDelete.Valid {
			updates["gmt_delete"] = nil
			updates["delete_user"] = nil
			updates["delete_name"] = nil
		}
		return db.Unscoped().Model(&model.SysTableRelation{}).
			Where("id = ?", existing.Id).
			Updates(updates).Error
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}

	id, err := newMigrationID(sf)
	if err != nil {
		return err
	}
	return db.Create(&model.SysTableRelation{
		Basic: model.Basic{
			Id:    id,
			State: true,
		},
		TableId:        parent.Id,
		RelatedTableId: child.Id,
		ReferenceKey:   seed.referenceKey,
		ForeignKey:     seed.foreignKey,
		RelationType:   seed.relationType,
	}).Error
}
