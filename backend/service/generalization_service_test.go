package service

import (
	"backend/dto/request"
	"backend/enum"
	"backend/internal/audit"
	error2 "backend/internal/errors"
	"backend/model"
	"backend/repository"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type generalizationRepoSpy struct {
	called       bool
	created      map[string]interface{}
	updated      map[string]interface{}
	rowExists    bool
	rowExistsErr error
}

func (r *generalizationRepoSpy) Query(*request.Basic, model.SysTable) (repository.GeneralizationListResult, error) {
	return repository.GeneralizationListResult{}, nil
}

func (r *generalizationRepoSpy) GetById(model.SysTable, int) (map[string]interface{}, error) {
	return nil, nil
}

func (r *generalizationRepoSpy) Create(_ model.SysTable, data map[string]interface{}) error {
	r.called = true
	r.created = copyMap(data)
	return nil
}

func (r *generalizationRepoSpy) RowExists(model.SysTable, int) (bool, error) {
	if r.rowExistsErr != nil {
		return false, r.rowExistsErr
	}
	return r.rowExists, nil
}

func (r *generalizationRepoSpy) Update(_ model.SysTable, _ int, data map[string]interface{}) error {
	r.called = true
	r.updated = copyMap(data)
	return nil
}

func (r *generalizationRepoSpy) SoftDelete(model.SysTable, int, map[string]interface{}) error {
	r.called = true
	return nil
}

func (r *generalizationRepoSpy) HardDelete(model.SysTable, int) error {
	r.called = true
	return nil
}

func copyMap(data map[string]interface{}) map[string]interface{} {
	copied := make(map[string]interface{}, len(data))
	for key, value := range data {
		copied[key] = value
	}
	return copied
}

func testUserContext() context.Context {
	return audit.WithAuditSubject(context.Background(), audit.NewAuditSubject(1, "generalization-test"))
}

func TestGeneralizationServiceRejectsProtectedTableWrites(t *testing.T) {
	protected := model.SysTable{TableCode: "sys_table"}
	tests := []struct {
		name string
		run  func(*GeneralizationService) error
	}{
		{
			name: "create",
			run: func(service *GeneralizationService) error {
				return service.Create(nil, protected, map[string]interface{}{"table_name": "bad"})
			},
		},
		{
			name: "update",
			run: func(service *GeneralizationService) error {
				return service.Update(nil, protected, 1, map[string]interface{}{"table_name": "bad"})
			},
		},
		{
			name: "delete",
			run: func(service *GeneralizationService) error {
				return service.Delete(nil, protected, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &generalizationRepoSpy{}
			service := NewGeneralizationService(repo, nil)

			err := tt.run(service)
			if err == nil {
				t.Fatal("expected protected table write to fail")
			}
			var adminErr *error2.ApplicationError
			if !errors.As(err, &adminErr) || adminErr.Kind != error2.KindInvalidArgument {
				t.Fatalf("expected bad request AdminError, got %T %v", err, err)
			}
			if !strings.Contains(adminErr.SafeMessage, "受保护的系统表") {
				t.Fatalf("expected protected table error message, got %q", adminErr.SafeMessage)
			}
			if repo.called {
				t.Fatal("protected table write reached repository")
			}
		})
	}
}

func TestGeneralizationServiceUpdateRejectsMissingRow(t *testing.T) {
	repo := &generalizationRepoSpy{rowExists: false}
	service := NewGeneralizationService(repo, nil)

	err := service.Update(nil, validationTypeTable(), 404, map[string]interface{}{"count": 1})
	if err != error2.ErrDataNotFound {
		t.Fatalf("expected data not found, got %v", err)
	}
	if repo.called {
		t.Fatal("missing row update reached repository update")
	}
}

func TestGeneralizationServiceDeleteRejectsMissingRow(t *testing.T) {
	repo := &generalizationRepoSpy{rowExists: false}
	service := NewGeneralizationService(repo, nil)

	err := service.Delete(nil, validationTypeTable(), 404)
	if err != error2.ErrDataNotFound {
		t.Fatalf("expected data not found, got %v", err)
	}
	if repo.called {
		t.Fatal("missing row delete reached repository delete")
	}
}

func TestIsProtectedTableCoversSystemAndCoreTables(t *testing.T) {
	protected := []string{"sys_user", "SYS_TABLE", "casbin_rule", "access_log"}
	for _, code := range protected {
		if !isProtectedTable(code) {
			t.Fatalf("expected %s to be protected", code)
		}
	}
	if isProtectedTable("smk_customer_order") {
		t.Fatal("expected custom low-code table to be writable")
	}
}

func TestValidateDataByBindingsRejectsInvalidFieldTypes(t *testing.T) {
	table := validationTypeTable()
	tests := []struct {
		name      string
		data      map[string]interface{}
		wantError string
	}{
		{
			name:      "integer",
			data:      map[string]interface{}{"count": "abc"},
			wantError: "数量必须是整数",
		},
		{
			name:      "integer fraction",
			data:      map[string]interface{}{"count": 1.5},
			wantError: "数量必须是整数",
		},
		{
			name:      "float",
			data:      map[string]interface{}{"price": "abc"},
			wantError: "价格必须是数字",
		},
		{
			name:      "bool",
			data:      map[string]interface{}{"enabled": "maybe"},
			wantError: "启用必须是布尔值",
		},
		{
			name:      "date",
			data:      map[string]interface{}{"biz_date": "2026/06/06"},
			wantError: "业务日期必须是日期",
		},
		{
			name:      "datetime",
			data:      map[string]interface{}{"started_at": "bad-date"},
			wantError: "开始时间必须是日期时间",
		},
		{
			name:      "time",
			data:      map[string]interface{}{"start_time": "25:00"},
			wantError: "开始时刻必须是时间",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDataByBindings(table, tt.data, true)
			if err == nil {
				t.Fatal("expected type validation error")
			}
			var adminErr *error2.ApplicationError
			if !errors.As(err, &adminErr) || adminErr.Kind != error2.KindInvalidArgument {
				t.Fatalf("expected bad request AdminError, got %T %v", err, err)
			}
			if !strings.Contains(adminErr.SafeMessage, tt.wantError) {
				t.Fatalf("expected %q in error message, got %q", tt.wantError, adminErr.SafeMessage)
			}
		})
	}
}

func TestValidateDataByBindingsAllowsTypedStringValues(t *testing.T) {
	data := map[string]interface{}{
		"count":      "12",
		"price":      "12.5",
		"enabled":    "true",
		"biz_date":   "2026-06-06",
		"started_at": "2026-06-06T12:30",
		"start_time": "12:30",
	}
	if err := validateDataByBindings(validationTypeTable(), data, true); err != nil {
		t.Fatalf("expected typed string values to pass, got %v", err)
	}
}

func TestValidateDataByBindingsRejectsEmptyNonNullableUpdates(t *testing.T) {
	table := model.SysTable{
		TableCode: "smk_validation",
		TableFields: []model.SysTableField{
			{FieldName: "数量", FieldCode: "count", FieldType: enum.IntFieldType, IsNull: false, IsInsertShow: true, IsUpdateShow: true},
		},
	}

	err := validateDataByBindings(table, map[string]interface{}{"count": ""}, false)
	if err == nil {
		t.Fatal("expected empty non-null update to fail")
	}
	var adminErr *error2.ApplicationError
	if !errors.As(err, &adminErr) || adminErr.Kind != error2.KindInvalidArgument {
		t.Fatalf("expected bad request AdminError, got %T %v", err, err)
	}
	if !strings.Contains(adminErr.SafeMessage, "数量不能为空") {
		t.Fatalf("expected required error, got %q", adminErr.SafeMessage)
	}
}

func TestNormalizeDataByFieldTypesConvertsEmptyNullableTypedValuesToNil(t *testing.T) {
	table := validationTypeTable()
	table.TableFields = append(table.TableFields, model.SysTableField{
		FieldName: "名称", FieldCode: "name", FieldType: enum.VarcharFieldType, IsNull: true, IsInsertShow: true, IsUpdateShow: true,
	})
	data := map[string]interface{}{
		"count":    "",
		"biz_date": "",
		"name":     "",
	}

	normalizeDataByFieldTypes(table, data)

	if data["count"] != nil {
		t.Fatalf("expected empty nullable int to become nil, got %#v", data["count"])
	}
	if data["biz_date"] != nil {
		t.Fatalf("expected empty nullable date to become nil, got %#v", data["biz_date"])
	}
	if data["name"] != "" {
		t.Fatalf("expected empty nullable varchar to stay empty string, got %#v", data["name"])
	}
}

func TestGeneralizationServiceNormalizesTypedStringValuesBeforeCreate(t *testing.T) {
	repo := &generalizationRepoSpy{}
	service := NewGeneralizationService(repo, nil)
	data := map[string]interface{}{
		"count":      "12",
		"price":      "12.5",
		"enabled":    "false",
		"biz_date":   "2026-06-06",
		"started_at": "2026-06-06T12:30",
		"start_time": "12:30",
	}

	if err := service.Create(testUserContext(), validationTypeTable(), data); err != nil {
		t.Fatalf("expected create to pass, got %v", err)
	}

	if _, ok := repo.created["count"].(int); !ok {
		t.Fatalf("expected count to be int, got %T", repo.created["count"])
	}
	if _, ok := repo.created["price"].(float64); !ok {
		t.Fatalf("expected price to be float64, got %T", repo.created["price"])
	}
	if _, ok := repo.created["enabled"].(bool); !ok {
		t.Fatalf("expected enabled to be bool, got %T", repo.created["enabled"])
	}
	for _, field := range []string{"biz_date", "started_at"} {
		if _, ok := repo.created[field].(time.Time); !ok {
			t.Fatalf("expected %s to be time.Time, got %T", field, repo.created[field])
		}
	}
	if repo.created["start_time"] != "12:30:00" {
		t.Fatalf("expected start_time to be normalized HH:mm:ss string, got %#v", repo.created["start_time"])
	}
}

func validationTypeTable() model.SysTable {
	return model.SysTable{
		TableCode: "smk_validation",
		TableFields: []model.SysTableField{
			{FieldName: "数量", FieldCode: "count", FieldType: enum.IntFieldType, IsNull: true, IsInsertShow: true, IsUpdateShow: true},
			{FieldName: "价格", FieldCode: "price", FieldType: enum.FloatFieldType, IsNull: true, IsInsertShow: true, IsUpdateShow: true},
			{FieldName: "启用", FieldCode: "enabled", FieldType: enum.BooleanFieldType, IsNull: true, IsInsertShow: true, IsUpdateShow: true},
			{FieldName: "业务日期", FieldCode: "biz_date", FieldType: enum.DateFieldType, IsNull: true, IsInsertShow: true, IsUpdateShow: true},
			{FieldName: "开始时间", FieldCode: "started_at", FieldType: enum.DatetimeFieldType, IsNull: true, IsInsertShow: true, IsUpdateShow: true},
			{FieldName: "开始时刻", FieldCode: "start_time", FieldType: enum.TimeFieldType, IsNull: true, IsInsertShow: true, IsUpdateShow: true},
		},
	}
}
