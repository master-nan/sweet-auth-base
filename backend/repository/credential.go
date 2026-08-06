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

type CredentialRepository interface {
	BasicRepository[model.Credential]
	GetCredentialList(context.Context, *request.Basic, model.SysTable) (response.ListResult[model.Credential], error)
	GetRuntimeCredential(context.Context, int) (CredentialRuntimeRecord, error)
}
