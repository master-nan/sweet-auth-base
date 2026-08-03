package service

import (
	"context"
	"errors"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"backend/dto/request"
	"backend/enum"
	"backend/internal/datapermission"
	myerrors "backend/internal/errors"
	"backend/model"
	"backend/repository"

	"github.com/gin-gonic/gin"
)

func TestLowCodeDataPermissionRuntimeRoutesLegacyCompatibilityExplicitly(t *testing.T) {
	table := lowCodeRuntimeTestTable()
	tests := []struct {
		name      string
		resources []model.DataResource
	}{
		{name: "resource is not configured"},
		{name: "resource exists but new permission is disabled", resources: []model.DataResource{
			lowCodeRuntimeResource(table.Id, false),
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var legacyCalls atomic.Int32
			var subjectCalls atomic.Int32
			runtime := newLowCodeDataPermissionRuntime(
				func(_ *gin.Context, tableId int) ([]model.DataResource, error) {
					if tableId != table.Id {
						t.Fatalf("table id = %d, want %d", tableId, table.Id)
					}
					return tt.resources, nil
				},
				func(*gin.Context, int) ([]model.DataOwnershipField, error) {
					t.Fatal("legacy route must not load new Ownership configuration")
					return nil, nil
				},
				func(*gin.Context, int) (datapermission.SubjectContext, error) {
					subjectCalls.Add(1)
					return datapermission.SubjectContext{}, nil
				},
				func(*gin.Context, datapermission.ResolverInput) (datapermission.DataScopeResult, error) {
					t.Fatal("legacy route must not call Resolver")
					return datapermission.DataScopeResult{}, nil
				},
				func(context.Context, datapermission.AdapterInput) (datapermission.AdapterExecution, error) {
					t.Fatal("legacy route must not call Metadata Adapter")
					return datapermission.AdapterExecution{}, nil
				},
				func(_ model.SysUser, menuId int, actualTable model.SysTable, action enum.SysMenuButtonEventAction) (*request.DataScope, error) {
					legacyCalls.Add(1)
					if menuId != 21 || actualTable.Id != table.Id || action != enum.ButtonActionQuery {
						t.Fatalf("unexpected legacy request: menu=%d table=%d action=%s", menuId, actualTable.Id, action)
					}
					return &request.DataScope{AllowAll: true}, nil
				},
			)

			resolution, err := runtime.Resolve(
				lowCodeRuntimeGinContext(101),
				table,
				21,
				model.DataPermissionOperationQuery,
				enum.ButtonActionQuery,
			)
			if err != nil {
				t.Fatalf("resolve legacy route: %v", err)
			}
			if resolution.permission.Mode != repository.GeneralizationPermissionLegacy ||
				resolution.permission.LegacyScope == nil || !resolution.permission.LegacyScope.AllowAll ||
				resolution.permission.AdapterExecution != nil {
				t.Fatalf("unexpected legacy resolution: %+v", resolution.permission)
			}
			if legacyCalls.Load() != 1 || subjectCalls.Load() != 0 {
				t.Fatalf("legacy calls=%d subject calls=%d, want 1/0", legacyCalls.Load(), subjectCalls.Load())
			}
		})
	}
}

func TestLowCodeDataPermissionRuntimeUsesTrustedResourceAndBuildsOnce(t *testing.T) {
	table := lowCodeRuntimeTestTable()
	resource := lowCodeRuntimeResource(table.Id, true)
	var subjectCalls atomic.Int32
	var resolverCalls atomic.Int32
	var adapterCalls atomic.Int32
	var legacyCalls atomic.Int32
	runtime := newLowCodeDataPermissionRuntime(
		func(*gin.Context, int) ([]model.DataResource, error) {
			return []model.DataResource{resource}, nil
		},
		func(_ *gin.Context, resourceId int) ([]model.DataOwnershipField, error) {
			if resourceId != resource.Id {
				t.Fatalf("resource id = %d, want %d", resourceId, resource.Id)
			}
			return []model.DataOwnershipField{lowCodeRuntimeOwnership(resource.Id)}, nil
		},
		func(_ *gin.Context, userId int) (datapermission.SubjectContext, error) {
			subjectCalls.Add(1)
			if userId != 101 {
				t.Fatalf("subject user = %d, want trusted user 101", userId)
			}
			return lowCodeRuntimeSubject(t, userId), nil
		},
		func(_ *gin.Context, input datapermission.ResolverInput) (datapermission.DataScopeResult, error) {
			resolverCalls.Add(1)
			if input.ResourceCode() != resource.ResourceCode || input.Operation() != model.DataPermissionOperationQuery {
				t.Fatalf("Resolver identity = %s/%s", input.ResourceCode(), input.Operation())
			}
			return lowCodeRuntimeFilteredResult(t, resource.ResourceCode, input.Operation()), nil
		},
		func(_ context.Context, input datapermission.AdapterInput) (datapermission.AdapterExecution, error) {
			adapterCalls.Add(1)
			return datapermission.BuildAdapterExecution(input)
		},
		func(model.SysUser, int, model.SysTable, enum.SysMenuButtonEventAction) (*request.DataScope, error) {
			legacyCalls.Add(1)
			return &request.DataScope{AllowAll: true}, nil
		},
	)

	resolution, err := runtime.Resolve(
		lowCodeRuntimeGinContext(101),
		table,
		21,
		model.DataPermissionOperationQuery,
		enum.ButtonActionQuery,
	)
	if err != nil {
		t.Fatalf("resolve new runtime: %v", err)
	}
	if resolution.permission.Mode != repository.GeneralizationPermissionAdapter ||
		resolution.permission.AdapterExecution == nil ||
		resolution.permission.AdapterExecution.Mode() != datapermission.AdapterExecutionModeApplyFilter ||
		resolution.permission.LegacyScope != nil {
		t.Fatalf("unexpected adapter resolution: %+v", resolution.permission)
	}
	if _, protected := resolution.ownershipFieldIds[501]; !protected {
		t.Fatal("metadata ownership field was not protected from generic updates")
	}
	if subjectCalls.Load() != 1 || resolverCalls.Load() != 1 || adapterCalls.Load() != 1 || legacyCalls.Load() != 0 {
		t.Fatalf(
			"calls subject=%d resolver=%d adapter=%d legacy=%d, want 1/1/1/0",
			subjectCalls.Load(), resolverCalls.Load(), adapterCalls.Load(), legacyCalls.Load(),
		)
	}
}

