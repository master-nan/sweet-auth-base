package repository

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"
	"context"
	"time"
)

// CredentialRuntimeRecord 是运行时解析凭证所需的最小安全投影。
// 它只能在内部 Credential Provider 中使用，不得作为配置 API 或普通 Service 的返回对象。
type CredentialRuntimeRecord struct {
	ID                int
	ExternalSystemID  int
	CredentialCode    string
	CredentialType    string
	Status            string
	State             bool
	SecretStorageRef  string `json:"-"`
	SecretCiphertext  string `json:"-"`
	SecretNonce       string `json:"-"`
	SecretFingerprint string `json:"-"`
	ExpiresAt         *time.Time
	Version           int
}

// CredentialRuntimeIdentity 是 Engine 构造 Provider 请求所需的非秘密凭证标识。
// 密文及安全存储材料仍只由 Credential Provider 读取。
type CredentialRuntimeIdentity struct {
	ID               int
	ExternalSystemID int
	CredentialCode   string
	CredentialType   string
}

type CredentialRepository interface {
	BasicRepository[model.Credential]
	GetCredentialList(context.Context, *request.Basic, model.SysTable) (response.ListResult[model.Credential], error)
	GetRuntimeCredentialIdentity(context.Context, int) (CredentialRuntimeIdentity, error)
	GetRuntimeCredential(context.Context, int) (CredentialRuntimeRecord, error)
}
