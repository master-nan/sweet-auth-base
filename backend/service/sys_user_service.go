/**
 * @Author: Nan
 * @Date: 2024/5/24 下午10:20
 */

package service

import (
	"backend/config"
	"backend/dto/request"
	"backend/dto/response"
	"backend/internal/cache"
	error2 "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jinzhu/copier"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type SysUserService struct {
	sysUserRepo     repository.SysUserRepository
	sysUserRoleRepo repository.SysUserRoleRepository
	sf              *utils.Snowflake
	sysUserCache    *cache.SysUserCache
	serverConfig    *config.Server
}

type userPasswordStateUpdate struct {
	Password          string
	IsReset           *bool
	PasswordChangedAt *model.CustomTime
}

func NewSysUserService(
	sysUserRepo repository.SysUserRepository,
	sysUserRoleRepo repository.SysUserRoleRepository,
	sf *utils.Snowflake,
	sysUserCache *cache.SysUserCache,
	serverConfig *config.Server,
) *SysUserService {
	return &SysUserService{
		sysUserRepo,
		sysUserRoleRepo,
		sf,
		sysUserCache,
		serverConfig,
	}
}

// GetByUserName 根据username获取用户信息
func (s *SysUserService) GetByUserName(username string) (model.SysUser, error) {
	data, err := s.sysUserCache.Get(username)
	if err == nil {
		return data, nil
	}
	if !errors.Is(err, cache.ErrCacheMiss) {
		return model.SysUser{}, err
	}
	data, err = s.sysUserRepo.GetByUserName(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.SysUser{}, nil
		}
		return model.SysUser{}, err
	}
	// 将用户按照id、username以及手机号缓存
	s.sysUserCache.Set(strconv.Itoa(data.Id), data)
	s.sysUserCache.Set(data.UserName, data)
	s.sysUserCache.Set(data.PhoneNumber, data)
	return data, nil
}

// GetById 根据id获取用户信息
func (s *SysUserService) GetById(id int) (model.SysUser, error) {
	data, err := s.sysUserCache.Get(strconv.Itoa(id))
	if err == nil {
		return data, nil
	}
	if !errors.Is(err, cache.ErrCacheMiss) {
		return model.SysUser{}, err
	}
	data, err = s.sysUserRepo.WithPreload("Roles").FindById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.SysUser{}, nil
		}
		return model.SysUser{}, err
	}
	// 将用户按照id、username以及手机号缓存
	s.sysUserCache.Set(strconv.Itoa(data.Id), data)
	s.sysUserCache.Set(data.UserName, data)
	s.sysUserCache.Set(data.PhoneNumber, data)
	return data, nil
}

func (s *SysUserService) GetUserList(basic *request.Basic, table model.SysTable) (response.ListResult[model.SysUser], error) {
	result, err := s.sysUserRepo.GetUserList(basic, table)
	return result, err
}

func (s *SysUserService) Create(ctx context.Context, req request.SysUserCreateReq) error {
	var data model.SysUser
	user, e := s.GetByUserName(req.UserName)
	if e != nil {
		return e
	}
	if user.Id != 0 {
		return error2.ErrUserExist
	}

	pwd := strings.TrimSpace(req.Password)
	if pwd == "" {
		return error2.ErrPasswordEmpty
	}
	now := model.CustomTime(time.Now().UTC())
	err := copier.Copy(&data, &req)
	if err != nil {
		zap.L().Error("结构体字段映射失败", zap.String("target", "SysUser"), zap.Error(err))
		return err
	}
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return err
	}
	data.Id = int(id)
	salt := strconv.Itoa(data.Id) + s.serverConfig.Conf.Salt
	data.Password = utils.Encryption(pwd, salt)
	data.PasswordChangedAt = &now
	tx := s.sysUserRepo.DBWithContext(ctx)
	return s.sysUserRepo.Create(tx, &data)
}

func (s *SysUserService) Update(ctx context.Context, req request.SysUserUpdateReq, selects ...string) error {
	tx := s.sysUserRepo.DBWithContext(ctx)
	err := s.sysUserRepo.WithSelect(selects...).Update(tx, &req, req.Id)
	if err != nil {
		return err
	}
	// 删除缓存
	s.RefreshCache(req.Id)
	return nil
}

// UpdatePassword 更新密码
func (s *SysUserService) UpdatePassword(ctx context.Context, req request.SysUserUpdatePasswordReq) error {
	user, err := s.GetById(req.Id)
	if err != nil {
		return err
	}
	if user.Id == 0 {
		return error2.ErrUserNotExist
	}

	pwd := strings.TrimSpace(req.Password)
	if pwd == "" {
		return error2.ErrPasswordEmpty
	}
	now := model.CustomTime(time.Now().UTC())
	// 存储加密后的密码
	salt := strconv.Itoa(user.Id) + s.serverConfig.Conf.Salt
	update := userPasswordStateUpdate{
		Password: utils.Encryption(pwd, salt), IsReset: utils.BoolPtr(false), PasswordChangedAt: &now,
	}

	tx := s.sysUserRepo.DBWithContext(ctx)
	err = s.sysUserRepo.Update(tx, &update, req.Id)
	if err != nil {
		return err
	}
	// 删除缓存
	s.RefreshCache(req.Id)
	return nil
}

