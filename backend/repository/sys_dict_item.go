/**
 * @Author: Nan
 * @Date: 2024/7/20 下午2:51
 */

package repository

import (
	"backend/model"
)

type SysDictItemRepository interface {
	BasicRepository[model.SysDictItem]
	GetSysDictItemsByDictId(int) ([]model.SysDictItem, error)
}
