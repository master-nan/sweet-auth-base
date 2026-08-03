package service

import (
	"context"
	"errors"
	"reflect"
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

type dataPermissionGeneralizationRepoSpy struct {
	mutex               sync.Mutex
	queryCalls          int
	detailCalls         int
	updateCalls         int
	softDeleteCalls     int
	hardDeleteCalls     int
	batchDeleteCalls    int
	permissionModes     []repository.GeneralizationPermissionMode
	executionOperations []string
}

func (repo *dataPermissionGeneralizationRepoSpy) Query(*request.Basic, model.SysTable) (repository.GeneralizationListResult, error) {
	return repository.GeneralizationListResult{}, nil
}

func (repo *dataPermissionGeneralizationRepoSpy) GetById(model.SysTable, int) (map[string]interface{}, error) {
	return nil, nil
}

func (repo *dataPermissionGeneralizationRepoSpy) Create(model.SysTable, map[string]interface{}) error {
	return nil
}

func (repo *dataPermissionGeneralizationRepoSpy) RowExists(model.SysTable, int) (bool, error) {
	return true, nil
}

func (repo *dataPermissionGeneralizationRepoSpy) RowMatchesDataScope(model.SysTable, int, *request.DataScope) (bool, error) {
	return true, nil
}

func (repo *dataPermissionGeneralizationRepoSpy) Update(model.SysTable, int, map[string]interface{}) error {
	return nil
}

func (repo *dataPermissionGeneralizationRepoSpy) SoftDelete(model.SysTable, int, map[string]interface{}) error {
	return nil
}

func (repo *dataPermissionGeneralizationRepoSpy) HardDelete(model.SysTable, int) error {
	return nil
}

func (repo *dataPermissionGeneralizationRepoSpy) GetFieldById(string, int, string) (interface{}, error) {
	return nil, nil
}

func (repo *dataPermissionGeneralizationRepoSpy) QueryWithPermission(
	_ *request.Basic,
	_ model.SysTable,
	permission repository.GeneralizationPermission,
) (repository.GeneralizationListResult, error) {
	repo.recordPermission(permission)
	repo.mutex.Lock()
	repo.queryCalls++
	repo.mutex.Unlock()
	return repository.GeneralizationListResult{}, nil
}

func (repo *dataPermissionGeneralizationRepoSpy) GetByIdWithPermission(
	_ model.SysTable,
	_ int,
	permission repository.GeneralizationPermission,
) (map[string]interface{}, error) {
	repo.recordPermission(permission)
	repo.mutex.Lock()
	repo.detailCalls++
	repo.mutex.Unlock()
	return map[string]interface{}{"id": int64(1)}, nil
}

func (repo *dataPermissionGeneralizationRepoSpy) UpdateWithPermission(
	_ model.SysTable,
	_ int,
	_ map[string]interface{},
	permission repository.GeneralizationPermission,
) (bool, error) {
	repo.recordPermission(permission)
	repo.mutex.Lock()
	repo.updateCalls++
	repo.mutex.Unlock()
	return true, nil
}

func (repo *dataPermissionGeneralizationRepoSpy) SoftDeleteWithPermission(
	_ model.SysTable,
	_ int,
	_ map[string]interface{},
	permission repository.GeneralizationPermission,
) (bool, error) {
	repo.recordPermission(permission)
	repo.mutex.Lock()
	repo.softDeleteCalls++
	repo.mutex.Unlock()
	return true, nil
}

func (repo *dataPermissionGeneralizationRepoSpy) HardDeleteWithPermission(
	_ model.SysTable,
	_ int,
	permission repository.GeneralizationPermission,
) (bool, error) {
	repo.recordPermission(permission)
	repo.mutex.Lock()
	repo.hardDeleteCalls++
	repo.mutex.Unlock()
	return true, nil
}

func (repo *dataPermissionGeneralizationRepoSpy) BatchSoftDeleteWithPermission(
	_ model.SysTable,
	_ []int,
	_ map[string]interface{},
	permission repository.GeneralizationPermission,
) (bool, error) {
	repo.recordPermission(permission)
	repo.mutex.Lock()
	repo.batchDeleteCalls++
	repo.mutex.Unlock()
	return true, nil
}

func (repo *dataPermissionGeneralizationRepoSpy) BatchHardDeleteWithPermission(
	_ model.SysTable,
	_ []int,
	permission repository.GeneralizationPermission,
) (bool, error) {
	repo.recordPermission(permission)
	repo.mutex.Lock()
	repo.batchDeleteCalls++
	repo.mutex.Unlock()
	return true, nil
}

func (repo *dataPermissionGeneralizationRepoSpy) recordPermission(
	permission repository.GeneralizationPermission,
) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()
	repo.permissionModes = append(repo.permissionModes, permission.Mode)
	if permission.AdapterExecution != nil {
		repo.executionOperations = append(repo.executionOperations, permission.AdapterExecution.Operation())
	}
}

