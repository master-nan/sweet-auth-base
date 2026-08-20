package service

import (
	"backend/dto/request"
	apperrors "backend/internal/errors"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"backend/enum"
	"backend/internal/database"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestDataResourceConfigServiceCreateAndQuery(t *testing.T) {
	t.Run("create defaults permission off and creates operations atomically", func(t *testing.T) {
		auditWriter := &testTransactionalAuditWriter{}
		service, db := newDataResourceConfigTestSubject(t, auditWriter)
		result, err := service.CreateResource(
			dataResourceConfigContext(),
			dataResourceCreateRequest(
				"service:tms.dispatch_task",
				model.DataPermissionOperationQuery,
				model.DataPermissionOperationDetail,
			),
		)
		if err != nil {
			t.Fatalf("create resource: %v", err)
		}
		if result.ResourceCode != "service:tms.dispatch_task" || result.PermissionEnabled {
			t.Fatalf("unexpected resource response: %+v", result)
		}

		var stored model.DataResource
		if err = db.Where("resource_code = ?", result.ResourceCode).First(&stored).Error; err != nil {
			t.Fatalf("reload resource: %v", err)
		}
		if stored.PermissionEnabled || stored.ServiceCode == nil || *stored.ServiceCode != "tms.dispatch_task" {
			t.Fatalf("unexpected stored resource: %+v", stored)
		}
		var operations []model.DataResourceOperation
		if err = db.Where("resource_id = ?", stored.Id).Order("operation").Find(&operations).Error; err != nil {
			t.Fatalf("reload operations: %v", err)
		}
		if len(operations) != 2 || operations[0].PermissionEnabled || operations[1].PermissionEnabled {
			t.Fatalf("unexpected operations: %+v", operations)
		}
		if records := auditWriter.snapshot(); len(records) != 1 ||
			records[0].Action != dataResourceCreateAction {
			t.Fatalf("unexpected audit records: %+v", records)
		}
	})

	t.Run("detail and page use response whitelist", func(t *testing.T) {
		service, db := newDataResourceConfigTestSubject(t, &testTransactionalAuditWriter{})
		resource := dataResourceFixture(101, "service:wms.stock_query")
		resource.Description = "internal configuration note"
		testutil.MustCreate(t, db, &resource)

		detail, err := service.GetResource(dataResourceConfigContext(), resource.Id)
		if err != nil {
			t.Fatalf("get resource: %v", err)
		}
		payload, err := json.Marshal(detail)
		if err != nil {
			t.Fatalf("marshal detail: %v", err)
		}
		for _, forbidden := range []string{"internal configuration note", "gmt_delete", "table_id", "service_code"} {
			if strings.Contains(string(payload), forbidden) {
				t.Fatalf("detail leaked %q: %s", forbidden, payload)
			}
		}

		page, err := service.PageResources(
			dataResourceConfigContext(),
			request.DataResourceQueryReq{
				DataPermissionConfigQueryReq: request.DataPermissionConfigQueryReq{
					Page:       1,
					Num:        10,
					QuickQuery: &request.QuickQuery{Keyword: "stock"},
				},
				ResourceType: model.DataResourceTypeBusinessService,
			},
		)
		if err != nil {
			t.Fatalf("page resources: %v", err)
		}
		if page.Total != 1 || len(page.Data) != 1 || page.Data[0].ResourceCode != resource.ResourceCode {
			t.Fatalf("unexpected resource page: %+v", page)
		}
	})
}

func TestDataResourceConfigServiceCreateValidation(t *testing.T) {
	t.Run("permission enable is rejected and not persisted", func(t *testing.T) {
		service, db := newDataResourceConfigTestSubject(t, &testTransactionalAuditWriter{})
		req := dataResourceCreateRequest("service:tms.permission_attempt")
		enabled := true
		req.PermissionEnabled = &enabled
		_, err := service.CreateResource(dataResourceConfigContext(), req)
		assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataResourcePermissionEnableDenied)
		assertDataResourceCount(t, db, 0)
	})

	t.Run("invalid resource fields are rejected", func(t *testing.T) {
		tests := []struct {
			name string
			edit func(*request.DataResourceCreateReq)
			code int
		}{
			{
				name: "path code",
				edit: func(req *request.DataResourceCreateReq) {
					req.ResourceCode = "/admin/order"
				},
				code: apperrors.ErrorCodeDataResourceCodeInvalid,
			},
			{
				name: "chinese code",
				edit: func(req *request.DataResourceCreateReq) {
					req.ResourceCode = "订单"
				},
				code: apperrors.ErrorCodeDataResourceCodeInvalid,
			},
			{
				name: "empty name",
				edit: func(req *request.DataResourceCreateReq) {
					req.Name = " "
				},
				code: apperrors.ErrorCodeDataResourceNameRequired,
			},
			{
				name: "invalid type",
				edit: func(req *request.DataResourceCreateReq) {
					req.ResourceType = "menu"
				},
				code: apperrors.ErrorCodeDataResourceTypeInvalid,
			},
			{
				name: "invalid target",
				edit: func(req *request.DataResourceCreateReq) {
					req.Target.ReferenceCode = nil
				},
				code: apperrors.ErrorCodeDataResourceTargetInvalid,
			},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				service, db := newDataResourceConfigTestSubject(t, &testTransactionalAuditWriter{})
				req := dataResourceCreateRequest("service:tms.validation")
				tc.edit(&req)
				_, err := service.CreateResource(dataResourceConfigContext(), req)
				assertDataResourceConfigError(t, err, tc.code)
				assertDataResourceCount(t, db, 0)
			})
		}
	})

	t.Run("duplicate code is a stable business error", func(t *testing.T) {
		service, db := newDataResourceConfigTestSubject(t, &testTransactionalAuditWriter{})
		resource := dataResourceFixture(102, "service:tms.duplicate")
		testutil.MustCreate(t, db, &resource)
		_, err := service.CreateResource(
			dataResourceConfigContext(),
			dataResourceCreateRequest(resource.ResourceCode),
		)
		assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataResourceCodeDuplicate)
		assertDataResourceCount(t, db, 1)
	})

	t.Run("audit failure rolls back resource and operations", func(t *testing.T) {
		service, db := newDataResourceConfigTestSubject(
			t,
			&testTransactionalAuditWriter{err: errors.New("audit unavailable")},
		)
		_, err := service.CreateResource(
			dataResourceConfigContext(),
			dataResourceCreateRequest(
				"service:tms.audit_rollback",
				model.DataPermissionOperationQuery,
			),
		)
		if err == nil {
			t.Fatal("expected audit failure")
		}
		assertDataResourceCount(t, db, 0)
		var operationCount int64
		if countErr := db.Model(&model.DataResourceOperation{}).Count(&operationCount).Error; countErr != nil {
			t.Fatalf("count rolled back operations: %v", countErr)
		}
		if operationCount != 0 {
			t.Fatalf("operation count = %d, want 0", operationCount)
		}
	})
}

