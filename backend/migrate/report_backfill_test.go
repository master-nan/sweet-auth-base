package main

import (
	"backend/internal/utils"
	"backend/model"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestBackfillReportDefinitionVersionsRepairsExistingVersions(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.ReportDefinition{}, &model.ReportDefinitionVersion{}); err != nil {
		t.Fatalf("migrate report tables: %v", err)
	}
	sf, err := utils.NewSnowflake(2)
	if err != nil {
		t.Fatalf("create snowflake: %v", err)
	}
	queryConfig := datatypes.JSON([]byte(`{"datasets":[]}`))
	layoutConfig := datatypes.JSON([]byte(`{"view":"sheet"}`))

	reportWithPublished := model.ReportDefinition{
		Basic:        model.Basic{Id: 1, State: true},
		Code:         "published_exists",
		Name:         "published_exists",
		Status:       reportMigrationStatusPublished,
		QueryConfig:  queryConfig,
		LayoutConfig: layoutConfig,
	}
	reportWithoutPublished := model.ReportDefinition{
		Basic:        model.Basic{Id: 2, State: true},
		Code:         "max_version_fallback",
		Name:         "max_version_fallback",
		Status:       reportMigrationStatusPublished,
		QueryConfig:  queryConfig,
		LayoutConfig: layoutConfig,
	}
	reportWithoutVersion := model.ReportDefinition{
		Basic:        model.Basic{Id: 3, State: true},
		Code:         "create_initial",
		Name:         "create_initial",
		Status:       reportMigrationStatusPublished,
		QueryConfig:  queryConfig,
		LayoutConfig: layoutConfig,
	}
	invalidReport := model.ReportDefinition{
		Basic:        model.Basic{Id: 4, State: true},
		Code:         "invalid_config",
		Name:         "invalid_config",
		Status:       reportMigrationStatusPublished,
		QueryConfig:  datatypes.JSON([]byte(`{invalid`)),
		LayoutConfig: layoutConfig,
	}
	if err := db.Create(&[]model.ReportDefinition{reportWithPublished, reportWithoutPublished, reportWithoutVersion, invalidReport}).Error; err != nil {
		t.Fatalf("seed reports: %v", err)
	}
	versions := []model.ReportDefinitionVersion{
		{Basic: model.Basic{Id: 101, State: false}, ReportId: 1, VersionNo: 1, ReportCode: reportWithPublished.Code, ReportName: reportWithPublished.Name, QueryConfig: queryConfig, LayoutConfig: layoutConfig, Status: reportMigrationStatusPublished},
		{Basic: model.Basic{Id: 102, State: true}, ReportId: 1, VersionNo: 2, ReportCode: reportWithPublished.Code, ReportName: reportWithPublished.Name, QueryConfig: queryConfig, LayoutConfig: layoutConfig, Status: reportMigrationStatusArchived},
		{Basic: model.Basic{Id: 201, State: true}, ReportId: 2, VersionNo: 1, ReportCode: reportWithoutPublished.Code, ReportName: reportWithoutPublished.Name, QueryConfig: queryConfig, LayoutConfig: layoutConfig, Status: reportMigrationStatusArchived},
		{Basic: model.Basic{Id: 202, State: true}, ReportId: 2, VersionNo: 2, ReportCode: reportWithoutPublished.Code, ReportName: reportWithoutPublished.Name, QueryConfig: queryConfig, LayoutConfig: layoutConfig, Status: reportMigrationStatusArchived},
	}
	if err := db.Create(&versions).Error; err != nil {
		t.Fatalf("seed versions: %v", err)
	}

	if err := backfillReportDefinitionVersions(db, sf); err != nil {
		t.Fatalf("backfill report versions: %v", err)
	}
	if err := backfillReportDefinitionVersions(db, sf); err != nil {
		t.Fatalf("backfill report versions should be idempotent: %v", err)
	}

	assertBackfilledReportPointer(t, db, 1, 101)
	assertBackfilledReportPointer(t, db, 2, 202)
	assertVersionStatus(t, db, 101, reportMigrationStatusPublished, true)
	assertVersionStatus(t, db, 102, reportMigrationStatusArchived, true)
	assertVersionStatus(t, db, 201, reportMigrationStatusArchived, true)
	assertVersionStatus(t, db, 202, reportMigrationStatusPublished, true)
	assertBackfillVersionCount(t, db, 3, 1)
	assertBackfilledReportPointerIsSet(t, db, 3)
	assertBackfillVersionCount(t, db, 4, 0)
	assertBackfilledReportPointer(t, db, 4, 0)
}

func assertBackfilledReportPointer(t *testing.T, db *gorm.DB, reportId int, wantVersionId int) {
	t.Helper()
	var report model.ReportDefinition
	if err := db.First(&report, reportId).Error; err != nil {
		t.Fatalf("load report %d: %v", reportId, err)
	}
	if report.PublishedVersionId != wantVersionId {
		t.Fatalf("report %d published_version_id=%d, want %d", reportId, report.PublishedVersionId, wantVersionId)
	}
}

func assertBackfilledReportPointerIsSet(t *testing.T, db *gorm.DB, reportId int) {
	t.Helper()
	var report model.ReportDefinition
	if err := db.First(&report, reportId).Error; err != nil {
		t.Fatalf("load report %d: %v", reportId, err)
	}
	if report.PublishedVersionId <= 0 {
		t.Fatalf("report %d published_version_id should be set", reportId)
	}
	var version model.ReportDefinitionVersion
	if err := db.First(&version, report.PublishedVersionId).Error; err != nil {
		t.Fatalf("load report %d backfilled version: %v", reportId, err)
	}
	if version.ReportId != reportId || version.VersionNo != 1 || version.Status != reportMigrationStatusPublished || !version.State {
		t.Fatalf("unexpected initial version for report %d: %#v", reportId, version)
	}
}

func assertVersionStatus(t *testing.T, db *gorm.DB, versionId int, wantStatus string, wantState bool) {
	t.Helper()
	var version model.ReportDefinitionVersion
	if err := db.First(&version, versionId).Error; err != nil {
		t.Fatalf("load version %d: %v", versionId, err)
	}
	if version.Status != wantStatus || version.State != wantState {
		t.Fatalf("version %d status/state = %s/%v, want %s/%v", versionId, version.Status, version.State, wantStatus, wantState)
	}
}

func assertBackfillVersionCount(t *testing.T, db *gorm.DB, reportId int, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(&model.ReportDefinitionVersion{}).Where("report_id = ?", reportId).Count(&count).Error; err != nil {
		t.Fatalf("count report %d versions: %v", reportId, err)
	}
	if count != want {
		t.Fatalf("report %d version count=%d, want %d", reportId, count, want)
	}
}
