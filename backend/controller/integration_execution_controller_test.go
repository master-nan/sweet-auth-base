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
	pageCalls      int
	detailCalls    int
	logPageCalls   int
	logDetailCalls int
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
) (response.IntegrationExecutionDetailRes, error) {
	s.detailCalls++
	return response.IntegrationExecutionDetailRes{
		IntegrationExecutionListRes: response.IntegrationExecutionListRes{Id: 1, ExecutionNo: "INT-1"},
	}, nil
}

func (s *integrationExecutionApplicationStub) PageLogs(
	_ context.Context,
	_ request.IntegrationLogQueryReq,
	_ model.SysTable,
	_ repository.GeneralizationPermission,
) (response.ListResult[response.IntegrationLogListRes], error) {
	s.logPageCalls++
	return response.ListResult[response.IntegrationLogListRes]{
		Data: []response.IntegrationLogListRes{{Id: 2, ExecutionNo: "INT-1", AttemptNo: 1}}, Total: 1,
	}, nil
}

func (s *integrationExecutionApplicationStub) GetLog(
	_ context.Context,
	_ int,
	_ model.SysTable,
	_ repository.GeneralizationPermission,
) (response.IntegrationLogDetailRes, error) {
	s.logDetailCalls++
	return response.IntegrationLogDetailRes{
		IntegrationLogListRes: response.IntegrationLogListRes{Id: 2, ExecutionNo: "INT-1", AttemptNo: 1},
	}, nil
}

type integrationExecutionTableProviderStub struct {
	tables map[string]model.SysTable
	err    error
}

func (s integrationExecutionTableProviderStub) GetTableByTableCode(code string) (model.SysTable, error) {
	if code != integrationExecutionTableCode && code != integrationLogTableCode {
		return model.SysTable{}, apperrors.ErrParamInvalid
	}
	return s.tables[code], s.err
}

type integrationExecutionPermissionResolverStub struct {
	operations []string
	tableCodes []string
	err        error
}

func (s *integrationExecutionPermissionResolverStub) ResolveDataPermission(
	_ *gin.Context,
	table model.SysTable,
	operation string,
) (repository.GeneralizationPermission, error) {
	s.operations = append(s.operations, operation)
	s.tableCodes = append(s.tableCodes, table.TableCode)
	return repository.GeneralizationPermission{}, s.err
}

func TestIntegrationExecutionControllerResolvesDataPermissionForQueryAndDetail(t *testing.T) {
	app := &integrationExecutionApplicationStub{}
	permissionResolver := &integrationExecutionPermissionResolverStub{}
	controller := newIntegrationExecutionController(
		app,
		integrationExecutionTableProviderStub{tables: map[string]model.SysTable{
			integrationExecutionTableCode: {Basic: model.Basic{Id: 50}, TableCode: integrationExecutionTableCode},
			integrationLogTableCode:       {Basic: model.Basic{Id: 51}, TableCode: integrationLogTableCode},
		}},
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

	logQueryContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	logQueryContext.Request = httptest.NewRequest("POST", "/admin/integration/log/query", strings.NewReader(`{"page":1,"num":10,"execution_id":1}`))
	logQueryContext.Request.Header.Set("Content-Type", "application/json")
	controller.QueryLogs(logQueryContext)

	logDetailContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	logDetailContext.Request = httptest.NewRequest("GET", "/admin/integration/log/2", nil)
	logDetailContext.Params = gin.Params{{Key: "id", Value: "2"}}
	controller.LogDetail(logDetailContext)

	if app.pageCalls != 1 || app.detailCalls != 1 || app.logPageCalls != 1 || app.logDetailCalls != 1 {
		t.Fatalf("service calls page=%d detail=%d log_page=%d log_detail=%d", app.pageCalls, app.detailCalls, app.logPageCalls, app.logDetailCalls)
	}
	if len(permissionResolver.operations) != 4 ||
		permissionResolver.operations[0] != model.DataPermissionOperationQuery ||
		permissionResolver.operations[1] != model.DataPermissionOperationDetail ||
		permissionResolver.operations[2] != model.DataPermissionOperationQuery ||
		permissionResolver.operations[3] != model.DataPermissionOperationDetail {
		t.Fatalf("permission operations = %v", permissionResolver.operations)
	}
	wantTables := []string{integrationExecutionTableCode, integrationExecutionTableCode, integrationLogTableCode, integrationLogTableCode}
	for index := range wantTables {
		if permissionResolver.tableCodes[index] != wantTables[index] {
			t.Fatalf("permission tables = %v", permissionResolver.tableCodes)
		}
	}
}

func TestIntegrationExecutionControllerStopsWhenPermissionResolutionFails(t *testing.T) {
	app := &integrationExecutionApplicationStub{}
	controller := newIntegrationExecutionController(
		app,
		integrationExecutionTableProviderStub{tables: map[string]model.SysTable{
			integrationExecutionTableCode: {Basic: model.Basic{Id: 50}, TableCode: integrationExecutionTableCode},
		}},
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
