/**
 * @Author: Nan
 * @Date: 2024/7/25 下午11:05
 */

package service

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/enum"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"context"
	"fmt"
	"strings"

	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SysRoleService 管理Role及菜单、按钮授权事务，不以角色名称代替Capability判断。
type SysRoleService struct {
	sysMenuButtonRepo     repository.SysMenuButtonRepository
	sysRoleRepo           repository.SysRoleRepository
	sysRoleMenuRepo       repository.SysRoleMenuRepository
	sysRoleMenuButtonRepo repository.SysRoleMenuButtonRepository
	casbinRuleRepo        repository.CasbinRuleRepository
	sf                    *utils.Snowflake
}

func NewSysRoleService(
	sysMenuButtonRepo repository.SysMenuButtonRepository,
	sysRoleRepo repository.SysRoleRepository,
	sysRoleMenuRepo repository.SysRoleMenuRepository,
	sysRoleMenuButtonRepo repository.SysRoleMenuButtonRepository,
	casbinRuleRepo repository.CasbinRuleRepository,
	sf *utils.Snowflake,
) *SysRoleService {
	return &SysRoleService{
		sysMenuButtonRepo,
		sysRoleRepo,
		sysRoleMenuRepo,
		sysRoleMenuButtonRepo,
		casbinRuleRepo,
		sf,
	}
}

func (s *SysRoleService) GetRoleById(id int) (model.SysRole, error) {
	data, err := s.sysRoleRepo.WithPreload("Menus", "Buttons").FindById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.SysRole{}, nil
		}
		return model.SysRole{}, err
	}
	return data, nil
}

func (s *SysRoleService) GetRoleList(basic *request.Basic, table model.SysTable) (response.ListResult[model.SysRole], error) {
	return s.sysRoleRepo.GetRoleList(basic, table)
}

func (s *SysRoleService) CreateRole(ctx context.Context, req request.RoleCreateReq) error {
	var role model.SysRole
	err := copier.Copy(&role, &req)
	if err != nil {
		zap.L().Error("结构体字段映射失败", zap.String("target", "SysRole"), zap.Error(err))
		return err
	}
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return err
	}
	role.Id = int(id)
	return s.sysRoleRepo.Create(s.sysRoleRepo.DBWithContext(ctx), &role)
}

func (s *SysRoleService) UpdateRole(ctx context.Context, req request.RoleUpdateReq) error {
	return s.sysRoleRepo.Update(s.sysRoleRepo.DBWithContext(ctx), &req, req.Id)
}

func (s *SysRoleService) DeleteRole(ctx context.Context, id int) error {
	return s.sysRoleRepo.DeleteById(s.sysRoleRepo.DBWithContext(ctx), id)
}

func (s *SysRoleService) GetRoleMenus(roleId int) ([]model.SysMenu, error) {
	menus, err := s.sysRoleMenuRepo.GetRoleMenus(roleId)
	if err != nil {
		return nil, err
	}
	return utils.BuildMenuTree(utils.SortMenuTree(menus), 0), nil
}

func (s *SysRoleService) GetRoleMenuButtons(roleId, menuId int) ([]model.SysMenuButton, error) {
	return s.sysRoleMenuButtonRepo.GetRoleMenuButtons(roleId, menuId)
}

// GetRoleMenuButtons 获取角色菜单按钮权限
func (s *SysMenuService) GetRoleMenuButtons(roleId, menuId int) ([]model.SysMenuButton, error) {
	return s.sysRoleMenuButtonRepo.GetRoleMenuButtons(roleId, menuId)
}

