package service

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/internal/audit"
	myerrors "backend/internal/errors"
	"backend/internal/queryscheme"
	"backend/model"
	"backend/repository"
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"gorm.io/gorm"
)

// GetScopeConfig 在当前用户拥有页面权限时返回Scope运行配置和菜单展示名称。
func (service *QuerySchemeService) GetScopeConfig(
	ctx context.Context,
	scopeCode string,
) (response.QueryScopeConfigRes, error) {
	_, config, menu, err := service.authorizeScope(ctx, scopeCode)
	if err != nil {
		return response.QueryScopeConfigRes{}, err
	}
	return scopeConfigResponse(menu.Id, strings.TrimSpace(scopeCode), menu.Title, config), nil
}

// Available 只返回当前用户可见的方案摘要，不加载完整Query Payload。
func (service *QuerySchemeService) Available(
	ctx context.Context,
	scopeCode string,
) ([]response.QuerySchemeSummaryRes, error) {
	actor, config, _, err := service.authorizeScope(ctx, scopeCode)
	if err != nil {
		return nil, err
	}
	values, err := service.repository.FindVisibleByScope(ctx, actor.UserID, actor.RoleIDs, strings.TrimSpace(scopeCode))
	if err != nil {
		return nil, myerrors.WrapDatabaseError(err)
	}
	result := make([]response.QuerySchemeSummaryRes, 0, len(values))
	for _, value := range values {
		payload, decodeErr := queryscheme.DecodePayload(value.QueryPayload)
		status := queryscheme.ValidationInvalid
		if decodeErr == nil && value.QuerySchemaVersion == queryscheme.SchemaVersion {
			schemaResult := queryscheme.ValidateSchema(queryscheme.Normalize(payload))
			status = schemaResult.Status
			if status == queryscheme.ValidationValid {
				metadataResult, metadataErr := service.validator.ValidateMetadata(ctx, config, payload)
				if metadataErr != nil {
					return nil, myerrors.WrapSystemError(metadataErr)
				}
				status = metadataResult.Status
			}
		}
		result = append(result, response.QuerySchemeSummaryRes{
			ID: value.Id, Name: value.Name, Type: value.SchemeType, IsDefault: value.IsDefault, Status: status,
		})
	}
	return result, nil
}

// Resolve 重新校验可见性、revision、Metadata和Binding；
// DEGRADED或INVALID方案不会返回可执行Query。
func (service *QuerySchemeService) Resolve(
	ctx context.Context,
	id int,
	req request.QuerySchemeResolveReq,
) (response.QuerySchemeResolveRes, error) {
	actor, config, _, err := service.authorizeScope(ctx, req.ScopeCode)
	if err != nil {
		return response.QuerySchemeResolveRes{}, err
	}
	value, err := service.visibleScheme(ctx, actor, id, strings.TrimSpace(req.ScopeCode))
	if err != nil {
		return response.QuerySchemeResolveRes{}, err
	}
	if req.ExpectedRevision != nil && value.Revision != *req.ExpectedRevision {
		return response.QuerySchemeResolveRes{}, myerrors.ErrQuerySchemeRevisionConflict
	}
	payload, decodeErr := queryscheme.DecodePayload(value.QueryPayload)
	if decodeErr != nil || value.QuerySchemaVersion != queryscheme.SchemaVersion {
		return resolveResponse(value, queryscheme.ValidationResult{
			Status: queryscheme.ValidationInvalid,
			Issues: []queryscheme.ValidationIssue{{Code: queryscheme.IssuePayloadInvalid, Message: "查询方案结构不合法"}},
		}, nil, nil), nil
	}
	payload = queryscheme.Normalize(payload)
	schemaResult := queryscheme.ValidateSchema(payload)
	if schemaResult.Status != queryscheme.ValidationValid {
		return resolveResponse(value, schemaResult, nil, payload.Bindings), nil
	}
	metadataResult, err := service.validator.ValidateMetadata(ctx, config, payload)
	if err != nil {
		return response.QuerySchemeResolveRes{}, myerrors.WrapSystemError(err)
	}
	if metadataResult.Status != queryscheme.ValidationValid {
		return resolveResponse(value, metadataResult, nil, payload.Bindings), nil
	}
	for _, binding := range payload.Bindings {
		if binding.Kind != queryscheme.BindingCurrentEmployee {
			continue
		}
		employeeID, employeeErr := service.repository.EmployeeID(ctx, actor.UserID)
		if employeeErr != nil && !errors.Is(employeeErr, gorm.ErrRecordNotFound) {
			return response.QuerySchemeResolveRes{}, myerrors.WrapDatabaseError(employeeErr)
		}
		actor.EmployeeID = employeeID
		break
	}
	resolved, err := service.bindings.Resolve(ctx, payload, config, actor)
	if err != nil {
		result := queryscheme.ValidationResult{Status: queryscheme.ValidationDegraded, Issues: []queryscheme.ValidationIssue{{
			Code: queryscheme.IssueBindingUnresolvable, Message: "动态条件当前无法解析",
		}}}
		return resolveResponse(value, result, nil, payload.Bindings), nil
	}
	return resolveResponse(value, metadataResult, &resolved, payload.Bindings), nil
}

