package service

import (
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/internal/queryscheme"
	"backend/model"
	"context"
)

func (service *QuerySchemeService) buildDetail(
	ctx context.Context,
	value model.QueryScheme,
) (response.QuerySchemeDetailRes, error) {
	payload, err := queryscheme.DecodePayload(value.QueryPayload)
	if err != nil || value.QuerySchemaVersion != queryscheme.SchemaVersion {
		return service.detailResponse(ctx, value, payload, queryscheme.ValidationResult{
			Status: queryscheme.ValidationInvalid,
			Issues: []queryscheme.ValidationIssue{{Code: queryscheme.IssuePayloadInvalid, Message: "查询方案结构不合法"}},
		}, nil)
	}
	payload = queryscheme.Normalize(payload)
	validation := queryscheme.ValidateSchema(payload)
	if validation.Status == queryscheme.ValidationValid {
		config, exists := service.scopes.Get(ctx, value.ScopeCode)
		if !exists {
			validation = queryscheme.ValidationResult{Status: queryscheme.ValidationInvalid, Issues: []queryscheme.ValidationIssue{{
				Code: queryscheme.IssueScopeUnavailable, Message: "查询范围已不可用",
			}}}
		} else {
			validation, err = service.validator.ValidateMetadata(ctx, config, payload)
			if err != nil {
				return response.QuerySchemeDetailRes{}, myerrors.WrapSystemError(err)
			}
		}
	}
	return service.detailResponse(ctx, value, payload, validation, nil)
}

func (service *QuerySchemeService) detailResponse(
	ctx context.Context,
	value model.QueryScheme,
	payload queryscheme.QuerySchemePayloadV1,
	validation queryscheme.ValidationResult,
	knownRoleIDs []int,
) (response.QuerySchemeDetailRes, error) {
	scopeLabels, err := service.repository.FindActiveScopeLabels(ctx, []string{value.ScopeCode})
	if err != nil {
		return response.QuerySchemeDetailRes{}, myerrors.WrapDatabaseError(err)
	}
	roleIDs := knownRoleIDs
	if value.SchemeType == model.QuerySchemeTypeRole && roleIDs == nil {
		roleIDs, err = service.repository.RoleIDs(service.repository.DBWithContext(ctx), value.Id)
		if err != nil {
			return response.QuerySchemeDetailRes{}, myerrors.WrapDatabaseError(err)
		}
	}
	item, err := service.listResponse(ctx, value, scopeLabels[value.ScopeCode], roleIDs)
	if err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	return response.QuerySchemeDetailRes{QuerySchemeListRes: item, Payload: payload, Issues: validation.Issues}, nil
}

func (service *QuerySchemeService) listResponse(
	ctx context.Context,
	value model.QueryScheme,
	scopeLabel string,
	roleIDs []int,
) (response.QuerySchemeListRes, error) {
	status := queryscheme.ValidationInvalid
	payload, err := queryscheme.DecodePayload(value.QueryPayload)
	if err == nil && value.QuerySchemaVersion == queryscheme.SchemaVersion {
		payload = queryscheme.Normalize(payload)
		validation := queryscheme.ValidateSchema(payload)
		status = validation.Status
		if status == queryscheme.ValidationValid {
			config, exists := service.scopes.Get(ctx, value.ScopeCode)
			if exists {
				validation, err = service.validator.ValidateMetadata(ctx, config, payload)
				if err != nil {
					return response.QuerySchemeListRes{}, myerrors.WrapSystemError(err)
				}
				status = validation.Status
			}
		}
	}
	creator := ""
	if value.CreateName != nil {
		creator = *value.CreateName
	}
	return response.QuerySchemeListRes{
		QuerySchemeSummaryRes: response.QuerySchemeSummaryRes{
			ID: value.Id, Name: value.Name, Type: value.SchemeType, IsDefault: value.IsDefault, Status: status,
		},
		ScopeCode: value.ScopeCode, ScopeLabel: scopeLabel, Enabled: value.Enabled,
		Creator: creator, RoleIDs: append([]int(nil), roleIDs...), Revision: value.Revision, UpdatedAt: value.GmtModify,
	}, nil
}
