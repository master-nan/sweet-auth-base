package security

import (
	"bytes"
	"testing"
)

func TestCredentialSecretProtectorEncryptsAndAuthenticates(t *testing.T) {
	protector, err := NewCredentialSecretProtectorWithKey("test-master-key-with-sufficient-entropy")
	if err != nil {
		t.Fatalf("create protector: %v", err)
	}
	plaintext := []byte(`{"token":"secret-value"}`)
	envelope, err := protector.Seal(plaintext)
	if err != nil {
		t.Fatalf("seal secret: %v", err)
	}
	if envelope.Ciphertext == string(plaintext) || envelope.StorageRef == "" || envelope.Fingerprint == "" {
		t.Fatalf("invalid envelope: %+v", envelope)
	}
	opened, err := protector.Open(envelope.Ciphertext, envelope.Nonce)
	if err != nil || !bytes.Equal(opened, plaintext) {
		t.Fatalf("open secret: %q %v", opened, err)
	}
	if _, err := protector.Open(envelope.Ciphertext+"x", envelope.Nonce); err == nil {
		t.Fatal("expected tampered ciphertext to fail")
	}
}
