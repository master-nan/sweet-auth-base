package utils

import (
	myerrors "backend/internal/errors"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	entrans "github.com/go-playground/validator/v10/translations/en"
	zhtrans "github.com/go-playground/validator/v10/translations/zh"
	"gorm.io/datatypes"
)

const (
	DefaultPageSize = 10
	MaxPageSize     = 5000
)

// ValidationIssue 是单个 DTO 字段错误的稳定内部表示。
type ValidationIssue struct {
	Field     string
	Rule      string
	Parameter string
	Message   string
}

// InitializeValidator 配置 Gin 使用的统一校验引擎。
func InitializeValidator(validate *validator.Validate) (map[string]ut.Translator, error) {
	if validate == nil {
		return nil, fmt.Errorf("validator engine is nil")
	}

	validate.SetTagName("binding")
	validate.RegisterTagNameFunc(validationFieldName)
	if err := validate.RegisterValidation("non_empty_json", validateNonEmptyJSON); err != nil {
		return nil, fmt.Errorf("register non_empty_json validation: %w", err)
	}

	zhLocale := zh.New()
	enLocale := en.New()
	universalTranslator := ut.New(enLocale, enLocale, zhLocale)

	zhTranslator, ok := universalTranslator.GetTranslator("zh")
	if !ok {
		return nil, fmt.Errorf("zh validator translator is unavailable")
	}
	enTranslator, ok := universalTranslator.GetTranslator("en")
	if !ok {
		return nil, fmt.Errorf("en validator translator is unavailable")
	}

	if err := zhtrans.RegisterDefaultTranslations(validate, zhTranslator); err != nil {
		return nil, fmt.Errorf("register zh validator translations: %w", err)
	}
	if err := entrans.RegisterDefaultTranslations(validate, enTranslator); err != nil {
		return nil, fmt.Errorf("register en validator translations: %w", err)
	}
	if err := registerValidationTranslation(validate, zhTranslator, "non_empty_json", "{0}不能为空"); err != nil {
		return nil, fmt.Errorf("register zh non_empty_json translation: %w", err)
	}
	if err := registerValidationTranslation(validate, enTranslator, "non_empty_json", "{0} cannot be empty"); err != nil {
		return nil, fmt.Errorf("register en non_empty_json translation: %w", err)
	}

	return map[string]ut.Translator{
		"zh": zhTranslator,
		"en": enTranslator,
	}, nil
}

// ValidationIssues 转换校验错误，不改变公共 API 响应结构。
func ValidationIssues(err error, translator ut.Translator) []ValidationIssue {
	var validationErrors validator.ValidationErrors
	if !stderrors.As(err, &validationErrors) {
		return nil
	}

	issues := make([]ValidationIssue, 0, len(validationErrors))
	for _, fieldErr := range validationErrors {
		message := ""
		if translator != nil {
			message = fieldErr.Translate(translator)
		}
		if strings.TrimSpace(message) == "" {
			message = fmt.Sprintf("%s校验失败(%s)", fieldErr.Field(), fieldErr.Tag())
		}
		issues = append(issues, ValidationIssue{
			Field:     fieldErr.Field(),
			Rule:      fieldErr.Tag(),
			Parameter: fieldErr.Param(),
			Message:   message,
		})
	}
	return issues
}

// ValidationErrorMessage 生成现有的逗号分隔客户端消息。
func ValidationErrorMessage(err error, translator ut.Translator) string {
	issues := ValidationIssues(err, translator)
	if len(issues) == 0 {
		return "参数错误"
	}

	messages := make([]string, 0, len(issues))
	for _, issue := range issues {
		messages = append(messages, issue.Message)
	}
	return strings.Join(messages, ",")
}

// ToValidationParameterError 保留校验原因，并使用平台参数错误。
func ToValidationParameterError(err error, translator ut.Translator) error {
	if err == nil {
		return nil
	}
	return myerrors.WrapParameterError(err, ValidationErrorMessage(err, translator))
}

// ValidateStruct 使用与 HTTP 绑定相同的引擎和响应语义校验 DTO。
func ValidateStruct(data any, translator ut.Translator) error {
	if err := binding.Validator.ValidateStruct(data); err != nil {
		return ToValidationParameterError(err, translator)
	}
	return nil
}

// ValidateEnum 校验无法通过静态 oneof 标签表达的枚举类值。
func ValidateEnum[T comparable](field string, value T, allowed ...T) error {
	for _, candidate := range allowed {
		if value == candidate {
			return nil
		}
	}
	field = strings.TrimSpace(field)
	if field == "" {
		field = "枚举字段"
	}
	return myerrors.NewParameterError(field + "取值不合法")
}

// ValidatePagination 校验显式分页值，并保留零值表示“使用默认值”的语义。
func ValidatePagination(page, pageSize int) error {
	if page < 0 {
		return myerrors.NewParameterError("page不能小于0")
	}
	if pageSize < 0 {
		return myerrors.NewParameterError("num不能小于0")
	}
	if pageSize > MaxPageSize {
		return myerrors.NewParameterError(fmt.Sprintf("num不能大于%d", MaxPageSize))
	}
	return nil
}

// ValidateDateRange 允许省略边界，并拒绝起止顺序颠倒的完整范围。
func ValidateDateRange(start, end time.Time) error {
	if start.IsZero() || end.IsZero() {
		return nil
	}
	if start.After(end) {
		return myerrors.NewParameterError("开始时间不能晚于结束时间")
	}
	return nil
}

func validationFieldName(field reflect.StructField) string {
	for _, tagName := range []string{"json", "form"} {
		name := strings.SplitN(field.Tag.Get(tagName), ",", 2)[0]
		if name != "" && name != "-" {
			return name
		}
	}
	return field.Name
}

func validateNonEmptyJSON(fieldLevel validator.FieldLevel) bool {
	field := fieldLevel.Field()
	if !field.IsValid() {
		return false
	}
	value, ok := field.Interface().(datatypes.JSON)
	if !ok || len(value) == 0 {
		return false
	}

	var decoded any
	if err := json.Unmarshal(value, &decoded); err != nil {
		return false
	}
	switch item := decoded.(type) {
	case []any:
		return len(item) > 0
	case map[string]any:
		return len(item) > 0
	default:
		return false
	}
}

func registerValidationTranslation(
	validate *validator.Validate,
	translator ut.Translator,
	tag string,
	message string,
) error {
	return validate.RegisterTranslation(
		tag,
		translator,
		func(current ut.Translator) error {
			return current.Add(tag, message, true)
		},
		func(current ut.Translator, fieldErr validator.FieldError) string {
			translated, _ := current.T(tag, fieldErr.Field())
			return translated
		},
	)
}
