/**
 * @Author: Nan
 * @Date: 2024/6/13 下午11:33
 */

package repository

import (
	"backend/dto/request"
	"backend/model"
)

type GeneralizationListResult struct {
	Data  []map[string]interface{} `json:"data"`
	Total int                      `json:"total"`
}

type GeneralizationRepository interface {
	Query(*request.Basic, model.SysTable) (GeneralizationListResult, error)
	GetById(table model.SysTable, id int) (map[string]interface{}, error)
	Create(table model.SysTable, data map[string]interface{}) error
	RowExists(table model.SysTable, id int) (bool, error)
	RowMatchesDataScope(table model.SysTable, id int, scope *request.DataScope) (bool, error)
	Update(table model.SysTable, id int, data map[string]interface{}) error
	SoftDelete(table model.SysTable, id int, deleteData map[string]interface{}) error
	HardDelete(table model.SysTable, id int) error
	GetFieldById(tableCode string, id int, fieldName string) (interface{}, error)
}
