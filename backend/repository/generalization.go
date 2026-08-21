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
	Query(context.Context, *request.Basic, model.SysTable) (GeneralizationListResult, error)
	GetById(context.Context, model.SysTable, int) (map[string]interface{}, error)
	Create(context.Context, model.SysTable, map[string]interface{}) error
	RowExists(context.Context, model.SysTable, int) (bool, error)
	Update(context.Context, model.SysTable, int, map[string]interface{}) error
	SoftDelete(context.Context, model.SysTable, int, map[string]interface{}) error
	HardDelete(context.Context, model.SysTable, int) error
}

type GeneralizationPermissionRepository interface {
	DBWithContext(context.Context) *gorm.DB
	QueryWithPermission(context.Context, *request.Basic, model.SysTable, GeneralizationPermission) (GeneralizationListResult, error)
	QueryWithPermissionDB(*gorm.DB, *request.Basic, model.SysTable, GeneralizationPermission) (GeneralizationListResult, error)
	GetByIdWithPermission(context.Context, model.SysTable, int, GeneralizationPermission) (map[string]interface{}, error)
	UpdateWithPermission(context.Context, model.SysTable, int, map[string]interface{}, GeneralizationPermission) (bool, error)
	SoftDeleteWithPermission(context.Context, model.SysTable, int, map[string]interface{}, GeneralizationPermission) (bool, error)
	HardDeleteWithPermission(context.Context, model.SysTable, int, GeneralizationPermission) (bool, error)
	BatchSoftDeleteWithPermission(*gorm.DB, model.SysTable, []int, map[string]interface{}, GeneralizationPermission) (bool, error)
	BatchHardDeleteWithPermission(*gorm.DB, model.SysTable, []int, GeneralizationPermission) (bool, error)
}
