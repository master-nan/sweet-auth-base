package security

import "strings"

func IsSensitiveFieldName(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if normalized == "" {
		return false
	}
	exact := map[string]bool{
		"password":        true,
		"passwd":          true,
		"pwd":             true,
		"access_token":    true,
		"access_tokens":   true,
		"refresh_token":   true,
		"refresh_tokens":  true,
		"id_token":        true,
		"id_tokens":       true,
		"token":           true,
		"secret":          true,
		"app_secret":      true,
		"client_secret":   true,
		"sender_password": true,
		"api_key":         true,
		"apikey":          true,
		"private_key":     true,
		"credential":      true,
		"credentials":     true,
	}
	if exact[normalized] {
		return true
	}
	return strings.Contains(normalized, "password") ||
		strings.Contains(normalized, "passwd") ||
		strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "secret") ||
		strings.Contains(normalized, "api_key") ||
		strings.Contains(normalized, "apikey") ||
		strings.Contains(normalized, "private_key") ||
		strings.Contains(normalized, "credential")
}

func IsManagedMetadataField(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "id",
		"gmt_create", "gmt_create_user", "gmt_create_name",
		"gmt_modify", "gmt_modify_user", "gmt_modify_name",
		"gmt_delete", "gmt_delete_user", "gmt_delete_name",
		"create_user", "create_name",
		"modify_user", "modify_name",
		"delete_user", "delete_name":
		return true
	default:
		return false
	}
}
