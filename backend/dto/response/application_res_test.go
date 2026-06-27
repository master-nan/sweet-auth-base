package response

import (
	"backend/model"
	"encoding/json"
	"testing"

	"github.com/jinzhu/copier"
)

func TestApplicationResDoesNotExposeSecrets(t *testing.T) {
	application := model.Application{
		Basic:      model.Basic{Id: 1, State: true},
		Name:       "Default Admin App",
		AppKey:     "sweet-admin",
		AppSecret:  "sweet-admin-secret",
		Expiration: 7200,
		DingKey:    "ding-key",
		DingSecret: "ding-secret",
		DingAppID:  "agent-id",
		Remark:     "demo",
	}

	var got ApplicationRes
	if err := copier.Copy(&got, &application); err != nil {
		t.Fatalf("copy application response: %v", err)
	}
	payload, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal application response: %v", err)
	}
	if got.AppKey != application.AppKey || got.Name != application.Name || got.Expiration != application.Expiration {
		t.Fatalf("expected non-sensitive fields to be preserved, got %+v", got)
	}
	if got.Id != application.Id || got.State != application.State {
		t.Fatalf("expected basic fields to be preserved, got %+v", got)
	}
	if string(payload) == "" || jsonContainsKey(payload, "app_secret") || jsonContainsKey(payload, "ding_secret") {
		t.Fatalf("expected secret fields to be absent, got %s", string(payload))
	}
}

func TestNewApplicationSecretResExposesOneTimeSecret(t *testing.T) {
	application := model.Application{
		Basic:      model.Basic{Id: 1, State: true},
		Name:       "Default Admin App",
		AppKey:     "sweet-admin",
		AppSecret:  "sweet-admin-secret",
		Expiration: 7200,
	}

	got := NewApplicationSecretRes(application)
	if got.AppSecret != application.AppSecret {
		t.Fatalf("expected one-time app secret to be returned, got %q", got.AppSecret)
	}
	if got.AppKey != application.AppKey || got.Id != application.Id {
		t.Fatalf("expected app identity to be preserved, got %+v", got)
	}
}

func jsonContainsKey(payload []byte, key string) bool {
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return false
	}
	_, ok := data[key]
	return ok
}