func TestLowCodeDataPermissionRuntimeFailsClosed(t *testing.T) {
	table := lowCodeRuntimeTestTable()
	resource := lowCodeRuntimeResource(table.Id, true)
	sentinel := errors.New("dependency failed")

	tests := []struct {
		name          string
		resources     []model.DataResource
		resolver      lowCodeResolver
		adapter       lowCodeMetadataAdapter
		wantErrorCode int
	}{
		{
			name:          "multiple resource routes conflict",
			resources:     []model.DataResource{resource, resource},
			wantErrorCode: myerrors.ErrorCodeDataPermissionRuntimeRouteConflict,
		},
		{
			name:      "Resolver failure is not downgraded to legacy",
			resources: []model.DataResource{resource},
			resolver: func(*gin.Context, datapermission.ResolverInput) (datapermission.DataScopeResult, error) {
				return datapermission.DataScopeResult{}, sentinel
			},
		},
		{
			name:      "not applicable from enabled resource is a route conflict",
			resources: []model.DataResource{resource},
			resolver: func(_ *gin.Context, input datapermission.ResolverInput) (datapermission.DataScopeResult, error) {
				return datapermission.NewNotApplicableResult(input.ResourceCode(), input.Operation())
			},
			wantErrorCode: myerrors.ErrorCodeDataPermissionRuntimeRouteConflict,
		},
		{
			name:      "Adapter failure does not return partial execution",
			resources: []model.DataResource{resource},
			adapter: func(context.Context, datapermission.AdapterInput) (datapermission.AdapterExecution, error) {
				return datapermission.AdapterExecution{}, sentinel
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var legacyCalls atomic.Int32
			resolver := tt.resolver
			if resolver == nil {
				resolver = func(_ *gin.Context, input datapermission.ResolverInput) (datapermission.DataScopeResult, error) {
					return lowCodeRuntimeFilteredResult(t, input.ResourceCode(), input.Operation()), nil
				}
			}
			adapter := tt.adapter
			if adapter == nil {
				adapter = func(_ context.Context, input datapermission.AdapterInput) (datapermission.AdapterExecution, error) {
					return datapermission.BuildAdapterExecution(input)
				}
			}
			runtime := newLowCodeDataPermissionRuntime(
				func(*gin.Context, int) ([]model.DataResource, error) { return tt.resources, nil },
				func(*gin.Context, int) ([]model.DataOwnershipField, error) {
					return []model.DataOwnershipField{lowCodeRuntimeOwnership(resource.Id)}, nil
				},
				func(*gin.Context, int) (datapermission.SubjectContext, error) {
					return lowCodeRuntimeSubject(t, 101), nil
				},
				resolver,
				adapter,
				func(model.SysUser, int, model.SysTable, enum.SysMenuButtonEventAction) (*request.DataScope, error) {
					legacyCalls.Add(1)
					return &request.DataScope{AllowAll: true}, nil
				},
			)

			_, err := runtime.Resolve(
				lowCodeRuntimeGinContext(101),
				table,
				21,
				model.DataPermissionOperationQuery,
				enum.ButtonActionQuery,
			)
			if err == nil {
				t.Fatal("expected fail-closed runtime error")
			}
			if legacyCalls.Load() != 0 {
				t.Fatalf("failed new runtime unexpectedly fell back to legacy %d times", legacyCalls.Load())
			}
			if tt.wantErrorCode != 0 {
				assertLowCodeRuntimeErrorCode(t, err, tt.wantErrorCode)
			}
		})
	}
}