func TestDataResourceConfigServiceUpdateDisableAndRemove(t *testing.T) {
	t.Run("updates mutable fields and protects permission switch", func(t *testing.T) {
		service, db := newDataResourceConfigTestSubject(t, &testTransactionalAuditWriter{})
		resource := dataResourceFixture(201, "service:tms.mutable")
		resource.Description = "old"
		testutil.MustCreate(t, db, &resource)
		name := "运输任务"
		description := "new"
		result, err := service.UpdateResource(dataResourceConfigContext(), request.DataResourceUpdateReq{
			Id:          resource.Id,
			Name:        &name,
			Description: &description,
		})
		if err != nil {
			t.Fatalf("update mutable fields: %v", err)
		}
		if result.Name != name {
			t.Fatalf("updated name = %q, want %q", result.Name, name)
		}
		var stored model.DataResource
		if err = db.First(&stored, resource.Id).Error; err != nil {
			t.Fatalf("reload updated resource: %v", err)
		}
		if stored.Description != description {
			t.Fatalf("stored description = %q, want %q", stored.Description, description)
		}

		enabled := true
		_, err = service.UpdateResource(dataResourceConfigContext(), request.DataResourceUpdateReq{
			Id:                resource.Id,
			PermissionEnabled: &enabled,
		})
		assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataResourcePermissionEnableDenied)
	})

	t.Run("resource code is immutable", func(t *testing.T) {
		service, db := newDataResourceConfigTestSubject(t, &testTransactionalAuditWriter{})
		resource := dataResourceFixture(202, "service:tms.identity")
		testutil.MustCreate(t, db, &resource)
		changed := "service:tms.changed"
		_, err := service.UpdateResource(dataResourceConfigContext(), request.DataResourceUpdateReq{
			Id:           resource.Id,
			ResourceCode: &changed,
		})
		assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataResourceFieldImmutable)
	})

	t.Run("semantic type may change only before references exist", func(t *testing.T) {
		service, db := newDataResourceConfigTestSubject(t, &testTransactionalAuditWriter{})
		resource := dataResourceFixture(203, "service:tms.unused")
		testutil.MustCreate(t, db, &resource)
		resourceType := model.DataResourceTypeReport
		adapter := "report_filter"
		reportId := 77
		result, err := service.UpdateResource(dataResourceConfigContext(), request.DataResourceUpdateReq{
			Id:           resource.Id,
			ResourceType: &resourceType,
			Target:       &request.DataResourceTargetReq{ReferenceId: &reportId},
			AdapterCode:  &adapter,
		})
		if err != nil {
			t.Fatalf("change unused resource semantics: %v", err)
		}
		if result.ResourceType != model.DataResourceTypeReport ||
			result.Target.ReferenceId == nil ||
			*result.Target.ReferenceId != reportId {
			t.Fatalf("unexpected changed resource: %+v", result)
		}

		operation := dataResourceOperationFixture(204, resource.Id, model.DataPermissionOperationRun)
		testutil.MustCreate(t, db, &operation)
		resourceType = model.DataResourceTypeBusinessService
		serviceCode := "tms.changed"
		_, err = service.UpdateResource(dataResourceConfigContext(), request.DataResourceUpdateReq{
			Id:           resource.Id,
			ResourceType: &resourceType,
			Target:       &request.DataResourceTargetReq{ReferenceCode: &serviceCode},
		})
		assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataResourceFieldImmutable)
	})

	t.Run("disable cascades state and permission switches without deletion", func(t *testing.T) {
		service, db := newDataResourceConfigTestSubject(t, &testTransactionalAuditWriter{})
		resource := dataResourceFixture(205, "service:tms.disable")
		resource.PermissionEnabled = true
		operation := dataResourceOperationFixture(206, resource.Id, model.DataPermissionOperationQuery)
		operation.PermissionEnabled = true
		testutil.MustCreate(t, db, &resource)
		testutil.MustCreate(t, db, &operation)

		if err := service.DisableResource(dataResourceConfigContext(), resource.Id); err != nil {
			t.Fatalf("disable resource: %v", err)
		}
		var storedResource model.DataResource
		if err := db.First(&storedResource, resource.Id).Error; err != nil {
			t.Fatalf("reload disabled resource: %v", err)
		}
		var storedOperation model.DataResourceOperation
		if err := db.First(&storedOperation, operation.Id).Error; err != nil {
			t.Fatalf("reload disabled operation: %v", err)
		}
		if storedResource.State || storedResource.PermissionEnabled ||
			storedOperation.State || storedOperation.PermissionEnabled {
			t.Fatalf("disable did not cascade: resource=%+v operation=%+v", storedResource, storedOperation)
		}
	})

	t.Run("referenced resource cannot be removed and unreferenced removal is soft", func(t *testing.T) {
		service, db := newDataResourceConfigTestSubject(t, &testTransactionalAuditWriter{})
		referenced := dataResourceFixture(207, "service:tms.referenced")
		operation := dataResourceOperationFixture(208, referenced.Id, model.DataPermissionOperationQuery)
		unreferenced := dataResourceFixture(209, "service:tms.removable")
		testutil.MustCreate(t, db, &referenced)
		testutil.MustCreate(t, db, &operation)
		testutil.MustCreate(t, db, &unreferenced)

		err := service.RemoveResource(dataResourceConfigContext(), referenced.Id)
		assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataResourceReferenced)
		if err = service.RemoveResource(dataResourceConfigContext(), unreferenced.Id); err != nil {
			t.Fatalf("soft remove unreferenced resource: %v", err)
		}
		var activeCount int64
		if err = db.Model(&model.DataResource{}).Where("id = ?", unreferenced.Id).Count(&activeCount).Error; err != nil {
			t.Fatalf("count active removed resource: %v", err)
		}
		var allCount int64
		if err = db.Unscoped().Model(&model.DataResource{}).Where("id = ?", unreferenced.Id).Count(&allCount).Error; err != nil {
			t.Fatalf("count all removed resource: %v", err)
		}
		if activeCount != 0 || allCount != 1 {
			t.Fatalf("soft delete counts active=%d all=%d", activeCount, allCount)
		}
	})
}

