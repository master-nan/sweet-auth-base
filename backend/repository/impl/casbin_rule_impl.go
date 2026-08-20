/**
 * @Author: Nan
 * @Date: 2024/6/22 下午2:31
 */

package impl

import (
	"backend/internal/database"
	"backend/model"
	"backend/repository"

	"gorm.io/gorm"
)

type CasbinRuleRepositoryImpl struct {
	db       *gorm.DB
	enforcer repository.CasbinPolicyEnforcer
}

func NewCasbinRuleRepositoryImpl(PrimaryDB *database.PrimaryDB, enforcer repository.CasbinPolicyEnforcer) *CasbinRuleRepositoryImpl {
	return &CasbinRuleRepositoryImpl{
		db:       PrimaryDB.DB,
		enforcer: enforcer,
	}
}

func (c CasbinRuleRepositoryImpl) AddPolicy(params ...interface{}) (bool, error) {
	return c.enforcer.AddPolicy(params...)
}

func (c CasbinRuleRepositoryImpl) RemovePolicy(params ...interface{}) (bool, error) {
	return c.enforcer.RemovePolicy(params...)
}

func (c CasbinRuleRepositoryImpl) GetFilteredPolicy(fieldIndex int, fieldValues ...string) ([][]string, error) {
	return c.enforcer.GetFilteredPolicy(fieldIndex, fieldValues...)
}

func (c CasbinRuleRepositoryImpl) ReplaceSubjectPolicies(subject string, policies [][]string) error {
	_, err := c.enforcer.UpdateFilteredPolicies(policies, 0, subject)
	return err
}

func (c CasbinRuleRepositoryImpl) UpsertPolicyWithDB(db *gorm.DB, subject, path, method string) error {
	var count int64
	if err := db.Model(&model.CasbinRule{}).
		Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?", "p", subject, path, method).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return db.Create(&model.CasbinRule{PType: "p", V0: subject, V1: path, V2: method}).Error
}

func (c CasbinRuleRepositoryImpl) RemovePolicyWithDB(db *gorm.DB, subject, path, method string) error {
	return db.Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?", "p", subject, path, method).
		Delete(&model.CasbinRule{}).Error
}

func (c CasbinRuleRepositoryImpl) ReloadPolicy() error {
	return c.enforcer.LoadPolicy()
}