func (service *QuerySchemeService) List(
	ctx context.Context,
	req request.QuerySchemeManagementQueryReq,
) (response.ListResult[response.QuerySchemeListRes], error) {
	actor, err := service.actor(ctx)
	if err != nil {
		return response.ListResult[response.QuerySchemeListRes]{}, err
	}
	sharedManager, err := service.repository.HasSharedManageCapability(ctx, actor.UserID, queryscheme.SharedManageCapability)
	if err != nil {
		return response.ListResult[response.QuerySchemeListRes]{}, myerrors.WrapDatabaseError(err)
	}
	if req.ScopeCode != "" {
		if _, _, _, err := service.authorizeScope(ctx, req.ScopeCode); err != nil {
			return response.ListResult[response.QuerySchemeListRes]{}, err
		}
	}
	page, err := service.repository.List(ctx, repository.QuerySchemeListFilter{
		Name: req.Name, ScopeCode: strings.TrimSpace(req.ScopeCode), SchemeType: req.SchemeType,
		Enabled: req.Enabled, Page: req.Page, Num: req.Num, UserID: actor.UserID,
		RoleIDs: actor.RoleIDs, SharedManager: sharedManager,
	})
	if err != nil {
		return response.ListResult[response.QuerySchemeListRes]{}, myerrors.WrapDatabaseError(err)
	}
	scopeCodes := make([]string, 0, len(page.Data))
	roleSchemeIDs := make([]int, 0, len(page.Data))
	seenScopes := make(map[string]struct{}, len(page.Data))
	for _, value := range page.Data {
		if _, exists := seenScopes[value.ScopeCode]; !exists {
			seenScopes[value.ScopeCode] = struct{}{}
			scopeCodes = append(scopeCodes, value.ScopeCode)
		}
		if value.SchemeType == model.QuerySchemeTypeRole {
			roleSchemeIDs = append(roleSchemeIDs, value.Id)
		}
	}
	scopeLabels, err := service.repository.FindActiveScopeLabels(ctx, scopeCodes)
	if err != nil {
		return response.ListResult[response.QuerySchemeListRes]{}, myerrors.WrapDatabaseError(err)
	}
	roleIDsByScheme, err := service.repository.FindRoleIDsBySchemeIDs(ctx, roleSchemeIDs)
	if err != nil {
		return response.ListResult[response.QuerySchemeListRes]{}, myerrors.WrapDatabaseError(err)
	}
	result := make([]response.QuerySchemeListRes, 0, len(page.Data))
	for _, value := range page.Data {
		item, err := service.listResponse(ctx, value, scopeLabels[value.ScopeCode], roleIDsByScheme[value.Id])
		if err != nil {
			return response.ListResult[response.QuerySchemeListRes]{}, err
		}
		result = append(result, item)
	}
	return response.ListResult[response.QuerySchemeListRes]{Data: result, Total: int(page.Total)}, nil
}

