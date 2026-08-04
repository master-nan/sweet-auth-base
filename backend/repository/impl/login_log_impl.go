/**
 * @Author: Nan
 * @Date: 2024/6/3 下午4:31
 */

package impl

import (
	"backend/internal/database"
	"backend/model"
	"context"
	"gorm.io/gorm"
)

type LoginLogRepositoryImpl struct {
	db *gorm.DB
	*BasicRepositoryImpl[model.LoginLog]
}

func NewLoginLogRepositoryImpl(PrimaryDB *database.PrimaryDB) *LoginLogRepositoryImpl {
	return &LoginLogRepositoryImpl{
		PrimaryDB.DB,
		NewBasicRepositoryImpl(PrimaryDB.DB, &model.LoginLog{}),
	}
}

func (l *LoginLogRepositoryImpl) CreateLoginLogContext(ctx context.Context, log *model.LoginLog) error {
	return l.db.WithContext(ctx).Create(log).Error
}
