/**
 * @Author: Nan
 * @Date: 2024/6/10 上午12:14
 */

package repository

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"
	"gorm.io/gorm"
)

type SysTableRepository interface {
	BasicRepository[model.SysTable]
	GetTableById(int) (model.SysTable, error)
	GetTableByTableCode(string) (model.SysTable, error)
	GetTableList(*request.Basic, model.SysTable) (response.ListResult[model.SysTable], error)

	FetchTableMetadata(string, string) ([]model.TableColumnMate, error)
	FetchTableIndexMetadata(string, string) ([]model.TableIndexMate, error)

	Model([]model.SysTableField) interface{}

	// DropTableIndex
	// 所有数据库操作
	DropTableIndex(*gorm.DB, string, string) error
	DropTable(*gorm.DB, string) error
	DropTableColumn(*gorm.DB, string, string) error
	ModifyTableColumn(*gorm.DB, string, string, string) error
	ChangeTableColumn(*gorm.DB, string, string, string, string) error
	CreateTableColumn(*gorm.DB, string, string, string) error
	CreateTable(*gorm.DB, string, any) error
	CreateView(*gorm.DB, string, string) error
	CreateTableIndex(*gorm.DB, bool, string, string, string) error
}
