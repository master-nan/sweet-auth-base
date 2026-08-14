package middleware

import (
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type responseBaselineController struct{}

func (responseBaselineController) success(ctx *gin.Context) {
	ctx.Set(
		"response",
		response.NewResponse().
			SetData(map[string]string{"name": "baseline"}).
			SetTotal(1),
	)
}

func (responseBaselineController) businessError(ctx *gin.Context) {
	rootErr := stderrors.New("internal conflict detail")
	err := myerrors.WrapApplicationError(rootErr, myerrors.KindConflict, myerrors.CategoryBusiness, 81001, "记录冲突")
	_ = ctx.Error(fmt.Errorf("service: %w", err))
}

func (responseBaselineController) unknownError(ctx *gin.Context) {
	_ = ctx.Error(stderrors.New("pq: password=do-not-expose"))
}

func (responseBaselineController) parameterError(ctx *gin.Context) {
	_, err := strconv.Atoi("not-an-id")
	_ = ctx.Error(err)
}

func TestResponseHandlerWritesControllerSuccessResponse(t *testing.T) {
	recorder := performResponseRequest(t, responseBaselineController{}.success)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	var payload response.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !payload.Success || payload.Code != http.StatusOK || payload.Total != 1 {
		t.Fatalf("unexpected success response: %#v", payload)
	}
}

func TestResponseHandlerWritesWrappedBusinessError(t *testing.T) {
	recorder := performResponseRequest(t, responseBaselineController{}.businessError)

	assertAdminErrorResponse(
		t,
		recorder,
		http.StatusConflict,
		81001,
		"记录冲突",
	)
	if strings.Contains(recorder.Body.String(), "internal conflict detail") {
		t.Fatalf("internal cause leaked to client: %s", recorder.Body.String())
	}
}

func TestResponseHandlerSanitizesUnknownError(t *testing.T) {
	recorder := performResponseRequest(t, responseBaselineController{}.unknownError)

	assertAdminErrorResponse(
		t,
		recorder,
		http.StatusInternalServerError,
		10000,
		"系统异常",
	)
	if strings.Contains(recorder.Body.String(), "do-not-expose") {
		t.Fatalf("raw system error leaked to client: %s", recorder.Body.String())
	}
}

func TestResponseHandlerClassifiesRawParameterError(t *testing.T) {
	recorder := performResponseRequest(t, responseBaselineController{}.parameterError)

	assertAdminErrorResponse(
		t,
		recorder,
		http.StatusBadRequest,
		20003,
		"参数错误",
	)
}

func TestHTTPErrorTranslationMapsApplicationKinds(t *testing.T) {
	tests := []struct {
		kind   myerrors.Kind
		status int
	}{
		{kind: myerrors.KindInvalidArgument, status: http.StatusBadRequest},
		{kind: myerrors.KindUnauthenticated, status: http.StatusUnauthorized},
		{kind: myerrors.KindForbidden, status: http.StatusForbidden},
		{kind: myerrors.KindNotFound, status: http.StatusNotFound},
		{kind: myerrors.KindConflict, status: http.StatusConflict},
		{kind: myerrors.KindUnprocessable, status: http.StatusUnprocessableEntity},
		{kind: myerrors.KindPayloadTooLarge, status: http.StatusRequestEntityTooLarge},
		{kind: myerrors.KindRateLimited, status: http.StatusTooManyRequests},
		{kind: myerrors.KindDependencyFailed, status: http.StatusBadGateway},
		{kind: myerrors.KindUnavailable, status: http.StatusServiceUnavailable},
		{kind: myerrors.KindTimeout, status: http.StatusGatewayTimeout},
		{kind: myerrors.KindInternal, status: http.StatusInternalServerError},
	}
	for _, test := range tests {
		if got := httpStatusForErrorKind(test.kind); got != test.status {
			t.Fatalf("kind %q mapped to %d, want %d", test.kind, got, test.status)
		}
	}
}

func TestInternalBusinessErrorIsLoggedAsServerFailure(t *testing.T) {
	err := myerrors.WrapApplicationError(nil, myerrors.KindInternal, myerrors.CategoryBusiness, 81002, "处理失败")
	if !shouldLogApplicationError(err, true) {
		t.Fatal("internal application failure must be logged even when its domain category is business")
	}
}

func performResponseRequest(t *testing.T, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()

	restoreLogger := zap.ReplaceGlobals(zap.NewNop())
	t.Cleanup(restoreLogger)

	engine := gin.New()
	engine.Use(ResponseHandler())
	engine.GET("/baseline", handler)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/baseline", nil)
	engine.ServeHTTP(recorder, request)
	return recorder
}

func assertAdminErrorResponse(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	statusCode int,
	errorCode int,
	errorMessage string,
) {
	t.Helper()

	if recorder.Code != statusCode {
		t.Fatalf("expected status %d, got %d: %s", statusCode, recorder.Code, recorder.Body.String())
	}
	var payload response.AdminError
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode admin error: %v", err)
	}
	if payload.Success ||
		payload.StatusCode != statusCode ||
		payload.ErrorCode != errorCode ||
		payload.ErrorMessage != errorMessage {
		t.Fatalf("unexpected admin error: %#v", payload)
	}
}
