/**
 * @Author: Nan
 * @Date: 2024/6/3 下午2:50
 */

package repository

import (
	"backend/model"
)

type SysConfigureRepository interface {
	BasicRepository[model.SysConfigure]
	GetSysConfigure() (model.SysConfigure, error)
}
