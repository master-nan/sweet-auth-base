/**
 * @Author: Nan
 * @Date: 2024/7/22 上午10:18
 */

package repository

import "backend/model"

type SysTableIndexFieldRepository interface {
	BasicRepository[model.SysTableIndexField]
}
