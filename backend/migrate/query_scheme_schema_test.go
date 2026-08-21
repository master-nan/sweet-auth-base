package main

import (
	testutil "backend/internal/test"
	"backend/model"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func TestQuerySchemePostgres16ConstraintsAndIdempotency(t *testing.T) {
	db, cleanup := openQuerySchemePostgresSchema(t)
	defer cleanup()
	if err := db.AutoMigrate(&model.SysUser{}, &model.SysRole{}, &model.SysMenu{}); err != nil {
		t.Fatalf("migrate prerequisites: %v", err)
	}
	if err := migrateQuerySchemeSchema(db); err != nil {
		t.Fatalf("first migration: %v", err)
	}
	if err := migrateQuerySchemeSchema(db); err != nil {
		t.Fatalf("second migration: %v", err)
	}
	var version int
	if err := db.Raw("SHOW server_version_num").Scan(&version).Error; err != nil || version < 160000 {
		t.Fatalf("PostgreSQL 16 required, version=%d err=%v", version, err)
	}

	user := model.SysUser{Basic: model.Basic{Id: 1, State: true}, UserName: "scheme-owner"}
	role := model.SysRole{Basic: model.Basic{Id: 2, State: true}, Name: "scheme-role"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	scope := "system.user.list"
	menu := model.SysMenu{Basic: model.Basic{Id: 3, State: true}, Name: "scope-menu", QueryScopeCode: &scope}
	if err := db.Create(&menu).Error; err != nil {
		t.Fatal(err)
	}
	duplicateMenu := menu
	duplicateMenu.Id = 4
	duplicateMenu.Name = "scope-menu-copy"
	assertPostgresQuerySchemeRejected(t, db.Create(&duplicateMenu).Error, "duplicate active scope")

	personal := postgresQueryScheme(10, "mine", scope, model.QuerySchemeTypePersonal)
	personal.OwnerUserID = &user.Id
	personal.IsDefault = true
	if err := db.Create(&personal).Error; err != nil {
		t.Fatalf("create personal scheme: %v", err)
	}
	duplicateDefault := postgresQueryScheme(11, "mine-two", scope, model.QuerySchemeTypePersonal)
	duplicateDefault.OwnerUserID = &user.Id
	duplicateDefault.IsDefault = true
	assertPostgresQuerySchemeRejected(t, db.Create(&duplicateDefault).Error, "duplicate personal default")

	invalidOwner := postgresQueryScheme(12, "public-owner", scope, model.QuerySchemeTypePublic)
	invalidOwner.OwnerUserID = &user.Id
	assertPostgresQuerySchemeRejected(t, db.Create(&invalidOwner).Error, "shared owner")
	invalidRevision := postgresQueryScheme(13, "bad-revision", scope, model.QuerySchemeTypePublic)
	invalidRevision.Revision = -1
	assertPostgresQuerySchemeRejected(t, db.Create(&invalidRevision).Error, "revision")
	large := postgresQueryScheme(14, "large", scope, model.QuerySchemeTypePublic)
	large.QueryPayload = datatypes.JSON([]byte(`{"expressions":[],"quick_query":{"keyword":"` + strings.Repeat("x", 33000) + `"},"order":{},"bindings":[]}`))
	assertPostgresQuerySchemeRejected(t, db.Create(&large).Error, "payload size")

	roleScheme := postgresQueryScheme(15, "role", scope, model.QuerySchemeTypeRole)
	if err := db.Create(&roleScheme).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.QuerySchemeRole{SchemeID: roleScheme.Id, RoleID: role.Id}).Error; err != nil {
		t.Fatalf("create role relation: %v", err)
	}
	assertPostgresQuerySchemeRejected(t, db.Create(&model.QuerySchemeRole{SchemeID: roleScheme.Id, RoleID: 999999}).Error, "role FK")

	if err := db.Delete(&personal).Error; err != nil {
		t.Fatal(err)
	}
	reusedName := postgresQueryScheme(16, personal.Name, scope, model.QuerySchemeTypePersonal)
	reusedName.OwnerUserID = &user.Id
	if err := db.Create(&reusedName).Error; err != nil {
		t.Fatalf("soft-deleted name should be reusable: %v", err)
	}
}

func TestRetireDictionaryQueryScopeIsIdempotentAndIsolated(t *testing.T) {
	db := migrateTestDB(t)
	if err := migrateQuerySchemeSchema(db); err != nil {
		t.Fatalf("initial query scheme migration: %v", err)
	}
	dictionaryScope := retiredDictionaryQueryScope
	otherScope := "system.user.list"
	menus := []model.SysMenu{
		{Basic: model.Basic{Id: 301, State: true}, Name: "develop_dictionary", QueryScopeCode: &dictionaryScope},
		{Basic: model.Basic{Id: 302, State: true}, Name: "system_user", QueryScopeCode: &otherScope},
	}
	if err := db.Create(&menus).Error; err != nil {
		t.Fatalf("create scope menus: %v", err)
	}
	ownerID := 8
	schemes := []model.QueryScheme{
		postgresQueryScheme(401, "personal", dictionaryScope, model.QuerySchemeTypePersonal),
		postgresQueryScheme(402, "public", dictionaryScope, model.QuerySchemeTypePublic),
		postgresQueryScheme(403, "role", dictionaryScope, model.QuerySchemeTypeRole),
		postgresQueryScheme(404, "default", dictionaryScope, model.QuerySchemeTypePageDefault),
		postgresQueryScheme(405, "other", otherScope, model.QuerySchemeTypePublic),
	}
	schemes[0].OwnerUserID = &ownerID
	schemes[0].IsDefault = true
	schemes[3].IsDefault = true
	if err := db.Create(&schemes).Error; err != nil {
		t.Fatalf("create query schemes: %v", err)
	}
	if err := db.Create(&model.QuerySchemeRole{SchemeID: 403, RoleID: 9}).Error; err != nil {
		t.Fatalf("create role relation: %v", err)
	}

	for pass := 1; pass <= 2; pass++ {
		if err := migrateQuerySchemeSchema(db); err != nil {
			t.Fatalf("retire dictionary scope pass %d: %v", pass, err)
		}
	}

	var retired []model.QueryScheme
	if err := db.Unscoped().Where("scope_code = ?", dictionaryScope).Find(&retired).Error; err != nil {
		t.Fatalf("load retired schemes: %v", err)
	}
	if len(retired) != 4 {
		t.Fatalf("retired scheme count=%d, want 4", len(retired))
	}
	for _, scheme := range retired {
		if scheme.State || scheme.IsDefault || !scheme.GmtDelete.Valid || scheme.Revision != 2 {
			t.Fatalf("scheme was not retired once: %+v", scheme)
		}
	}
	var roleCount int64
	if err := db.Model(&model.QuerySchemeRole{}).Where("scheme_id = ?", 403).Count(&roleCount).Error; err != nil || roleCount != 0 {
		t.Fatalf("retired role relation count=%d err=%v", roleCount, err)
	}
	var dictionaryMenu, otherMenu model.SysMenu
	if err := db.First(&dictionaryMenu, 301).Error; err != nil || dictionaryMenu.QueryScopeCode != nil {
		t.Fatalf("dictionary menu scope=%v err=%v", dictionaryMenu.QueryScopeCode, err)
	}
	if err := db.First(&otherMenu, 302).Error; err != nil || otherMenu.QueryScopeCode == nil || *otherMenu.QueryScopeCode != otherScope {
		t.Fatalf("other menu scope=%v err=%v", otherMenu.QueryScopeCode, err)
	}
	var other model.QueryScheme
	if err := db.First(&other, 405).Error; err != nil || !other.State || other.Revision != 1 {
		t.Fatalf("other scope scheme=%+v err=%v", other, err)
	}
}

func TestQuerySchemePostgresConcurrentPageDefaultHasOneWinner(t *testing.T) {
	db, cleanup := openQuerySchemePostgresSchema(t)
	defer cleanup()
	if err := db.AutoMigrate(&model.SysUser{}, &model.SysRole{}, &model.SysMenu{}); err != nil {
		t.Fatal(err)
	}
	if err := migrateQuerySchemeSchema(db); err != nil {
		t.Fatal(err)
	}
	scope := "system.role.list"
	start := make(chan struct{})
	errorsByWriter := make(chan error, 2)
	var wait sync.WaitGroup
	for id := 21; id <= 22; id++ {
		id := id
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			value := postgresQueryScheme(id, fmt.Sprintf("default-%d", id), scope, model.QuerySchemeTypePageDefault)
			value.IsDefault = true
			errorsByWriter <- db.Create(&value).Error
		}()
	}
	close(start)
	wait.Wait()
	close(errorsByWriter)
	successes := 0
	for err := range errorsByWriter {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent defaults successes=%d, want 1", successes)
	}
}

