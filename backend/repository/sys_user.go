/**
 * @Author: Nan
 * @Date: 2024/6/3 下午6:07
 */

package repository

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"
	"context"
	"errors"

	"gorm.io/gorm"
)

var ErrAmbiguousAuthenticationPrincipal = errors.New("authentication principal matches multiple users")

// AuthenticationUserRepository 是认证链读取账号安全事实的权威Repository边界，
// 为保证认证事实实时生效而绕过通用User缓存。
type AuthenticationUserRepository interface {
	DBWithContext(context.Context) *gorm.DB
	FindAuthenticationByPrincipal(context.Context, string) (model.SysUser, error)
	FindAuthenticationByPhone(context.Context, string) (model.SysUser, error)
	FindAuthenticationByEmail(context.Context, string) (model.SysUser, error)
	FindAuthenticationByID(context.Context, int) (model.SysUser, error)
	FindAuthenticationByIDForUpdate(*gorm.DB, int) (model.SysUser, error)
	UpdateAuthenticationState(*gorm.DB, int, map[string]any) error
}

type SysUserRepository interface {
	BasicRepository[model.SysUser]
	GetByUserName(string) (model.SysUser, error)
	GetUserList(*request.Basic, model.SysTable) (response.ListResult[model.SysUser], error)
}
