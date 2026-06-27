package utils

import (
	"backend/enum"
	myerrors "backend/internal/errors"
	"backend/model"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"regexp"
	"sort"
	"strings"
	"time"

	"errors"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin/binding"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"

	"reflect"

	"github.com/gin-gonic/gin"
)

func GetStructSqlSelect(data interface{}) []string {
	var arr []string
	refData := reflect.ValueOf(data)
	for i := 0; i < refData.NumField(); i++ {
		tag := refData.Type().Field(i).Tag
		value := refData.Field(i).Interface() //获取属性的值
		refValue := reflect.ValueOf(value)    //获取值的类型
		if refValue.Kind() == reflect.Ptr {   //是否为指针  判断是否为nil
			if !refValue.IsNil() {
				arr = append(arr, tag.Get("json"))
			}
		} else {
			if value != "" {
				arr = append(arr, tag.Get("json"))
			}
		}
	}
	return arr
}

func IsStructEmpty(source interface{}, target interface{}) bool {
	return reflect.DeepEqual(source, target)
}

func SaveSession(ctx *gin.Context, key string, value interface{}) {
	session := sessions.Default(ctx)
	option := sessions.Options{Path: "/", MaxAge: 3600}
	session.Options(option)
	session.Set(key, value)
	session.Save()
}

func DeleteSession(ctx *gin.Context, key string) {
	session := sessions.Default(ctx)
	session.Delete(key)
	session.Save()
}

func GetSessionString(ctx *gin.Context, key string) string {
	session := sessions.Default(ctx)
	value := session.Get(key)
	if value == nil {
		return ""
	}
	return value.(string)
}

func ClearSession(ctx *gin.Context) {
	session := sessions.Default(ctx)
	session.Clear()
}

// RandString 生成随机大写字符串
func RandString(len int) string {
	bytes := make([]byte, len)
	for i := 0; i < len; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(26))
		if err != nil {
			panic(err)
		}
		bytes[i] = byte(num.Int64() + 65)
	}
	return string(bytes)
}

func Encryption(password string, salt string) string {
	str := fmt.Sprintf("%s%s", password, salt)
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

func IsEmpty(s interface{}) bool {
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return true
	}
	return isZero(v)
}

func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Func, reflect.Map, reflect.Slice, reflect.Interface, reflect.Ptr, reflect.Chan:
		return v.IsNil()
	case reflect.Array, reflect.String:
		return v.Len() == 0
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if !isZero(v.Field(i)) {
				return false
			}
		}
		return true
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Complex64, reflect.Complex128:
		return v.Complex() == 0
	default:
		// 未处理的类型
		return false
	}
}

func TranslateError(err validator.FieldError) string {
	// 你可以根据err.Tag()来判断是哪种验证错误，然后返回不同的错误信息
	switch err.Tag() {
	case "required":
		return fmt.Sprintf("%s 是必填项", err.Field())
	default:
		return fmt.Sprintf("%s 验证错误", err.Field())
	}
}

// BoolPtr 辅助函数用于创建各种类型的指针
func BoolPtr(b bool) *bool {
	return &b
}

// IntPtr 辅助函数用于创建各种类型的指针
func IntPtr(i int) *int {
	return &i
}

// StringPtr 辅助函数用于创建各种类型的指针
func StringPtr(s string) *string {
	return &s
}

func StringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func IsSuperAdmin(user model.SysUser) bool {
	for _, role := range user.Roles {
		if role.Name == "super_admin" {
			return true
		}
	}
	return false
}

func SqlTypeFromFieldType(fieldType enum.SysTableFieldType) string {
	switch fieldType {
	case enum.BigIntFieldType:
		return "bigint"
	case enum.IntFieldType:
		return "integer"
	case enum.VarcharFieldType:
		return "varchar" // 长度将在外部指定
	case enum.DatetimeFieldType:
		return "timestamp"
	case enum.BooleanFieldType:
		return "boolean"
	case enum.TextFieldType:
		return "text"
	case enum.DateFieldType:
		return "date"
	case enum.TimeFieldType:
		return "time"
	case enum.JsonFieldType:
		return "jsonb"
	case enum.FloatFieldType:
		return "numeric"
	case enum.TinyintFieldType:
		return "smallint"
	default:
		return "text"
	}
}

