package controller

import (
	"backend/dto/request"
	"backend/dto/response"
	apperrors "backend/internal/errors"
	"backend/model"
	"backend/repository"
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type integrationExecutionApplicationStub struct {
	integrationExecutionApplication
	pageCalls   int
	detailCalls int
	operations  []string
}

func (s *integrationExecutionApplicationStub) PageExecution(
	_ context.Context,
	_ request.IntegrationExecutionQueryReq,
	_ model.SysTable,
	_ repository.GeneralizationPermission,
) (response.ListResult[response.IntegrationExecutionListRes], error) {
	s.pageCalls++
	return response.ListResult[response.IntegrationExecutionListRes]{
		Data: []response.IntegrationExecutionListRes{{Id: 1, ExecutionNo: "INT-1"}}, Total: 1,
	}, nil
}

func (s *integrationExecutionApplicationStub) GetExecution(
	_ context.Context,
	_ int,
	_ model.SysTable,
	_ repository.GeneralizationPermission,
	_ model.SysTable,
	_ repository.GeneralizationPermission,
) (response.IntegrationExecutionDetailRes, error) {
	s.detailCalls++
	return response.IntegrationExecutionDetailRes{
		IntegrationExecutionListRes: response.IntegrationExecutionListRes{Id: 1, ExecutionNo: "INT-1"},
	}, nil
}

type integrationExecutionTableProviderStub struct {
	table model.SysTable
	err   error
}

func (s integrationExecutionTableProviderStub) GetTableByTableCode(code string) (model.SysTable, error) {
	if code != integrationExecutionTableCode && code != integrationLogTableCode {
		return model.SysTable{}, apperrors.ErrParamInvalid
	}
	return s.table, s.err
}

type integrationExecutionPermissionResolverStub struct {
	operations []string
	err        error
}

func (s *integrationExecutionPermissionResolverStub) ResolveDataPermission(
	_ *gin.Context,
	_ model.SysTable,
	operation string,
) (repository.GeneralizationPermission, error) {
	s.operations = append(s.operations, operation)
	return repository.GeneralizationPermission{}, s.err
}

func TestIntegrationExecutionControllerResolvesDataPermissionForQueryAndDetail(t *testing.T) {
	app := &integrationExecutionApplicationStub{}
	permissionResolver := &integrationExecutionPermissionResolverStub{}
	controller := newIntegrationExecutionController(
		app,
		integrationExecutionTableProviderStub{table: model.SysTable{Basic: model.Basic{Id: 50}, TableCode: integrationExecutionTableCode}},
		permissionResolver,
		nil,
	)

	queryContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	queryContext.Request = httptest.NewRequest("POST", "/admin/integration/execution/query", strings.NewReader(`{"page":1,"num":10}`))
	queryContext.Request.Header.Set("Content-Type", "application/json")
	controller.Query(queryContext)

	detailContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	detailContext.Request = httptest.NewRequest("GET", "/admin/integration/execution/1", nil)
	detailContext.Params = gin.Params{{Key: "id", Value: "1"}}
	controller.Detail(detailContext)

	if app.pageCalls != 1 || app.detailCalls != 1 {
		t.Fatalf("service calls page=%d detail=%d", app.pageCalls, app.detailCalls)
	}
	if len(permissionResolver.operations) != 3 ||
		permissionResolver.operations[0] != model.DataPermissionOperationQuery ||
		permissionResolver.operations[1] != model.DataPermissionOperationDetail ||
		permissionResolver.operations[2] != model.DataPermissionOperationDetail {
		t.Fatalf("permission operations = %v", permissionResolver.operations)
	}
}

func TestIntegrationExecutionControllerStopsWhenPermissionResolutionFails(t *testing.T) {
	app := &integrationExecutionApplicationStub{}
	controller := newIntegrationExecutionController(
		app,
		integrationExecutionTableProviderStub{table: model.SysTable{Basic: model.Basic{Id: 50}, TableCode: integrationExecutionTableCode}},
		&integrationExecutionPermissionResolverStub{err: apperrors.ErrDataPermissionRuntimeFailed},
		nil,
	)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/admin/integration/execution/query", strings.NewReader(`{"page":1,"num":10}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	controller.Query(ctx)
	if app.pageCalls != 0 || len(ctx.Errors) != 1 {
		t.Fatalf("permission failure service calls=%d errors=%v", app.pageCalls, ctx.Errors)
	}
}