func TestDataResourceConfigServiceOperations(t *testing.T) {
	t.Run("add list and duplicate protection", func(t *testing.T) {
		service, db := newDataResourceConfigTestSubject(t, &testTransactionalAuditWriter{})
		resource := dataResourceFixture(301, "service:tms.operations")
		testutil.MustCreate(t, db, &resource)
		items := []request.DataResourceOperationCreateItemReq{
			{Operation: model.DataPermissionOperationQuery},
			{Operation: model.DataPermissionOperationDetail},
		}
		rows, err := service.AddResourceOperations(dataResourceConfigContext(), request.DataResourceOperationBatchReq{
			ResourceId: resource.Id,
			Items:      items,
		})
		if err != nil {
			t.Fatalf("add resource operations: %v", err)
		}
		if len(rows) != 2 {
			t.Fatalf("operation rows = %d, want 2", len(rows))
		}
		_, err = service.AddResourceOperations(dataResourceConfigContext(), request.DataResourceOperationBatchReq{
			ResourceId: resource.Id,
			Items:      []request.DataResourceOperationCreateItemReq{{Operation: model.DataPermissionOperationQuery}},
		})
		assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataResourceOperationDuplicate)
	})

	t.Run("invalid batch and missing or disabled resource leave no partial rows", func(t *testing.T) {
		service, db := newDataResourceConfigTestSubject(t, &testTransactionalAuditWriter{})
		resource := dataResourceFixture(302, "service:tms.batch")
		disabled := dataResourceFixture(303, "service:tms.disabled")
		testutil.MustCreate(t, db, &resource)
		testutil.MustCreate(t, db, &disabled)
		if err := db.Model(&model.DataResource{}).
			Where("id = ?", disabled.Id).
			Update("state", false).Error; err != nil {
			t.Fatalf("disable resource fixture: %v", err)
		}

		_, err := service.AddResourceOperations(dataResourceConfigContext(), request.DataResourceOperationBatchReq{
			ResourceId: resource.Id,
			Items: []request.DataResourceOperationCreateItemReq{
				{Operation: model.DataPermissionOperationQuery},
				{Operation: "free_sql"},
			},
		})
		assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataResourceOperationInvalid)
		assertOperationCount(t, db, resource.Id, 0)

		enabled := true
		_, err = service.AddResourceOperations(dataResourceConfigContext(), request.DataResourceOperationBatchReq{
			ResourceId: resource.Id,
			Items: []request.DataResourceOperationCreateItemReq{
				{
					Operation:         model.DataPermissionOperationQuery,
					PermissionEnabled: &enabled,
				},
			},
		})
		assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataResourcePermissionEnableDenied)
		assertOperationCount(t, db, resource.Id, 0)

		_, err = service.AddResourceOperations(dataResourceConfigContext(), request.DataResourceOperationBatchReq{
			ResourceId: 999,
			Items:      []request.DataResourceOperationCreateItemReq{{Operation: model.DataPermissionOperationQuery}},
		})
		assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataResourceNotFound)

		_, err = service.AddResourceOperations(dataResourceConfigContext(), request.DataResourceOperationBatchReq{
			ResourceId: disabled.Id,
			Items:      []request.DataResourceOperationCreateItemReq{{Operation: model.DataPermissionOperationQuery}},
		})
		assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataResourceStateInvalid)
	})

	t.Run("replace is atomic and disables operations omitted from the set", func(t *testing.T) {
		auditWriter := &testTransactionalAuditWriter{}
		service, db := newDataResourceConfigTestSubject(t, auditWriter)
		resource := dataResourceFixture(304, "service:tms.replace")
		query := dataResourceOperationFixture(305, resource.Id, model.DataPermissionOperationQuery)
		detail := dataResourceOperationFixture(306, resource.Id, model.DataPermissionOperationDetail)
		testutil.MustCreate(t, db, &resource)
		testutil.MustCreate(t, db, &query)
		testutil.MustCreate(t, db, &detail)

		rows, err := service.ReplaceResourceOperations(
			dataResourceConfigContext(),
			request.DataResourceOperationBatchReq{
				ResourceId: resource.Id,
				Items: []request.DataResourceOperationCreateItemReq{
					{Operation: model.DataPermissionOperationQuery},
					{Operation: model.DataPermissionOperationExport},
				},
			},
		)
		if err != nil {
			t.Fatalf("replace resource operations: %v", err)
		}
		states := make(map[string]bool, len(rows))
		for _, row := range rows {
			states[row.Operation] = row.State
		}
		if !states[model.DataPermissionOperationQuery] ||
			!states[model.DataPermissionOperationExport] ||
			states[model.DataPermissionOperationDetail] {
			t.Fatalf("unexpected operation states: %+v", states)
		}

		auditWriter.err = errors.New("audit unavailable")
		_, err = service.ReplaceResourceOperations(
			dataResourceConfigContext(),
			request.DataResourceOperationBatchReq{
				ResourceId: resource.Id,
				Items: []request.DataResourceOperationCreateItemReq{
					{Operation: model.DataPermissionOperationRun},
				},
			},
		)
		if err == nil {
			t.Fatal("expected audit rollback")
		}
		var runCount int64
		if err = db.Model(&model.DataResourceOperation{}).
			Where("resource_id = ? AND operation = ?", resource.Id, model.DataPermissionOperationRun).
			Count(&runCount).Error; err != nil {
			t.Fatalf("count rolled back run operation: %v", err)
		}
		if runCount != 0 {
			t.Fatalf("run operation count = %d, want 0 after rollback", runCount)
		}
	})

	t.Run("referenced operation may be disabled but not removed", func(t *testing.T) {
		service, db := newDataResourceConfigTestSubject(t, &testTransactionalAuditWriter{})
		resource := dataResourceFixture(307, "service:tms.granted")
		operation := dataResourceOperationFixture(308, resource.Id, model.DataPermissionOperationQuery)
		grant := model.DataGrant{
			Basic:       model.Basic{Id: 309, State: true},
			SubjectType: model.DataGrantSubjectTypeRole,
			SubjectId:   1,
			ResourceId:  resource.Id,
			Operation:   operation.Operation,
			PolicyId:    99,
		}
		testutil.MustCreate(t, db, &resource)
		testutil.MustCreate(t, db, &operation)
		testutil.MustCreate(t, db, &grant)

		err := service.RemoveResourceOperation(dataResourceConfigContext(), operation.Id)
		assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataResourceOperationReferenced)
		if err = service.DisableResourceOperation(dataResourceConfigContext(), operation.Id); err != nil {
			t.Fatalf("disable referenced operation: %v", err)
		}
		var stored model.DataResourceOperation
		if err = db.First(&stored, operation.Id).Error; err != nil {
			t.Fatalf("reload disabled operation: %v", err)
		}
		if stored.State {
			t.Fatal("referenced operation should be disabled")
		}
	})

	t.Run("unreferenced operation removal uses soft delete", func(t *testing.T) {
		service, db := newDataResourceConfigTestSubject(t, &testTransactionalAuditWriter{})
		resource := dataResourceFixture(310, "service:tms.unused_operation")
		operation := dataResourceOperationFixture(311, resource.Id, model.DataPermissionOperationQuery)
		testutil.MustCreate(t, db, &resource)
		testutil.MustCreate(t, db, &operation)

		if err := service.RemoveResourceOperation(dataResourceConfigContext(), operation.Id); err != nil {
			t.Fatalf("remove unreferenced operation: %v", err)
		}
		var activeCount int64
		if err := db.Model(&model.DataResourceOperation{}).Where("id = ?", operation.Id).Count(&activeCount).Error; err != nil {
			t.Fatalf("count active operation: %v", err)
		}
		var allCount int64
		if err := db.Unscoped().Model(&model.DataResourceOperation{}).Where("id = ?", operation.Id).Count(&allCount).Error; err != nil {
			t.Fatalf("count all operations: %v", err)
		}
		if activeCount != 0 || allCount != 1 {
			t.Fatalf("soft delete counts active=%d all=%d", activeCount, allCount)
		}
	})
}