func TestGeneralizationServiceUsesIndependentOperationsAndOneRuntimeResolution(t *testing.T) {
	table := lowCodeRuntimeTestTable()
	table.TableFields = append(table.TableFields,
		model.SysTableField{Basic: model.Basic{Id: 502, State: true}, TableId: table.Id, FieldCode: "name", FieldType: enum.VarcharFieldType, IsListShow: true, IsUpdateShow: true},
		model.SysTableField{Basic: model.Basic{Id: 503, State: true}, TableId: table.Id, FieldCode: "gmt_delete", FieldType: enum.DatetimeFieldType},
	)
	resource := lowCodeRuntimeResource(table.Id, true)
	var subjectCalls atomic.Int32
	var resolverCalls atomic.Int32
	var adapterCalls atomic.Int32
	var operationsMutex sync.Mutex
	operations := make([]string, 0, 5)
	runtime := newLowCodeDataPermissionRuntime(
		func(*gin.Context, int) ([]model.DataResource, error) { return []model.DataResource{resource}, nil },
		func(*gin.Context, int) ([]model.DataOwnershipField, error) {
			return []model.DataOwnershipField{lowCodeRuntimeOwnership(resource.Id)}, nil
		},
		func(_ *gin.Context, userId int) (datapermission.SubjectContext, error) {
			subjectCalls.Add(1)
			return lowCodeRuntimeSubject(t, userId), nil
		},
		func(_ *gin.Context, input datapermission.ResolverInput) (datapermission.DataScopeResult, error) {
			resolverCalls.Add(1)
			operationsMutex.Lock()
			operations = append(operations, input.Operation())
			operationsMutex.Unlock()
			return datapermission.NewAllResult(input.ResourceCode(), input.Operation())
		},
		func(_ context.Context, input datapermission.AdapterInput) (datapermission.AdapterExecution, error) {
			adapterCalls.Add(1)
			return datapermission.BuildAdapterExecution(input)
		},
		func(model.SysUser, int, model.SysTable, enum.SysMenuButtonEventAction) (*request.DataScope, error) {
			return nil, errors.New("new runtime must not use legacy scope")
		},
	)
	repo := &dataPermissionGeneralizationRepoSpy{}
	service := NewGeneralizationServiceWithDataPermission(repo, nil, runtime)
	ctx := lowCodeRuntimeGinContext(101)

	if _, err := service.QueryWithDataPermission(ctx, &request.Basic{Page: 1, Num: 10}, table, 21, model.DataPermissionOperationQuery, enum.ButtonActionQuery); err != nil {
		t.Fatalf("query: %v", err)
	}
	if _, err := service.QueryWithDataPermission(ctx, &request.Basic{Page: 1, Num: 10}, table, 21, model.DataPermissionOperationExport, enum.ButtonActionExport); err != nil {
		t.Fatalf("export query: %v", err)
	}
	if _, err := service.GetByIdWithDataPermission(ctx, table, 1, 21, model.DataPermissionOperationDetail, enum.ButtonActionDetail); err != nil {
		t.Fatalf("detail: %v", err)
	}
	if err := service.UpdateWithDataPermission(ctx, table, 1, map[string]interface{}{"name": "updated"}, 21); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := service.DeleteWithDataPermission(ctx, table, 1, 21, enum.ButtonActionDelete); err != nil {
		t.Fatalf("delete: %v", err)
	}

	wantOperations := []string{
		model.DataPermissionOperationQuery,
		model.DataPermissionOperationExport,
		model.DataPermissionOperationDetail,
		model.DataPermissionOperationUpdate,
		model.DataPermissionOperationDelete,
	}
	if !reflect.DeepEqual(operations, wantOperations) {
		t.Fatalf("Resolver operations = %v, want %v", operations, wantOperations)
	}
	if subjectCalls.Load() != 5 || resolverCalls.Load() != 5 || adapterCalls.Load() != 5 {
		t.Fatalf("runtime calls subject=%d resolver=%d adapter=%d, want one per request",
			subjectCalls.Load(), resolverCalls.Load(), adapterCalls.Load())
	}
	if repo.queryCalls != 2 || repo.detailCalls != 1 || repo.updateCalls != 1 || repo.softDeleteCalls != 1 {
		t.Fatalf("repository calls query=%d detail=%d update=%d delete=%d",
			repo.queryCalls, repo.detailCalls, repo.updateCalls, repo.softDeleteCalls)
	}
	if !reflect.DeepEqual(repo.executionOperations, wantOperations) {
		t.Fatalf("repository operations = %v, want %v", repo.executionOperations, wantOperations)
	}
}

