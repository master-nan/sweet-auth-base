/**
 * @Author: Nan
 * @Date: 2024/7/11 上午11:24
 */

package repository

import (
	"backend/dto/request"
	"backend/model"
	"context"
	"errors"

	"gorm.io/gorm"
)

var (
	ErrInvalidField       = errors.New("repository: invalid field")
	ErrInvalidFieldValues = errors.New("repository: field values must be a slice or array")
)

// BasicRepository 定义类型化 Repository 的共享持久化基线。
//
// 实现可以组合数据库查询、执行 CRUD 操作并对结果分页。
// 业务规则、权限决策和 Service 调用不属于 Repository 职责。
// 方法将校验或数据库错误返回 Service 层，不在此处转换为 API 错误。
type BasicRepository[T any] interface {
	ExecuteTx(ctx context.Context, fn func(tx *gorm.DB) error) error
	DBWithContext(context.Context) *gorm.DB
	QueryWhere(string) *gorm.DB
	Count(*gorm.DB) (int64, error)
	PaginateAndCountAsync(*request.Basic, interface{}, model.SysTable) (int64, error)
	Create(*gorm.DB, interface{}) error
	Update(*gorm.DB, interface{}, int) error
	DeleteById(*gorm.DB, int) error
	DeleteByField(*gorm.DB, string, interface{}) error
	DeleteByIds(*gorm.DB, []int) error
	DeleteByFieldIn(*gorm.DB, string, []interface{}) error
	FindById(id int) (T, error)
	FindByIdWithDB(*gorm.DB, int) (T, error)
	FindByIdForUpdate(*gorm.DB, int) (T, error)
	FindListById(id int) ([]T, error)
	FindByField(field string, value interface{}) (T, error)
	FindByFieldWithDB(*gorm.DB, string, interface{}) (T, error)
	FindListByField(field string, value interface{}) ([]T, error)
	FindListByFieldIn(field string, value interface{}) ([]T, error)
	UpdateFields(*gorm.DB, int, map[string]any) (bool, error)
	UpdateFieldsByRevision(*gorm.DB, int, int, map[string]any) (bool, error)
	WithPreload(...string) BasicRepository[T]
	WithUnscoped() BasicRepository[T]
	WithSelect(...string) BasicRepository[T]
	WithOmit(...string) BasicRepository[T]
	WithContext(context.Context) BasicRepository[T]
}