func TestDataResourceConfigServiceBoundariesAndErrorConversion(t *testing.T) {
	serviceType := reflect.TypeOf(DataResourceConfigService{})
	allowedDependencies := map[string]struct{}{
		"resourceRepo":  {},
		"operationRepo": {},
		"ownershipRepo": {},
		"grantRepo":     {},
		"sf":            {},
		"auditWriter":   {},
	}
	for i := 0; i < serviceType.NumField(); i++ {
		field := serviceType.Field(i)
		if _, ok := allowedDependencies[field.Name]; !ok {
			t.Fatalf("unexpected service dependency %q", field.Name)
		}
		for _, forbidden := range []string{"menu", "casbin", "resolver", "provider"} {
			if strings.Contains(strings.ToLower(field.Type.String()), forbidden) {
				t.Fatalf("service dependency %q crosses boundary: %s", field.Name, field.Type)
			}
		}
	}

	service, db := newDataResourceConfigTestSubject(t, &testTransactionalAuditWriter{})
	if err := db.Migrator().DropTable(&model.DataResource{}); err != nil {
		t.Fatalf("drop resource table: %v", err)
	}
	_, err := service.GetResource(dataResourceConfigContext(), 1)
	clientErr, _ := apperrors.Classify(err)
	if clientErr == nil ||
		clientErr.Category != apperrors.CategoryDatabase ||
		clientErr.SafeMessage != "系统异常" ||
		strings.Contains(clientErr.SafeMessage, "sys_data_resource") {
		t.Fatalf("unexpected converted database error: %+v cause=%v", clientErr, err)
	}
}

