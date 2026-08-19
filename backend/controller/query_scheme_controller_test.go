package controller

import (
	"backend/dto/request"
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/internal/queryscheme"
	"backend/middleware"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type querySchemeApplicationStub struct {
	querySchemeApplication
	available []response.QuerySchemeSummaryRes
	sharedErr error
}

func (stub querySchemeApplicationStub) Available(context.Context, string) ([]response.QuerySchemeSummaryRes, error) {
	return stub.available, nil
}

func (stub querySchemeApplicationStub) CreateShared(context.Context, request.QuerySchemeSharedCreateReq) (response.QuerySchemeDetailRes, error) {
	return response.QuerySchemeDetailRes{}, stub.sharedErr
}

func TestQuerySchemeRuntimeSummaryDoesNotExposePayloadOrRevision(t *testing.T) {
	controller := &QuerySchemeController{service: querySchemeApplicationStub{available: []response.QuerySchemeSummaryRes{{
		ID: 10, Name: "我的方案", IsDefault: true, Status: queryscheme.ValidationValid,
	}}}}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/admin/runtime/query-schemes/available?scope_code=system.user.list", nil)
	controller.Available(ctx)
	value, exists := ctx.Get("response")
	if !exists {
		t.Fatal("unified response missing")
	}
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"query_payload", "revision", "owner_user_id", "role_ids"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("runtime summary leaked %s: %s", forbidden, raw)
		}
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func TestQuerySchemeSharedFailureUsesStableApplicationError(t *testing.T) {
	router := gin.New()
	router.Use(middleware.ResponseHandler())
	controller := &QuerySchemeController{service: querySchemeApplicationStub{sharedErr: myerrors.ErrQuerySchemeSharedForbidden}}
	router.POST("/shared", controller.CreateShared)
	body := `{"name":"public","scope_code":"system.user.list","scheme_type":"PUBLIC","query_payload":{"expressions":[],"quick_query":{"keyword":""},"order":{"field":"","is_asc":false},"bindings":[]},"enabled":true}`
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/shared", strings.NewReader(body)))
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "无权管理共享查询方案") || strings.Contains(strings.ToLower(recorder.Body.String()), "sql") {
		t.Fatalf("unsafe or unstable response: %s", recorder.Body.String())
	}
}
