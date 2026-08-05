package security

import (
	"backend/config"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

var (
	ErrCredentialSecretRequired        = errors.New("credential secret is required")
	ErrCredentialSecretKeyRequired     = errors.New("credential secret encryption key is required")
	ErrCredentialSecretEnvelopeInvalid = errors.New("credential secret envelope is invalid")
)

const credentialSecretKeyContext = "sweet-platform:integration-credential:v1:"

// CredentialSecretEnvelope 是凭证密钥的服务端加密信封，不对 API 暴露。
type CredentialSecretEnvelope struct {
	StorageRef  string
	Ciphertext  string
	Nonce       string
	Fingerprint string
}

// CredentialSecretProtector 使用服务端主密钥保护集成凭证，不保存明文。
type CredentialSecretProtector struct {
	aead cipher.AEAD
}

func NewCredentialSecretProtector(server *config.Server) (*CredentialSecretProtector, error) {
	if server == nil {
		return nil, ErrCredentialSecretKeyRequired
	}
	return NewCredentialSecretProtectorWithKey(server.Session.Secret)
}

func NewCredentialSecretProtectorWithKey(masterKey string) (*CredentialSecretProtector, error) {
	if strings.TrimSpace(masterKey) == "" {
		return nil, ErrCredentialSecretKeyRequired
	}
	key := sha256.Sum256([]byte(credentialSecretKeyContext + masterKey))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("create credential secret cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create credential secret gcm: %w", err)
	}
	return &CredentialSecretProtector{aead: aead}, nil
}

func (p *CredentialSecretProtector) Seal(plaintext []byte) (CredentialSecretEnvelope, error) {
	if p == nil || p.aead == nil {
		return CredentialSecretEnvelope{}, ErrCredentialSecretKeyRequired
	}
	if len(plaintext) == 0 {
		return CredentialSecretEnvelope{}, ErrCredentialSecretRequired
	}
	nonce := make([]byte, p.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return CredentialSecretEnvelope{}, fmt.Errorf("generate credential nonce: %w", err)
	}
	refBytes := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, refBytes); err != nil {
		return CredentialSecretEnvelope{}, fmt.Errorf("generate credential storage reference: %w", err)
	}
	fingerprint := sha256.Sum256(plaintext)
	return CredentialSecretEnvelope{
		StorageRef:  hex.EncodeToString(refBytes),
		Ciphertext:  base64.RawStdEncoding.EncodeToString(p.aead.Seal(nil, nonce, plaintext, nil)),
		Nonce:       base64.RawStdEncoding.EncodeToString(nonce),
		Fingerprint: hex.EncodeToString(fingerprint[:]),
	}, nil
}

// Open 仅供后续受控执行器读取凭证，配置 API 不调用该方法。
func (p *CredentialSecretProtector) Open(ciphertext, nonce string) ([]byte, error) {
	if p == nil || p.aead == nil {
		return nil, ErrCredentialSecretKeyRequired
	}
	cipherBytes, err := base64.RawStdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, ErrCredentialSecretEnvelopeInvalid
	}
	nonceBytes, err := base64.RawStdEncoding.DecodeString(nonce)
	if err != nil || len(nonceBytes) != p.aead.NonceSize() {
		return nil, ErrCredentialSecretEnvelopeInvalid
	}
	plaintext, err := p.aead.Open(nil, nonceBytes, cipherBytes, nil)
	if err != nil {
		return nil, ErrCredentialSecretEnvelopeInvalid
	}
	return plaintext, nil
}
