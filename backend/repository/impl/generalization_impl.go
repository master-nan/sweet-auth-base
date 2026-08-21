/**
 * @Author: Nan
 * @Date: 2024/6/13 下午11:34
 */

package impl

import (
	"backend/dto/request"
	"backend/enum"
	"backend/internal/database"
	myerrors "backend/internal/errors"
	"backend/internal/security"
	"backend/model"
	"backend/repository"
	"backend/repository/util"
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

type GeneralizationRepositoryImpl struct {
	db *gorm.DB
}

var errGeneralizationBatchPermissionMismatch = errors.New("generalization batch permission mismatch")

func NewGeneralizationRepositoryImpl(PrimaryDB *database.PrimaryDB) *GeneralizationRepositoryImpl {
	return &GeneralizationRepositoryImpl{
		db: PrimaryDB.DB,
	}
}

func (g *GeneralizationRepositoryImpl) DBWithContext(ctx context.Context) *gorm.DB {
	return g.db.WithContext(ctx)
}

func (g *GeneralizationRepositoryImpl) Query(ctx context.Context, basic *request.Basic, table model.SysTable) (repository.GeneralizationListResult, error) {
	result, err := util.DynamicQuery(g.db.WithContext(ctx), basic, table)
	if err != nil {
		return repository.GeneralizationListResult{}, err
	}
	return result, nil
}

func (g *GeneralizationRepositoryImpl) QueryWithPermission(
	ctx context.Context,
	basic *request.Basic,
	table model.SysTable,
	permission repository.GeneralizationPermission,
) (repository.GeneralizationListResult, error) {
	return g.QueryWithPermissionDB(g.db.WithContext(ctx), basic, table, permission)
}

func (g *GeneralizationRepositoryImpl) QueryWithPermissionDB(
	db *gorm.DB,
	basic *request.Basic,
	table model.SysTable,
	permission repository.GeneralizationPermission,
) (repository.GeneralizationListResult, error) {
	result, err := util.DynamicQueryWithPermission(db, basic, table, permission)
	if err != nil {
		return repository.GeneralizationListResult{}, err
	}
	return result, nil
}

func (g *GeneralizationRepositoryImpl) GetById(ctx context.Context, table model.SysTable, id int) (map[string]interface{}, error) {
	return g.getByIdWithQuery(activeRowQuery(g.db.WithContext(ctx).Table(table.TableCode), table), table, id)
}

func (g *GeneralizationRepositoryImpl) GetByIdWithPermission(
	ctx context.Context,
	table model.SysTable,
	id int,
	permission repository.GeneralizationPermission,
) (map[string]interface{}, error) {
	query, err := util.ApplyGeneralizationPermission(
		activeRowQuery(g.db.WithContext(ctx).Table(table.TableCode), table),
		permission,
		table,
	)
	if err != nil {
		return nil, err
	}
	return g.getByIdWithQuery(query, table, id)
}

func (g *GeneralizationRepositoryImpl) getByIdWithQuery(
	query *gorm.DB,
	table model.SysTable,
	id int,
) (map[string]interface{}, error) {
	if id <= 0 {
		return nil, myerrors.ErrDataNotFound
	}
	selectParts := detailSelectParts(table)
	if len(selectParts) == 0 {
		return nil, myerrors.ErrDataNotFound
	}
	var result map[string]interface{}
	err := query.
		Select(strings.Join(selectParts, ",")).
		Where("id = ?", id).
		Take(&result).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, myerrors.ErrDataNotFound
	}
	if err != nil {
		return nil, err
	}
	return normalizeGeneralizationRecord(result), nil
}

func (g *GeneralizationRepositoryImpl) Create(ctx context.Context, table model.SysTable, data map[string]interface{}) error {
	return g.db.WithContext(ctx).Table(table.TableCode).Create(data).Error
}

func (g *GeneralizationRepositoryImpl) RowExists(ctx context.Context, table model.SysTable, id int) (bool, error) {
	var exists int
	query := activeRowQuery(g.db.WithContext(ctx).Table(table.TableCode), table).Select("1").Where("id = ?", id).Limit(1)
	err := query.Scan(&exists).Error
	if err != nil {
		return false, err
	}
	return exists == 1, nil
}

func (g *GeneralizationRepositoryImpl) Update(ctx context.Context, table model.SysTable, id int, data map[string]interface{}) error {
	return activeRowQuery(g.db.WithContext(ctx).Table(table.TableCode), table).Where("id = ?", id).Updates(data).Error
}

