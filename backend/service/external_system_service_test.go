package service

import (
	"backend/dto/request"
	"backend/enum"
	"backend/internal/audit"
	apperrors "backend/internal/errors"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"backend/internal/database"
	"gorm.io/gorm"
)

type externalSystemAuditWriter struct {
	mu       sync.Mutex
	records  []TransactionalAuditRecord
	subjects []audit.AuditSubject
}

func (w *externalSystemAuditWriter) RecordTransactionalAuditContext(
	ctx context.Context,
	_ *gorm.DB,
	record TransactionalAuditRecord,
) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	subject, _ := audit.GetAuditSubject(ctx)
	w.records = append(w.records, record)
	w.subjects = append(w.subjects, subject)
	return nil
}

func TestExternalSystemServiceLifecycleAndAudit(t *testing.T) {
	writer := &externalSystemAuditWriter{}
	svc, db := newExternalSystemTestSubject(t, writer)
	ctx := audit.WithAuditSubject(context.Background(), audit.NewAuditSubject(9, "integration-admin"))

	created, err := svc.Create(ctx, externalSystemCreateRequest("demo_erp"))
	if err != nil {
		t.Fatalf("create external system: %v", err)
	}
	if created.Status != model.ExternalSystemStatusDraft || created.Revision != 1 {
		t.Fatalf("unexpected created state: %+v", created)
	}

	changedCode := "other_code"
	_, err = svc.Update(ctx, created.Id, request.ExternalSystemUpdateReq{
		SystemCode: &changedCode,
		Revision:   created.Revision,
	})
	if !errors.Is(err, apperrors.ErrExternalSystemFieldImmutable) {
		t.Fatalf("immutable code error = %v", err)
	}

	name := "生产 ERP"
	updated, err := svc.Update(ctx, created.Id, request.ExternalSystemUpdateReq{
		Name:     &name,
		Revision: created.Revision,
	})
	if err != nil {
		t.Fatalf("update external system: %v", err)
	}
	if updated.Name != name || updated.Revision != 2 {
		t.Fatalf("unexpected update result: %+v", updated)
	}

	enabled, err := svc.Enable(ctx, created.Id, updated.Revision)
	if err != nil {
		t.Fatalf("enable external system: %v", err)
	}
	if enabled.Status != model.ExternalSystemStatusEnabled {
		t.Fatalf("unexpected enabled state: %+v", enabled)
	}
	disabled, err := svc.Disable(ctx, created.Id, enabled.Revision)
	if err != nil {
		t.Fatalf("disable external system: %v", err)
	}
	if disabled.Status != model.ExternalSystemStatusDisabled {
		t.Fatalf("unexpected disabled state: %+v", disabled)
	}
	if _, err := svc.Disable(ctx, created.Id, disabled.Revision); err != nil {
		t.Fatalf("idempotent disable: %v", err)
	}

	var stored model.ExternalSystem
	if err := db.First(&stored, created.Id).Error; err != nil {
		t.Fatalf("reload external system: %v", err)
	}
	if stored.SystemCode != "demo_erp" || stored.Revision != 4 {
		t.Fatalf("unexpected stored external system: %+v", stored)
	}
	if len(writer.records) != 4 || writer.subjects[0].UserID != 9 {
		t.Fatalf("unexpected audit records: %+v subjects=%+v", writer.records, writer.subjects)
	}
}

func TestExternalSystemServiceValidationUniquenessAndReferences(t *testing.T) {
	svc, db := newExternalSystemTestSubject(t, &externalSystemAuditWriter{})
	ctx := context.Background()
	created, err := svc.Create(ctx, externalSystemCreateRequest("demo_tms"))
	if err != nil {
		t.Fatalf("create fixture: %v", err)
	}
	if _, err := svc.Create(ctx, externalSystemCreateRequest("demo_tms")); !errors.Is(err, apperrors.ErrExternalSystemCodeDuplicate) {
		t.Fatalf("duplicate code error = %v", err)
	}
	invalidCode := externalSystemCreateRequest("中文 编码")
	if _, err := svc.Create(ctx, invalidCode); !errors.Is(err, apperrors.ErrExternalSystemCodeInvalid) {
		t.Fatalf("invalid code error = %v", err)
	}
	invalidURL := externalSystemCreateRequest("private_url")
	invalidURL.BaseURL = "http://127.0.0.1/internal"
	if _, err := svc.Create(ctx, invalidURL); !errors.Is(err, apperrors.ErrExternalSystemBaseURLInvalid) {
		t.Fatalf("invalid URL error = %v", err)
	}
	if _, err := svc.Enable(ctx, created.Id, created.Revision+1); !errors.Is(err, apperrors.ErrExternalSystemRevisionConflict) {
		t.Fatalf("revision conflict error = %v", err)
	}

	if err := db.Exec("CREATE TABLE integration_interface_definition (id INTEGER PRIMARY KEY, external_system_id INTEGER, gmt_delete DATETIME)").Error; err != nil {
		t.Fatalf("create future reference table: %v", err)
	}
	if err := db.Exec("INSERT INTO integration_interface_definition (id, external_system_id) VALUES (?, ?)", 1, created.Id).Error; err != nil {
		t.Fatalf("insert future reference: %v", err)
	}
	nextType := model.ExternalSystemTypeTMS
	if _, err := svc.Update(ctx, created.Id, request.ExternalSystemUpdateReq{
		SystemType: &nextType,
		Revision:   created.Revision,
	}); !errors.Is(err, apperrors.ErrExternalSystemReferenced) {
		t.Fatalf("referenced type change error = %v", err)
	}
}

