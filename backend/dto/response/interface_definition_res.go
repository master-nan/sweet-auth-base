package response

import "backend/model"

type InterfaceSystemSummaryRes struct {
	Id         int    `json:"id"`
	SystemCode string `json:"system_code"`
	Name       string `json:"name"`
}

type InterfaceDefinitionListRes struct {
	Id             int                       `json:"id"`
	ExternalSystem InterfaceSystemSummaryRes `json:"external_system"`
	InterfaceCode  string                    `json:"interface_code"`
	Name           string                    `json:"name"`
	Version        int                       `json:"version"`
	Protocol       string                    `json:"protocol"`
	HTTPMethod     string                    `json:"http_method"`
	PathSummary    string                    `json:"path_summary"`
	Status         string                    `json:"status"`
	Revision       int                       `json:"revision"`
	GmtModify      model.CustomTime          `json:"gmt_modify"`
}

type InterfaceDefinitionDetailRes struct {
	InterfaceDefinitionListRes
	RelativePath   string           `json:"relative_path"`
	CredentialID   *int             `json:"credential_id,omitempty"`
	TimeoutSeconds int              `json:"timeout_seconds"`
	ResponseLimit  int64            `json:"response_limit"`
	RetryPolicyID  *int             `json:"retry_policy_id,omitempty"`
	Description    string           `json:"description"`
	GmtCreate      model.CustomTime `json:"gmt_create"`
}

func NewInterfaceDefinitionListRes(value model.InterfaceDefinition, system model.ExternalSystem) InterfaceDefinitionListRes {
	return InterfaceDefinitionListRes{
		Id:             value.Id,
		ExternalSystem: InterfaceSystemSummaryRes{Id: system.Id, SystemCode: system.SystemCode, Name: system.Name},
		InterfaceCode:  value.InterfaceCode, Name: value.Name, Version: value.Version,
		Protocol: value.Protocol, HTTPMethod: value.HTTPMethod, PathSummary: value.RelativePath,
		Status: value.Status, Revision: value.Revision, GmtModify: value.GmtModify,
	}
}

func NewInterfaceDefinitionDetailRes(value model.InterfaceDefinition, system model.ExternalSystem) InterfaceDefinitionDetailRes {
	return InterfaceDefinitionDetailRes{
		InterfaceDefinitionListRes: NewInterfaceDefinitionListRes(value, system),
		RelativePath:               value.RelativePath, CredentialID: value.CredentialID,
		TimeoutSeconds: value.TimeoutSeconds, ResponseLimit: value.ResponseLimit,
		RetryPolicyID: value.RetryPolicyID, Description: value.Description, GmtCreate: value.GmtCreate,
	}
}