func newDataResourceConfigTestSubject(
	t *testing.T,
	auditWriter TransactionalAuditWriter,
) (*DataResourceConfigService, *gorm.DB) {
	t.Helper()
	db := testutil.OpenSQLite(
		t,
		&model.DataResource{},
		&model.DataResourceOperation{},
		&model.DataOwnershipField{},
		&model.DataGrant{},
	)
	primaryDB := &database.PrimaryDB{DB: db}
	sf, err := utils.NewSnowflake(1)
	if err != nil {
		t.Fatalf("create snowflake: %v", err)
	}
	return NewDataResourceConfigService(
		impl.NewDataResourceRepositoryImpl(primaryDB),
		impl.NewDataResourceOperationRepositoryImpl(primaryDB),
		impl.NewDataOwnershipFieldRepositoryImpl(primaryDB),
		impl.NewDataGrantRepositoryImpl(primaryDB),
		sf,
		auditWriter,
	), db
}

func dataResourceCreateRequest(
	resourceCode string,
	operations ...string,
) request.DataResourceCreateReq {
	serviceCode := strings.TrimPrefix(resourceCode, "service:")
	items := make([]request.DataResourceOperationCreateItemReq, 0, len(operations))
	for _, operation := range operations {
		items = append(items, request.DataResourceOperationCreateItemReq{Operation: operation})
	}
	return request.DataResourceCreateReq{
		ResourceCode: resourceCode,
		Name:         "测试资源",
		ResourceType: model.DataResourceTypeBusinessService,
		Target: request.DataResourceTargetReq{
			ReferenceCode: &serviceCode,
		},
		AdapterCode: "registered_filter",
		Operations:  items,
	}
}

