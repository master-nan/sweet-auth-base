package utils

import (
	"backend/model"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"
	"sort"
	"strings"

	"github.com/gin-gonic/gin/binding"
	ut "github.com/go-playground/universal-translator"

	"reflect"

	"github.com/gin-gonic/gin"
)

func Encryption(password string, salt string) string {
	str := fmt.Sprintf("%s%s", password, salt)
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
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
		return ToValidationParameterError(err, translator)
	}
	SanitizeData(data)
	return nil
}

func ValidatorQuery[T any](ctx *gin.Context, data *T, translator ut.Translator) error {
	err := ctx.ShouldBindQuery(data)
	if err != nil {
		return ToValidationParameterError(err, translator)
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
