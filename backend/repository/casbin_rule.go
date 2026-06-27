/**
 * @Author: Nan
 * @Date: 2024/6/22 下午2:30
 */

package repository

import "backend/model"

type CasbinRuleRepository interface {
	BasicRepository[model.CasbinRule]
	AddPolicy(params ...interface{}) (bool, error)
	RemovePolicy(params ...interface{}) (bool, error)
	RemoveFilteredPolicy(fieldIndex int, fieldValues ...string) (bool, error)
	GetFilteredPolicy(fieldIndex int, fieldValues ...string) ([][]string, error)
}
