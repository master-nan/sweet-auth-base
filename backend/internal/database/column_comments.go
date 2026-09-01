package database

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type metadataColumnComment struct {
	TableCode string `gorm:"column:table_code"`
	FieldCode string `gorm:"column:field_code"`
	FieldName string `gorm:"column:field_name"`
}

type physicalColumnComment struct {
	TableName string         `gorm:"column:table_name"`
	FieldCode string         `gorm:"column:field_code"`
	Comment   sql.NullString `gorm:"column:comment"`
}

// SetColumnComment updates one PostgreSQL column comment with safely quoted identifiers.
func SetColumnComment(db *gorm.DB, tableCode, fieldCode, comment string) error {
	if db == nil || db.Dialector.Name() != "postgres" {
		return nil
	}
	tableName := db.NamingStrategy.TableName(strings.TrimSpace(tableCode))
	statement := fmt.Sprintf(
		"COMMENT ON COLUMN %s.%s IS %s",
		pq.QuoteIdentifier(tableName),
		pq.QuoteIdentifier(strings.TrimSpace(fieldCode)),
		pq.QuoteLiteral(strings.TrimSpace(comment)),
	)
	return db.Exec(statement).Error
}

// SyncMetadataColumnComments makes physical comments match the field names used by the UI.
// An empty tableCode synchronizes every registered physical table in the current schema.
func SyncMetadataColumnComments(db *gorm.DB, tableCode string) (int, error) {
	if db == nil || db.Dialector.Name() != "postgres" {
		return 0, nil
	}

	tableTable := pq.QuoteIdentifier(db.NamingStrategy.TableName("SysTable"))
	fieldTable := pq.QuoteIdentifier(db.NamingStrategy.TableName("SysTableField"))
	query := fmt.Sprintf(`
SELECT t.table_code, f.field_code, btrim(f.field_name) AS field_name
FROM %s t
JOIN %s f ON f.table_id = t.id
WHERE t.gmt_delete IS NULL
  AND f.gmt_delete IS NULL
  AND btrim(f.field_name) <> ''`, tableTable, fieldTable)
	args := make([]any, 0, 1)
	if tableCode = strings.TrimSpace(tableCode); tableCode != "" {
		query += " AND t.table_code = ?"
		args = append(args, tableCode)
	}

	var metadataRows []metadataColumnComment
	if err := db.Raw(query, args...).Scan(&metadataRows).Error; err != nil {
		return 0, fmt.Errorf("query metadata column comments: %w", err)
	}
	var physicalRows []physicalColumnComment
	if err := db.Raw(`
SELECT c.relname AS table_name,
       a.attname AS field_code,
       col_description(c.oid, a.attnum) AS comment
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
JOIN pg_attribute a ON a.attrelid = c.oid
WHERE n.nspname = current_schema()
  AND c.relkind IN ('r', 'p', 'v', 'm')
  AND a.attnum > 0
  AND NOT a.attisdropped`).Scan(&physicalRows).Error; err != nil {
		return 0, fmt.Errorf("query physical column comments: %w", err)
	}

	physicalComments := make(map[string]sql.NullString, len(physicalRows))
	for _, row := range physicalRows {
		physicalComments[row.TableName+"\x00"+row.FieldCode] = row.Comment
	}

	updated := 0
	for _, row := range metadataRows {
		physicalName := db.NamingStrategy.TableName(row.TableCode)
		current, exists := physicalComments[physicalName+"\x00"+row.FieldCode]
		if !exists || current.Valid && current.String == row.FieldName {
			continue
		}
		if err := SetColumnComment(db, row.TableCode, row.FieldCode, row.FieldName); err != nil {
			return updated, fmt.Errorf("comment %s.%s: %w", row.TableCode, row.FieldCode, err)
		}
		updated++
	}
	return updated, nil
}