func (service *QuerySchemeService) Detail(ctx context.Context, id int) (response.QuerySchemeDetailRes, error) {
	actor, err := service.actor(ctx)
	if err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	value, err := service.repository.FindByIDWithDB(service.repository.DBWithContext(ctx), id, false)
	if err != nil {
		return response.QuerySchemeDetailRes{}, querySchemeLookupError(err)
	}
	if _, _, _, err := service.authorizeScope(ctx, value.ScopeCode); err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	sharedManager, managerErr := service.repository.HasSharedManageCapability(ctx, actor.UserID, queryscheme.SharedManageCapability)
	if managerErr != nil {
		return response.QuerySchemeDetailRes{}, myerrors.WrapDatabaseError(managerErr)
	}
	visible, err := service.isVisible(ctx, actor, value)
	if err != nil {
		return response.QuerySchemeDetailRes{}, myerrors.WrapDatabaseError(err)
	}
	if !visible && !(sharedManager && sharedSchemeType(value.SchemeType)) {
		return response.QuerySchemeDetailRes{}, myerrors.ErrQuerySchemeNotFound
	}
	return service.buildDetail(ctx, value)
}

func (service *QuerySchemeService) CopyToPersonal(
	ctx context.Context,
	id int,
	req request.QuerySchemeCopyReq,
) (response.QuerySchemeDetailRes, error) {
	actor, _, _, err := service.authorizeScope(ctx, req.ScopeCode)
	if err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	source, err := service.visibleScheme(ctx, actor, id, strings.TrimSpace(req.ScopeCode))
	if err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	if source.SchemeType == model.QuerySchemeTypePersonal {
		return response.QuerySchemeDetailRes{}, myerrors.ErrQuerySchemeTypeInvalid
	}
	payload, err := queryscheme.DecodePayload(source.QueryPayload)
	if err != nil {
		return response.QuerySchemeDetailRes{}, myerrors.ErrQuerySchemeInvalid
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return response.QuerySchemeDetailRes{}, myerrors.WrapSystemError(err)
	}
	return service.CreatePersonal(ctx, request.QuerySchemePersonalCreateReq{
		Name: req.Name, ScopeCode: req.ScopeCode, Payload: raw, IsDefault: req.IsDefault,
	})
}

func (service *QuerySchemeService) authorizeScope(
	ctx context.Context,
	scopeCode string,
) (queryscheme.Subject, queryscheme.ScopeConfig, model.SysMenu, error) {
	actor, err := service.actor(ctx)
	if err != nil {
		return queryscheme.Subject{}, queryscheme.ScopeConfig{}, model.SysMenu{}, err
	}
	config, menu, err := service.authorizeScopeForActor(ctx, actor, scopeCode)
	return actor, config, menu, err
}

func (service *QuerySchemeService) authorizeScopeForActor(
	ctx context.Context,
	actor queryscheme.Subject,
	scopeCode string,
) (queryscheme.ScopeConfig, model.SysMenu, error) {
	scopeCode = strings.TrimSpace(scopeCode)
	config, exists := service.scopes.Get(ctx, scopeCode)
	if !exists {
		return queryscheme.ScopeConfig{}, model.SysMenu{}, myerrors.ErrQuerySchemeScopeNotFound
	}
	menu, err := service.repository.FindActiveScopeMenu(ctx, actor.UserID, scopeCode)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return queryscheme.ScopeConfig{}, model.SysMenu{}, myerrors.ErrQuerySchemeScopeForbidden
	}
	if err != nil {
		return queryscheme.ScopeConfig{}, model.SysMenu{}, myerrors.WrapDatabaseError(err)
	}
	return config, menu, nil
}

func (service *QuerySchemeService) actor(ctx context.Context) (queryscheme.Subject, error) {
	subject, ok := audit.GetAuditSubject(ctx)
	if !ok {
		return queryscheme.Subject{}, myerrors.ErrUserNotLogin
	}
	roleIDs, err := service.repository.ActiveRoleIDs(ctx, subject.UserID)
	if err != nil {
		return queryscheme.Subject{}, myerrors.WrapDatabaseError(err)
	}
	return queryscheme.Subject{UserID: subject.UserID, RoleIDs: roleIDs}, nil
}

func (service *QuerySchemeService) requireSharedManager(ctx context.Context, userID int) error {
	allowed, err := service.repository.HasSharedManageCapability(ctx, userID, queryscheme.SharedManageCapability)
	if err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	if !allowed {
		return myerrors.ErrQuerySchemeSharedForbidden
	}
	return nil
}