// UpdateAccessTokens 替换当前token字符串
func UpdateAccessTokens(existingTokens string, newToken string) string {
	// 分隔符
	delimiter := ","
	var tokens []string
	// 检查是否有现有的Tokens，防止创建一个包含空字符串的slice
	if existingTokens != "" {
		tokens = strings.Split(existingTokens, delimiter)
	}

	// 确保只保留最近的4个token（因为我们将添加一个新的）
	if len(tokens) >= 5 {
		tokens = tokens[1:] // 删除最老的Token
	}
	// 添加新的Token
	tokens = append(tokens, newToken)
	// 将更新后的Token列表连接成一个新的字符串
	updatedTokens := strings.Join(tokens, delimiter)
	return updatedTokens
}

// ContainsToken 查找token是否在当前token集合内
func ContainsToken(existingTokens string, newToken string) bool {
	// 如果现有的tokens字符串为空，直接返回false
	if existingTokens == "" {
		return false
	}
	// 分隔符
	delimiter := ","
	// 分割现有tokens
	tokens := strings.Split(existingTokens, delimiter)

	// 检查newToken是否在tokens切片中
	for _, token := range tokens {
		if token == newToken {
			return true // 找到了，返回true
		}
	}
	return false // 没有找到，返回false
}

func ValidatorBody[T any](ctx *gin.Context, data *T, translator ut.Translator) error {
	err := ctx.ShouldBindBodyWith(data, binding.JSON)
	if err != nil {
		if err == io.EOF {
			// 客户端请求体为空
			return myerrors.ErrParamInvalid
		}
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			// 如果是验证错误，则翻译错误信息
			var errorMessages []string
			for _, e := range ve {
				errMsg := e.Translate(translator)
				errorMessages = append(errorMessages, errMsg)
			}
			return myerrors.NewBadRequestError(strings.Join(errorMessages, ","))
		}
		return err
	}
	SanitizeData(data)
	return nil
}

func ValidatorQuery[T any](ctx *gin.Context, data *T, translator ut.Translator) error {
	err := ctx.ShouldBindQuery(data)
	if err != nil {
		if err == io.EOF {
			// 客户端请求体为空
			return myerrors.ErrParamInvalid
		}
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			// 如果是验证错误，则翻译错误信息
			var errorMessages []string
			for _, e := range ve {
				errMsg := e.Translate(translator)
				errorMessages = append(errorMessages, errMsg)
			}

			return myerrors.NewBadRequestError(strings.Join(errorMessages, ","))
		}
		return err
	}
	SanitizeData(data)
	return nil
}

// RandInt64 生成随机100以内随机数
func RandInt64() int64 {
	num, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
		panic(err)
	}
	return num.Int64()
}

func ToInterfaceSlice(slice interface{}) []interface{} {
	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice {
		panic("ToInterfaceSlice: not a slice")
	}
	interfaceSlice := make([]interface{}, v.Len())
	for i := 0; i < v.Len(); i++ {
		interfaceSlice[i] = v.Index(i).Interface()
	}
	return interfaceSlice
}

// BuildMenuTree 递归构建树形结构
func BuildMenuTree(menus []model.SysMenu, pid int) []model.SysMenu {
	var tree []model.SysMenu
	for _, menu := range menus {
		if menu.Pid == pid {
			menu.Children = BuildMenuTree(menus, menu.Id)
			tree = append(tree, menu)
		}
	}
	return tree
}