func TestExternalSystemServiceQueryAndDTOWhitelist(t *testing.T) {
	svc, _ := newExternalSystemTestSubject(t, &externalSystemAuditWriter{})
	ctx := context.Background()
	created, err := svc.Create(ctx, externalSystemCreateRequest("demo_hr"))
	if err != nil {
		t.Fatalf("create fixture: %v", err)
	}
	page, err := svc.Page(ctx, request.ExternalSystemQueryReq{
		Page:       1,
		Num:        10,
		QuickQuery: &request.QuickQuery{Keyword: "demo"},
	}, externalSystemQueryTableForTest())
	if err != nil {
		t.Fatalf("page external systems: %v", err)
	}
	if page.Total != 1 || len(page.Data) != 1 || page.Data[0].BaseURLSummary != "https://api.example.com" {
		t.Fatalf("unexpected page: %+v", page)
	}
	payload, err := json.Marshal(page.Data[0])
	if err != nil {
		t.Fatalf("marshal list DTO: %v", err)
	}
	for _, forbidden := range []string{"base_url\"", "gmt_delete", "delete_user", "create_user"} {
		if strings.Contains(string(payload), forbidden) {
			t.Fatalf("list DTO leaked %q: %s", forbidden, payload)
		}
	}
	detail, err := svc.Get(ctx, created.Id)
	if err != nil || detail.BaseURL != "https://api.example.com" {
		t.Fatalf("unexpected detail: %+v err=%v", detail, err)
	}
}

func newExternalSystemTestSubject(
	t *testing.T,
	auditWriter StandardContextAuditWriter,
) (*ExternalSystemService, *gorm.DB) {
	t.Helper()
	db := testutil.OpenSQLite(t, &model.ExternalSystem{})
	sf, err := utils.NewSnowflake(1)
	if err != nil {
		t.Fatalf("create snowflake: %v", err)
	}
	repository := impl.NewExternalSystemRepositoryImpl(&database.PrimaryDB{DB: db})
	return NewExternalSystemService(repository, sf, auditWriter), db
}

func externalSystemCreateRequest(code string) request.ExternalSystemCreateReq {
	return request.ExternalSystemCreateReq{
		SystemCode:      code,
		Name:            "测试外部系统",
		SystemType:      model.ExternalSystemTypeERP,
		BaseURL:         "https://api.example.com/",
		OwnerIdentifier: "owner-001",
		OwnerName:       "实施负责人",
		Description:     "用于集成配置测试",
	}
}

func externalSystemQueryTableForTest() model.SysTable {
	return model.SysTable{
		TableCode: "integration_external_system",
		TableFields: []model.SysTableField{
			{Basic: model.Basic{State: true}, FieldCode: "system_code", FieldType: enum.VarcharFieldType, IsQuickSearch: true},
			{Basic: model.Basic{State: true}, FieldCode: "name", FieldType: enum.VarcharFieldType, IsQuickSearch: true},
			{Basic: model.Basic{State: true}, FieldCode: "system_type", FieldType: enum.VarcharFieldType},
			{Basic: model.Basic{State: true}, FieldCode: "status", FieldType: enum.VarcharFieldType},
			{Basic: model.Basic{State: true}, FieldCode: "owner_identifier", FieldType: enum.VarcharFieldType},
			{Basic: model.Basic{State: true}, FieldCode: "owner_name", FieldType: enum.VarcharFieldType},
		},
	}
}