func dataResourceFixture(id int, code string) model.DataResource {
	serviceCode := strings.TrimPrefix(code, "service:")
	return model.DataResource{
		Basic:             model.Basic{Id: id, State: true},
		ResourceCode:      code,
		Name:              "测试资源",
		ResourceType:      model.DataResourceTypeBusinessService,
		ServiceCode:       &serviceCode,
		AdapterCode:       "registered_filter",
		PermissionEnabled: false,
	}
}

func dataResourceOperationFixture(id, resourceId int, operation string) model.DataResourceOperation {
	return model.DataResourceOperation{
		Basic:             model.Basic{Id: id, State: true},
		ResourceId:        resourceId,
		Operation:         operation,
		PermissionEnabled: false,
	}
}

func dataResourceConfigContext() *gin.Context {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/admin/data-resource", nil)
	return ctx
}

func dataResourceConfigTable() model.SysTable {
	return model.SysTable{
		TableCode: "sys_data_resource",
		TableFields: []model.SysTableField{
			{
				FieldCode:        "resource_code",
				FieldType:        enum.VarcharFieldType,
				IsListShow:       true,
				IsQuickSearch:    true,
				IsAdvancedSearch: true,
				IsSort:           true,
			},
			{
				FieldCode:        "name",
				FieldType:        enum.VarcharFieldType,
				IsListShow:       true,
				IsQuickSearch:    true,
				IsAdvancedSearch: true,
				IsSort:           true,
			},
			{
				FieldCode:        "resource_type",
				FieldType:        enum.VarcharFieldType,
				IsListShow:       true,
				IsAdvancedSearch: true,
				IsSort:           true,
			},
		},
	}
}

func assertDataResourceConfigError(t *testing.T, err error, expectedCode int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error code %d", expectedCode)
	}
	clientErr, _ := apperrors.Classify(err)
	if clientErr == nil || clientErr.Code != expectedCode {
		t.Fatalf("error = %+v, want code %d, cause=%v", clientErr, expectedCode, err)
	}
}

func assertDataResourceCount(t *testing.T, db *gorm.DB, expected int64) {
	t.Helper()
	var count int64
	if err := db.Model(&model.DataResource{}).Count(&count).Error; err != nil {
		t.Fatalf("count resources: %v", err)
	}
	if count != expected {
		t.Fatalf("resource count = %d, want %d", count, expected)
	}
}

func assertOperationCount(t *testing.T, db *gorm.DB, resourceId int, expected int64) {
	t.Helper()
	var count int64
	if err := db.Model(&model.DataResourceOperation{}).
		Where("resource_id = ?", resourceId).
		Count(&count).Error; err != nil {
		t.Fatalf("count operations: %v", err)
	}
	if count != expected {
		t.Fatalf("operation count = %d, want %d", count, expected)
	}
}
