package response

import (
	"backend/model"
	"net/url"
)

type ExternalSystemListRes struct {
	Id              int              `json:"id"`
	SystemCode      string           `json:"system_code"`
	Name            string           `json:"name"`
	SystemType      string           `json:"system_type"`
	BaseURLSummary  string           `json:"base_url_summary"`
	OwnerIdentifier string           `json:"owner_identifier"`
	OwnerName       string           `json:"owner_name"`
	Status          string           `json:"status"`
	Revision        int              `json:"revision"`
	GmtModify       model.CustomTime `json:"gmt_modify"`
}

type ExternalSystemDetailRes struct {
	ExternalSystemListRes
	BaseURL     string           `json:"base_url"`
	Description string           `json:"description"`
	GmtCreate   model.CustomTime `json:"gmt_create"`
}

func NewExternalSystemListRes(value model.ExternalSystem) ExternalSystemListRes {
	return ExternalSystemListRes{
		Id:              value.Id,
		SystemCode:      value.SystemCode,
		Name:            value.Name,
		SystemType:      value.SystemType,
		BaseURLSummary:  externalSystemBaseURLSummary(value.BaseURL),
		OwnerIdentifier: value.OwnerIdentifier,
		OwnerName:       value.OwnerName,
		Status:          value.Status,
		Revision:        value.Revision,
		GmtModify:       value.GmtModify,
	}
}

func NewExternalSystemDetailRes(value model.ExternalSystem) ExternalSystemDetailRes {
	return ExternalSystemDetailRes{
		ExternalSystemListRes: NewExternalSystemListRes(value),
		BaseURL:               value.BaseURL,
		Description:           value.Description,
		GmtCreate:             value.GmtCreate,
	}
}

func externalSystemBaseURLSummary(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}
