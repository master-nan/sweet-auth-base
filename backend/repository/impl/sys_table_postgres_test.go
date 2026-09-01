package impl

import (
	"backend/internal/database"
	testutil "backend/internal/test"
	"backend/model"
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func TestSysTableRepositoryPostgreSQLDDLAndRollback(t *testing.T) {
	dsn := testutil.PostgreSQLDSN(t)
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	schemaName := fmt.Sprintf("metadata_ddl_%d", time.Now().UnixNano())
	if err := admin.Exec(fmt.Sprintf(`CREATE SCHEMA %q`, schemaName)).Error; err != nil {
		t.Fatalf("create metadata DDL schema: %v", err)
	}
	t.Cleanup(func() {
		_ = admin.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS %q CASCADE`, schemaName)).Error
	})

	db, err := gorm.Open(postgres.Open(postgresClaimDSN(t, dsn, schemaName)), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
		Logger:         logger.Default.LogMode(logger.Silent),
		NowFunc:        model.Now,
	})
	if err != nil {
		t.Fatalf("open isolated metadata DDL schema: %v", err)
	}
	if err := db.Exec(`CREATE TABLE metadata_ddl_target (id bigint PRIMARY KEY, code varchar(64) NOT NULL)`).Error; err != nil {
		t.Fatalf("create metadata target: %v", err)
	}

	repo := NewSysTableRepositoryImpl(&database.PrimaryDB{DB: db})
	if err := repo.CreateTableColumn(db, "metadata_ddl_target", "display_name", "varchar(128)"); err != nil {
		t.Fatalf("add metadata column: %v", err)
	}
	if !repo.HasTableColumn(db, "metadata_ddl_target", "display_name") {
		t.Fatal("PostgreSQL metadata column was not created")
	}
	if err := repo.ChangeTableColumn(db, "metadata_ddl_target", "display_name", "title", "varchar(256)"); err != nil {
		t.Fatalf("rename metadata column: %v", err)
	}
	if repo.HasTableColumn(db, "metadata_ddl_target", "display_name") || !repo.HasTableColumn(db, "metadata_ddl_target", "title") {
		t.Fatal("PostgreSQL metadata column rename did not converge")
	}
	if err := repo.SetTableColumnComment(db, "metadata_ddl_target", "title", "显示标题"); err != nil {
		t.Fatalf("set metadata column comment: %v", err)
	}
	columns, err := repo.FetchTableMetadata(t.Context(), db, schemaName, "metadata_ddl_target")
	if err != nil {
		t.Fatalf("fetch metadata column comments: %v", err)
	}
	foundComment := false
	for _, column := range columns {
		if column.ColumnName == "title" && column.ColumnComment == "显示标题" {
			foundComment = true
			break
		}
	}
	if !foundComment {
		t.Fatalf("physical column comment not returned: %+v", columns)
	}
	if err := db.Exec(`CREATE TABLE sys_table (
		id bigint PRIMARY KEY, table_code varchar(128) NOT NULL, gmt_delete timestamptz
	)`).Error; err != nil {
		t.Fatalf("create table metadata fixture: %v", err)
	}
	if err := db.Exec(`CREATE TABLE sys_table_field (
		id bigint PRIMARY KEY, table_id bigint NOT NULL, field_code varchar(128) NOT NULL,
		field_name varchar(128) NOT NULL, gmt_delete timestamptz
	)`).Error; err != nil {
		t.Fatalf("create field metadata fixture: %v", err)
	}
	if err := db.Exec(`INSERT INTO sys_table (id, table_code) VALUES (1, 'metadata_ddl_target')`).Error; err != nil {
		t.Fatalf("insert table metadata fixture: %v", err)
	}
	if err := db.Exec(`INSERT INTO sys_table_field (id, table_id, field_code, field_name)
		VALUES (1, 1, 'title', '同步标题')`).Error; err != nil {
		t.Fatalf("insert field metadata fixture: %v", err)
	}
	if updated, err := database.SyncMetadataColumnComments(db, "metadata_ddl_target"); err != nil || updated != 1 {
		t.Fatalf("sync metadata column comments updated=%d err=%v", updated, err)
	}
	columns, err = repo.FetchTableMetadata(t.Context(), db, schemaName, "metadata_ddl_target")
	if err != nil {
		t.Fatalf("fetch synchronized column comments: %v", err)
	}
	foundComment = false
	for _, column := range columns {
		if column.ColumnName == "title" && column.ColumnComment == "同步标题" {
			foundComment = true
			break
		}
	}
	if !foundComment {
		t.Fatalf("metadata column comment was not synchronized: %+v", columns)
	}

	if err := repo.CreateTableIndex(db, true, "uidx_metadata_ddl_code", "metadata_ddl_target", "code"); err != nil {
		t.Fatalf("create metadata index: %v", err)
	}
	var indexCount int64
	if err := db.Raw(`SELECT count(*) FROM pg_indexes WHERE schemaname = ? AND indexname = ?`, schemaName, "uidx_metadata_ddl_code").Scan(&indexCount).Error; err != nil || indexCount != 1 {
		t.Fatalf("metadata index count=%d err=%v", indexCount, err)
	}
	if err := repo.DropTableIndex(db, "uidx_metadata_ddl_code", "metadata_ddl_target"); err != nil {
		t.Fatalf("drop metadata index: %v", err)
	}

	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin metadata DDL transaction: %v", tx.Error)
	}
	if err := repo.CreateTableColumn(tx, "metadata_ddl_target", "rolled_back", "varchar(32)"); err != nil {
		t.Fatalf("add rollback probe column: %v", err)
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("rollback metadata DDL transaction: %v", err)
	}
	if repo.HasTableColumn(db, "metadata_ddl_target", "rolled_back") {
		t.Fatal("rolled-back PostgreSQL metadata DDL remained visible")
	}

	if err := repo.DropTableColumn(db, "metadata_ddl_target", "title"); err != nil {
		t.Fatalf("drop metadata column: %v", err)
	}
	if repo.HasTableColumn(db, "metadata_ddl_target", "title") {
		t.Fatal("PostgreSQL metadata column was not dropped")
	}
}
