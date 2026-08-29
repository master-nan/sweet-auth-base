package database

import (
	"backend/internal/utils"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type snowflakeIntFixture struct {
	ID   int `gorm:"primaryKey;autoIncrement:false"`
	Name string
}

type snowflakeUintFixture struct {
	ID   uint `gorm:"primaryKey;autoIncrement:false"`
	Name string
}

func TestRegisterSnowflakeIDsAssignsZeroSinglePrimaryKeys(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:snowflake_callback?mode=memory&cache=shared"), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open sqlite handle: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	sf, err := utils.NewSnowflake(3)
	if err != nil {
		t.Fatalf("create snowflake: %v", err)
	}
	if err := RegisterSnowflakeIDs(db, sf); err != nil {
		t.Fatalf("register callback: %v", err)
	}
	if err := db.AutoMigrate(&snowflakeIntFixture{}, &snowflakeUintFixture{}); err != nil {
		t.Fatalf("migrate fixtures: %v", err)
	}

	intRows := []snowflakeIntFixture{{Name: "first"}, {Name: "second"}}
	if err := db.Create(&intRows).Error; err != nil {
		t.Fatalf("create int rows: %v", err)
	}
	if intRows[0].ID <= 0 || intRows[1].ID <= 0 || intRows[0].ID == intRows[1].ID {
		t.Fatalf("unexpected int snowflake IDs: %+v", intRows)
	}

	uintRow := snowflakeUintFixture{Name: "third"}
	if err := db.Create(&uintRow).Error; err != nil {
		t.Fatalf("create uint row: %v", err)
	}
	if uintRow.ID == 0 {
		t.Fatal("uint snowflake ID was not assigned")
	}

	explicit := snowflakeIntFixture{ID: 42, Name: "reserved seed"}
	if err := db.Create(&explicit).Error; err != nil {
		t.Fatalf("create explicit ID row: %v", err)
	}
	if explicit.ID != 42 {
		t.Fatalf("explicit ID changed to %d", explicit.ID)
	}
}
