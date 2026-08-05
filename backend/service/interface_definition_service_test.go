package service

import (
	"backend/dto/request"
	"backend/enum"
	"backend/internal/database"
	apperrors "backend/internal/errors"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"gorm.io/gorm"
)

func TestInterfaceDefinitionServiceVersionLifecycleAndAudit(t *testing.T) {
	writer := &externalSystemAuditWriter{}
	svc, db, system := newInterfaceDefinitionTestSubject(t, writer)
	ctx := context.Background()

	created, err := svc.Create(ctx, interfaceDefinitionCreateRequest(system.Id, "order_query"))
	if err != nil {
		t.Fatalf("create interface definition: %v", err)
	}
	if created.Status != model.InterfaceDefinitionStatusDraft || created.Version != 1 || created.Revision != 1 {
		t.Fatalf("unexpected draft: %+v", created)
	}
	changedCode := "other_code"
	if _, err := svc.Update(ctx, created.Id, request.InterfaceDefinitionUpdateReq{InterfaceCode: &changedCode, Revision: created.Revision}); !errors.Is(err, apperrors.ErrInterfaceFieldImmutable) {
		t.Fatalf("immutable interface code error = %v", err)
	}
	name := "订单明细查询"
	updated, err := svc.Update(ctx, created.Id, request.InterfaceDefinitionUpdateReq{Name: &name, Revision: created.Revision})
	if err != nil || updated.Name != name || updated.Revision != 2 {
		t.Fatalf("update draft = %+v err=%v", updated, err)
	}
	enabled, err := svc.Enable(ctx, created.Id, updated.Revision)
	if err != nil || enabled.Status != model.InterfaceDefinitionStatusEnabled {
		t.Fatalf("enable v1 = %+v err=%v", enabled, err)
	}
	if _, err := svc.Update(ctx, enabled.Id, request.InterfaceDefinitionUpdateReq{Name: &name, Revision: enabled.Revision}); !errors.Is(err, apperrors.ErrInterfaceStatusInvalid) {
		t.Fatalf("enabled update error = %v", err)
	}
	v2, err := svc.CreateVersion(ctx, enabled.Id, enabled.Revision)
	if err != nil || v2.Version != 2 || v2.Status != model.InterfaceDefinitionStatusDraft || v2.Revision != 1 {
		t.Fatalf("create v2 = %+v err=%v", v2, err)
	}
	if _, err := svc.Enable(ctx, v2.Id, v2.Revision); !errors.Is(err, apperrors.ErrInterfaceEnabledVersionConflict) {
		t.Fatalf("enabled version conflict = %v", err)
	}
	disabled, err := svc.Disable(ctx, enabled.Id, enabled.Revision)
	if err != nil || disabled.Status != model.InterfaceDefinitionStatusDisabled {
		t.Fatalf("disable v1 = %+v err=%v", disabled, err)
	}
	v2Enabled, err := svc.Enable(ctx, v2.Id, v2.Revision)
	if err != nil || v2Enabled.Status != model.InterfaceDefinitionStatusEnabled {
		t.Fatalf("enable v2 = %+v err=%v", v2Enabled, err)
	}

	var count int64
	if err := db.Model(&model.InterfaceDefinition{}).Where("external_system_id = ? AND interface_code = ?", system.Id, "order_query").Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("version count=%d err=%v", count, err)
	}
	if len(writer.records) != 6 {
		t.Fatalf("audit count=%d want 6", len(writer.records))
	}
}

func TestInterfaceDefinitionServiceValidationReferencesAndDTOWhitelist(t *testing.T) {
	svc, _, system := newInterfaceDefinitionTestSubject(t, &externalSystemAuditWriter{})
	ctx := context.Background()
	invalidPath := interfaceDefinitionCreateRequest(system.Id, "unsafe_path")
	invalidPath.RelativePath = "https://outside.example.com/orders"
	if _, err := svc.Create(ctx, invalidPath); !errors.Is(err, apperrors.ErrInterfacePathInvalid) {
		t.Fatalf("full URL error = %v", err)
	}
	invalidPath.RelativePath = "/api/%2e%2e/private"
	if _, err := svc.Create(ctx, invalidPath); !errors.Is(err, apperrors.ErrInterfacePathInvalid) {
		t.Fatalf("encoded traversal error = %v", err)
	}
	credentialID := 99
	invalidReference := interfaceDefinitionCreateRequest(system.Id, "credential_query")
	invalidReference.CredentialID = &credentialID
	if _, err := svc.Create(ctx, invalidReference); !errors.Is(err, apperrors.ErrInterfaceCredentialInvalid) {
		t.Fatalf("credential reference error = %v", err)
	}

	created, err := svc.Create(ctx, interfaceDefinitionCreateRequest(system.Id, "stock_query"))
	if err != nil {
		t.Fatalf("create valid interface: %v", err)
	}
	if _, err := svc.Create(ctx, interfaceDefinitionCreateRequest(system.Id, "stock_query")); !errors.Is(err, apperrors.ErrInterfaceCodeDuplicate) {
		t.Fatalf("duplicate interface error = %v", err)
	}
	detail, err := svc.Get(ctx, created.Id)
	if err != nil {
		t.Fatalf("get detail: %v", err)
	}
	payload, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("marshal detail: %v", err)
	}
	for _, forbidden := range []string{"gmt_delete", "delete_user", "create_user", "state\"", "token", "password"} {
		if strings.Contains(strings.ToLower(string(payload)), forbidden) {
			t.Fatalf("detail DTO leaked %q: %s", forbidden, payload)
		}
	}
}

