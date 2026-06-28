/**
 * @Author: Nan
 * @Date: 2024/8/1 下午10:37
 */

package impl

import (
	"backend/enum"
	"backend/internal/database"
	"backend/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SysMenuButtonRepositoryImpl struct {
	db *gorm.DB
	*BasicRepositoryImpl[model.SysMenuButton]
}

func NewSysMenuButtonRepositoryImpl(PrimaryDB *database.PrimaryDB) *SysMenuButtonRepositoryImpl {
	return &SysMenuButtonRepositoryImpl{
		PrimaryDB.DB,
		NewBasicRepositoryImpl(PrimaryDB.DB, &model.SysMenuButton{}),
	}
}

func (s *SysMenuButtonRepositoryImpl) FindByMenuIdAndCode(tx *gorm.DB, menuId int, code string) (model.SysMenuButton, error) {
	var button model.SysMenuButton
	if tx == nil {
		tx = s.db
	}
	err := tx.Where("menu_id = ? AND code = ?", menuId, code).First(&button).Error
	return button, err
}

func (s *SysMenuButtonRepositoryImpl) Create(tx *gorm.DB, entity interface{}) error {
	if tx == nil {
		tx = s.db
	}
	tableName := tableNameForModel(tx, &model.SysMenuButton{})
	switch buttons := entity.(type) {
	case *model.SysMenuButton:
		return tx.Table(tableName).Create(sysMenuButtonCreateMap(tx, buttons)).Error
	case *[]model.SysMenuButton:
		rows := make([]map[string]interface{}, 0, len(*buttons))
		for i := range *buttons {
			rows = append(rows, sysMenuButtonCreateMap(tx, &(*buttons)[i]))
		}
		return tx.Table(tableName).Create(rows).Error
	case []model.SysMenuButton:
		rows := make([]map[string]interface{}, 0, len(buttons))
		for i := range buttons {
			rows = append(rows, sysMenuButtonCreateMap(tx, &buttons[i]))
		}
		return tx.Table(tableName).Create(rows).Error
	default:
		return tx.Model(&model.SysMenuButton{}).Create(entity).Error
	}
}

func sysMenuButtonCreateMap(tx *gorm.DB, button *model.SysMenuButton) map[string]interface{} {
	normalizeSysMenuButtonDefaults(button)
	now := model.CustomTime(model.Now())
	gmtCreate := button.GmtCreate
	if gmtCreate.IsZero() {
		gmtCreate = now
	}
	gmtModify := button.GmtModify
	if gmtModify.IsZero() {
		gmtModify = now
	}
	row := map[string]interface{}{
		"id":            button.Id,
		"gmt_create":    gmtCreate,
		"gmt_modify":    gmtModify,
		"state":         true,
		"menu_id":       button.MenuId,
		"name":          button.Name,
		"code":          button.Code,
		"memo":          button.Memo,
		"position":      button.Position,
		"event_type":    button.EventType,
		"event_action":  button.EventAction,
		"icon":          button.Icon,
		"color":         button.Color,
		"display_mode":  button.DisplayMode,
		"sequence":      button.Sequence,
		"path":          button.Path,
		"method":        button.Method,
		"params_schema": button.ParamsSchema,
		"confirm_text":  button.ConfirmText,
		"disable_when":  button.DisableWhen,
		"is_button":     button.IsButton,
		"is_hidden":     button.IsHidden,
		"is_disabled":   button.IsDisabled,
		"before_hooks":  button.BeforeHooks,
		"after_hooks":   button.AfterHooks,
	}
	if button.State {
		row["state"] = button.State
	}
	if button.CreateUser != nil {
		row["create_user"] = button.CreateUser
	}
	if button.ModifyUser != nil {
		row["modify_user"] = button.ModifyUser
	}
	ctx, ok := tx.Statement.Context.(*gin.Context)
	if ok {
		if userValue, exists := ctx.Get("user"); exists {
			if user, ok := userValue.(model.SysUser); ok {
				row["create_user"] = user.Id
			}
		}
	}
	return row
}

func normalizeSysMenuButtonDefaults(button *model.SysMenuButton) {
	displayMode, ok := enum.NormalizeSysMenuButtonDisplayMode(string(button.DisplayMode))
	if !ok {
		displayMode = enum.ButtonDisplayAuto
	}
	button.DisplayMode = displayMode
}

func (s *SysMenuButtonRepositoryImpl) FindLegacyLowCodeButtons(tx *gorm.DB, menuId int) ([]model.SysMenuButton, error) {
	var buttons []model.SysMenuButton
	if tx == nil {
		tx = s.db
	}
	err := tx.Where("menu_id = ? AND code >= ? AND code < ?", menuId, "system_", "system`").Find(&buttons).Error
	return buttons, err
}

func (s *SysMenuButtonRepositoryImpl) UpdateMenuButtonFields(tx *gorm.DB, id int, values map[string]any) error {
	if tx == nil {
		tx = s.db
	}
	return tx.Model(&model.SysMenuButton{}).Where("id = ?", id).Updates(values).Error
}
