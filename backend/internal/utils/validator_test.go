package utils

import (
	"backend/dto/response"
	stderrors "errors"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"gorm.io/datatypes"
)

type validatorBodyFixture struct {
	Name string `json:"name" binding:"required"`
}

type validatorQueryFixture struct {
	Page int `form:"page" binding:"required,min=1"`
}

type validatorRuleFixture struct {
	Required string `json:"required_value" binding:"required"`
	Min      string `json:"min_value" binding:"min=2"`
	Max      string `json:"max_value" binding:"max=2"`
	Enum     string `json:"enum_value" binding:"oneof=draft published"`
	GT       int    `json:"gt_value" binding:"gt=0"`
	GTE      int    `json:"gte_value" binding:"gte=0"`
	LT       int    `json:"lt_value" binding:"lt=5"`
	LTE      int    `json:"lte_value" binding:"lte=5"`
}

type validatorJSONFixture struct {
	Config datatypes.JSON `json:"config" binding:"required,non_empty_json"`
}

var (
	validatorTranslatorOnce sync.Once
	validatorTranslator     ut.Translator
	validatorTranslatorErr  error
)

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
	if !stderrors.As(err, &adminErr) || !strings.Contains(adminErr.ErrorMessage, "name") {
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

func TestInitializeValidatorSupportsBaselineRules(t *testing.T) {
	validate := validator.New()
	translators, err := InitializeValidator(validate)
	if err != nil {
		t.Fatalf("initialize validator: %v", err)
	}

	err = validate.Struct(validatorRuleFixture{
		Max:  "long",
		Enum: "unknown",
		GTE:  -1,
		LT:   5,
		LTE:  6,
	})
	if err == nil {
		t.Fatal("expected baseline validation rules to fail")
	}

	issues := ValidationIssues(err, translators["zh"])
	got := make(map[string]string, len(issues))
	for _, issue := range issues {
		got[issue.Field] = issue.Rule
	}
	expected := map[string]string{
		"required_value": "required",
		"min_value":      "min",
		"max_value":      "max",
		"enum_value":     "oneof",
		"gt_value":       "gt",
		"gte_value":      "gte",
		"lt_value":       "lt",
		"lte_value":      "lte",
	}
	for field, rule := range expected {
		if got[field] != rule {
			t.Fatalf("expected %s to fail %s, got %q in %#v", field, rule, got[field], got)
		}
	}
}

func TestInitializeValidatorPreservesNonEmptyJSONRule(t *testing.T) {
	validate := validator.New()
	if _, err := InitializeValidator(validate); err != nil {
		t.Fatalf("initialize validator: %v", err)
	}

	if err := validate.Struct(validatorJSONFixture{Config: datatypes.JSON(`{"enabled":true}`)}); err != nil {
		t.Fatalf("expected non-empty JSON object to be accepted: %v", err)
	}
	if err := validate.Struct(validatorJSONFixture{Config: datatypes.JSON(`[]`)}); err == nil {
		t.Fatal("expected empty JSON array to be rejected")
	}
}

func TestValidateStructUsesJSONFieldNameAndParameterResponse(t *testing.T) {
	err := ValidateStruct(
		validatorRuleFixture{
			Required: "present",
			Min:      "ok",
			Max:      "ok",
			Enum:     "invalid",
			GT:       1,
			GTE:      0,
			LT:       4,
			LTE:      5,
		},
		newValidatorTranslator(t),
	)
	assertValidatorParameterError(t, err)

	var adminErr *response.AdminError
	if !stderrors.As(err, &adminErr) || !strings.Contains(adminErr.ErrorMessage, "enum_value") {
		t.Fatalf("expected JSON field name in validation response, got %#v", adminErr)
	}
}

func TestValidateEnum(t *testing.T) {
	if err := ValidateEnum("status", "draft", "draft", "published"); err != nil {
		t.Fatalf("expected enum value to be accepted: %v", err)
	}

	err := ValidateEnum("status", "unknown", "draft", "published")
	assertValidatorParameterError(t, err)
	var adminErr *response.AdminError
	if !stderrors.As(err, &adminErr) || adminErr.ErrorMessage != "status取值不合法" {
		t.Fatalf("unexpected enum validation error: %#v", adminErr)
	}
}

func TestValidatePagination(t *testing.T) {
	validCases := []struct {
		page int
		num  int
	}{
		{page: 0, num: 0},
		{page: 1, num: DefaultPageSize},
		{page: 2, num: MaxPageSize},
	}
	for _, testCase := range validCases {
		if err := ValidatePagination(testCase.page, testCase.num); err != nil {
			t.Fatalf("expected pagination %d/%d to be accepted: %v", testCase.page, testCase.num, err)
		}
	}

	for _, testCase := range []struct {
		page int
		num  int
	}{
		{page: -1, num: 10},
		{page: 1, num: -1},
		{page: 1, num: MaxPageSize + 1},
	} {
		assertValidatorParameterError(t, ValidatePagination(testCase.page, testCase.num))
	}
}

func TestValidateDateRange(t *testing.T) {
	start := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	assertValidatorParameterError(t, ValidateDateRange(start, end))

	if err := ValidateDateRange(end, start); err != nil {
		t.Fatalf("expected ordered date range to be accepted: %v", err)
	}
	if err := ValidateDateRange(time.Time{}, start); err != nil {
		t.Fatalf("expected omitted date boundary to be accepted: %v", err)
	}
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

	validatorTranslatorOnce.Do(func() {
		validate, ok := binding.Validator.Engine().(*validator.Validate)
		if !ok {
			validatorTranslatorErr = stderrors.New("gin validator engine is unavailable")
			return
		}
		translators, err := InitializeValidator(validate)
		if err != nil {
			validatorTranslatorErr = err
			return
		}
		validatorTranslator = translators["zh"]
	})
	if validatorTranslatorErr != nil {
		t.Fatalf("initialize validator translator: %v", validatorTranslatorErr)
	}
	return validatorTranslator
}