func TestInterfaceDefinitionServicePageIncludesSystemSummary(t *testing.T) {
	svc, _, system := newInterfaceDefinitionTestSubject(t, &externalSystemAuditWriter{})
	if _, err := svc.Create(context.Background(), interfaceDefinitionCreateRequest(system.Id, "shipment_query")); err != nil {
		t.Fatalf("create fixture: %v", err)
	}
	page, err := svc.Page(context.Background(), request.InterfaceDefinitionQueryReq{
		Page: 1, Num: 10, QuickQuery: &request.QuickQuery{Keyword: "shipment"}, ExternalSystemID: system.Id,
	}, interfaceDefinitionQueryTableForTest())
	if err != nil || page.Total != 1 || len(page.Data) != 1 {
		t.Fatalf("page = %+v err=%v", page, err)
	}
	if page.Data[0].ExternalSystem.SystemCode != system.SystemCode || page.Data[0].PathSummary != "/api/orders" {
		t.Fatalf("unexpected list DTO: %+v", page.Data[0])
	}
}

func newInterfaceDefinitionTestSubject(t *testing.T, writer StandardContextAuditWriter) (*InterfaceDefinitionService, *gorm.DB, model.ExternalSystem) {
	t.Helper()
	db := testutil.OpenSQLite(t, &model.ExternalSystem{}, &model.InterfaceDefinition{})
	system := model.ExternalSystem{
		Basic: model.Basic{Id: 100, State: true}, SystemCode: "demo_erp", Name: "Demo ERP",
		SystemType: model.ExternalSystemTypeERP, BaseURL: "https://api.example.com",
		OwnerIdentifier: "owner-1", OwnerName: "实施负责人", Status: model.ExternalSystemStatusEnabled, Revision: 1,
	}
	if err := db.Create(&system).Error; err != nil {
		t.Fatalf("create external system fixture: %v", err)
	}
	sf, err := utils.NewSnowflake(1)
	if err != nil {
		t.Fatalf("create snowflake: %v", err)
	}
	primary := &database.PrimaryDB{DB: db}
	return NewInterfaceDefinitionService(
		impl.NewInterfaceDefinitionRepositoryImpl(primary), impl.NewExternalSystemRepositoryImpl(primary), sf, writer,
	), db, system
}

func interfaceDefinitionCreateRequest(systemID int, code string) request.InterfaceDefinitionCreateReq {
	return request.InterfaceDefinitionCreateReq{
		ExternalSystemID: systemID, InterfaceCode: code, Name: "订单查询", Protocol: model.InterfaceProtocolHTTPS,
		HTTPMethod: model.InterfaceMethodGET, RelativePath: "/api/orders", TimeoutSeconds: 30,
		ResponseLimit: 10 * 1024 * 1024, Description: "集成接口定义测试",
	}
}

func interfaceDefinitionQueryTableForTest() model.SysTable {
	return model.SysTable{TableCode: "integration_interface_definition", TableFields: []model.SysTableField{
		{Basic: model.Basic{State: true}, FieldCode: "interface_code", FieldType: enum.VarcharFieldType, IsQuickSearch: true},
		{Basic: model.Basic{State: true}, FieldCode: "name", FieldType: enum.VarcharFieldType, IsQuickSearch: true},
		{Basic: model.Basic{State: true}, FieldCode: "external_system_id", FieldType: enum.BigIntFieldType},
		{Basic: model.Basic{State: true}, FieldCode: "http_method", FieldType: enum.VarcharFieldType},
		{Basic: model.Basic{State: true}, FieldCode: "status", FieldType: enum.VarcharFieldType},
	}}
}
