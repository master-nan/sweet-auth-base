package utils

import (
	"backend/dto/response"
	stderrors "errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zhtrans "github.com/go-playground/validator/v10/translations/zh"
)

type validatorBodyFixture struct {
	Name string `json:"name" binding:"required"`
}

type validatorQueryFixture struct {
	Page int `form:"page" binding:"required,min=1"`
}

func TestValidatorBodyReturnsParameterErrorForMalformedJSON(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{"name":`))
	ctx.Request.Header.Set("Content-Type", binding.MIMEJSON)

	var data validatorBodyFixture
	err := ValidatorBody(ctx, &data, newValidatorTranslator(t))
	assertValidatorParameterError(t, err)

	var adminErr *response.AdminError
	if !stderrors.As(err, &adminErr) || adminErr.Cause == nil {
		t.Fatalf("expected malformed JSON cause to be retained: %#v", err)
	}
	if adminErr.ErrorMessage != "参数错误" {
		t.Fatalf("malformed JSON details must not be exposed: %#v", adminErr)
	}
}

func TestValidatorBodyReturnsTranslatedParameterError(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{}`))
	ctx.Request.Header.Set("Content-Type", binding.MIMEJSON)

	var data validatorBodyFixture
	err := ValidatorBody(ctx, &data, newValidatorTranslator(t))
	assertValidatorParameterError(t, err)

	var adminErr *response.AdminError
	if !stderrors.As(err, &adminErr) || !strings.Contains(adminErr.ErrorMessage, "Name") {
		t.Fatalf("expected translated validation message, got %#v", adminErr)
	}
}

func TestValidatorQueryReturnsParameterErrorForInvalidType(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("GET", "/?page=invalid", nil)

	var data validatorQueryFixture
	err := ValidatorQuery(ctx, &data, newValidatorTranslator(t))
	assertValidatorParameterError(t, err)
}

func assertValidatorParameterError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("expected validator error")
	}
	var adminErr *response.AdminError
	if !stderrors.As(err, &adminErr) {
		t.Fatalf("expected AdminError, got %T", err)
	}
	if adminErr.Category != response.ErrorCategoryParameter ||
		adminErr.StatusCode != 400 ||
		adminErr.ErrorCode != 20003 {
		t.Fatalf("unexpected parameter error: %#v", adminErr)
	}
}

func newValidatorTranslator(t *testing.T) ut.Translator {
	t.Helper()

	validate, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		t.Fatal("gin validator engine is unavailable")
	}
	locale := zh.New()
	translator, _ := ut.New(locale, locale).GetTranslator("zh")
	if err := zhtrans.RegisterDefaultTranslations(validate, translator); err != nil {
		t.Fatalf("register validator translations: %v", err)
	}
	return translator
}
