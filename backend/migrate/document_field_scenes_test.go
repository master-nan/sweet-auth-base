package main

import (
	"slices"
	"testing"

	"backend/enum"
	testutil "backend/internal/test"
	"backend/model"
	"github.com/google/go-cmp/cmp"
	"gorm.io/gorm"
)

func TestDocumentFieldScenesV28Upgrade(t *testing.T) {
	steps := migrationSteps()
	i := slices.IndexFunc(steps, func(s migrationStep) bool { return s.version == 28 })
	if i < 0 || steps[i].name != "document_field_scenes" {
		t.Fatal("v28 document_field_scenes is not registered")
	}
	for _, engine := range []struct {
		name string
		open func(*testing.T) *gorm.DB
	}{
		{"SQLite", func(t *testing.T) *gorm.DB { return testutil.OpenSQLite(t) }},
		{"PostgreSQL", openMigrationLedgerPostgreSQL},
	} {
		t.Run(engine.name, func(t *testing.T) {
			db := engine.open(t)
			if err := db.AutoMigrate(&model.SysTableRelation{}); err != nil {
				t.Fatalf("create v27 relation metadata: %v", err)
			}
			testutil.MustCreate(t, db, &[]model.SysTableRelation{
				{Basic: model.Basic{Id: 1, State: true}, TableId: 30, RelatedTableId: 20, RelationType: enum.OneToMany, DocumentTab: "Lines"},
				{Basic: model.Basic{Id: 2, State: true}, TableId: 30, RelatedTableId: 40, RelationType: enum.OneToMany},
			})
			// Model auto-migration would create the v28 columns before the upgrade under test.
			if err := db.Exec(`CREATE TABLE sys_table_field (
				id BIGINT PRIMARY KEY, table_id BIGINT NOT NULL, field_code TEXT NOT NULL,
				is_list_show BOOLEAN NOT NULL, is_insert_show BOOLEAN NOT NULL,
				is_update_show BOOLEAN NOT NULL, gmt_delete TIMESTAMP
			)`).Error; err != nil {
				t.Fatalf("create legacy field metadata: %v", err)
			}
			if err := db.Exec(`INSERT INTO sys_table_field
				(id, table_id, field_code, is_list_show, is_insert_show, is_update_show) VALUES
				(101, 10, 'caption', TRUE, FALSE, FALSE),
				(102, 20, 'caption', TRUE, FALSE, FALSE),
				(103, 10, 'archived_note', FALSE, FALSE, FALSE),
				(104, 10, 'gmt_create', FALSE, FALSE, FALSE),
				(105, 10, 'password', FALSE, FALSE, FALSE),
				(106, 40, 'ordinary_note', FALSE, FALSE, FALSE)`).Error; err != nil {
				t.Fatalf("seed legacy field metadata: %v", err)
			}

			yes, no := true, false
			type flags struct {
				ID               int
				DetailVisible    *bool
				IsInsertReadonly *bool
				IsUpdateReadonly *bool
			}
			runAndCheck := func(standaloneCaption *bool) {
				t.Helper()
				if err := db.Transaction(steps[i].run); err != nil {
					t.Fatalf("run v28: %v", err)
				}
				var got []flags
				if err := db.Table("sys_table_field").Order("id").Find(&got).Error; err != nil {
					t.Fatalf("read upgraded field scenes: %v", err)
				}
				want := []flags{
					{101, standaloneCaption, &no, &no},
					{102, nil, &no, &no},
					{103, &yes, &no, &no},
					{104, nil, &no, &no},
					{105, nil, &no, &no},
					{106, &yes, &no, &no},
				}
				if diff := cmp.Diff(want, got); diff != "" {
					t.Fatalf("v28 field scenes (-want +got):\n%s", diff)
				}
			}
			runAndCheck(&yes)
			result := db.Table("sys_table_field").Where("id = ?", 101).UpdateColumn("detail_visible", false)
			if result.Error != nil || result.RowsAffected != 1 {
				t.Fatalf("set explicit false: rows=%d err=%v", result.RowsAffected, result.Error)
			}
			runAndCheck(&no)
		})
	}
}
