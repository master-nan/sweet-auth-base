package queryscheme

import (
	"backend/dto/request"
	"backend/model"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// BindingKind 是后端允许持久化的动态值种类，不接受任意客户端扩展字符串。
type BindingKind string

const (
	BindingToday           BindingKind = "TODAY"
	BindingStartOfWeek     BindingKind = "START_OF_WEEK"
	BindingEndOfWeek       BindingKind = "END_OF_WEEK"
	BindingStartOfMonth    BindingKind = "START_OF_MONTH"
	BindingEndOfMonth      BindingKind = "END_OF_MONTH"
	BindingCurrentUser     BindingKind = "CURRENT_USER"
	BindingCurrentEmployee BindingKind = "CURRENT_EMPLOYEE"
)

var allBindingKinds = []BindingKind{
	BindingToday, BindingStartOfWeek, BindingEndOfWeek, BindingStartOfMonth,
	BindingEndOfMonth, BindingCurrentUser, BindingCurrentEmployee,
}

func BindingKinds() []BindingKind {
	return append([]BindingKind(nil), allBindingKinds...)
}

func (kind BindingKind) Valid() bool {
	for _, allowed := range allBindingKinds {
		if kind == allowed {
			return true
		}
	}
	return false
}

// Clock 使日期Binding可使用确定时间进行测试，同时避免业务代码直接读取全局时钟。
type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time { return model.Now() }

func SystemClock() Clock { return systemClock{} }

// BindingResolver 在Resolve阶段复制表达式并替换Binding目标值，
// 不查询Employee Repository，也不会修改已保存的Scheme Payload。
type BindingResolver struct {
	clock    Clock
	location *time.Location
}

func NewBindingResolver(clock Clock, location *time.Location) *BindingResolver {
	if clock == nil {
		clock = SystemClock()
	}
	if location == nil {
		location = model.AppLocation()
	}
	return &BindingResolver{clock: clock, location: location}
}

// Resolve 只处理Scope允许的Binding，并在表达式副本上按JSON Pointer写值；
// CURRENT_EMPLOYEE仅使用Subject中的服务端解析结果，未绑定时失败关闭。
func (resolver *BindingResolver) Resolve(
	_ context.Context,
	payload QuerySchemePayloadV1,
	config ScopeConfig,
	subject Subject,
) (ResolvedQuery, error) {
	if resolver == nil || resolver.clock == nil || resolver.location == nil {
		return ResolvedQuery{}, fmt.Errorf("query scheme binding resolver is unavailable")
	}
	raw, err := json.Marshal(map[string]any{"expressions": payload.Expressions})
	if err != nil {
		return ResolvedQuery{}, err
	}
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return ResolvedQuery{}, err
	}
	for _, binding := range payload.Bindings {
		if !config.AllowsBinding(binding.Kind) {
			return ResolvedQuery{}, fmt.Errorf("binding kind is not allowed")
		}
		value, err := resolver.resolveValue(binding, subject)
		if err != nil {
			return ResolvedQuery{}, err
		}
		if err := setJSONPointer(root, binding.Pointer, value); err != nil {
			return ResolvedQuery{}, err
		}
	}
	resolvedRaw, err := json.Marshal(root["expressions"])
	if err != nil {
		return ResolvedQuery{}, err
	}
	var resolved []request.ExpressionGroup
	if err := json.Unmarshal(resolvedRaw, &resolved); err != nil {
		return ResolvedQuery{}, err
	}
	return ResolvedQuery{
		Expressions: resolved,
		QuickQuery:  payload.QuickQuery,
		Order:       payload.Order,
	}, nil
}

func (resolver *BindingResolver) resolveValue(binding Binding, subject Subject) (any, error) {
	if err := validateBindingParams(binding); err != nil {
		return nil, err
	}
	now := resolver.clock.Now().In(resolver.location)
	switch binding.Kind {
	case BindingToday:
		return now.AddDate(0, 0, intValue(binding.Params.DayOffset)).Format(time.DateOnly), nil
	case BindingStartOfWeek, BindingEndOfWeek:
		date := now.AddDate(0, 0, intValue(binding.Params.WeekOffset)*7)
		weekday := (int(date.Weekday()) + 6) % 7
		start := date.AddDate(0, 0, -weekday)
		if binding.Kind == BindingEndOfWeek {
			start = start.AddDate(0, 0, 6)
		}
		return start.Format(time.DateOnly), nil
	case BindingStartOfMonth, BindingEndOfMonth:
		date := now.AddDate(0, intValue(binding.Params.MonthOffset), 0)
		start := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, resolver.location)
		if binding.Kind == BindingEndOfMonth {
			start = start.AddDate(0, 1, -1)
		}
		return start.Format(time.DateOnly), nil
	case BindingCurrentUser:
		if subject.UserID <= 0 {
			return nil, fmt.Errorf("current user is unavailable")
		}
		return subject.UserID, nil
	case BindingCurrentEmployee:
		if subject.EmployeeID == nil || *subject.EmployeeID <= 0 {
			return nil, fmt.Errorf("current employee is unavailable")
		}
		return *subject.EmployeeID, nil
	default:
		return nil, fmt.Errorf("unsupported binding kind")
	}
}

func validateBindingParams(binding Binding) error {
	if !binding.Kind.Valid() {
		return fmt.Errorf("unsupported binding kind")
	}
	params := binding.Params
	switch binding.Kind {
	case BindingToday:
		if params.WeekOffset != nil || params.MonthOffset != nil || !bounded(params.DayOffset, -366, 366) {
			return fmt.Errorf("invalid day offset")
		}
	case BindingStartOfWeek, BindingEndOfWeek:
		if params.DayOffset != nil || params.MonthOffset != nil || !bounded(params.WeekOffset, -52, 52) {
			return fmt.Errorf("invalid week offset")
		}
	case BindingStartOfMonth, BindingEndOfMonth:
		if params.DayOffset != nil || params.WeekOffset != nil || !bounded(params.MonthOffset, -120, 120) {
			return fmt.Errorf("invalid month offset")
		}
	case BindingCurrentUser, BindingCurrentEmployee:
		if params.DayOffset != nil || params.WeekOffset != nil || params.MonthOffset != nil {
			return fmt.Errorf("identity binding does not accept parameters")
		}
	}
	return nil
}

func bounded(value *int, min, max int) bool {
	return value == nil || (*value >= min && *value <= max)
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func setJSONPointer(root any, pointer string, value any) error {
	parts := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
	if pointer == "" || !strings.HasPrefix(pointer, "/expressions/") || len(parts) < 5 {
		return fmt.Errorf("invalid binding pointer")
	}
	current := root
	for index, raw := range parts {
		part := strings.ReplaceAll(strings.ReplaceAll(raw, "~1", "/"), "~0", "~")
		last := index == len(parts)-1
		switch node := current.(type) {
		case map[string]any:
			if last {
				if _, exists := node[part]; !exists {
					return fmt.Errorf("binding pointer target does not exist")
				}
				node[part] = value
				return nil
			}
			next, exists := node[part]
			if !exists {
				return fmt.Errorf("binding pointer target does not exist")
			}
			current = next
		case []any:
			position, err := strconv.Atoi(part)
			if err != nil || position < 0 || position >= len(node) {
				return fmt.Errorf("binding pointer index is invalid")
			}
			if last {
				node[position] = value
				return nil
			}
			current = node[position]
		default:
			return fmt.Errorf("binding pointer traverses a scalar")
		}
	}
	return fmt.Errorf("binding pointer is incomplete")
}