func TestLowCodeDataPermissionRuntimeSupportsConcurrentRequestIsolation(t *testing.T) {
	table := lowCodeRuntimeTestTable()
	resource := lowCodeRuntimeResource(table.Id, true)
	runtime := newLowCodeDataPermissionRuntime(
		func(*gin.Context, int) ([]model.DataResource, error) { return []model.DataResource{resource}, nil },
		func(*gin.Context, int) ([]model.DataOwnershipField, error) {
			return []model.DataOwnershipField{lowCodeRuntimeOwnership(resource.Id)}, nil
		},
		func(_ *gin.Context, userId int) (datapermission.SubjectContext, error) {
			return lowCodeRuntimeSubject(t, userId), nil
		},
		func(_ *gin.Context, input datapermission.ResolverInput) (datapermission.DataScopeResult, error) {
			return lowCodeRuntimeFilteredResult(t, input.ResourceCode(), input.Operation()), nil
		},
		func(_ context.Context, input datapermission.AdapterInput) (datapermission.AdapterExecution, error) {
			return datapermission.BuildAdapterExecution(input)
		},
		func(model.SysUser, int, model.SysTable, enum.SysMenuButtonEventAction) (*request.DataScope, error) {
			return nil, errors.New("legacy route must not be used")
		},
	)

	var wait sync.WaitGroup
	for index := 0; index < 32; index++ {
		userId := 100 + index
		wait.Add(1)
		go func() {
			defer wait.Done()
			resolution, err := runtime.Resolve(
				lowCodeRuntimeGinContext(userId), table, 21,
				model.DataPermissionOperationQuery, enum.ButtonActionQuery,
			)
			if err != nil {
				t.Errorf("concurrent resolve for user %d: %v", userId, err)
				return
			}
			if resolution.permission.AdapterExecution == nil ||
				resolution.permission.AdapterExecution.Mode() != datapermission.AdapterExecutionModeApplyFilter {
				t.Errorf("unexpected concurrent resolution for user %d", userId)
			}
		}()
	}
	wait.Wait()
}

func lowCodeRuntimeTestTable() model.SysTable {
	return model.SysTable{
		Basic:     model.Basic{Id: 9001, State: true},
		TableCode: "permission_demo",
		TableFields: []model.SysTableField{
			{Basic: model.Basic{Id: 501, State: true}, TableId: 9001, FieldCode: "owner_org_id", FieldType: enum.BigIntFieldType, IsAdvancedSearch: true},
		},
	}
}

func lowCodeRuntimeResource(tableId int, enabled bool) model.DataResource {
	return model.DataResource{
		Basic:             model.Basic{Id: 301, State: true},
		ResourceCode:      "permission_demo",
		ResourceType:      model.DataResourceTypeLowCodeTable,
		TableId:           &tableId,
		AdapterCode:       "metadata_filter",
		PermissionEnabled: enabled,
	}
}

func lowCodeRuntimeOwnership(resourceId int) model.DataOwnershipField {
	fieldId := 501
	return model.DataOwnershipField{
		Basic:         model.Basic{Id: 401, State: true},
		ResourceId:    resourceId,
		OwnershipCode: "owner_org",
		DimensionId:   201,
		BindingType:   model.DataOwnershipBindingTypeMetadataField,
		TableFieldId:  &fieldId,
		ValueType:     string(datapermission.DataScopeValueTypeBigint),
	}
}

func lowCodeRuntimeGinContext(userId int) *gin.Context {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("GET", "/admin/generalization/query/code/permission_demo", nil)
	ctx.Set("id", userId)
	ctx.Set("user", model.SysUser{Basic: model.Basic{Id: userId, State: true}})
	return ctx
}

func lowCodeRuntimeSubject(t *testing.T, userId int) datapermission.SubjectContext {
	t.Helper()
	employeeId := userId + 1000
	subject, err := datapermission.NewSubjectContext(userId, []int{7}, &employeeId, "2026-08-03")
	if err != nil {
		t.Fatalf("create SubjectContext: %v", err)
	}
	return subject
}

func lowCodeRuntimeFilteredResult(
	t *testing.T,
	resourceCode string,
	operation string,
) datapermission.DataScopeResult {
	t.Helper()
	condition, err := datapermission.NewDataScopeCondition(datapermission.DataScopeConditionInput{
		OwnershipCode: "owner_org",
		DimensionId:   201,
		Operator:      datapermission.DataScopeOperatorIn,
		ValueType:     datapermission.DataScopeValueTypeBigint,
		Values:        []any{int64(11), int64(12)},
	})
	if err != nil {
		t.Fatalf("create Condition: %v", err)
	}
	group, err := datapermission.NewDataScopeConditionGroup([]datapermission.DataScopeCondition{condition})
	if err != nil {
		t.Fatalf("create ConditionGroup: %v", err)
	}
	result, err := datapermission.NewFilteredResult(resourceCode, operation, []datapermission.DataScopeConditionGroup{group})
	if err != nil {
		t.Fatalf("create DataScopeResult: %v", err)
	}
	return result
}

func assertLowCodeRuntimeErrorCode(t *testing.T, err error, want int) {
	t.Helper()
	clientError, classified := myerrors.ToClientError(err)
	if !classified || clientError.ErrorCode != want {
		t.Fatalf("error = %v, code = %v, want %d", err, clientError, want)
	}
}
