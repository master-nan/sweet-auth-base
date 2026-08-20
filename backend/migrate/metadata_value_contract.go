package main

import (
	"backend/enum"
	"fmt"

	"gorm.io/gorm"
)

const (
	removedApproximateNumericType = 2
	removedNarrowIntegerType      = 9
)

func migrateMetadataValueContract(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`UPDATE sys_table_field
			SET numeric_precision = CASE
					WHEN numeric_precision > 0 THEN numeric_precision
					WHEN field_length > 0 THEN field_length
					ELSE 38
				END,
				numeric_scale = CASE
					WHEN (CASE WHEN numeric_scale > 0 THEN numeric_scale WHEN field_decimal_length > 0 THEN field_decimal_length ELSE 18 END)
						> (CASE WHEN numeric_precision > 0 THEN numeric_precision WHEN field_length > 0 THEN field_length ELSE 38 END)
					THEN (CASE WHEN numeric_precision > 0 THEN numeric_precision WHEN field_length > 0 THEN field_length ELSE 38 END)
					ELSE (CASE WHEN numeric_scale > 0 THEN numeric_scale WHEN field_decimal_length > 0 THEN field_decimal_length ELSE 18 END)
				END
			WHERE field_type = ?`, removedApproximateNumericType).Error; err != nil {
			return err
		}
		if err := tx.Exec(`UPDATE sys_table_field SET field_type = ? WHERE field_type = ?`,
			enum.DecimalFieldType, removedApproximateNumericType).Error; err != nil {
			return err
		}
		if err := tx.Exec(`UPDATE sys_table_field SET field_type = ? WHERE field_type = ?`,
			enum.SmallIntFieldType, removedNarrowIntegerType).Error; err != nil {
			return err
		}
		if err := tx.Exec(`DELETE FROM sys_dict_item WHERE item_code IN (?, ?)`,
			"sys_table_field_type_float", "sys_table_field_type_tinyint").Error; err != nil {
			return err
		}
		if tx.Dialector.Name() != "postgres" {
			return nil
		}
		statements := []string{
			`ALTER TABLE sys_table_field DROP CONSTRAINT IF EXISTS chk_sys_table_field_storage_type`,
			fmt.Sprintf(`ALTER TABLE sys_table_field ADD CONSTRAINT chk_sys_table_field_storage_type CHECK
				(field_type IN (%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d))`,
				enum.BigIntFieldType, enum.VarcharFieldType, enum.TextFieldType, enum.BooleanFieldType,
				enum.DateFieldType, enum.DatetimeFieldType, enum.TimeFieldType, enum.JsonFieldType,
				enum.IntFieldType, enum.SmallIntFieldType, enum.DecimalFieldType),
			`ALTER TABLE sys_table_field DROP CONSTRAINT IF EXISTS chk_sys_table_field_smallint_default`,
			fmt.Sprintf(`ALTER TABLE sys_table_field ADD CONSTRAINT chk_sys_table_field_smallint_default CHECK
				(field_type <> %d OR default_value IS NULL OR default_value::numeric BETWEEN -32768 AND 32767)`, enum.SmallIntFieldType),
			`ALTER TABLE sys_table_field DROP CONSTRAINT IF EXISTS chk_sys_table_field_numeric_shape`,
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