func (service *QuerySchemeService) validateRoles(ctx context.Context, schemeType model.QuerySchemeType, roleIDs []int) error {
	if schemeType != model.QuerySchemeTypeRole {
		if len(roleIDs) != 0 {
			return myerrors.ErrQuerySchemeRoleInvalid
		}
		return nil
	}
	if len(roleIDs) == 0 || len(roleIDs) > queryscheme.MaxRoleCount {
		return myerrors.ErrQuerySchemeRoleInvalid
	}
	unique := make(map[int]struct{}, len(roleIDs))
	for _, roleID := range roleIDs {
		if roleID <= 0 {
			return myerrors.ErrQuerySchemeRoleInvalid
		}
		unique[roleID] = struct{}{}
	}
	if len(unique) != len(roleIDs) {
		return myerrors.ErrQuerySchemeRoleInvalid
	}
	count, err := service.repository.CountActiveRoles(ctx, roleIDs)
	if err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	if count != int64(len(roleIDs)) {
		return myerrors.ErrQuerySchemeRoleInvalid
	}
	return nil
}

func (service *QuerySchemeService) visibleScheme(
	ctx context.Context,
	actor queryscheme.Subject,
	id int,
	scopeCode string,
) (model.QueryScheme, error) {
	value, err := service.repository.FindByIDWithDB(service.repository.DBWithContext(ctx), id, false)
	if err != nil {
		return model.QueryScheme{}, querySchemeLookupError(err)
	}
	if value.ScopeCode != scopeCode {
		return model.QueryScheme{}, myerrors.ErrQuerySchemeNotFound
	}
	visible, err := service.isVisible(ctx, actor, value)
	if err != nil {
		return model.QueryScheme{}, myerrors.WrapDatabaseError(err)
	}
	if !visible {
		return model.QueryScheme{}, myerrors.ErrQuerySchemeNotFound
	}
	return value, nil
}

func (service *QuerySchemeService) isVisible(ctx context.Context, actor queryscheme.Subject, value model.QueryScheme) (bool, error) {
	switch value.SchemeType {
	case model.QuerySchemeTypePersonal:
		return value.OwnerUserID != nil && *value.OwnerUserID == actor.UserID, nil
	case model.QuerySchemeTypePublic, model.QuerySchemeTypePageDefault:
		return value.Enabled, nil
	case model.QuerySchemeTypeRole:
		if !value.Enabled {
			return false, nil
		}
		roleIDs, err := service.repository.RoleIDs(service.repository.DBWithContext(ctx), value.Id)
		if err != nil {
			return false, err
		}
		owned := make(map[int]struct{}, len(actor.RoleIDs))
		for _, roleID := range actor.RoleIDs {
			owned[roleID] = struct{}{}
		}
		for _, roleID := range roleIDs {
			if _, exists := owned[roleID]; exists {
				return true, nil
			}
		}
	}
	return false, nil
}

func scopeConfigResponse(menuID int, scopeCode, scopeLabel string, config queryscheme.ScopeConfig) response.QueryScopeConfigRes {
	bindings := append([]queryscheme.BindingKind(nil), config.AllowedDynamicBindings...)
	sort.Slice(bindings, func(i, j int) bool { return bindings[i] < bindings[j] })
	return response.QueryScopeConfigRes{
		MenuID: menuID, ScopeCode: scopeCode, ScopeLabel: scopeLabel, TableCode: config.TableCode, QuickDateField: config.QuickDateField,
		QuickPresets: config.QuickPresets, VirtualSortFields: config.AllowedVirtualSortFields,
		DynamicBindingKinds: bindings,
	}
}

func resolveResponse(
	value model.QueryScheme,
	validation queryscheme.ValidationResult,
	resolved *queryscheme.ResolvedQuery,
	bindings []queryscheme.Binding,
) response.QuerySchemeResolveRes {
	kinds := make([]queryscheme.BindingKind, 0, len(bindings))
	seen := make(map[queryscheme.BindingKind]struct{}, len(bindings))
	for _, binding := range bindings {
		if _, exists := seen[binding.Kind]; !exists {
			seen[binding.Kind] = struct{}{}
			kinds = append(kinds, binding.Kind)
		}
	}
	return response.QuerySchemeResolveRes{
		Scheme: response.QuerySchemeResolveSourceRes{
			ID: value.Id, Name: value.Name, Type: value.SchemeType, Revision: value.Revision, IsDefault: value.IsDefault,
		},
		ValidationStatus: validation.Status, Issues: validation.Issues, ResolvedQuery: resolved,
		Bindings: append([]queryscheme.Binding(nil), bindings...), BindingKinds: kinds,
	}
}
