package response

import (
	"backend/model"
	"encoding/json"
	"strings"
	"testing"
)

func TestIntegrationExecutionResponseDTOWhitelist(t *testing.T) {
	startedAt := model.Now()
	value := model.IntegrationExecution{
		Basic: model.Basic{Id: 1, State: true}, ExecutionNo: "INT-001",
		ExternalSystemID: 10, ExternalSystemCode: "demo_hr", ExternalSystemName: "Demo HR",
		InterfaceDefinitionID: 20, InterfaceCode: "org_list", InterfaceName: "组织列表", InterfaceVersion: 1,
		TriggerSource: model.IntegrationTriggerSourceManual, Status: model.IntegrationExecutionStatusRunning,
		IdempotencyScope: "acceptance", IdempotencyKey: "request-001",
		InputHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		StartedAt: &startedAt, Revision: 2,
	}
	log := model.IntegrationLog{
		Basic: model.Basic{Id: 2, State: true}, ExecutionID: value.Id, AttemptNo: 1,
		Status: model.IntegrationLogStatusRunning, StartedAt: startedAt,
		ResultCertainty: model.IntegrationResultCertaintyUnknown, RequestID: "request-id", TraceID: "trace-id",
	}
	payload, err := json.Marshal(NewIntegrationExecutionDetailRes(value, []model.IntegrationLog{log}))
	if err != nil {
		t.Fatalf("marshal execution response: %v", err)
	}
	text := strings.ToLower(string(payload))
	for _, forbidden := range []string{
		"gmt_delete", "create_user", "modify_user", "delete_user", "authorization",
		"secret", "ciphertext", "nonce", "payload", "external_system_id\"", "interface_definition_id\"",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("response leaked %q: %s", forbidden, payload)
		}
	}
	for _, required := range []string{"execution_no", "external_system", "interface", "attempts", "input_hash"} {
		if !strings.Contains(text, required) {
			t.Fatalf("response missing %q: %s", required, payload)
		}
	}
}
