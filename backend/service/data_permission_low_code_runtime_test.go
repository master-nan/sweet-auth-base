package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"backend/enum"
	"backend/internal/audit"
	"backend/internal/datapermission"
	myerrors "backend/internal/errors"
	"backend/model"
)

func TestLowCodeDataPermissionRuntimeReturnsNotApplicableWithoutLegacyFallback(t *testing.T) {
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
			var subjectCalls atomic.Int32
			var adapterCalls atomic.Int32
			runtime := newLowCodeDataPermissionRuntime(
				func(_ context.Context, tableId int) ([]model.DataResource, error) {
					if tableId != table.Id {
						t.Fatalf("table id = %d, want %d", tableId, table.Id)
					}
					return tt.resources, nil
				},
				func(context.Context, int) ([]model.DataOwnershipField, error) {
					t.Fatal("not_applicable route must not load Ownership configuration")
					return nil, nil
				},
				func(context.Context, int) (datapermission.SubjectContext, error) {
					subjectCalls.Add(1)
					return datapermission.SubjectContext{}, nil
				},
				func(context.Context, datapermission.ResolverInput) (datapermission.DataScopeResult, error) {
					t.Fatal("not_applicable route must not call Resolver")
					return datapermission.DataScopeResult{}, nil
				},
				func(_ context.Context, input datapermission.AdapterInput) (datapermission.AdapterExecution, error) {
					adapterCalls.Add(1)
					return datapermission.BuildAdapterExecution(input)
				},
			)

			resolution, err := runtime.Resolve(
				lowCodeRuntimeContext(101),
				table,
				model.DataPermissionOperationQuery,
			)
			if err != nil {
				t.Fatalf("resolve not_applicable route: %v", err)
			}
			if resolution.permission.AdapterExecution == nil ||
				resolution.permission.AdapterExecution.Mode() != datapermission.AdapterExecutionModeNotApplicable {
				t.Fatalf("unexpected not_applicable resolution: %+v", resolution.permission)
			}
			if adapterCalls.Load() != 1 || subjectCalls.Load() != 0 {
				t.Fatalf("adapter calls=%d subject calls=%d, want 1/0", adapterCalls.Load(), subjectCalls.Load())
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
	runtime := newLowCodeDataPermissionRuntime(
		func(context.Context, int) ([]model.DataResource, error) {
			return []model.DataResource{resource}, nil
		},
		func(_ context.Context, resourceId int) ([]model.DataOwnershipField, error) {
			if resourceId != resource.Id {
				t.Fatalf("resource id = %d, want %d", resourceId, resource.Id)
			}
			return []model.DataOwnershipField{lowCodeRuntimeOwnership(resource.Id)}, nil
		},
		func(_ context.Context, userId int) (datapermission.SubjectContext, error) {
			subjectCalls.Add(1)
			if userId != 101 {
				t.Fatalf("subject user = %d, want trusted user 101", userId)
			}
			return lowCodeRuntimeSubject(t, userId), nil
		},
		func(_ context.Context, input datapermission.ResolverInput) (datapermission.DataScopeResult, error) {
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
	)

	resolution, err := runtime.Resolve(
		lowCodeRuntimeContext(101),
		table,
		model.DataPermissionOperationQuery,
	)
	if err != nil {
		t.Fatalf("resolve new runtime: %v", err)
	}
	if resolution.permission.AdapterExecution == nil ||
		resolution.permission.AdapterExecution.Mode() != datapermission.AdapterExecutionModeApplyFilter {
		t.Fatalf("unexpected adapter resolution: %+v", resolution.permission)
	}
	if _, protected := resolution.ownershipFieldIds[501]; !protected {
		t.Fatal("metadata ownership field was not protected from generic updates")
	}
	if subjectCalls.Load() != 1 || resolverCalls.Load() != 1 || adapterCalls.Load() != 1 {
		t.Fatalf(
			"calls subject=%d resolver=%d adapter=%d, want 1/1/1",
			subjectCalls.Load(), resolverCalls.Load(), adapterCalls.Load(),
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
			resolver: func(context.Context, datapermission.ResolverInput) (datapermission.DataScopeResult, error) {
				return datapermission.DataScopeResult{}, sentinel
			},
		},
		{
			name:      "not applicable from enabled resource is a route conflict",
			resources: []model.DataResource{resource},
			resolver: func(_ context.Context, input datapermission.ResolverInput) (datapermission.DataScopeResult, error) {
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
			resolver := tt.resolver
			if resolver == nil {
				resolver = func(_ context.Context, input datapermission.ResolverInput) (datapermission.DataScopeResult, error) {
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
				func(context.Context, int) ([]model.DataResource, error) { return tt.resources, nil },
				func(context.Context, int) ([]model.DataOwnershipField, error) {
					return []model.DataOwnershipField{lowCodeRuntimeOwnership(resource.Id)}, nil
				},
				func(context.Context, int) (datapermission.SubjectContext, error) {
					return lowCodeRuntimeSubject(t, 101), nil
				},
				resolver,
				adapter,
			)

			_, err := runtime.Resolve(
				lowCodeRuntimeContext(101),
				table,
				model.DataPermissionOperationQuery,
			)
			if err == nil {
				t.Fatal("expected fail-closed runtime error")
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
		func(context.Context, int) ([]model.DataResource, error) { return []model.DataResource{resource}, nil },
		func(context.Context, int) ([]model.DataOwnershipField, error) {
			return []model.DataOwnershipField{lowCodeRuntimeOwnership(resource.Id)}, nil
		},
		func(_ context.Context, userId int) (datapermission.SubjectContext, error) {
			return lowCodeRuntimeSubject(t, userId), nil
		},
		func(_ context.Context, input datapermission.ResolverInput) (datapermission.DataScopeResult, error) {
			return lowCodeRuntimeFilteredResult(t, input.ResourceCode(), input.Operation()), nil
		},
		func(_ context.Context, input datapermission.AdapterInput) (datapermission.AdapterExecution, error) {
			return datapermission.BuildAdapterExecution(input)
		},
	)

	var wait sync.WaitGroup
	for index := 0; index < 32; index++ {
		userId := 100 + index
		wait.Add(1)
		go func() {
			defer wait.Done()
			resolution, err := runtime.Resolve(
				lowCodeRuntimeContext(userId), table,
				model.DataPermissionOperationQuery,
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

func lowCodeRuntimeContext(userId int) context.Context {
	return audit.WithAuditSubject(context.Background(), audit.NewAuditSubject(userId, "runtime-test"))
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
	clientError, classified := myerrors.Classify(err)
	if !classified || clientError.Code != want {
		t.Fatalf("error = %v, code = %v, want %d", err, clientError, want)
	}
}
