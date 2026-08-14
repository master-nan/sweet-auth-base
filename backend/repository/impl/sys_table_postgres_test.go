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
