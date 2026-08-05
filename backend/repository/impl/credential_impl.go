package impl

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/internal/database"
	"backend/model"
	"backend/repository"
	"context"
)

type CredentialRepositoryImpl struct {
	*BasicRepositoryImpl[model.Credential]
}

func NewCredentialRepositoryImpl(primaryDB *database.PrimaryDB) *CredentialRepositoryImpl {
	return &CredentialRepositoryImpl{
		BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.Credential{}),
	}
}

func (r *CredentialRepositoryImpl) GetCredentialList(
	ctx context.Context,
	basic *request.Basic,
	table model.SysTable,
) (response.ListResult[model.Credential], error) {
	var values []model.Credential
	total, err := r.WithContext(ctx).PaginateAndCountAsync(basic, &values, table)
	return response.ListResult[model.Credential]{Data: values, Total: int(total)}, err
}

var _ repository.CredentialRepository = (*CredentialRepositoryImpl)(nil)
