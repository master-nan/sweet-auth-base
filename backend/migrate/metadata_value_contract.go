package main

import (
	"backend/enum"
	"backend/model"
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

const (
	historicalSmallIntFieldType = 12
	historicalDecimalFieldType  = 13
)

func migrateMetadataValueContract(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if tx.Dialector.Name() == "postgres" {
			for _, constraint := range []string{
				"chk_sys_table_field_storage_type",
				"chk_sys_table_field_smallint_default",
				"chk_sys_table_field_numeric_shape",
			} {
				if err := tx.Exec(fmt.Sprintf(`ALTER TABLE sys_table_field DROP CONSTRAINT IF EXISTS %s`, constraint)).Error; err != nil {
					return err
				}
			}
		}
		if err := tx.Exec(`UPDATE sys_table_field
			SET numeric_precision = CASE
					WHEN numeric_precision BETWEEN 1 AND 1000 THEN numeric_precision
					WHEN field_length BETWEEN 1 AND 1000 THEN field_length
					ELSE 38
				END,
				numeric_scale = CASE
					WHEN numeric_precision BETWEEN 1 AND 1000 AND numeric_scale BETWEEN 0 AND numeric_precision
						THEN numeric_scale
					WHEN field_decimal_length > 0 AND field_decimal_length <=
						(CASE WHEN field_length BETWEEN 1 AND 1000 THEN field_length ELSE 38 END)
						THEN field_decimal_length
					WHEN 18 > (CASE WHEN field_length BETWEEN 1 AND 1000 THEN field_length ELSE 38 END)
						THEN (CASE WHEN field_length BETWEEN 1 AND 1000 THEN field_length ELSE 38 END)
					ELSE 18
				END
			WHERE field_type IN (?, ?)`, enum.DecimalFieldType, historicalDecimalFieldType).Error; err != nil {
			return err
		}
		if err := tx.Exec(`UPDATE sys_table_field SET field_type = ? WHERE field_type = ?`,
			enum.DecimalFieldType, historicalDecimalFieldType).Error; err != nil {
			return err
		}
		if err := tx.Exec(`UPDATE sys_table_field SET field_type = ? WHERE field_type = ?`,
			enum.SmallIntFieldType, historicalSmallIntFieldType).Error; err != nil {
			return err
		}
		if err := migrateCanonicalFieldTypeDictionary(tx); err != nil {
			return err
		}
		if tx.Dialector.Name() != "postgres" {
			return nil
		}
		statements := []string{
			fmt.Sprintf(`ALTER TABLE sys_table_field ADD CONSTRAINT chk_sys_table_field_storage_type CHECK
				(field_type IN (%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d))`,
				enum.BigIntFieldType, enum.DecimalFieldType, enum.VarcharFieldType, enum.TextFieldType,
				enum.BooleanFieldType, enum.DateFieldType, enum.DatetimeFieldType, enum.TimeFieldType,
				enum.SmallIntFieldType, enum.JsonFieldType, enum.IntFieldType),
			fmt.Sprintf(`ALTER TABLE sys_table_field ADD CONSTRAINT chk_sys_table_field_smallint_default CHECK
				(field_type <> %d OR default_value IS NULL OR default_value::numeric BETWEEN -32768 AND 32767)`, enum.SmallIntFieldType),
			fmt.Sprintf(`ALTER TABLE sys_table_field ADD CONSTRAINT chk_sys_table_field_numeric_shape CHECK
				(field_type <> %d OR (numeric_precision BETWEEN 1 AND 1000 AND numeric_scale BETWEEN 0 AND numeric_precision))`, enum.DecimalFieldType),
			`ALTER TABLE sys_table_field DROP CONSTRAINT IF EXISTS chk_sys_table_field_list_width`,
			`ALTER TABLE sys_table_field ADD CONSTRAINT chk_sys_table_field_list_width CHECK (list_width IS NULL OR list_width > 0)`,
			`ALTER TABLE sys_table_field DROP CONSTRAINT IF EXISTS chk_sys_table_field_logical_type`,
			`ALTER TABLE sys_table_field ADD CONSTRAINT chk_sys_table_field_logical_type CHECK
				(logical_type IS NULL OR logical_type = '' OR logical_type IN ('plain','integer','decimal','money','percent','boolean','enum','date','datetime','relation'))`,
			`ALTER TABLE sys_table_field DROP CONSTRAINT IF EXISTS chk_sys_table_field_display_format`,
			`ALTER TABLE sys_table_field ADD CONSTRAINT chk_sys_table_field_display_format CHECK
				(display_format IS NULL OR display_format = '' OR display_format IN ('plain','integer','decimal','money','percent','date','datetime','dictionary','relation'))`,
		}
		for _, statement := range statements {
			if err := tx.Exec(statement).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func migrateCanonicalFieldTypeDictionary(tx *gorm.DB) error {
	var dict model.SysDict
	if err := tx.Unscoped().Where("dict_code = ?", "sys_table_field_type").First(&dict).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}
	if err := tx.Unscoped().Model(&model.SysDict{}).Where("id = ?", dict.Id).
		Updates(map[string]any{"dict_name": "字段类型", "state": true, "gmt_delete": nil}).Error; err != nil {
		return err
	}
	canonical := map[string]struct {
		name  string
		value string
	}{
		"sys_table_field_type_bigint":   {name: "大数字", value: "1"},
		"sys_table_field_type_decimal":  {name: "精确小数", value: "2"},
		"sys_table_field_type_varchar":  {name: "字符串", value: "3"},
		"sys_table_field_type_text":     {name: "文本", value: "4"},
		"sys_table_field_type_boolean":  {name: "布尔", value: "5"},
		"sys_table_field_type_date":     {name: "日期", value: "6"},
		"sys_table_field_type_datetime": {name: "日期时间", value: "7"},
		"sys_table_field_type_time":     {name: "时间", value: "8"},
		"sys_table_field_type_smallint": {name: "小整数", value: "9"},
		"sys_table_field_type_json":     {name: "JSON", value: "10"},
		"sys_table_field_type_int":      {name: "数字", value: "11"},
	}
	codes := make([]string, 0, len(canonical))
	for code, item := range canonical {
		codes = append(codes, code)
		if err := tx.Unscoped().Model(&model.SysDictItem{}).
			Where("dict_id = ? AND item_code = ?", dict.Id, code).
			Updates(map[string]any{"item_name": item.name, "item_value": item.value, "state": true, "gmt_delete": nil}).Error; err != nil {
			return err
		}
	}
	return tx.Unscoped().Where("dict_id = ? AND item_code NOT IN ?", dict.Id, codes).Delete(&model.SysDictItem{}).Error
}

func migrateCanonicalRuntimeContract(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := migrateOrganizationSelectorAliases(tx); err != nil {
			return err
		}
		return removeLegacyLowCodeMenuButtons(tx)
	})
}

