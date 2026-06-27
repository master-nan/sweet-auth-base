/**
 * @Author: Nan
 * @Date: 2024/7/20 下午3:48
 */

package repository

import (
	"backend/model"
)

type SysTableRelationRepository interface {
	BasicRepository[model.SysTableRelation]
	GetTableRelationsByTableId(int) ([]model.SysTableRelation, error)
}
