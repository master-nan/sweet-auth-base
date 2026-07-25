/**
 * @Author: Nan
 * @Date: 2024/7/11 上午11:24
 */

package repository

import (
	"backend/dto/request"
	"backend/model"
	"errors"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var (
	ErrInvalidField       = errors.New("repository: invalid field")
	ErrInvalidFieldValues = errors.New("repository: field values must be a slice or array")
)

// BasicRepository defines the shared persistence baseline for typed repositories.
//
// Implementations may compose database queries, perform CRUD operations and
// paginate results. Business rules, permission decisions and service calls do
// not belong in repositories. Methods return validation or database errors to
// the service layer without translating them into API errors.
type BasicRepository[T any] interface {
	ExecuteTx(ctx *gin.Context, fn func(tx *gorm.DB) error) error
	DBWithContext(*gin.Context) *gorm.DB
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
	FindListById(id int) ([]T, error)
	FindByField(field string, value interface{}) (T, error)
	FindListByField(field string, value interface{}) ([]T, error)
	FindListByFieldIn(field string, value interface{}) ([]T, error)
	WithPreload(...string) BasicRepository[T]
	WithUnscoped() BasicRepository[T]
	WithSelect(...string) BasicRepository[T]
	WithOmit(...string) BasicRepository[T]
	WithContext(*gin.Context) BasicRepository[T]
}
