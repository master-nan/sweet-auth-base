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
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
	"gorm.io/gorm"
)

type SysUserService struct {
	sysUserRepo  repository.SysUserRepository
	sf           *utils.Snowflake
	sysUserCache *cache.SysUserCache
	serverConfig *config.Server
}

func NewSysUserService(
	sysUserRepo repository.SysUserRepository,
	sf *utils.Snowflake,
	sysUserCache *cache.SysUserCache,
	serverConfig *config.Server,
) *SysUserService {
	return &SysUserService{
		sysUserRepo,
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

func (s *SysUserService) Create(ctx *gin.Context, req request.SysUserCreateReq) error {
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
		fmt.Println("Error during struct mapping:", err)
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

func (s *SysUserService) Update(ctx *gin.Context, req request.SysUserUpdateReq, selects ...string) error {
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
func (s *SysUserService) UpdatePassword(ctx *gin.Context, req request.SysUserUpdatePasswordReq) error {
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
	req.PasswordChangedAt = &now
	req.IsReset = utils.BoolPtr(false)
	// 存储加密后的密码
	salt := strconv.Itoa(user.Id) + s.serverConfig.Conf.Salt
	req.Password = utils.Encryption(pwd, salt)

	tx := s.sysUserRepo.DBWithContext(ctx)
	err = s.sysUserRepo.Update(tx, &req, req.Id)
	if err != nil {
		return err
	}
	// 删除缓存
	s.RefreshCache(req.Id)
	return nil
}

func (s *SysUserService) ResetPassword(ctx *gin.Context, id int, password string) error {
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
	req := request.SysUserUpdateReq{
		Id:           id,
		AccessTokens: "",
		IsReset:      utils.BoolPtr(true),
	}
	salt := strconv.Itoa(user.Id) + s.serverConfig.Conf.Salt
	updatePasswordReq := request.SysUserUpdatePasswordReq{
		Id:                id,
		Password:          utils.Encryption(pwd, salt),
		IsReset:           req.IsReset,
		PasswordChangedAt: &now,
	}

	tx := s.sysUserRepo.DBWithContext(ctx)
	if err := s.sysUserRepo.Update(tx, &updatePasswordReq, id); err != nil {
		return err
	}
	if err := s.sysUserRepo.WithSelect("access_tokens").Update(tx, &req, id); err != nil {
		return err
	}
	s.RefreshCache(id)
	return nil
}

func (s *SysUserService) Delete(ctx *gin.Context, id int) error {
	tx := s.sysUserRepo.DBWithContext(ctx)
	err := s.sysUserRepo.DeleteById(tx, id)
	if err != nil {
		return err
	}
	// 删除缓存
	s.DeleteCache(id)
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

func (s *SysUserService) GetAll(ctx *gin.Context) ([]model.SysUser, error) {
	var userList []model.SysUser
	err := s.sysUserRepo.DBWithContext(ctx).Find(&userList).Error
	if err != nil {
		return nil, err
	}
	return userList, nil
}
