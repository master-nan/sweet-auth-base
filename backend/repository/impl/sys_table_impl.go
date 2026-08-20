/**
 * @Author: Nan
 * @Date: 2024/6/10 上午12:16
 */

package impl

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/internal/database"
	error2 "backend/internal/errors"
	platformmetadata "backend/internal/metadata"
	"backend/model"
	"backend/repository/util"
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type SysTableRepositoryImpl struct {
	db *gorm.DB
	*BasicRepositoryImpl[model.SysTable]
}

func NewSysTableRepositoryImpl(PrimaryDB *database.PrimaryDB) *SysTableRepositoryImpl {
	return &SysTableRepositoryImpl{
		PrimaryDB.DB,
		NewBasicRepositoryImpl(PrimaryDB.DB, &model.SysTable{}),
	}
}

func (s *SysTableRepositoryImpl) GetTableById(ctx context.Context, i int) (model.SysTable, error) {
	var table model.SysTable
	err := s.db.WithContext(ctx).Preload("TableFields", func(db *gorm.DB) *gorm.DB {
		return db.Order("sequence")
	}).Preload("TableRelations").Preload("TableIndexes.IndexFields").Where("id = ?", i).First(&table).Error
	return table, err
}

func (s *SysTableRepositoryImpl) GetTableByTableCode(ctx context.Context, code string) (model.SysTable, error) {
	var table model.SysTable
	err := s.db.WithContext(ctx).Preload("TableFields", func(db *gorm.DB) *gorm.DB {
		return db.Order("sequence")
	}).Preload("TableRelations").Preload("TableIndexes.IndexFields").Where("table_code = ? ", code).First(&table).Error
	return table, err
}

func (s *SysTableRepositoryImpl) FindMetadataIdentity(db *gorm.DB, id int) (model.SysTable, error) {
	if db == nil {
		db = s.db
	}
	var table model.SysTable
	err := db.Select("id", "table_code", "state", "gmt_delete").Unscoped().Where("id = ?", id).First(&table).Error
	return table, err
}

func (s *SysTableRepositoryImpl) GetTableList(ctx context.Context, basic *request.Basic, table model.SysTable) (response.ListResult[model.SysTable], error) {
	var repo response.ListResult[model.SysTable]
	var sysTableList []model.SysTable
	total, err := s.WithContext(ctx).PaginateAndCountAsync(basic, &sysTableList, table)
	zap.L().Debug("sysTableList", zap.Any("sysTableList", sysTableList))
	repo.Data = sysTableList
	repo.Total = int(total)
	return repo, err
}

func (s *SysTableRepositoryImpl) ListRuntimeTables(ctx context.Context) ([]model.SysTable, error) {
	var tables []model.SysTable
	err := s.db.WithContext(ctx).
		Preload("TableFields", func(db *gorm.DB) *gorm.DB {
			return db.Where("state = ?", true).Order("sequence, field_code")
		}).
		Preload("TableRelations", "state = ?", true).
		Where("state = ?", true).
		Order("table_code").
		Find(&tables).Error
	return tables, err
}

func (s *SysTableRepositoryImpl) HasTableColumn(db *gorm.DB, tableCode, fieldCode string) bool {
	if db == nil {
		db = s.db
	}
	tableName := util.GetTableName(db, tableCode)
	return db.Migrator().HasColumn(tableName, fieldCode)
}

func (s *SysTableRepositoryImpl) HasPhysicalTable(db *gorm.DB, tableCode string) bool {
	if db == nil {
		db = s.db
	}
	return db.Migrator().HasTable(util.GetTableName(db, tableCode))
}

// CreateTableIndex 创建实体表索引
func (s *SysTableRepositoryImpl) CreateTableIndex(tx *gorm.DB, isUnique bool, indexName string, tableCode string, fields string) error {
	var unique string
	if isUnique {
		unique = "UNIQUE "
	}
	tableName := util.GetTableName(tx, tableCode)
	createIndexSql := fmt.Sprintf("CREATE %sINDEX %s ON %s (%s)", unique, quoteSQLIdentifier(indexName), quoteSQLIdentifier(tableName), quoteSQLIdentifierList(fields))
	return tx.Exec(createIndexSql).Error
}

// DropTableIndex 删除实体表索引
func (s *SysTableRepositoryImpl) DropTableIndex(tx *gorm.DB, indexName string, tableCode string) error {
	dropIndexSQL := fmt.Sprintf("DROP INDEX IF EXISTS %s", quoteSQLIdentifier(indexName))
	return tx.Exec(dropIndexSQL).Error
}

func quoteSQLIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(strings.TrimSpace(identifier), `"`, `""`) + `"`
}

func quoteSQLIdentifierList(fields string) string {
	parts := strings.Split(fields, ",")
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		quoted = append(quoted, quoteSQLIdentifier(part))
	}
	return strings.Join(quoted, ",")
}

