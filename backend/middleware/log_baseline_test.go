package middleware

import (
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/model"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type captureAccessLogWriter struct {
	logs []model.AccessLog
	err  error
}

func (writer *captureAccessLogWriter) CreateAccessLog(_ context.Context, log model.AccessLog) error {
	writer.logs = append(writer.logs, log)
	return writer.err
}

func TestLogHandlerPropagatesContextAndRecordsSuccessfulAudit(t *testing.T) {
	writer := &captureAccessLogWriter{}
	engine := newLogBaselineEngine(t, writer)
	engine.GET("/baseline/success", func(ctx *gin.Context) {
		ctx.Set("user", model.SysUser{
			Basic:    model.Basic{Id: 42},
			UserName: "auditor",
		})
		SetAuditContext(ctx, AuditContext{
			MenuID:       1001,
			Action:       "baseline_query",
			ResourceType: "baseline_resource",
			ResourceCode: "baseline",
			ResourceID:   "88",
		})
		ctx.Set("response", response.NewResponse().SetData(gin.H{"id": 88}))
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/baseline/success", nil)
	request.Header.Set(RequestIDHeader, "request-001")
	request.Header.Set(TraceIDHeader, "trace-001")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get(RequestIDHeader); got != "request-001" {
		t.Fatalf("response request id = %q, want %q", got, "request-001")
	}
	if got := recorder.Header().Get(TraceIDHeader); got != "trace-001" {
		t.Fatalf("response trace id = %q, want %q", got, "trace-001")
	}

	log := onlyCapturedAccessLog(t, writer)
	if log.RequestId != "request-001" || log.TraceId != "trace-001" {
		t.Fatalf("unexpected correlation context: request_id=%q trace_id=%q", log.RequestId, log.TraceId)
	}
	if log.UserId != 42 || log.UserName != "auditor" {
		t.Fatalf("unexpected actor: user_id=%d user_name=%q", log.UserId, log.UserName)
	}
	if log.MenuId != 1001 ||
		log.Action != "baseline_query" ||
		log.ResourceType != "baseline_resource" ||
		log.ResourceCode != "baseline" ||
		log.ResourceId != "88" {
		t.Fatalf("unexpected audit context: %#v", log)
	}
	if !log.Success || log.Result != accessAuditResultSuccess {
		t.Fatalf("unexpected audit result: success=%t result=%q", log.Success, log.Result)
	}
	if log.ErrorCode != "" || log.ErrorMessage != "" {
		t.Fatalf("successful audit retained error: code=%q message=%q", log.ErrorCode, log.ErrorMessage)
	}
}

func TestLogHandlerRecordsFailedAuditWithSafeError(t *testing.T) {
	writer := &captureAccessLogWriter{}
	engine := newLogBaselineEngine(t, writer)
	engine.POST("/baseline/failure", func(ctx *gin.Context) {
		ctx.Set("user", model.SysUser{
			Basic:    model.Basic{Id: 7},
			UserName: "operator",
		})
		SetAuditContext(ctx, AuditContext{
			MenuID:       2002,
			Action:       "baseline_update",
			ResourceType: "baseline_resource",
			ResourceID:   "99",
		})
		_ = ctx.Error(myerrors.WrapApplicationError(nil, myerrors.KindConflict, myerrors.CategoryBusiness, 81001, "记录冲突"))
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/baseline/failure", nil)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusConflict)
	}
	log := onlyCapturedAccessLog(t, writer)
	if log.Success || log.Result != accessAuditResultFailed {
		t.Fatalf("unexpected audit result: success=%t result=%q", log.Success, log.Result)
	}
	if log.ErrorCode != "81001" || log.ErrorMessage != "记录冲突" {
		t.Fatalf("unexpected audit error: code=%q message=%q", log.ErrorCode, log.ErrorMessage)
	}
	if log.UserId != 7 || log.Action != "baseline_update" || log.MenuId != 2002 {
		t.Fatalf("failed audit lost actor or operation context: %#v", log)
	}
	if log.RequestId == "" || log.TraceId == "" {
		t.Fatalf("failed audit missing correlation context: %#v", log)
	}
}

func TestLogHandlerGeneratesCorrelationIDsAndRejectsUnsafeHeader(t *testing.T) {
	writer := &captureAccessLogWriter{}
	engine := newLogBaselineEngine(t, writer)
	engine.GET("/baseline/generated", func(ctx *gin.Context) {
		ctx.Set("response", response.NewResponse())
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/baseline/generated", nil)
	request.Header.Set(RequestIDHeader, "unsafe request id\n")
	engine.ServeHTTP(recorder, request)

	log := onlyCapturedAccessLog(t, writer)
	if log.RequestId == "" || log.TraceId == "" {
		t.Fatalf("generated correlation context is empty: %#v", log)
	}
	if log.RequestId != log.TraceId {
		t.Fatalf("trace id = %q, want generated request id %q", log.TraceId, log.RequestId)
	}
	if log.RequestId == "unsafe request id" {
		t.Fatalf("unsafe incoming request id was accepted")
	}
	if recorder.Header().Get(RequestIDHeader) != log.RequestId ||
		recorder.Header().Get(TraceIDHeader) != log.TraceId {
		t.Fatalf("generated IDs were not returned to caller")
	}
}

func TestLogHandlerDoesNotDuplicateCommittedTransactionalAudit(t *testing.T) {
	writer := &captureAccessLogWriter{}
	engine := newLogBaselineEngine(t, writer)
	engine.POST("/baseline/transactional-audit", func(ctx *gin.Context) {
		SetAuditContext(ctx, AuditContext{
			Action:       "bind_user",
			ResourceType: "org_employee",
			ResourceID:   "88",
		})
		MarkAccessAuditPersisted(ctx)
		ctx.Set("response", response.NewResponse().SetData(gin.H{"employee_id": 88}))
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/baseline/transactional-audit", nil)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if len(writer.logs) != 0 {
		t.Fatalf("request middleware duplicated %d transactional audit records", len(writer.logs))
	}
}

func newLogBaselineEngine(t *testing.T, writer accessLogWriter) *gin.Engine {
	t.Helper()
	restoreLogger := zap.ReplaceGlobals(zap.NewNop())
	t.Cleanup(restoreLogger)

	engine := gin.New()
	engine.Use(LogHandler(writer))
	engine.Use(ResponseHandler())
	return engine
}

func onlyCapturedAccessLog(t *testing.T, writer *captureAccessLogWriter) model.AccessLog {
	t.Helper()
	if len(writer.logs) != 1 {
		t.Fatalf("captured %d access logs, want 1", len(writer.logs))
	}
	return writer.logs[0]
}
