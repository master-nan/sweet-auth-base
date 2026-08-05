package response

import (
	"backend/model"
	"time"
)

type CredentialSystemSummaryRes struct {
	Id         int    `json:"id"`
	SystemCode string `json:"system_code"`
	Name       string `json:"name"`
}

type CredentialListRes struct {
	Id                 int                        `json:"id"`
	ExternalSystem     CredentialSystemSummaryRes `json:"external_system"`
	CredentialCode     string                     `json:"credential_code"`
	Name               string                     `json:"name"`
	CredentialType     string                     `json:"credential_type"`
	Status             string                     `json:"status"`
	EffectiveStatus    string                     `json:"effective_status"`
	FingerprintSummary string                     `json:"fingerprint_summary"`
	ExpiresAt          *time.Time                 `json:"expires_at"`
	Version            int                        `json:"version"`
	RotatedAt          *time.Time                 `json:"rotated_at"`
	Revision           int                        `json:"revision"`
	GmtModify          model.CustomTime           `json:"gmt_modify"`
}

type CredentialDetailRes struct {
	CredentialListRes
	Description string           `json:"description"`
	GmtCreate   model.CustomTime `json:"gmt_create"`
}

func NewCredentialListRes(value model.Credential, system model.ExternalSystem, now time.Time) CredentialListRes {
	return CredentialListRes{
		Id:                 value.Id,
		ExternalSystem:     CredentialSystemSummaryRes{Id: system.Id, SystemCode: system.SystemCode, Name: system.Name},
		CredentialCode:     value.CredentialCode,
		Name:               value.Name,
		CredentialType:     value.CredentialType,
		Status:             value.Status,
		EffectiveStatus:    CredentialEffectiveStatus(value, now),
		FingerprintSummary: credentialFingerprintSummary(value.SecretFingerprint),
		ExpiresAt:          value.ExpiresAt,
		Version:            value.Version,
		RotatedAt:          value.RotatedAt,
		Revision:           value.Revision,
		GmtModify:          value.GmtModify,
	}
}

func NewCredentialDetailRes(value model.Credential, system model.ExternalSystem, now time.Time) CredentialDetailRes {
	return CredentialDetailRes{
		CredentialListRes: NewCredentialListRes(value, system, now),
		Description:       value.Description,
		GmtCreate:         value.GmtCreate,
	}
}

func CredentialEffectiveStatus(value model.Credential, now time.Time) string {
	if value.Status != model.CredentialStatusRevoked && value.ExpiresAt != nil && !value.ExpiresAt.After(now) {
		return "expired"
	}
	return value.Status
}

func credentialFingerprintSummary(value string) string {
	if len(value) <= 12 {
		return value
	}
	return value[:12]
}
