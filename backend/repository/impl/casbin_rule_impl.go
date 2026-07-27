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
	db *gorm.DB
	*BasicRepositoryImpl[model.CasbinRule]
	enforcer repository.CasbinPolicyEnforcer
}

func NewCasbinRuleRepositoryImpl(PrimaryDB *database.PrimaryDB, enforcer repository.CasbinPolicyEnforcer) *CasbinRuleRepositoryImpl {
	return &CasbinRuleRepositoryImpl{
		db:                  PrimaryDB.DB,
		BasicRepositoryImpl: NewBasicRepositoryImpl(PrimaryDB.DB, &model.CasbinRule{}),
		enforcer:            enforcer,
	}
}

func (c CasbinRuleRepositoryImpl) AddPolicy(params ...interface{}) (bool, error) {
	return c.enforcer.AddPolicy(params...)
}

func (c CasbinRuleRepositoryImpl) RemovePolicy(params ...interface{}) (bool, error) {
	return c.enforcer.RemovePolicy(params...)
}

func (c CasbinRuleRepositoryImpl) RemoveFilteredPolicy(fieldIndex int, fieldValues ...string) (bool, error) {
	return c.enforcer.RemoveFilteredPolicy(fieldIndex, fieldValues...)
}

func (c CasbinRuleRepositoryImpl) GetFilteredPolicy(fieldIndex int, fieldValues ...string) ([][]string, error) {
	return c.enforcer.GetFilteredPolicy(fieldIndex, fieldValues...)
}

func (c CasbinRuleRepositoryImpl) ReplaceSubjectPolicies(subject string, policies [][]string) error {
	_, err := c.enforcer.UpdateFilteredPolicies(policies, 0, subject)
	return err
}
