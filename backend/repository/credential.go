package repository

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"
	"context"

	"gorm.io/gorm"
)

type CredentialRepository interface {
	DBWithContext(context.Context) *gorm.DB
	Create(*gorm.DB, *model.Credential) error
	FindByID(context.Context, int) (model.Credential, error)
	FindByIDForUpdate(*gorm.DB, int) (model.Credential, error)
	FindByIDs(context.Context, []int) ([]model.Credential, error)
	FindByCode(*gorm.DB, int, string) (model.Credential, error)
	Query(context.Context, request.CredentialQueryReq, model.SysTable) (response.ListResult[model.Credential], error)
	UpdateFields(*gorm.DB, int, int, map[string]any) (bool, error)
}
