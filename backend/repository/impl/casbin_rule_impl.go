/**
 * @Author: Nan
 * @Date: 2024/6/22 下午2:31
 */

package impl

import (
	"backend/internal/database"
	"backend/model"

	"github.com/casbin/casbin/v2"
	"gorm.io/gorm"
)

type CasbinRuleRepositoryImpl struct {
	db *gorm.DB
	*BasicRepositoryImpl[model.CasbinRule]
	enforcer *casbin.Enforcer
}

func NewCasbinRuleRepositoryImpl(PrimaryDB *database.PrimaryDB, enforcer *casbin.Enforcer) *CasbinRuleRepositoryImpl {
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
