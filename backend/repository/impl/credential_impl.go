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

// GetRuntimeCredentialIdentity 仅提供 Engine 构造 Provider 请求需要的非秘密标识字段。
func (r *CredentialRepositoryImpl) GetRuntimeCredentialIdentity(
	ctx context.Context,
	id int,
) (repository.CredentialRuntimeIdentity, error) {
	var value repository.CredentialRuntimeIdentity
	err := r.DBWithContext(ctx).
		Model(&model.Credential{}).
		Select([]string{"id", "external_system_id", "credential_code", "credential_type"}).
		Where("id = ?", id).
		First(&value).Error
	return value, err
}

// GetRuntimeCredential 仅读取执行期解析认证所需的固定字段。
// 配置 API 和普通 Service 不应调用此方法获取秘密材料。
func (r *CredentialRepositoryImpl) GetRuntimeCredential(
	ctx context.Context,
	id int,
) (repository.CredentialRuntimeRecord, error) {
	var value repository.CredentialRuntimeRecord
	err := r.DBWithContext(ctx).
		Model(&model.Credential{}).
		Select([]string{
			"id",
			"external_system_id",
			"credential_code",
			"credential_type",
			"status",
			"state",
			"secret_storage_ref",
			"secret_ciphertext",
			"secret_nonce",
			"secret_fingerprint",
			"expires_at",
			"version",
		}).
		Where("id = ?", id).
		First(&value).Error
	return value, err
}

var _ repository.CredentialRepository = (*CredentialRepositoryImpl)(nil)