// AssignPermissions 分配角色权限
func (s *SysRoleService) AssignPermissions(ctx context.Context, data request.RoleAssignPermissionsReq) error {
	role, err := s.sysRoleRepo.FindById(data.RoleId)
	if err != nil {
		return err
	}
	if role.Id == 0 {
		return nil
	}
	menuIDs := uniquePositiveInts(data.MenuIds)
	buttonIDs := uniquePositiveInts(data.ButtonIds)
	menuIDSet := intSet(menuIDs)
	var assignableButtons []model.SysMenuButton
	if len(buttonIDs) > 0 {
		buttons, err := s.sysMenuButtonRepo.FindListByFieldIn("id", buttonIDs)
		if err != nil {
			return err
		}
		assignableButtons = filterAssignableRoleButtons(buttons, menuIDSet)
	}
	oldPolicies, err := quiesceCasbinSubject(s.casbinRuleRepo, role.Name)
	if err != nil {
		return fmt.Errorf("casbin策略暂时撤下失败: %w", err)
	}

	err = RunInTransaction(ctx, s.sysRoleRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		// 删除旧的角色菜单
		if err := s.sysRoleMenuRepo.DeleteByField(tx, "role_id", data.RoleId); err != nil {
			return err
		}
		// 删除旧的角色按钮
		if err := s.sysRoleMenuButtonRepo.DeleteByField(tx, "role_id", data.RoleId); err != nil {
			return err
		}
		// 添加新的角色菜单
		for _, menuId := range menuIDs {
			roleMenu := model.SysRoleMenu{
				RoleId: data.RoleId,
				MenuId: menuId,
			}
			if err := s.sysRoleMenuRepo.Create(tx, &roleMenu); err != nil {
				return err
			}
		}
		// 添加新的角色按钮
		for _, button := range assignableButtons {
			roleMenuButton := model.SysRoleMenuButton{
				RoleId:   data.RoleId,
				MenuId:   button.MenuId,
				ButtonId: button.Id,
			}
			if err := s.sysRoleMenuButtonRepo.Create(tx, &roleMenuButton); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if restoreErr := s.casbinRuleRepo.ReplaceSubjectPolicies(role.Name, oldPolicies); restoreErr != nil {
			return fmt.Errorf("角色权限事务失败且casbin恢复失败: %v: %w", restoreErr, err)
		}
		return err
	}

	// 先构建新策略列表
	var newPolicies [][]string
	if len(assignableButtons) > 0 {
		policySet := make(map[string]struct{})
		for _, button := range assignableButtons {
			for _, policy := range buttonAPIPolicies(button) {
				key := fmt.Sprintf("%s|%s", policy.Path, policy.Method)
				if _, ok := policySet[key]; !ok {
					policySet[key] = struct{}{}
					newPolicies = append(newPolicies, []string{role.Name, policy.Path, policy.Method})
				}
			}
		}
	}

	if err = s.casbinRuleRepo.ReplaceSubjectPolicies(role.Name, newPolicies); err != nil {
		_ = s.casbinRuleRepo.ReplaceSubjectPolicies(role.Name, nil)
		return fmt.Errorf("casbin策略更新失败: %w", err)
	}

	return nil
}

type buttonAPIPolicy struct {
	Path   string
	Method string
}

func buttonAPIPolicies(button model.SysMenuButton) []buttonAPIPolicy {
	policies := make([]buttonAPIPolicy, 0, 2)
	path := strings.TrimSpace(button.Path)
	method := strings.ToUpper(strings.TrimSpace(button.Method))
	if path != "" && method != "" {
		policies = append(policies, buttonAPIPolicy{Path: path, Method: method})
	}

	action, ok := enum.NormalizeSysMenuButtonEventAction(button.EventAction)
	if path == "" && method == "" && ok && action == enum.ButtonActionDetail {
		policies = append(policies, buttonAPIPolicy{
			Path:   "/admin/generalization/detail/code/:code/:id",
			Method: "GET",
		})
	}
	return policies
}

func uniquePositiveInts(ids []int) []int {
	result := make([]int, 0, len(ids))
	seen := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func intSet(ids []int) map[int]bool {
	result := make(map[int]bool, len(ids))
	for _, id := range ids {
		result[id] = true
	}
	return result
}

func filterAssignableRoleButtons(buttons []model.SysMenuButton, menuIDSet map[int]bool) []model.SysMenuButton {
	if len(buttons) == 0 || len(menuIDSet) == 0 {
		return []model.SysMenuButton{}
	}
	result := make([]model.SysMenuButton, 0, len(buttons))
	for _, button := range buttons {
		if menuIDSet[button.MenuId] && button.State && !button.IsDisabled {
			result = append(result, button)
		}
	}
	return result
}