var organizationSelectorCanonical = map[string]string{
	"legal_entity_select": "legal_entity",
	"org_unit_select":     "org_unit",
	"employee_select":     "employee",
	"position_select":     "position",
}

func migrateOrganizationSelectorAliases(tx *gorm.DB) error {
	if tx.Migrator().HasTable(&model.DataDimensionDefinition{}) {
		for historical, canonical := range organizationSelectorCanonical {
			if err := tx.Unscoped().Model(&model.DataDimensionDefinition{}).
				Where("selector_type = ?", historical).Update("selector_type", canonical).Error; err != nil {
				return err
			}
		}
	}

	var fields []struct {
		ID            int
		LinkageConfig string
	}
	if err := tx.Unscoped().Model(&model.SysTableField{}).
		Select("id", "linkage_config").Where("linkage_config IS NOT NULL AND linkage_config <> ''").Scan(&fields).Error; err != nil {
		return err
	}
	for _, field := range fields {
		var value any
		if json.Unmarshal([]byte(field.LinkageConfig), &value) != nil || !canonicalizeSelectorAliases(value) {
			continue
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		if err := tx.Unscoped().Model(&model.SysTableField{}).Where("id = ?", field.ID).
			Update("linkage_config", string(encoded)).Error; err != nil {
			return err
		}
	}
	return nil
}

func canonicalizeSelectorAliases(value any) bool {
	changed := false
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			if text, ok := item.(string); ok {
				if canonical, exists := organizationSelectorCanonical[strings.ToLower(strings.TrimSpace(text))]; exists {
					typed[key] = canonical
					changed = true
					continue
				}
			}
			if canonicalizeSelectorAliases(item) {
				changed = true
			}
		}
	case []any:
		for _, item := range typed {
			if canonicalizeSelectorAliases(item) {
				changed = true
			}
		}
	}
	return changed
}