func (s *SysTableRepositoryImpl) CreateTable(tx *gorm.DB, tableCode string, model any) error {
	tableName := util.GetTableName(tx, tableCode)
	if tx.Migrator().HasTable(tableName) {
		return error2.ErrTableExist
	}
	return tx.Table(tableName).AutoMigrate(model)
}

func (s *SysTableRepositoryImpl) CreateView(tx *gorm.DB, viewCode string, sql string) error {
	viewName := util.GetTableName(tx, viewCode)
	query, err := platformmetadata.ValidateReadOnlyQuery(sql)
	if errors.Is(err, platformmetadata.ErrReadOnlyQueryEmpty) {
		return error2.ErrTableViewSQLEmpty
	}
	if err != nil {
		return err
	}
	createSQL := fmt.Sprintf("CREATE OR REPLACE VIEW %s AS %s", quoteSQLIdentifier(viewName), query)
	return tx.Exec(createSQL).Error
}

func (s *SysTableRepositoryImpl) DropTable(tx *gorm.DB, tableCode string) error {
	tableName := util.GetTableName(tx, tableCode)
	// 检查表是否存在
	if tx.Migrator().HasTable(tableName) {
		// 删除表
		err := tx.Migrator().DropTable(tableName)
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateTableColumn 添加实体字段
func (s *SysTableRepositoryImpl) CreateTableColumn(tx *gorm.DB, tableCode string, fieldCode string, sqlType string) error {
	tableName := util.GetTableName(tx, tableCode)
	addColumnSQL := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", quoteSQLIdentifier(tableName), quoteSQLIdentifier(fieldCode), sqlType)
	zap.L().Info("[DDL] 添加字段", zap.String("sql", addColumnSQL))
	return tx.Exec(addColumnSQL).Error
}

// DropTableColumn 删除实体字段
func (s *SysTableRepositoryImpl) DropTableColumn(tx *gorm.DB, tableCode string, fieldCode string) error {
	tableName := util.GetTableName(tx, tableCode)
	dropColumnSQL := fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", quoteSQLIdentifier(tableName), quoteSQLIdentifier(fieldCode))
	zap.L().Info("[DDL] 删除字段", zap.String("sql", dropColumnSQL))
	return tx.Exec(dropColumnSQL).Error
}

func (s *SysTableRepositoryImpl) ModifyTableColumn(tx *gorm.DB, tableCode string, fieldCode string, sqlType string) error {
	tableName := util.GetTableName(tx, tableCode)
	return alterPostgresColumn(tx, tableName, fieldCode, sqlType)
}

func (s *SysTableRepositoryImpl) ChangeTableColumn(tx *gorm.DB, tableCode string, originalFieldCode string, fieldCode string, sqlType string) error {
	tableName := util.GetTableName(tx, tableCode)
	renameColumnSQL := fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s;", quoteSQLIdentifier(tableName), quoteSQLIdentifier(originalFieldCode), quoteSQLIdentifier(fieldCode))
	zap.L().Info("[DDL] 重命名字段", zap.String("sql", renameColumnSQL))
	if err := tx.Exec(renameColumnSQL).Error; err != nil {
		return err
	}
	return alterPostgresColumn(tx, tableName, fieldCode, sqlType)
}

func (s *SysTableRepositoryImpl) FetchTableMetadata(ctx context.Context, db *gorm.DB, tableSchema string, tableCode string) ([]model.TableColumnMate, error) {
	var columns []model.TableColumnMate
	if db == nil {
		db = s.db
	}
	tableName := util.GetTableName(db, tableCode)
	query := `
SELECT
	c.column_name AS "COLUMN_NAME",
	c.ordinal_position AS "ORDINAL_POSITION",
	c.column_default AS "COLUMN_DEFAULT",
	c.is_nullable AS "IS_NULLABLE",
	c.data_type AS "DATA_TYPE",
	c.character_maximum_length AS "CHARACTER_MAXIMUM_LENGTH",
	c.character_octet_length AS "CHARACTER_OCTET_LENGTH",
	c.numeric_precision AS "NUMERIC_PRECISION",
	c.numeric_scale AS "NUMERIC_SCALE",
	c.datetime_precision AS "DATETIME_PRECISION",
	COALESCE(c.udt_name, c.data_type) AS "COLUMN_TYPE",
	CASE WHEN pk.column_name IS NULL THEN '' ELSE 'PRI' END AS "COLUMN_KEY",
	'' AS "EXTRA",
	'' AS "COLUMN_COMMENT"
FROM information_schema.columns c
LEFT JOIN (
	SELECT kcu.table_schema, kcu.table_name, kcu.column_name
	FROM information_schema.table_constraints tc
	JOIN information_schema.key_column_usage kcu
		ON tc.constraint_name = kcu.constraint_name
		AND tc.table_schema = kcu.table_schema
		AND tc.table_name = kcu.table_name
	WHERE tc.constraint_type = 'PRIMARY KEY'
) pk
	ON pk.table_schema = c.table_schema
	AND pk.table_name = c.table_name
	AND pk.column_name = c.column_name
WHERE c.table_schema = ? AND c.table_name = ?
ORDER BY c.ordinal_position;`
	err := db.WithContext(ctx).Raw(query, normalizePostgresSchema(tableSchema), tableName).Scan(&columns).Error
	if err != nil {
		return []model.TableColumnMate{}, err
	}
	return columns, nil
}

func (s *SysTableRepositoryImpl) FetchTableIndexMetadata(ctx context.Context, db *gorm.DB, tableSchema string, tableCode string) ([]model.TableIndexMate, error) {
	var indexes []model.TableIndexMate
	if db == nil {
		db = s.db
	}
	tableName := util.GetTableName(db, tableCode)
	query := `
SELECT
	a.attname AS "COLUMN_NAME",
	i.relname AS "INDEX_NAME",
	(NOT ix.indisunique) AS "NON_UNIQUE",
	key.ordinal_position AS "ORDINAL_POSITION"
FROM pg_class t
JOIN pg_namespace n ON n.oid = t.relnamespace
JOIN pg_index ix ON ix.indrelid = t.oid
JOIN pg_class i ON i.oid = ix.indexrelid
CROSS JOIN LATERAL unnest(ix.indkey) WITH ORDINALITY AS key(attnum, ordinal_position)
JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = key.attnum
WHERE n.nspname = ? AND t.relname = ? AND ix.indisprimary = false
ORDER BY i.relname, key.ordinal_position;`
	err := db.WithContext(ctx).Raw(query, normalizePostgresSchema(tableSchema), tableName).Scan(&indexes).Error
	if err != nil {
		return []model.TableIndexMate{}, err
	}
	return indexes, nil
}

func normalizePostgresSchema(schema string) string {
	schema = strings.TrimSpace(schema)
	if schema == "" {
		return "public"
	}
	return schema
}

func alterPostgresColumn(tx *gorm.DB, tableName string, fieldCode string, sqlType string) error {
	typeSQL, defaultSQL, notNull := parsePostgresColumnDefinition(sqlType)
	table := quoteSQLIdentifier(tableName)
	field := quoteSQLIdentifier(fieldCode)

	alterTypeSQL := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;", table, field, typeSQL)
	zap.L().Info("[DDL] 修改字段类型", zap.String("sql", alterTypeSQL))
	if err := tx.Exec(alterTypeSQL).Error; err != nil {
		return err
	}
	if defaultSQL != nil {
		setDefaultSQL := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s;", table, field, *defaultSQL)
		zap.L().Info("[DDL] 修改字段默认值", zap.String("sql", setDefaultSQL))
		if err := tx.Exec(setDefaultSQL).Error; err != nil {
			return err
		}
	} else {
		dropDefaultSQL := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT;", table, field)
		zap.L().Info("[DDL] 清理字段默认值", zap.String("sql", dropDefaultSQL))
		if err := tx.Exec(dropDefaultSQL).Error; err != nil {
			return err
		}
	}
	if notNull {
		setNotNullSQL := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL;", table, field)
		zap.L().Info("[DDL] 设置字段非空", zap.String("sql", setNotNullSQL))
		return tx.Exec(setNotNullSQL).Error
	}
	dropNotNullSQL := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL;", table, field)
	zap.L().Info("[DDL] 设置字段可空", zap.String("sql", dropNotNullSQL))
	return tx.Exec(dropNotNullSQL).Error
}

func parsePostgresColumnDefinition(sqlType string) (string, *string, bool) {
	normalized := strings.TrimSpace(sqlType)
	notNull := strings.Contains(strings.ToUpper(normalized), " NOT NULL")
	normalized = strings.ReplaceAll(normalized, " NOT NULL", "")
	normalized = strings.ReplaceAll(normalized, " not null", "")
	normalized = strings.TrimSuffix(normalized, " NULL")
	normalized = strings.TrimSuffix(normalized, " null")

	upper := strings.ToUpper(normalized)
	defaultIndex := strings.Index(upper, " DEFAULT ")
	if defaultIndex < 0 {
		return strings.TrimSpace(normalized), nil, notNull
	}
	typeSQL := strings.TrimSpace(normalized[:defaultIndex])
	defaultSQL := strings.TrimSpace(normalized[defaultIndex+len(" DEFAULT "):])
	return typeSQL, &defaultSQL, notNull
}

func (s *SysTableRepositoryImpl) Model(data []model.SysTableField) interface{} {
	// 动态创建结构体类型
	dynamicType := util.CreateDynamicStruct(data)
	// 创建实例
	dynamicModel := reflect.New(dynamicType).Interface()
	return dynamicModel
}
