/**
 * @Author: Nan
 * @Date: 2024/6/3 下午6:08
 */

package impl

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/internal/database"
	"backend/model"
	"backend/repository"
	"context"
	"strings"

	"gorm.io/gorm"
)

func (s *SysUserRepositoryImpl) FindAuthenticationByPrincipal(ctx context.Context, principal string) (model.SysUser, error) {
	var users []model.SysUser
	value := strings.TrimSpace(principal)
	err := s.db.WithContext(ctx).
		Where("user_name = ? OR phone_number = ? OR email = ?", value, value, value).
		Preload("Roles").
		Limit(2).
		Find(&users).Error
	if err != nil {
		return model.SysUser{}, err
	}
	if len(users) == 0 {
		return model.SysUser{}, gorm.ErrRecordNotFound
	}
	if len(users) > 1 {
		return model.SysUser{}, repository.ErrAmbiguousAuthenticationPrincipal
	}
	return users[0], nil
}

func (s *SysUserRepositoryImpl) FindAuthenticationByPhone(ctx context.Context, phone string) (model.SysUser, error) {
	return s.findAuthenticationByExactField(ctx, "phone_number", phone)
}

func (s *SysUserRepositoryImpl) FindAuthenticationByEmail(ctx context.Context, email string) (model.SysUser, error) {
	return s.findAuthenticationByExactField(ctx, "email", email)
}

func (s *SysUserRepositoryImpl) findAuthenticationByExactField(ctx context.Context, field, value string) (model.SysUser, error) {
	var users []model.SysUser
	err := s.db.WithContext(ctx).
		Where(field+" = ?", strings.TrimSpace(value)).
		Preload("Roles").
		Limit(2).
		Find(&users).Error
	if err != nil {
		return model.SysUser{}, err
	}
	if len(users) == 0 {
		return model.SysUser{}, gorm.ErrRecordNotFound
	}
	if len(users) > 1 {
		return model.SysUser{}, repository.ErrAmbiguousAuthenticationPrincipal
	}
	return users[0], nil
}

func (s *SysUserRepositoryImpl) FindAuthenticationByID(ctx context.Context, id int) (model.SysUser, error) {
	var user model.SysUser
	err := s.db.WithContext(ctx).Preload("Roles").First(&user, id).Error
	return user, err
}

func (s *SysUserRepositoryImpl) FindAuthenticationByIDForUpdate(tx *gorm.DB, id int) (model.SysUser, error) {
	return s.FindByIdForUpdate(tx, id)
}

func (s *SysUserRepositoryImpl) UpdateAuthenticationState(tx *gorm.DB, id int, fields map[string]any) error {
	return tx.Model(&model.SysUser{}).Where("id = ?", id).Updates(fields).Error
}

type SysUserRepositoryImpl struct {
	db *gorm.DB
	*BasicRepositoryImpl[model.SysUser]
}

func NewSysUserRepositoryImpl(PrimaryDB *database.PrimaryDB) *SysUserRepositoryImpl {
	return &SysUserRepositoryImpl{PrimaryDB.DB, NewBasicRepositoryImpl(PrimaryDB.DB, &model.SysUser{})}
}

func (s *SysUserRepositoryImpl) GetByUserName(username string) (model.SysUser, error) {
	var user model.SysUser
	err := s.db.Where(&model.SysUser{UserName: username}).Or(&model.SysUser{PhoneNumber: username}).Or(&model.SysUser{Email: username}).Preload("Roles").First(&user).Error
	return user, err
}

func (s *SysUserRepositoryImpl) GetUserList(basic *request.Basic, table model.SysTable) (response.ListResult[model.SysUser], error) {
	var repo response.ListResult[model.SysUser]
	var sysUserList []model.SysUser
	total, err := s.PaginateAndCountAsync(basic, &sysUserList, table)
	repo.Data = sysUserList
	repo.Total = int(total)
	return repo, err
}
