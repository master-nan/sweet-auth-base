/**
 * @Author: Nan
 * @Date: 2025/2/7 21:55
 */

package impl

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/internal/database"
	"backend/model"
	"backend/repository/util"
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

func (s *SmsLogImpl) GetSmsLogList(basic *request.Basic) (response.ListResult[model.SmsLog], error) {
	var repo response.ListResult[model.SmsLog]
	var table model.SysTable
	query := util.ExecuteQuery(s.db, basic, table)
	var smsLogList []model.SmsLog
	var total int64 = 0
	err := query.Find(&smsLogList).Limit(-1).Offset(-1).Count(&total).Error
	repo.Data = smsLogList
	repo.Total = int(total)
	return repo, err
}
