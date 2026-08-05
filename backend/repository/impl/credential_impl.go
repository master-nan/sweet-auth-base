package impl

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/internal/database"
	"backend/model"
	"backend/repository"
	"backend/repository/util"
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CredentialRepositoryImpl struct {
	db *gorm.DB
}

func NewCredentialRepositoryImpl(primaryDB *database.PrimaryDB) *CredentialRepositoryImpl {
	return &CredentialRepositoryImpl{db: primaryDB.DB}
}

func (r *CredentialRepositoryImpl) DBWithContext(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

func (r *CredentialRepositoryImpl) Create(tx *gorm.DB, value *model.Credential) error {
	return tx.Create(value).Error
}

func (r *CredentialRepositoryImpl) FindByID(ctx context.Context, id int) (model.Credential, error) {
	var value model.Credential
	err := r.db.WithContext(ctx).First(&value, id).Error
	return value, err
}

func (r *CredentialRepositoryImpl) FindByIDForUpdate(tx *gorm.DB, id int) (model.Credential, error) {
	var value model.Credential
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&value, id).Error
	return value, err
}

func (r *CredentialRepositoryImpl) FindByIDs(ctx context.Context, ids []int) ([]model.Credential, error) {
	if len(ids) == 0 {
		return []model.Credential{}, nil
	}
	var values []model.Credential
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&values).Error
	return values, err
}

func (r *CredentialRepositoryImpl) FindByCode(tx *gorm.DB, systemID int, code string) (model.Credential, error) {
	var value model.Credential
	err := tx.Where("external_system_id = ? AND credential_code = ?", systemID, code).First(&value).Error
	return value, err
}

func (r *CredentialRepositoryImpl) Query(
	ctx context.Context,
	req request.CredentialQueryReq,
	table model.SysTable,
) (response.ListResult[model.Credential], error) {
	basic := req.ToBasic()
	query := util.ExecuteQuery(r.db.WithContext(ctx).Model(&model.Credential{}), &basic, table)
	if req.ExternalSystemID > 0 {
		query = query.Where("external_system_id = ?", req.ExternalSystemID)
	}
	if req.CredentialType != "" {
		query = query.Where("credential_type = ?", req.CredentialType)
	}
	if status := strings.TrimSpace(req.Status); status != "" {
		if status == "expired" {
			query = query.Where("status <> ? AND expires_at IS NOT NULL AND expires_at <= ?", model.CredentialStatusRevoked, time.Now())
		} else {
			query = query.Where("status = ?", status)
		}
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Limit(-1).Offset(-1).Count(&total).Error; err != nil {
		return response.ListResult[model.Credential]{}, err
	}
	var values []model.Credential
	if err := query.Find(&values).Error; err != nil {
		return response.ListResult[model.Credential]{}, err
	}
	return response.ListResult[model.Credential]{Data: values, Total: int(total)}, nil
}

func (r *CredentialRepositoryImpl) UpdateFields(tx *gorm.DB, id, revision int, updates map[string]any) (bool, error) {
	result := tx.Model(&model.Credential{}).
		Where("id = ? AND revision = ?", id, revision).
		Updates(updates)
	return result.RowsAffected == 1, result.Error
}

var _ repository.CredentialRepository = (*CredentialRepositoryImpl)(nil)