func openQuerySchemePostgresSchema(t *testing.T) (*gorm.DB, func()) {
	t.Helper()
	dsn := testutil.PostgreSQLDSN(t)
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	schemaName := fmt.Sprintf("query_scheme_%d", time.Now().UnixNano())
	if err := admin.Exec(fmt.Sprintf(`CREATE SCHEMA "%s"`, schemaName)).Error; err != nil {
		t.Fatalf("create schema: %v", err)
	}
	db, err := gorm.Open(postgres.Open(postgresDSNWithSearchPath(t, dsn, schemaName)), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true}, DisableForeignKeyConstraintWhenMigrating: true,
		Logger: logger.Default.LogMode(logger.Silent), NowFunc: model.Now,
	})
	if err != nil {
		t.Fatalf("open isolated schema: %v", err)
	}
	return db, func() { _ = admin.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS "%s" CASCADE`, schemaName)).Error }
}

func postgresQueryScheme(id int, name, scope string, schemeType model.QuerySchemeType) model.QueryScheme {
	return model.QueryScheme{
		Basic: model.Basic{Id: id, State: true}, Name: name, ScopeCode: scope, SchemeType: schemeType,
		QuerySchemaVersion: 1, QueryPayload: datatypes.JSON([]byte(`{"expressions":[],"quick_query":{"keyword":""},"order":{"field":"","is_asc":false},"bindings":[]}`)),
		Enabled: true, Revision: 1,
	}
}

func assertPostgresQuerySchemeRejected(t *testing.T, err error, scenario string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected PostgreSQL to reject %s", scenario)
	}
}
