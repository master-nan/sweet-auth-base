package main

import (
	testutil "backend/internal/test"
	"backend/model"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func TestRemoveLegacyDataPermissionSchemaIsIdempotentAndPreservesNewDomain(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.SysTable{},
		&model.SysTableField{},
		&model.SysTableRelation{},
		&model.SysTableIndex{},
		&model.SysTableIndexField{},
		&model.SysMenuButton{},
		&model.SysRoleMenuButton{},
		&model.SysDictItem{},
		&model.CasbinRule{},
	); err != nil {
		t.Fatalf("migrate platform models: %v", err)
	}
	if err := migrateDataPermissionSchema(db); err != nil {
		t.Fatalf("migrate new data permission schema: %v", err)
	}
	for _, table := range legacyDataPermissionTables {
		if err := db.Exec(`CREATE TABLE "` + table + `" (id integer primary key)`).Error; err != nil {
			t.Fatalf("create legacy table %s: %v", table, err)
		}
	}
	legacyMetadata := model.SysTable{Basic: model.Basic{Id: 8001, State: true}, TableCode: "sys_data_dimension", TableName: "旧维度"}
	if err := db.Create(&legacyMetadata).Error; err != nil {
		t.Fatalf("seed legacy metadata: %v", err)
	}
	legacyButton := model.SysMenuButton{Basic: model.Basic{Id: 8002, State: true}, Code: "system_data_permission_debug", Path: "/admin/data-permission/debug", Method: "GET"}
	if err := db.Create(&legacyButton).Error; err != nil {
		t.Fatalf("seed legacy button: %v", err)
	}
	if err := db.Create(&model.SysRoleMenuButton{RoleId: 1, MenuId: 1, ButtonId: legacyButton.Id}).Error; err != nil {
		t.Fatalf("seed legacy button grant: %v", err)
	}
	if err := db.Create(&model.CasbinRule{PType: "p", V0: "super_admin", V1: legacyButton.Path, V2: legacyButton.Method}).Error; err != nil {
		t.Fatalf("seed legacy casbin rule: %v", err)
	}

	for attempt := 0; attempt < 2; attempt++ {
		if err := removeLegacyDataPermissionSchema(db); err != nil {
			t.Fatalf("cleanup attempt %d: %v", attempt+1, err)
		}
	}
	for _, table := range legacyDataPermissionTables {
		if db.Migrator().HasTable(table) {
			t.Fatalf("legacy table %s still exists", table)
		}
	}
	for _, table := range dataPermissionDomainTableNames() {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("new data permission table %s was removed", table)
		}
	}
	var count int64
	if err := db.Unscoped().Model(&model.SysTable{}).Where("table_code = ?", legacyMetadata.TableCode).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("legacy metadata count=%d err=%v", count, err)
	}
	if err := db.Unscoped().Model(&model.SysMenuButton{}).Where("id = ?", legacyButton.Id).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("legacy button count=%d err=%v", count, err)
	}
}

func TestRemoveLegacyDataPermissionSchemaPostgreSQL(t *testing.T) {
	dsn := testutil.PostgreSQLDSN(t)

	adminDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open PostgreSQL test database: %v", err)
	}
	schemaName := fmt.Sprintf("legacy_data_permission_cleanup_test_%d", time.Now().UnixNano())
	if err = adminDB.Exec(fmt.Sprintf(`CREATE SCHEMA %q`, schemaName)).Error; err != nil {
		t.Fatalf("create PostgreSQL test schema: %v", err)
	}
	t.Cleanup(func() {
		_ = adminDB.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS %q CASCADE`, schemaName)).Error
	})

	db, err := gorm.Open(postgres.Open(postgresDSNWithSearchPath(t, dsn, schemaName)), &gorm.Config{
		NamingStrategy:                           schema.NamingStrategy{SingularTable: true},
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   logger.Default.LogMode(logger.Silent),
		NowFunc:                                  model.Now,
	})
	if err != nil {
		t.Fatalf("open isolated PostgreSQL cleanup schema: %v", err)
	}
	createDataPermissionPostgresPrerequisites(t, db, schemaName)
	if err = db.Exec(`ALTER TABLE "sys_table" ADD COLUMN "table_code" varchar(128)`).Error; err != nil {
		t.Fatalf("extend PostgreSQL sys_table cleanup prerequisite: %v", err)
	}
	if err = migrateDataPermissionSchema(db); err != nil {
		t.Fatalf("migrate new data permission schema: %v", err)
	}
	for _, table := range legacyDataPermissionTables {
		if err = db.Exec(fmt.Sprintf(`CREATE TABLE %q (id bigint PRIMARY KEY)`, table)).Error; err != nil {
			t.Fatalf("create legacy PostgreSQL table %s: %v", table, err)
		}
	}

	for attempt := 0; attempt < 2; attempt++ {
		if err = removeLegacyDataPermissionSchema(db); err != nil {
			t.Fatalf("PostgreSQL cleanup attempt %d: %v", attempt+1, err)
		}
	}
	for _, table := range legacyDataPermissionTables {
		if db.Migrator().HasTable(table) {
			t.Fatalf("legacy PostgreSQL table %s still exists", table)
		}
	}
	for _, table := range dataPermissionDomainTableNames() {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("new PostgreSQL data permission table %s was removed", table)
		}
	}
}

func dataPermissionDomainTableNames() []string {
	return []string{
		"sys_data_dimension_definition",
		"sys_data_resource",
		"sys_data_resource_operation",
		"sys_data_ownership_field",
		"sys_data_policy",
		"sys_data_policy_rule",
		"sys_data_grant",
	}
}
