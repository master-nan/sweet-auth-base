package repository

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"
	"context"
)

type CredentialRepository interface {
	BasicRepository[model.Credential]
	GetCredentialList(context.Context, *request.Basic, model.SysTable) (response.ListResult[model.Credential], error)
}