func TestGeneralizationServiceDeniesOwnershipMutationAndRuntimeFailureBeforeWrite(t *testing.T) {
	table := lowCodeRuntimeTestTable()
	resource := lowCodeRuntimeResource(table.Id, true)
	buildRuntime := func(resolver lowCodeResolver) *LowCodeDataPermissionRuntime {
		return newLowCodeDataPermissionRuntime(
			func(*gin.Context, int) ([]model.DataResource, error) { return []model.DataResource{resource}, nil },
			func(*gin.Context, int) ([]model.DataOwnershipField, error) {
				return []model.DataOwnershipField{lowCodeRuntimeOwnership(resource.Id)}, nil
			},
			func(_ *gin.Context, userId int) (datapermission.SubjectContext, error) {
				return lowCodeRuntimeSubject(t, userId), nil
			},
			resolver,
			func(_ context.Context, input datapermission.AdapterInput) (datapermission.AdapterExecution, error) {
				return datapermission.BuildAdapterExecution(input)
			},
			func(model.SysUser, int, model.SysTable, enum.SysMenuButtonEventAction) (*request.DataScope, error) {
				return nil, errors.New("legacy route must not be used")
			},
		)
	}
	ctx := lowCodeRuntimeGinContext(101)

	repo := &dataPermissionGeneralizationRepoSpy{}
	service := NewGeneralizationServiceWithDataPermission(repo, nil, buildRuntime(
		func(_ *gin.Context, input datapermission.ResolverInput) (datapermission.DataScopeResult, error) {
			return lowCodeRuntimeFilteredResult(t, input.ResourceCode(), input.Operation()), nil
		},
	))
	err := service.UpdateWithDataPermission(ctx, table, 1, map[string]interface{}{"owner_org_id": 12}, 21)
	assertLowCodeRuntimeErrorCode(t, err, myerrors.ErrorCodeDataPermissionOwnershipUpdateDenied)
	if repo.updateCalls != 0 {
		t.Fatal("ownership mutation reached repository")
	}

	repo = &dataPermissionGeneralizationRepoSpy{}
	service = NewGeneralizationServiceWithDataPermission(repo, nil, buildRuntime(
		func(*gin.Context, datapermission.ResolverInput) (datapermission.DataScopeResult, error) {
			return datapermission.DataScopeResult{}, errors.New("Resolver failed")
		},
	))
	if err = service.UpdateWithDataPermission(ctx, table, 1, map[string]interface{}{"name": "leaked"}, 21); err == nil {
		t.Fatal("Resolver failure was ignored")
	}
	if repo.updateCalls != 0 {
		t.Fatal("Resolver failure reached repository update")
	}
}

func TestGeneralizationServiceBatchDeleteResolvesOnceAndUsesOneAtomicRepositoryCall(t *testing.T) {
	table := lowCodeRuntimeTestTable()
	table.TableFields = append(table.TableFields,
		model.SysTableField{Basic: model.Basic{Id: 503, State: true}, TableId: table.Id, FieldCode: "gmt_delete", FieldType: enum.DatetimeFieldType},
	)
	resource := lowCodeRuntimeResource(table.Id, true)
	var subjectCalls atomic.Int32
	var resolverCalls atomic.Int32
	runtime := newLowCodeDataPermissionRuntime(
		func(*gin.Context, int) ([]model.DataResource, error) { return []model.DataResource{resource}, nil },
		func(*gin.Context, int) ([]model.DataOwnershipField, error) {
			return []model.DataOwnershipField{lowCodeRuntimeOwnership(resource.Id)}, nil
		},
		func(_ *gin.Context, userId int) (datapermission.SubjectContext, error) {
			subjectCalls.Add(1)
			return lowCodeRuntimeSubject(t, userId), nil
		},
		func(_ *gin.Context, input datapermission.ResolverInput) (datapermission.DataScopeResult, error) {
			resolverCalls.Add(1)
			if input.Operation() != model.DataPermissionOperationDelete {
				t.Fatalf("batch delete operation = %s", input.Operation())
			}
			return datapermission.NewAllResult(input.ResourceCode(), input.Operation())
		},
		func(_ context.Context, input datapermission.AdapterInput) (datapermission.AdapterExecution, error) {
			return datapermission.BuildAdapterExecution(input)
		},
		func(model.SysUser, int, model.SysTable, enum.SysMenuButtonEventAction) (*request.DataScope, error) {
			return nil, errors.New("legacy route must not be used")
		},
	)
	repo := &dataPermissionGeneralizationRepoSpy{}
	service := NewGeneralizationServiceWithDataPermission(repo, nil, runtime)

	if err := service.BatchDeleteWithDataPermission(
		lowCodeRuntimeGinContext(101), table, []int{1, 2, 2}, 21, enum.ButtonActionBatchDelete,
	); err != nil {
		t.Fatalf("batch delete: %v", err)
	}
	if subjectCalls.Load() != 1 || resolverCalls.Load() != 1 || repo.batchDeleteCalls != 1 {
		t.Fatalf("batch request calls subject=%d resolver=%d repository=%d, want 1/1/1",
			subjectCalls.Load(), resolverCalls.Load(), repo.batchDeleteCalls)
	}
}

var _ repository.GeneralizationRepository = (*dataPermissionGeneralizationRepoSpy)(nil)
var _ repository.GeneralizationPermissionRepository = (*dataPermissionGeneralizationRepoSpy)(nil)
