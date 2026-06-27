package security

import "testing"

func TestIsSensitiveFieldName(t *testing.T) {
	sensitive := []string{
		"password",
		"sender_password",
		"access_token",
		"refresh_tokens",
		"client_secret",
		"api_key",
		"private_key",
		"credential",
		"oauthCredential",
	}
	for _, name := range sensitive {
		if !IsSensitiveFieldName(name) {
			t.Fatalf("expected %q to be treated as sensitive", name)
		}
	}

	ordinary := []string{
		"",
		"name",
		"tenant_id",
		"foreign_key",
		"public_key",
		"sequence",
	}
	for _, name := range ordinary {
		if IsSensitiveFieldName(name) {
			t.Fatalf("expected %q to be treated as ordinary", name)
		}
	}
}

func TestIsManagedMetadataField(t *testing.T) {
	managed := []string{
		"id",
		"gmt_create",
		"gmt_create_user",
		"gmt_modify",
		"gmt_modify_user",
		"gmt_delete",
		"gmt_delete_user",
		"create_user",
		"create_name",
		"modify_user",
		"modify_name",
		"delete_user",
		"delete_name",
	}
	for _, name := range managed {
		if !IsManagedMetadataField(name) {
			t.Fatalf("expected %q to be treated as managed metadata", name)
		}
	}

	ordinary := []string{
		"",
		"name",
		"tenant_id",
		"customer_id",
		"status",
		"remark",
	}
	for _, name := range ordinary {
		if IsManagedMetadataField(name) {
			t.Fatalf("expected %q to be treated as business metadata", name)
		}
	}
}
