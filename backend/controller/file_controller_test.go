package controller

import (
	"backend/enum"
	myerrors "backend/internal/errors"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestContentDispositionEscapesFileName(t *testing.T) {
	header := contentDisposition("attachment", "测试 file \"name\".txt")
	if !strings.HasPrefix(header, "attachment; filename*=UTF-8''") {
		t.Fatalf("unexpected content disposition: %s", header)
	}
	encoded := strings.TrimPrefix(header, "attachment; filename*=UTF-8''")
	if strings.Contains(encoded, "\"") || strings.Contains(encoded, " ") {
		t.Fatalf("unsafe content disposition: %s", header)
	}
}

func TestDeleteBusinessActionCannotBeDowngraded(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("DELETE", "/admin/file/1?table_code=orders&record_id=2&action=detail", nil)
	_, _, err := parseFileBusinessContext(ctx, enum.ButtonActionDelete, false)
	if !errors.Is(err, myerrors.ErrParamInvalid) {
		t.Fatalf("expected action downgrade rejection, got %v", err)
	}
}

func TestDetailBusinessActionAllowsExplicitOperation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("GET", "/admin/file/1?table_code=orders&record_id=2&action=update", nil)
	business, found, err := parseFileBusinessContext(ctx, enum.ButtonActionDetail, true)
	if err != nil || !found || business.Action != enum.ButtonActionUpdate {
		t.Fatalf("unexpected business context: %+v found=%v err=%v", business, found, err)
	}
}