func (g *GeneralizationRepositoryImpl) UpdateWithPermission(
	ctx context.Context,
	table model.SysTable,
	id int,
	data map[string]interface{},
	permission repository.GeneralizationPermission,
) (bool, error) {
	query, err := util.ApplyGeneralizationPermission(
		activeRowQuery(g.db.WithContext(ctx).Table(table.TableCode), table),
		permission,
		table,
	)
	if err != nil {
		return false, err
	}
	result := query.Where("id = ?", id).Updates(data)
	return result.RowsAffected > 0, result.Error
}

func (g *GeneralizationRepositoryImpl) SoftDelete(ctx context.Context, table model.SysTable, id int, deleteData map[string]interface{}) error {
	return activeRowQuery(g.db.WithContext(ctx).Table(table.TableCode), table).Where("id = ?", id).Updates(deleteData).Error
}

func (g *GeneralizationRepositoryImpl) SoftDeleteWithPermission(
	ctx context.Context,
	table model.SysTable,
	id int,
	deleteData map[string]interface{},
	permission repository.GeneralizationPermission,
) (bool, error) {
	query, err := util.ApplyGeneralizationPermission(
		activeRowQuery(g.db.WithContext(ctx).Table(table.TableCode), table),
		permission,
		table,
	)
	if err != nil {
		return false, err
	}
	result := query.Where("id = ?", id).Updates(deleteData)
	return result.RowsAffected > 0, result.Error
}

func (g *GeneralizationRepositoryImpl) HardDelete(ctx context.Context, table model.SysTable, id int) error {
	return g.db.WithContext(ctx).Table(table.TableCode).Where("id = ?", id).Delete(nil).Error
}

func (g *GeneralizationRepositoryImpl) HardDeleteWithPermission(
	ctx context.Context,
	table model.SysTable,
	id int,
	permission repository.GeneralizationPermission,
) (bool, error) {
	query, err := util.ApplyGeneralizationPermission(g.db.WithContext(ctx).Table(table.TableCode), permission, table)
	if err != nil {
		return false, err
	}
	result := query.Where("id = ?", id).Delete(nil)
	return result.RowsAffected > 0, result.Error
}

func (g *GeneralizationRepositoryImpl) BatchSoftDeleteWithPermission(
	tx *gorm.DB,
	table model.SysTable,
	ids []int,
	deleteData map[string]interface{},
	permission repository.GeneralizationPermission,
) (bool, error) {
	if len(ids) == 0 {
		return false, nil
	}
	query, err := util.ApplyGeneralizationPermission(activeRowQuery(tx.Table(table.TableCode), table), permission, table)
	if err != nil {
		return false, err
	}
	result := query.Where("id IN ?", ids).Updates(deleteData)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected != int64(len(ids)) {
		return false, nil
	}
	return true, nil
}

func (g *GeneralizationRepositoryImpl) BatchHardDeleteWithPermission(
	tx *gorm.DB,
	table model.SysTable,
	ids []int,
	permission repository.GeneralizationPermission,
) (bool, error) {
	if len(ids) == 0 {
		return false, nil
	}
	query, err := util.ApplyGeneralizationPermission(tx.Table(table.TableCode), permission, table)
	if err != nil {
		return false, err
	}
	result := query.Where("id IN ?", ids).Delete(nil)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected != int64(len(ids)) {
		return false, nil
	}
	return true, nil
}

func activeRowQuery(db *gorm.DB, table model.SysTable) *gorm.DB {
	if hasGeneralizationTableField(table, "gmt_delete") {
		return db.Where("gmt_delete IS NULL")
	}
	return db
}

func hasGeneralizationTableField(table model.SysTable, fieldCode string) bool {
	for _, field := range table.TableFields {
		if field.FieldCode == fieldCode {
			return true
		}
	}
	return false
}

func detailSelectParts(table model.SysTable) []string {
	result := make([]string, 0, len(table.TableFields))
	for _, field := range table.TableFields {
		fieldCode := strings.TrimSpace(field.FieldCode)
		if fieldCode == "" || security.IsSensitiveFieldName(fieldCode) {
			continue
		}
		if field.FieldCategory == enum.CalculatedField || field.FieldCategory == enum.VirtualField {
			continue
		}
		expression := util.QuoteIdentifier(fieldCode)
		if field.FieldType == enum.DecimalFieldType {
			expression = "CAST(" + expression + " AS text) AS " + util.QuoteIdentifier(fieldCode)
		}
		result = append(result, expression)
	}
	return result
}

func normalizeGeneralizationRecord(record map[string]interface{}) map[string]interface{} {
	for key, value := range record {
		if security.IsSensitiveFieldName(key) {
			delete(record, key)
			continue
		}
		switch v := value.(type) {
		case time.Time:
			if v.IsZero() {
				record[key] = ""
			} else {
				record[key] = v.Format(time.DateTime)
			}
		case []byte:
			record[key] = string(v)
		}
	}
	return record
}
