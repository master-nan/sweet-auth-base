/**
 * @Author: Nan
 * @Date: 2025/2/7 21:55
 */

package impl

import (
	"backend/internal/database"
	"backend/model"
	"context"
	"gorm.io/gorm"
)

type SmsLogImpl struct {
	db *gorm.DB
	*BasicRepositoryImpl[model.SmsLog]
}

func NewSmsLogImpl(primaryDB *database.PrimaryDB) *SmsLogImpl {
	return &SmsLogImpl{
		primaryDB.DB,
		NewBasicRepositoryImpl(primaryDB.DB, &model.SmsLog{}),
	}
}

func (s *SmsLogImpl) CreateSmsLogContext(ctx context.Context, log *model.SmsLog) error {
	return s.db.WithContext(ctx).Create(log).Error
}
