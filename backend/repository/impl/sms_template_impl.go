/**
 * @Author: Nan
 * @Date: 2025/2/7 21:56
 */

package impl

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/internal/database"
	"backend/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type SmsTemplateImpl struct {
	db *gorm.DB
	*BasicRepositoryImpl[model.SmsTemplate]
}

func NewSmsTemplateImpl(primaryDB *database.PrimaryDB) *SmsTemplateImpl {
	return &SmsTemplateImpl{
		primaryDB.DB,
		NewBasicRepositoryImpl(primaryDB.DB, &model.SmsTemplate{}),
	}
}

func (s *SmsTemplateImpl) GetSmsTemplateList(basic *request.Basic, table model.SysTable) (response.ListResult[model.SmsTemplate], error) {
	var repo response.ListResult[model.SmsTemplate]
	var smsTemplateList []model.SmsTemplate
	total, err := s.PaginateAndCountAsync(basic, &smsTemplateList, table)
	zap.L().Debug("smsTemplateList", zap.Any("smsTemplateList", smsTemplateList))
	repo.Data = smsTemplateList
	repo.Total = int(total)
	return repo, err
}
