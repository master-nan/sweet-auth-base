/**
 * @Author: Nan
 * @Date: 2024/6/10 上午12:14
 */

package repository

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"
	"context"
	"gorm.io/gorm"
)

type SysTableRepository interface {
	BasicRepository[model.SysTable]
	GetTableById(context.Context, int) (model.SysTable, error)
	GetTableByTableCode(context.Context, string) (model.SysTable, error)
	FindMetadataIdentity(*gorm.DB, int) (model.SysTable, error)
	GetTableList(context.Context, *request.Basic, model.SysTable) (response.ListResult[model.SysTable], error)
	ListRuntimeTables(context.Context) ([]model.SysTable, error)
	QueryRuntimeRelationOptions(context.Context, RuntimeRelationOptionQuery) ([]RuntimeRelationOption, int, error)
	HasPhysicalTable(*gorm.DB, string) bool
	HasTableColumn(*gorm.DB, string, string) bool

	FetchTableMetadata(context.Context, *gorm.DB, string, string) ([]model.TableColumnMate, error)
	FetchTableIndexMetadata(context.Context, *gorm.DB, string, string) ([]model.TableIndexMate, error)

	Model([]model.SysTableField) interface{}

	// 表结构相关数据库操作
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

type RuntimeRelationOptionQuery struct {
	TableCode      string
	ValueField     string
	DisplayField   string
	ParentField    string
	Keyword        string
	Page           int
	Num            int
	SelectedValues []string
	Filters        map[string]interface{}
	HasState       bool
	HasDeletedAt   bool
}

type RuntimeRelationOption struct {
	Value       string
	Label       string
	ParentValue string
}