func (s *SysUserService) ResetPassword(ctx context.Context, id int, password string) error {
	user, err := s.GetById(id)
	if err != nil {
		return err
	}
	if user.Id == 0 {
		return error2.ErrUserNotExist
	}

	pwd := strings.TrimSpace(password)
	if pwd == "" {
		return error2.ErrPasswordEmpty
	}
	now := model.CustomTime(time.Now().UTC())
	salt := strconv.Itoa(user.Id) + s.serverConfig.Conf.Salt
	updatePasswordReq := userPasswordStateUpdate{
		Password:          utils.Encryption(pwd, salt),
		IsReset:           utils.BoolPtr(true),
		PasswordChangedAt: &now,
	}

	tx := s.sysUserRepo.DBWithContext(ctx)
	if err := RunInTransaction(ctx, tx, func(tx *gorm.DB) error {
		if err := s.sysUserRepo.Update(tx, &updatePasswordReq, id); err != nil {
			return err
		}
		_, err := s.sysUserRepo.UpdateFields(tx, id, map[string]any{"access_tokens": ""})
		return err
	}); err != nil {
		return err
	}
	s.RefreshCache(id)
	return nil
}

func (s *SysUserService) Delete(ctx context.Context, id int) error {
	tx := s.sysUserRepo.DBWithContext(ctx)
	err := s.sysUserRepo.DeleteById(tx, id)
	if err != nil {
		return err
	}
	// 删除缓存
	s.DeleteCache(id)
	return nil
}

func (s *SysUserService) AssignRoles(ctx context.Context, userId int, roleIds []int) error {
	if userId <= 0 {
		return error2.ErrParamInvalid
	}
	user, err := s.GetById(userId)
	if err != nil {
		return err
	}
	if user.Id == 0 {
		return error2.ErrUserNotExist
	}
	normalizedRoleIds := make([]int, 0, len(roleIds))
	roleIDSet := make(map[int]bool, len(roleIds))
	for _, roleId := range roleIds {
		if roleId <= 0 || roleIDSet[roleId] {
			continue
		}
		roleIDSet[roleId] = true
		normalizedRoleIds = append(normalizedRoleIds, roleId)
	}
	if len(normalizedRoleIds) == 0 {
		return error2.NewBadRequestError("用户至少需要绑定一个角色")
	}
	tx := s.sysUserRepo.DBWithContext(ctx)
	var count int64
	if err := tx.Model(&model.SysRole{}).Where("id IN ?", normalizedRoleIds).Count(&count).Error; err != nil {
		return err
	}
	if int(count) != len(normalizedRoleIds) {
		return error2.NewBadRequestError("存在无效角色")
	}
	err = RunInTransaction(ctx, tx, func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userId).Delete(&model.SysUserRole{}).Error; err != nil {
			return err
		}
		if len(normalizedRoleIds) == 0 {
			return nil
		}
		userRoles := make([]model.SysUserRole, 0, len(normalizedRoleIds))
		for _, roleId := range normalizedRoleIds {
			userRoles = append(userRoles, model.SysUserRole{
				UserId: userId,
				RoleId: roleId,
			})
		}
		return tx.Create(&userRoles).Error
	})
	if err != nil {
		return err
	}
	s.RefreshCache(userId)
	return nil
}

// RefreshCache 刷新缓存
func (s *SysUserService) RefreshCache(userId int) {
	go func() {
		data, err := s.sysUserRepo.WithPreload("Roles").FindById(userId)
		if err == nil {
			if data.Id != 0 {
				s.sysUserCache.Set(strconv.Itoa(data.Id), data)
				s.sysUserCache.Set(data.UserName, data)
				s.sysUserCache.Set(data.PhoneNumber, data)
			}
		}
	}()
}

func (s *SysUserService) DeleteCache(userId int) {
	go func() {
		data, _ := s.GetById(userId)
		if data.Id != 0 {
			s.sysUserCache.Delete(strconv.Itoa(data.Id))
			s.sysUserCache.Delete(data.UserName)
			s.sysUserCache.Delete(data.PhoneNumber)
		}
	}()
}

func PasswordChangeRequirement(user model.SysUser, cfg model.SysConfigure, now time.Time) (bool, string) {
	if user.IsReset {
		return true, "initial_reset"
	}
	expireDays := cfg.PasswordExpireTime
	if expireDays <= 0 {
		return false, ""
	}
	changedAt := passwordChangedAt(user)
	if changedAt.IsZero() {
		return true, "unknown_changed_at"
	}
	if now.After(changedAt.AddDate(0, 0, expireDays)) {
		return true, "expired"
	}
	return false, ""
}

func passwordChangedAt(user model.SysUser) time.Time {
	if user.PasswordChangedAt != nil && !time.Time(*user.PasswordChangedAt).IsZero() {
		return time.Time(*user.PasswordChangedAt)
	}
	return time.Time(user.GmtModify)
}

func boolPtr(value bool) *bool {
	return &value
}

// 获取所有用户

func (s *SysUserService) GetAll(ctx context.Context) ([]model.SysUser, error) {
	var userList []model.SysUser
	err := s.sysUserRepo.DBWithContext(ctx).Find(&userList).Error
	if err != nil {
		return nil, err
	}
	return userList, nil
}
