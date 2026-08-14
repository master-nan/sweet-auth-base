package utils

import (
	"backend/enum"
	"backend/model"
	"fmt"
	"math"
	"strconv"
	"strings"
)

func HasTableField(table model.SysTable, fieldCode string) bool {
	for _, field := range table.TableFields {
		if field.FieldCode == fieldCode {
			return true
		}
	}
	return false
}

func IntFromAny(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int8:
		return int(v), true
	case int16:
		return int(v), true
	case int32:
		return int(v), true
	case int64:
		return int64ToInt(v)
	case uint:
		return uint64ToInt(uint64(v))
	case uint8:
		return int(v), true
	case uint16:
		return int(v), true
	case uint32:
		return uint64ToInt(uint64(v))
	case uint64:
		return uint64ToInt(v)
	case float32:
		return integralFloatToInt(float64(v))
	case float64:
		return integralFloatToInt(v)
	case string:
		id, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, false
		}
		return id, true
	case fmt.Stringer:
		id, err := strconv.Atoi(strings.TrimSpace(v.String()))
		if err != nil {
			return 0, false
		}
		return id, true
	default:
		return 0, false
	}
}

func int64ToInt(value int64) (int, bool) {
	converted := int(value)
	if int64(converted) != value {
		return 0, false
	}
	return converted, true
}

func uint64ToInt(value uint64) (int, bool) {
	converted := int(value)
	if converted < 0 || uint64(converted) != value {
		return 0, false
	}
	return converted, true
}

func integralFloatToInt(value float64) (int, bool) {
	if math.IsNaN(value) || math.IsInf(value, 0) || math.Trunc(value) != value {
		return 0, false
	}
	converted := int(value)
	if float64(converted) != value {
		return 0, false
	}
	return converted, true
}

func FlattenMenus(menus []model.SysMenu) []model.SysMenu {
	result := make([]model.SysMenu, 0, len(menus))
	var walk func(items []model.SysMenu)
	walk = func(items []model.SysMenu) {
		for _, item := range items {
			result = append(result, item)
			if len(item.Children) > 0 {
				walk(item.Children)
			}
		}
	}
	walk(menus)
	return result
}

func MenuAllowsTableCode(menu model.SysMenu, tableCode string) bool {
	tableCode = strings.TrimSpace(tableCode)
	if tableCode == "" {
		return false
	}
	return strings.TrimSpace(menu.TableCode) == tableCode
}

func MenuHasGrantedReadAction(menu model.SysMenu, action string) bool {
	for _, button := range menu.MenuButtons {
		if MenuButtonAllowsReadAction(button, menu.Id, action) {
			return true
		}
	}
	return false
}

func MenuButtonAllowsReadAction(button model.SysMenuButton, menuId int, targetAction string) bool {
	if button.MenuId != menuId || !button.State || button.IsDisabled {
		return false
	}
	action, ok := enum.NormalizeSysMenuButtonEventAction(button.EventAction)
	if !ok || !enum.IsReadMenuButtonAction(action) {
		return false
	}
	return string(action) == strings.TrimSpace(targetAction)
}
