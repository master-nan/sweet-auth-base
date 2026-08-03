/**
 * @Author: Nan
 * @Date: 2024/7/11 上午11:25
 */

package impl

import (
	"backend/dto/request"
	"backend/model"
	"backend/repository"
	"backend/repository/util"
	"fmt"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BasicRepositoryImpl[T any] struct {
	db       *gorm.DB
	preloads []string
	selects  []string
	omits    []string
	model    *T
	unscoped bool
	ctx      *gin.Context
}

func NewBasicRepositoryImpl[T any](db *gorm.DB, model *T) *BasicRepositoryImpl[T] {
	return &BasicRepositoryImpl[T]{
		db:    db,
		model: model,
	}
}

func (b *BasicRepositoryImpl[T]) ExecuteTx(ctx *gin.Context, fn func(tx *gorm.DB) error) error {
	return b.db.WithContext(ctx).Transaction(func(tx *gorm.DB) (err error) {
		defer func() {
			if r := recover(); r != nil {
				if e, ok := r.(error); ok {
					zap.L().Error("Recovered from panic:", zap.Error(e))
					err = e
				} else {
					zap.L().Error("Recovered from panic:", zap.Any("recover:", r))
					err = fmt.Errorf("panic: %v", r)
				}
			}
		}()
		return fn(tx) // 执行传入的函数
	})
}

func (b *BasicRepositoryImpl[T]) DBWithContext(ctx *gin.Context) *gorm.DB {
	return b.db.WithContext(ctx)
}

func (b *BasicRepositoryImpl[T]) QueryWhere(filter string) *gorm.DB {
	return b.baseQuery().Model(b.model).Where(filter)
}

func (b *BasicRepositoryImpl[T]) Count(query *gorm.DB) (int64, error) {
	var total int64
	err := query.Limit(-1).Offset(-1).Count(&total).Error
	return total, err
}

func (b *BasicRepositoryImpl[T]) PaginateAndCountAsync(basic *request.Basic, result interface{}, table model.SysTable) (int64, error) {
	if basic == nil {
		basic = &request.Basic{}
	}
	query := util.ExecuteQuery(b.baseQuery(), basic, table)
	query = query.Model(b.model)
	if basic.IncludeDeleted {
		query = query.Unscoped()
	}

	type countResult struct {
		total int64
		err   error
	}
	countChan := make(chan countResult, 1)
	dataErrChan := make(chan error, 1)

	// 异步查询总数
	go func() {
		// 为计数查询创建独立的 query 对象
		countQuery := query.Session(&gorm.Session{})
		t, e := b.Count(countQuery)
		countChan <- countResult{total: t, err: e}
	}()

	// 分页查询
	go func() {
		// 为数据查询创建独立的 query 对象
		dataQuery := b.applyReadOptions(query.Session(&gorm.Session{}))
		if e := dataQuery.Find(result).Error; e != nil {
			zap.L().Error("数据查询出错", zap.Error(e))
			dataErrChan <- e
		} else {
			dataErrChan <- nil
		}
	}()

	var total int64
	for i := 0; i < 2; i++ {
		select {
		case count := <-countChan:
			if count.err != nil {
				return 0, count.err
			}
			total = count.total
		case err := <-dataErrChan:
			if err != nil {
				return 0, err
			}
		}
	}
	return total, nil
}

func (b *BasicRepositoryImpl[T]) Create(tx *gorm.DB, entity interface{}) error {
	return tx.Model(b.model).Create(entity).Error
}

func (b *BasicRepositoryImpl[T]) Update(tx *gorm.DB, entity interface{}, id int) error {
	entityMap := make(map[string]interface{})
	err := mapstructure.Decode(entity, &entityMap)
	if err != nil {
		return err
	}
	delete(entityMap, "Basic")
	delete(entityMap, "basic")
	query := tx.Model(b.model).Where("id = ?", id).Omit("id")
	// 是否更新已删除的记录
	if b.unscoped {
		query = query.Unscoped()
	}
	// 指定更新某些字段
	if b.selects != nil {
		query = query.Select(b.selects)
	}
	return query.Updates(entityMap).Error
}

func (b *BasicRepositoryImpl[T]) DeleteById(tx *gorm.DB, id int) error {
	return tx.Where("id = ?", id).Delete(b.model).Error
}

func (b *BasicRepositoryImpl[T]) DeleteByField(tx *gorm.DB, field string, value interface{}) error {
	if err := validateRepositoryField(field); err != nil {
		return err
	}
	return tx.Where(fmt.Sprintf("%s = ?", field), value).Delete(b.model).Error
}

func (b *BasicRepositoryImpl[T]) DeleteByIds(tx *gorm.DB, ids []int) error {
	return tx.Where("id in ?", ids).Delete(b.model).Error
}

func (b *BasicRepositoryImpl[T]) DeleteByFieldIn(tx *gorm.DB, field string, values []interface{}) error {
	if err := validateRepositoryField(field); err != nil {
		return err
	}
	return tx.Where(fmt.Sprintf("%s in ?", field), values).Delete(b.model).Error
}

func (b *BasicRepositoryImpl[T]) FindById(id int) (T, error) {
	var entity T
	query := b.applyReadOptions(b.baseQuery())
	err := query.First(&entity, id).Error
	return entity, err
}

func (b *BasicRepositoryImpl[T]) FindListById(id int) ([]T, error) {
	var entity []T
	query := b.applyReadOptions(b.baseQuery())
	err := query.Find(&entity, id).Error
	return entity, err
}

func (b *BasicRepositoryImpl[T]) FindByField(field string, value interface{}) (T, error) {
	var entity T
	if err := validateRepositoryField(field); err != nil {
		return entity, err
	}
	query := b.applyReadOptions(b.baseQuery())
	err := query.Where(fmt.Sprintf("%s = ?", field), value).First(&entity).Error
	return entity, err
}

func (b *BasicRepositoryImpl[T]) FindListByField(field string, value interface{}) ([]T, error) {
	var entity []T
	if err := validateRepositoryField(field); err != nil {
		return nil, err
	}
	query := b.applyReadOptions(b.baseQuery())
	err := query.Where(fmt.Sprintf("%s = ?", field), value).Find(&entity).Error
	return entity, err
}

func (b *BasicRepositoryImpl[T]) FindListByFieldIn(field string, values interface{}) ([]T, error) {
	var entities []T
	if err := validateRepositoryField(field); err != nil {
		return nil, err
	}
	// 检查是否为切片类型
	val := reflect.ValueOf(values)
	if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
		return nil, fmt.Errorf("%w: got %T", repository.ErrInvalidFieldValues, values)
	}
	// 转换为[]interface{}
	valueSlice := make([]interface{}, val.Len())
	for i := 0; i < val.Len(); i++ {
		valueSlice[i] = val.Index(i).Interface()
	}
	query := b.applyReadOptions(b.baseQuery())
	err := query.Model(b.model).Where(fmt.Sprintf("%s IN ?", field), valueSlice).Find(&entities).Error
	return entities, err
}

