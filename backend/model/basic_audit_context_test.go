package model

import (
	"backend/internal/audit"
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type basicAuditContextFixture struct {
	Basic
	Name string `gorm:"size:64"`
}

func TestBasicHooksReadAuditSubjectFromStandardContext(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&basicAuditContextFixture{}); err != nil {
		t.Fatalf("migrate fixture: %v", err)
	}

	ctx := audit.WithAuditSubject(context.Background(), audit.NewAuditSubject(41, "model-hook-user"))
	row := basicAuditContextFixture{Name: "created"}
	if err := db.WithContext(ctx).Create(&row).Error; err != nil {
		t.Fatalf("create fixture: %v", err)
	}
	if row.CreateUser == nil || *row.CreateUser != 41 {
		t.Fatalf("create user = %v, want 41", row.CreateUser)
	}

	if err := db.WithContext(ctx).Model(&basicAuditContextFixture{}).Where("name = ?", "created").Update("name", "updated").Error; err != nil {
		t.Fatalf("update fixture: %v", err)
	}
	var updated basicAuditContextFixture
	if err := db.Where("name = ?", "updated").First(&updated).Error; err != nil {
		t.Fatalf("load updated fixture: %v", err)
	}
	if updated.ModifyUser == nil || *updated.ModifyUser != 41 {
		t.Fatalf("modify user = %v, want 41", updated.ModifyUser)
	}
}

func TestBasicHooksDoNotInventAuditSubject(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&basicAuditContextFixture{}); err != nil {
		t.Fatalf("migrate fixture: %v", err)
	}

	row := basicAuditContextFixture{Name: "without-subject"}
	if err := db.WithContext(context.Background()).Create(&row).Error; err != nil {
		t.Fatalf("create fixture: %v", err)
	}
	if row.CreateUser != nil || row.CreateName != nil {
		t.Fatalf("unexpected audit subject: user=%v name=%v", row.CreateUser, row.CreateName)
	}
}
