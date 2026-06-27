/**
 * @Author: Nan
 * @Date: 2024/6/7 下午10:54
 */

package initialize

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	entrans "github.com/go-playground/validator/v10/translations/en"
	zhtrans "github.com/go-playground/validator/v10/translations/zh"
	"gorm.io/datatypes"
)

func InitValidators() (map[string]ut.Translator, error) {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		translators := make(map[string]ut.Translator)
		// 注册自定义字段名称提取函数
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})
		// 注册自定义验证标签
		v.RegisterValidation("non_empty_json", validateNonEmptyJSON)
		// 创建翻译器
		zhT := zh.New()
		enT := en.New()
		uni := ut.New(enT, zhT)

		// 获取翻译器
		zhTrans, _ := uni.GetTranslator("zh")
		enTrans, _ := uni.GetTranslator("en")

		// 注册翻译
		_ = zhtrans.RegisterDefaultTranslations(v, zhTrans)
		_ = entrans.RegisterDefaultTranslations(v, enTrans)

		// 注册自定义验证标签的翻译
		registerCustomValidationTranslation(v, zhTrans, "non_empty_json", "{0}不能为空")
		registerCustomValidationTranslation(v, enTrans, "non_empty_json", "{0} cannot be empty")

		translators["zh"] = zhTrans
		translators["en"] = enTrans
		return translators, nil
	}
	return map[string]ut.Translator{}, nil
}

// 自定义验证标签函数
func validateNonEmptyJSON(fl validator.FieldLevel) bool {
	field := fl.Field().Interface().(datatypes.JSON)
	// 检查是否为nil
	if field == nil {
		return false
	}
	// 检查是否为空数组[]或空对象{}
	var js interface{}
	if err := json.Unmarshal(field, &js); err != nil {
		return false
	}
	// 检查数组类型
	if arr, ok := js.([]interface{}); ok {
		return len(arr) > 0
	}
	// 检查对象类型
	if obj, ok := js.(map[string]interface{}); ok {
		return len(obj) > 0
	}
	return false
}

// registerCustomValidationTranslation 注册自定义验证标签的翻译
func registerCustomValidationTranslation(v *validator.Validate, trans ut.Translator, tag string, message string) error {
	return v.RegisterTranslation(tag, trans,
		func(ut ut.Translator) error {
			return ut.Add(tag, message, true)
		},
		func(ut ut.Translator, fe validator.FieldError) string {
			t, _ := ut.T(tag, fe.Field())
			return t
		},
	)
}
