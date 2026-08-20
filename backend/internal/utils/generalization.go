package utils

import (
	"backend/model"
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

func MenuAllowsTableCode(menu model.SysMenu, tableCode string) bool {
	tableCode = strings.TrimSpace(tableCode)
	if tableCode == "" {
		return false
	}
	return strings.TrimSpace(menu.TableCode) == tableCode
}
