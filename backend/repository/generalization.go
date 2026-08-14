/**
 * @Author: Nan
 * @Date: 2024/6/13 下午11:33
 */

package repository

import (
	"backend/dto/request"
	"backend/internal/datapermission"
	"backend/model"
	"context"

	"gorm.io/gorm"
)

type GeneralizationListResult struct {
	Data  []map[string]interface{} `json:"data"`
	Total int                      `json:"total"`
}

// GeneralizationPermission 将服务端构建的 Adapter 执行结果传入 Repository。
// 客户端请求 DTO 无法填充此字段。
type GeneralizationPermission struct {
	AdapterExecution *datapermission.AdapterExecution
}

type GeneralizationRepository interface {
	Query(*request.Basic, model.SysTable) (GeneralizationListResult, error)
	GetById(table model.SysTable, id int) (map[string]interface{}, error)
	Create(table model.SysTable, data map[string]interface{}) error
	RowExists(table model.SysTable, id int) (bool, error)
	Update(table model.SysTable, id int, data map[string]interface{}) error
	SoftDelete(table model.SysTable, id int, deleteData map[string]interface{}) error
	HardDelete(table model.SysTable, id int) error
}

type GeneralizationPermissionRepository interface {
	DBWithContext(context.Context) *gorm.DB
	QueryWithPermission(*request.Basic, model.SysTable, GeneralizationPermission) (GeneralizationListResult, error)
	GetByIdWithPermission(model.SysTable, int, GeneralizationPermission) (map[string]interface{}, error)
	UpdateWithPermission(model.SysTable, int, map[string]interface{}, GeneralizationPermission) (bool, error)
	SoftDeleteWithPermission(model.SysTable, int, map[string]interface{}, GeneralizationPermission) (bool, error)
	HardDeleteWithPermission(model.SysTable, int, GeneralizationPermission) (bool, error)
	BatchSoftDeleteWithPermission(*gorm.DB, model.SysTable, []int, map[string]interface{}, GeneralizationPermission) (bool, error)
	BatchHardDeleteWithPermission(*gorm.DB, model.SysTable, []int, GeneralizationPermission) (bool, error)
}
