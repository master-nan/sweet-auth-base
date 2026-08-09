package response

import (
	"backend/model"
	"encoding/json"
	"strings"
	"testing"

	"gorm.io/datatypes"
)

func TestIntegrationExecutionResponseDTOWhitelist(t *testing.T) {
	startedAt := model.Now()
	value := model.IntegrationExecution{
		Basic: model.Basic{Id: 1, State: true}, ExecutionNo: "INT-001",
		ExternalSystemID: 10, ExternalSystemCode: "demo_hr", ExternalSystemName: "Demo HR",
		InterfaceDefinitionID: 20, InterfaceCode: "org_list", InterfaceName: "组织列表", InterfaceVersion: 1,
		TriggerSource: model.IntegrationTriggerSourceManual, Status: model.IntegrationExecutionStatusRunning,
		IdempotencyScope: "acceptance", IdempotencyKey: "request-001",
		InputHash:            "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		InputSnapshot:        datatypes.JSON([]byte(`{"version":1,"path_params":{"employee_id":"10001"},"query_params":{"page":["1"]},"headers":{"x-correlation-id":["corr-1"]},"json_body":{"name":"private-business-value"}}`)),
		InputSnapshotVersion: 1, InputSnapshotSize: 180,
		RetryPolicySnapshotVersion: 1,
		RetryPolicySnapshot:        datatypes.JSON([]byte(`{"version":1,"policy_code":"safe_retry","policy_version":2,"max_attempts":3}`)),
		CurrentAttempt:             1, RetryReasonCode: "retry_allowed",
		StartedAt: &startedAt, Revision: 2,
	}
	payload, err := json.Marshal(NewIntegrationExecutionDetailRes(value))
	if err != nil {
		t.Fatalf("marshal execution response: %v", err)
	}
	text := strings.ToLower(string(payload))
	for _, forbidden := range []string{
		"gmt_delete", "create_user", "modify_user", "delete_user", "authorization",
		"secret", "ciphertext", "nonce", "payload", "external_system_id\"", "interface_definition_id\"",
		`"attempts":`, "http_status", "result_certainty", "request_id", "trace_id",
		"employee_id", "10001", "corr-1", "private-business-value", "json_body", "path_params",
		"remote_idempotency_key",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("response leaked %q: %s", forbidden, payload)
		}
	}
	for _, required := range []string{
		"execution_no", "external_system", "interface", "current_attempt", "max_attempts", "attempts_remaining",
		"retry_reason_code", "safe_retry", "input_hash", "input_summary", "snapshot_version", "has_body",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("response missing %q: %s", required, payload)
		}
	}
	detail := NewIntegrationExecutionDetailRes(value)
	if detail.InputSummary.PathCount != 1 || detail.InputSummary.QueryCount != 1 ||
		detail.InputSummary.HeaderCount != 1 || !detail.InputSummary.HasBody ||
		detail.MaxAttempts != 3 || detail.AttemptsRemaining != 2 || detail.RetryPolicy == nil {
		t.Fatalf("input summary = %+v", detail.InputSummary)
	}
}

func TestIntegrationLogResponseDTOWhitelist(t *testing.T) {
	startedAt := model.Now()
	value := model.IntegrationLog{
		Basic:     model.Basic{Id: 2, State: true},
		AttemptNo: 1, Status: model.IntegrationLogStatusSucceeded, StartedAt: startedAt,
		WorkerID: "worker-with-sensitive-context", ResponseContentType: "application/json",
		CredentialCode: "hr-token", CredentialVersion: "v2", CredentialFingerprintSummary: "abcd...wxyz",
		ResultSummary: "safe summary", ResultHash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		RequestID: "request-id", TraceID: "trace-id",
		Retryable: true, RetryReasonCode: "retry_allowed", RetryDelayMs: 2000,
		RetryScheduledAt: &startedAt, RetryAfterSource: "http_delta",
		Execution: model.IntegrationExecution{
			ExecutionNo: "INT-001", ExternalSystemCode: "demo_hr", InterfaceCode: "org_list",
		},
	}
	payload, err := json.Marshal(NewIntegrationLogDetailRes(value))
	if err != nil {
		t.Fatalf("marshal log response: %v", err)
	}
	text := strings.ToLower(string(payload))
	for _, forbidden := range []string{
		"worker-with-sensitive-context", "authorization", "cookie", "api_key", "token-value",
		"ciphertext", "nonce", "storage_reference", "payload", "response_body",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("log response leaked %q: %s", forbidden, payload)
		}
	}
	for _, required := range []string{
		"execution_no", "attempt_no", "result_hash", "credential_code", "credential_version",
		"retryable", "retry_reason_code", "retry_delay_ms", "retry_scheduled_at", "retry_after_source",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("log response missing %q: %s", required, payload)
		}
	}
}
