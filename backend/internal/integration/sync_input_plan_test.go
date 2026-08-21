package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	myerrors "backend/internal/errors"
)

func TestNormalizeSyncExecutionInputPlanUsesSnapshotContract(t *testing.T) {
	contract := syncPlanContract(t, false)
	plan := SyncExecutionInputPlan{
		Version: SyncExecutionInputPlanVersionV1,
		StaticInput: ExecutionInputValues{
			Headers:  map[string][]string{"X-Correlation-ID": {"sweet"}},
			JSONBody: json.RawMessage(`{"tenant":"north"}`),
		},
		WindowStartBinding: &SyncWindowBinding{Location: InputLocationQuery, Code: "updated_from", Format: SyncTimeFormatRFC3339},
		WindowEndBinding:   &SyncWindowBinding{Location: InputLocationQuery, Code: "updated_to", Format: SyncTimeFormatRFC3339},
	}
	raw, _ := json.Marshal(plan)
	normalized, summary, err := NormalizeSyncExecutionInputPlan(raw, contract, "POST", "/employees", 3, "timestamp")
	if err != nil {
		t.Fatalf("normalize plan: %v", err)
	}
	if summary.Version != 1 || summary.StaticParameterCount != 2 || !summary.HasWindowBindings {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	decoded, err := DecodeSyncExecutionInputPlan(normalized)
	if err != nil || len(decoded.StaticInput.Headers["X-Correlation-ID"]) != 1 || decoded.StaticInput.Headers["X-Correlation-ID"][0] != "sweet" {
		t.Fatalf("normalized plan=%+v err=%v", decoded, err)
	}
}

func TestNormalizeSyncExecutionInputPlanRejectsUnsafeOrOverlappingInput(t *testing.T) {
	contract := syncPlanContract(t, false)
	tests := []SyncExecutionInputPlan{
		{Version: 1, StaticInput: ExecutionInputValues{QueryParams: map[string][]string{"unknown": {"x"}}}, WindowStartBinding: &SyncWindowBinding{Location: "query", Code: "updated_from", Format: "rfc3339"}, WindowEndBinding: &SyncWindowBinding{Location: "query", Code: "updated_to", Format: "rfc3339"}},
		{Version: 1, StaticInput: ExecutionInputValues{QueryParams: map[string][]string{"updated_from": {"x"}}}, WindowStartBinding: &SyncWindowBinding{Location: "query", Code: "updated_from", Format: "rfc3339"}, WindowEndBinding: &SyncWindowBinding{Location: "query", Code: "updated_to", Format: "rfc3339"}},
	}
	for index, plan := range tests {
		raw, _ := json.Marshal(plan)
		if _, _, err := NormalizeSyncExecutionInputPlan(raw, contract, "POST", "/employees", 3, "timestamp"); !errors.Is(err, myerrors.ErrSyncInputPlanContractMismatch) {
			t.Fatalf("case %d error=%v", index, err)
		}
	}
	if _, _, err := NormalizeSyncExecutionInputPlan([]byte(`{"version":1,"static_input":{},"window_start_binding":{"location":"header","code":"Authorization","format":"rfc3339"},"window_end_binding":{"location":"query","code":"updated_to","format":"rfc3339"}}`), contract, "POST", "/employees", 3, "timestamp"); !errors.Is(err, myerrors.ErrSyncInputPlanContractMismatch) {
		t.Fatalf("authorization binding error=%v", err)
	}
	sensitivePlan := SyncExecutionInputPlan{Version: 1, StaticInput: ExecutionInputValues{Headers: map[string][]string{"X-Correlation-ID": {"secret"}}, JSONBody: json.RawMessage(`{"tenant":"north"}`)}, WindowStartBinding: &SyncWindowBinding{Location: "query", Code: "updated_from", Format: "rfc3339"}, WindowEndBinding: &SyncWindowBinding{Location: "query", Code: "updated_to", Format: "rfc3339"}}
	sensitiveRaw, _ := json.Marshal(sensitivePlan)
	if _, _, err := NormalizeSyncExecutionInputPlan(sensitiveRaw, syncPlanContract(t, true), "POST", "/employees", 3, "timestamp"); !errors.Is(err, myerrors.ErrSyncInputPlanContractMismatch) {
		t.Fatalf("sensitive static input error=%v", err)
	}
	invalidFormat := SyncExecutionInputPlan{Version: 1, StaticInput: ExecutionInputValues{JSONBody: json.RawMessage(`{"tenant":"north"}`)}, WindowStartBinding: &SyncWindowBinding{Location: "query", Code: "updated_from", Format: "iso8601"}, WindowEndBinding: &SyncWindowBinding{Location: "query", Code: "updated_to", Format: "rfc3339"}}
	invalidRaw, _ := json.Marshal(invalidFormat)
	if _, _, err := NormalizeSyncExecutionInputPlan(invalidRaw, contract, "POST", "/employees", 3, "timestamp"); !errors.Is(err, myerrors.ErrSyncInputPlanContractMismatch) {
		t.Fatalf("binding format error=%v", err)
	}
}

func TestStaticSyncConsumerRegistryValidationAndIsolation(t *testing.T) {
	registry, err := NewStaticSyncConsumerRegistry(SyncConsumerRegistration{Metadata: SyncConsumerMetadata{Code: "test_sync", Version: 1, Name: "Test", Status: SyncConsumerStatusEnabled, ContentTypes: []string{"application/json"}, MaxResponseBytes: 1 << 20, MaxDuration: 10 * time.Second, CheckpointModes: []string{"timestamp"}}, Consumer: SyncResultConsumerFunc(func(context.Context, SyncConsumptionRequest) (SyncConsumptionResult, error) {
		return NewSyncConsumptionResult(true, "", 1, 0, "")
	})})
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := registry.ValidateReference(SyncConsumerReference{Code: "test_sync", Version: 1, ContentType: "application/json; charset=utf-8", ResponseLimit: 1024, CheckpointMode: "timestamp", RequestTimeout: 30 * time.Second, LeaseDuration: IntegrationDefaultLeaseDuration})
	if err != nil || metadata.Code != "test_sync" {
		t.Fatalf("validate registry: metadata=%+v err=%v", metadata, err)
	}
	metadata.ContentTypes[0] = "text/plain"
	if registry.ListMetadata()[0].ContentTypes[0] != "application/json" {
		t.Fatal("registry metadata leaked mutable slice")
	}
	if _, err := registry.ValidateReference(SyncConsumerReference{Code: "missing", Version: 1}); !errors.Is(err, myerrors.ErrSyncConsumerNotRegistered) {
		t.Fatalf("missing consumer error=%v", err)
	}
	if _, err := registry.ValidateReference(SyncConsumerReference{Code: "test_sync", Version: 1, ContentType: "application/json", ResponseLimit: 1024, CheckpointMode: "timestamp", RequestTimeout: 120 * time.Second, LeaseDuration: 165 * time.Second}); !errors.Is(err, myerrors.ErrSyncLeaseBudgetInsufficient) {
		t.Fatalf("lease budget error=%v", err)
	}
}

func TestMaterializeSyncExecutionInputPlan(t *testing.T) {
	raw := []byte(`{"version":1,"static_input":{"query_params":{"tenant":["sweet"]}},"window_start_binding":{"location":"query","code":"updated_from","format":"rfc3339"},"window_end_binding":{"location":"body","code":"updated_to","format":"unix_milliseconds"}}`)
	start := time.Date(2026, 8, 10, 1, 2, 3, 456000000, time.UTC)
	end := start.Add(time.Hour)
	input, err := MaterializeSyncExecutionInputPlan(raw, &start, &end)
	if err != nil {
		t.Fatal(err)
	}
	if got := input.QueryParams["updated_from"]; len(got) != 1 || got[0] != start.Format(time.RFC3339Nano) {
		t.Fatalf("start binding=%v", got)
	}
	var body map[string]any
	if err := json.Unmarshal(input.JSONBody, &body); err != nil || body["updated_to"].(float64) != float64(end.UnixMilli()) {
		t.Fatalf("body=%s err=%v", input.JSONBody, err)
	}
	if input.QueryParams["tenant"][0] != "sweet" {
		t.Fatal("static input was not preserved")
	}
}

func TestSyncExecutionInputPlanV1RemainsBounded(t *testing.T) {
	raw := []byte(`{"version":1,"static_input":{"json_body":{"tenant":"north"}},"window_start_binding":{"location":"query","code":"updated_from","format":"rfc3339"},"window_end_binding":{"location":"query","code":"updated_to","format":"rfc3339"}}`)
	normalized, summary, err := NormalizeSyncExecutionInputPlan(raw, syncPlanContract(t, false), "POST", "/employees", 3, "timestamp")
	if err != nil {
		t.Fatal(err)
	}
	if summary.Version != SyncExecutionInputPlanVersionV1 || summary.WindowMode != SyncWindowModeBoundedWindow || !summary.ResponseBounded {
		t.Fatalf("unexpected V1 summary: %+v", summary)
	}
	if bytes.Contains(normalized, []byte(`"window_mode"`)) {
		t.Fatalf("V1 representation changed: %s", normalized)
	}
}

func TestSyncExecutionInputPlanV2WindowModes(t *testing.T) {
	bounded := []byte(`{"version":2,"window_mode":"bounded_window","static_input":{"json_body":{"tenant":"north"}},"window_start_binding":{"location":"query","code":"updated_from","format":"rfc3339"},"window_end_binding":{"location":"query","code":"updated_to","format":"rfc3339"}}`)
	if _, summary, err := NormalizeSyncExecutionInputPlan(bounded, syncPlanContract(t, false), "POST", "/employees", 3, "timestamp"); err != nil || !summary.ResponseBounded {
		t.Fatalf("V2 bounded plan: summary=%+v err=%v", summary, err)
	}
	lowerContract := syncLowerBoundPlanContract(t)
	lower := []byte(`{"version":2,"window_mode":"lower_bound_only","static_input":{},"window_start_binding":{"location":"query","code":"changed_since","format":"rfc3339"}}`)
	normalized, summary, err := NormalizeSyncExecutionInputPlan(lower, lowerContract, "GET", "/employees", 4, "timestamp")
	if err != nil || summary.WindowMode != SyncWindowModeLowerBoundOnly || summary.ResponseBounded || !summary.HasWindowBindings {
		t.Fatalf("V2 lower-bound plan: summary=%+v err=%v", summary, err)
	}
	start := time.Date(2026, 8, 10, 9, 50, 0, 0, time.UTC)
	end := start.Add(70 * time.Minute)
	input, err := MaterializeSyncExecutionInputPlan(normalized, &start, &end)
	if err != nil {
		t.Fatal(err)
	}
	if got := input.QueryParams["changed_since"]; len(got) != 1 || got[0] != start.Format(time.RFC3339Nano) {
		t.Fatalf("lower bound=%v", got)
	}
	if len(input.QueryParams) != 1 {
		t.Fatalf("lower-bound plan injected an end parameter: %+v", input.QueryParams)
	}
}

func TestSyncExecutionInputPlanV2LowerBoundRejectsEndBinding(t *testing.T) {
	raw := []byte(`{"version":2,"window_mode":"lower_bound_only","static_input":{},"window_start_binding":{"location":"query","code":"updated_from","format":"rfc3339"},"window_end_binding":{"location":"query","code":"updated_to","format":"rfc3339"}}`)
	if _, _, err := NormalizeSyncExecutionInputPlan(raw, syncPlanContract(t, false), "POST", "/employees", 3, "timestamp"); !errors.Is(err, myerrors.ErrSyncInputPlanInvalid) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSyncExecutionInputPlanV1CannotBehaveAsLowerBound(t *testing.T) {
	raw := []byte(`{"version":1,"static_input":{},"window_start_binding":{"location":"query","code":"updated_from","format":"rfc3339"}}`)
	if _, err := DecodeSyncExecutionInputPlan(raw); !errors.Is(err, myerrors.ErrSyncInputPlanInvalid) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func syncPlanContract(t *testing.T, sensitive bool) []byte {
	t.Helper()
	value := InterfaceInputContract{Version: 1, Parameters: []InputParameterDefinition{
		{Code: "updated_from", Location: "query", DataType: "string", Required: true, MaxLength: 64},
		{Code: "updated_to", Location: "query", DataType: "string", Required: true, MaxLength: 64},
		{Code: "X-Correlation-ID", Location: "header", DataType: "string", MaxLength: 64, Sensitive: sensitive},
		{Code: "tenant", Location: "body", DataType: "string", Required: true, MaxLength: 64},
	}}
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func syncLowerBoundPlanContract(t *testing.T) []byte {
	t.Helper()
	value := InterfaceInputContract{Version: 1, Parameters: []InputParameterDefinition{
		{Code: "changed_since", Location: "query", DataType: "string", Required: true, MaxLength: 64},
	}}
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
