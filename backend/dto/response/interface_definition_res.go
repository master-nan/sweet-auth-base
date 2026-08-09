package response

import (
	"backend/model"
	"encoding/json"
	"time"
)

type InterfaceSystemSummaryRes struct {
	Id         int    `json:"id"`
	SystemCode string `json:"system_code"`
	Name       string `json:"name"`
}

type InterfaceDefinitionListRes struct {
	Id              int                       `json:"id"`
	ExternalSystem  InterfaceSystemSummaryRes `json:"external_system"`
	InterfaceCode   string                    `json:"interface_code"`
	Name            string                    `json:"name"`
	Version         int                       `json:"version"`
	Protocol        string                    `json:"protocol"`
	HTTPMethod      string                    `json:"http_method"`
	PathSummary     string                    `json:"path_summary"`
	Status          string                    `json:"status"`
	EffectiveStatus string                    `json:"effective_status"`
	Revision        int                       `json:"revision"`
	GmtModify       model.CustomTime          `json:"gmt_modify"`
}

type InterfaceCredentialSummaryRes struct {
	Id              int    `json:"id"`
	CredentialCode  string `json:"credential_code"`
	Name            string `json:"name"`
	CredentialType  string `json:"credential_type"`
	EffectiveStatus string `json:"effective_status"`
}

type InterfaceRetryPolicySummaryRes struct {
	Id         int    `json:"id"`
	PolicyCode string `json:"policy_code"`
	PolicyName string `json:"policy_name"`
	Version    int    `json:"version"`
	Status     string `json:"status"`
}

type InterfaceDefinitionDetailRes struct {
	InterfaceDefinitionListRes
	RelativePath   string                          `json:"relative_path"`
	InputContract  json.RawMessage                 `json:"input_contract"`
	CredentialID   *int                            `json:"credential_id,omitempty"`
	Credential     *InterfaceCredentialSummaryRes  `json:"credential,omitempty"`
	TimeoutSeconds int                             `json:"timeout_seconds"`
	ResponseLimit  int64                           `json:"response_limit"`
	RetryPolicyID  *int                            `json:"retry_policy_id,omitempty"`
	RetryPolicy    *InterfaceRetryPolicySummaryRes `json:"retry_policy,omitempty"`
	Description    string                          `json:"description"`
	GmtCreate      model.CustomTime                `json:"gmt_create"`
}

func NewInterfaceDefinitionListRes(value model.InterfaceDefinition, system model.ExternalSystem, credential *model.Credential, now time.Time, policies ...*model.RetryPolicy) InterfaceDefinitionListRes {
	var policy *model.RetryPolicy
	if len(policies) > 0 {
		policy = policies[0]
	}
	return InterfaceDefinitionListRes{
		Id:             value.Id,
		ExternalSystem: InterfaceSystemSummaryRes{Id: system.Id, SystemCode: system.SystemCode, Name: system.Name},
		InterfaceCode:  value.InterfaceCode, Name: value.Name, Version: value.Version,
		Protocol: value.Protocol, HTTPMethod: value.HTTPMethod, PathSummary: value.RelativePath,
		Status: value.Status, EffectiveStatus: InterfaceDefinitionEffectiveStatus(value, system, credential, now, policy),
		Revision: value.Revision, GmtModify: value.GmtModify,
	}
}

func NewInterfaceDefinitionDetailRes(value model.InterfaceDefinition, system model.ExternalSystem, credential *model.Credential, now time.Time, policies ...*model.RetryPolicy) InterfaceDefinitionDetailRes {
	var policy *model.RetryPolicy
	if len(policies) > 0 {
		policy = policies[0]
	}
	result := InterfaceDefinitionDetailRes{
		InterfaceDefinitionListRes: NewInterfaceDefinitionListRes(value, system, credential, now, policy),
		RelativePath:               value.RelativePath, CredentialID: value.CredentialID,
		InputContract:  append(json.RawMessage(nil), value.InputContract...),
		TimeoutSeconds: value.TimeoutSeconds, ResponseLimit: value.ResponseLimit,
		RetryPolicyID: value.RetryPolicyID, Description: value.Description, GmtCreate: value.GmtCreate,
	}
	if credential != nil {
		result.Credential = &InterfaceCredentialSummaryRes{
			Id: credential.Id, CredentialCode: credential.CredentialCode, Name: credential.Name,
			CredentialType:  credential.CredentialType,
			EffectiveStatus: CredentialEffectiveStatus(*credential, now),
		}
	}
	if policy != nil {
		result.RetryPolicy = &InterfaceRetryPolicySummaryRes{
			Id: policy.Id, PolicyCode: policy.PolicyCode, PolicyName: policy.PolicyName,
			Version: policy.Version, Status: policy.Status,
		}
	}
	return result
}

func InterfaceDefinitionEffectiveStatus(value model.InterfaceDefinition, system model.ExternalSystem, credential *model.Credential, now time.Time, policies ...*model.RetryPolicy) string {
	if value.Status != model.InterfaceDefinitionStatusEnabled {
		return value.Status
	}
	if system.Status != model.ExternalSystemStatusEnabled {
		return "unavailable"
	}
	if value.CredentialID != nil && (credential == nil || credential.Status != model.CredentialStatusActive || CredentialEffectiveStatus(*credential, now) == "expired") {
		return "unavailable"
	}
	if value.RetryPolicyID != nil {
		if len(policies) == 0 || policies[0] == nil || policies[0].Status != model.RetryPolicyStatusEnabled {
			return "unavailable"
		}
	}
	return value.Status
}