func SortMenuTree(menus []model.SysMenu) []model.SysMenu {
	// 递归排序所有子菜单
	for i := range menus {
		if len(menus[i].Children) > 0 {
			menus[i].Children = SortMenuTree(menus[i].Children)
		}
	}
	// 排序当前层级
	sort.Slice(menus, func(i, j int) bool {
		return menus[i].Sequence < menus[j].Sequence
	})
	return menus
}

func SanitizeData(data any) {
	val := reflect.ValueOf(data)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return
	}
	val = val.Elem()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		switch field.Kind() {
		case reflect.String:
			escapedStr := SanitizeInput(field.String())
			field.SetString(escapedStr)
		case reflect.Struct:
			SanitizeData(field.Addr().Interface())
		case reflect.Slice:
			if field.Type().Elem().Kind() == reflect.String {
				for j := 0; j < field.Len(); j++ {
					escapedStr := SanitizeInput(field.Index(j).String())
					field.Index(j).SetString(escapedStr)
				}
			} else if field.Type().Elem().Kind() == reflect.Struct {
				for j := 0; j < field.Len(); j++ {
					element := field.Index(j).Addr().Interface()
					SanitizeData(element)
				}
			}
		case reflect.Map:
			if field.Type().Key().Kind() == reflect.String && field.Type().Elem().Kind() == reflect.String {
				iter := field.MapRange()
				for iter.Next() {
					key := iter.Key()
					val := iter.Value()
					escapedVal := SanitizeInput(val.String())
					field.SetMapIndex(key, reflect.ValueOf(escapedVal))
				}
			}
		}
	}
}

func SanitizeInput(input string) string {
	replacements := map[string]string{
		"\n": "\\n",
		"\r": "\\r",
		"\t": "\\t",
	}
	for old, new := range replacements {
		input = strings.ReplaceAll(input, old, new)
	}
	// 移除潜在的 XSS 脚本标签，但不对整体做 HTML 编码
	// 避免将 JSON 字段中的 " 转为 &#34;
	input = stripDangerousTags(input)
	return input
}

// stripDangerousTags 移除可能的 XSS 危险标签，如 <script>、<iframe> 等
func stripDangerousTags(input string) string {
	// 移除 <script>...</script> 及其内容
	reScript := regexp.MustCompile(`(?i)<script[^>]*>[\s\S]*?</script>`)
	input = reScript.ReplaceAllString(input, "")
	// 移除 <iframe>、<object>、<embed>、<link>、<meta> 等内联标签
	reDangerous := regexp.MustCompile(`(?i)<\s*/?\s*(script|iframe|object|embed|link|meta|form|base)[^>]*>`)
	input = reDangerous.ReplaceAllString(input, "")
	// 移除 javascript: 协议
	reJS := regexp.MustCompile(`(?i)javascript\s*:`)
	input = reJS.ReplaceAllString(input, "")
	// 移除事件属性 on*=
	reEvent := regexp.MustCompile(`(?i)\s+on\w+\s*=`)
	input = reEvent.ReplaceAllString(input, " ")
	return input
}

func CustomTimeDecodeHook(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
	if from.Kind() == reflect.String && to == reflect.TypeOf(model.CustomTime{}) {
		parsedTime, err := time.Parse(time.RFC3339, data.(string))
		if err != nil {
			return nil, err
		}
		return model.CustomTime(parsedTime), nil
	}
	return data, nil
}

func EscapeString(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

// GenerateSecretKey 生成指定长度的随机密钥
func GenerateSecretKey(length int) (string, error) {
	key := make([]byte, length)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(key), nil
}

// GenerateRandomNumber 生成 N 位随机数字验证码
func GenerateRandomNumber(n int) string {
	bytes := make([]byte, n)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	var sb strings.Builder
	for _, b := range bytes {
		sb.WriteByte('0' + (b % 10))
	}
	return sb.String()
}

// IsMobile 正则验证手机号
func IsMobile(mobile string) bool {
	// 正则验证手机号
	mobilePattern := `^1[3-9]\d{9}$`
	return regexp.MustCompile(mobilePattern).MatchString(mobile)
}
