package impl

import (
	"backend/internal/database"
	"backend/model"
	"backend/repository"

	"gorm.io/gorm"
)

type RetryPolicyRepositoryImpl struct {
	*BasicRepositoryImpl[model.RetryPolicy]
}

func NewRetryPolicyRepositoryImpl(primaryDB *database.PrimaryDB) *RetryPolicyRepositoryImpl {
	return &RetryPolicyRepositoryImpl{
		BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.RetryPolicy{}),
	}
}

func (r *RetryPolicyRepositoryImpl) NextVersion(tx *gorm.DB, code string) (int, error) {
	var maxVersion int
	err := tx.Model(&model.RetryPolicy{}).
		Where("policy_code = ?", code).
		Select("COALESCE(MAX(version), 0)").
		Scan(&maxVersion).Error
	return maxVersion + 1, err
}

func (r *RetryPolicyRepositoryImpl) HasEnabledVersion(tx *gorm.DB, code string, excludeID int) (bool, error) {
	var count int64
	err := tx.Model(&model.RetryPolicy{}).
		Where("policy_code = ? AND status = ? AND id <> ?", code, model.RetryPolicyStatusEnabled, excludeID).
		Count(&count).Error
	return count > 0, err
}

func (r *RetryPolicyRepositoryImpl) CountEnabledInterfaceReferences(tx *gorm.DB, id int) (int64, error) {
	var count int64
	err := tx.Model(&model.InterfaceDefinition{}).
		Where("retry_policy_id = ? AND status = ?", id, model.InterfaceDefinitionStatusEnabled).
		Count(&count).Error
	return count, err
}

var _ repository.RetryPolicyRepository = (*RetryPolicyRepositoryImpl)(nil)
