package main

import (
	"database/sql"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type timestampColumnContract struct {
	TableSchema   string
	TableName     string
	ColumnName    string
	ColumnDefault sql.NullString
}

type sequenceIDContract struct {
	TableSchema  string
	TableName    string
	ColumnName   string
	Identity     string
	SequenceName sql.NullString
}

// migrateCanonicalTimeAndIDContract 把既有东八区墙上时间转换成有明确时区的时刻，
// 并移除单列 id 主键的数据库自增入口。历史固定 ID 保持不变。
func migrateCanonicalTimeAndIDContract(db *gorm.DB) error {
	if db.Dialector.Name() != "postgres" {
		return nil
	}
	if err := db.Exec(`SET LOCAL TIME ZONE 'Asia/Shanghai'`).Error; err != nil {
		return fmt.Errorf("set migration timezone: %w", err)
	}
	if err := migrateTimestampColumns(db); err != nil {
		return err
	}
	return removeDatabaseGeneratedIDs(db)
}

func migrateTimestampColumns(db *gorm.DB) error {
	var columns []timestampColumnContract
	if err := db.Raw(`
SELECT c.table_schema, c.table_name, c.column_name, c.column_default
FROM information_schema.columns c
JOIN information_schema.tables t
  ON t.table_schema = c.table_schema AND t.table_name = c.table_name
WHERE c.table_schema = current_schema()
  AND t.table_type = 'BASE TABLE'
  AND c.data_type = 'timestamp without time zone'
ORDER BY c.table_name, c.ordinal_position
`).Scan(&columns).Error; err != nil {
		return fmt.Errorf("inspect timestamp columns: %w", err)
	}
	for _, column := range columns {
		tableName, err := quotePostgresQualifiedIdentifier(column.TableSchema + "." + column.TableName)
		if err != nil {
			return err
		}
		columnName, err := quotePostgresIdentifier(column.ColumnName)
		if err != nil {
			return err
		}
		if column.ColumnDefault.Valid && strings.TrimSpace(column.ColumnDefault.String) != "" {
			if err := db.Exec(fmt.Sprintf(
				"ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT",
				tableName,
				columnName,
			)).Error; err != nil {
				return fmt.Errorf("drop timestamp default %s.%s: %w", column.TableName, column.ColumnName, err)
			}
		}
		if err := db.Exec(fmt.Sprintf(
			"ALTER TABLE %s ALTER COLUMN %s TYPE timestamptz USING %s AT TIME ZONE 'Asia/Shanghai'",
			tableName,
			columnName,
			columnName,
		)).Error; err != nil {
			return fmt.Errorf("convert timestamp %s.%s: %w", column.TableName, column.ColumnName, err)
		}
		if column.ColumnDefault.Valid && strings.TrimSpace(column.ColumnDefault.String) != "" {
			if err := db.Exec(fmt.Sprintf(
				"ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s",
				tableName,
				columnName,
				column.ColumnDefault.String,
			)).Error; err != nil {
				return fmt.Errorf("restore timestamp default %s.%s: %w", column.TableName, column.ColumnName, err)
			}
		}
	}
	return nil
}

func removeDatabaseGeneratedIDs(db *gorm.DB) error {
	var columns []sequenceIDContract
	if err := db.Raw(`
SELECT c.table_schema,
       c.table_name,
       c.column_name,
       c.is_identity AS identity,
       pg_get_serial_sequence(format('%I.%I', c.table_schema, c.table_name), c.column_name) AS sequence_name
FROM information_schema.columns c
JOIN information_schema.table_constraints tc
  ON tc.table_schema = c.table_schema
 AND tc.table_name = c.table_name
 AND tc.constraint_type = 'PRIMARY KEY'
JOIN information_schema.key_column_usage kcu
  ON kcu.constraint_schema = tc.constraint_schema
 AND kcu.constraint_name = tc.constraint_name
 AND kcu.table_schema = tc.table_schema
 AND kcu.table_name = tc.table_name
 AND kcu.column_name = c.column_name
WHERE c.table_schema = current_schema()
  AND c.column_name = 'id'
  AND c.data_type IN ('bigint', 'integer', 'smallint')
  AND (c.is_identity = 'YES' OR c.column_default LIKE 'nextval(%')
  AND NOT EXISTS (
    SELECT 1
    FROM information_schema.key_column_usage other
    WHERE other.constraint_schema = tc.constraint_schema
      AND other.constraint_name = tc.constraint_name
      AND other.table_schema = tc.table_schema
      AND other.table_name = tc.table_name
      AND other.column_name <> c.column_name
  )
ORDER BY c.table_name
`).Scan(&columns).Error; err != nil {
		return fmt.Errorf("inspect generated ID columns: %w", err)
	}
	for _, column := range columns {
		tableName, err := quotePostgresQualifiedIdentifier(column.TableSchema + "." + column.TableName)
		if err != nil {
			return err
		}
		columnName, err := quotePostgresIdentifier(column.ColumnName)
		if err != nil {
			return err
		}
		if column.Identity == "YES" {
			if err := db.Exec(fmt.Sprintf(
				"ALTER TABLE %s ALTER COLUMN %s DROP IDENTITY IF EXISTS",
				tableName,
				columnName,
			)).Error; err != nil {
				return fmt.Errorf("drop ID identity %s.%s: %w", column.TableName, column.ColumnName, err)
			}
		}
		if err := db.Exec(fmt.Sprintf(
			"ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT",
			tableName,
			columnName,
		)).Error; err != nil {
			return fmt.Errorf("drop ID default %s.%s: %w", column.TableName, column.ColumnName, err)
		}
		if column.SequenceName.Valid && strings.TrimSpace(column.SequenceName.String) != "" {
			sequenceName, err := quotePostgresQualifiedIdentifier(column.SequenceName.String)
			if err != nil {
				return err
			}
			if err := db.Exec("DROP SEQUENCE IF EXISTS " + sequenceName).Error; err != nil {
				return fmt.Errorf("drop ID sequence %s: %w", column.SequenceName.String, err)
			}
		}
	}
	return nil
}