func (b *BasicRepositoryImpl[T]) WithPreload(preloads ...string) repository.BasicRepository[T] {
	newImpl := b.clone()
	// 判断perloads是否为空，如果为空则直接返回
	if len(preloads) == 0 {
		return newImpl
	}
	newImpl.preloads = append(newImpl.preloads, preloads...)
	return newImpl
}

func (b *BasicRepositoryImpl[T]) WithUnscoped() repository.BasicRepository[T] {
	newImpl := b.clone()
	newImpl.unscoped = true
	return newImpl
}

func (b *BasicRepositoryImpl[T]) WithSelect(selects ...string) repository.BasicRepository[T] {
	newImpl := b.clone()
	// 判断selects是否为空，如果为空则直接返回
	if len(selects) == 0 {
		return newImpl
	}
	newImpl.selects = append(newImpl.selects, selects...)
	return newImpl
}

func (b *BasicRepositoryImpl[T]) WithOmit(omits ...string) repository.BasicRepository[T] {
	newImpl := b.clone()
	// 判断omits是否为空，如果为空则直接返回
	if len(omits) == 0 {
		return newImpl
	}
	newImpl.omits = append(newImpl.omits, omits...)
	return newImpl
}

func (b *BasicRepositoryImpl[T]) WithContext(ctx *gin.Context) repository.BasicRepository[T] {
	newImpl := b.clone()
	newImpl.ctx = ctx
	return newImpl
}

func (b *BasicRepositoryImpl[T]) baseQuery() *gorm.DB {
	query := b.db
	if b.ctx != nil {
		query = query.WithContext(b.ctx)
	}
	if b.unscoped {
		query = query.Unscoped()
	}
	return query
}

func (b *BasicRepositoryImpl[T]) applyReadOptions(query *gorm.DB) *gorm.DB {
	if len(b.selects) > 0 {
		query = query.Select(b.selects)
	}
	if len(b.omits) > 0 {
		query = query.Omit(b.omits...)
	}
	for _, preload := range b.preloads {
		query = query.Preload(preload)
	}
	return query
}

func (b *BasicRepositoryImpl[T]) clone() *BasicRepositoryImpl[T] {
	newImpl := *b
	newImpl.preloads = append([]string(nil), b.preloads...)
	newImpl.selects = append([]string(nil), b.selects...)
	newImpl.omits = append([]string(nil), b.omits...)
	return &newImpl
}

func validateRepositoryField(field string) error {
	field = strings.TrimSpace(field)
	if field == "" {
		return fmt.Errorf("%w: field is empty", repository.ErrInvalidField)
	}
	for _, part := range strings.Split(field, ".") {
		if !isRepositoryIdentifier(part) {
			return fmt.Errorf("%w: %q", repository.ErrInvalidField, field)
		}
	}
	return nil
}

func isRepositoryIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for i, char := range value {
		if char == '_' || char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' {
			continue
		}
		if i > 0 && char >= '0' && char <= '9' {
			continue
		}
		return false
	}
	return true
}
