/**
 * @Author: Nan
 * @Date: 2024/7/20 下午3:48
 */

package repository

import (
	"backend/model"
	"context"
)

type SysTableRelationRepository interface {
	BasicRepository[model.SysTableRelation]
	GetTableRelationsByTableId(context.Context, int) ([]model.SysTableRelation, error)
}
