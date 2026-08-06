package integration

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	myerrors "backend/internal/errors"
)

func TestExecutionInputSnapshotNormalizationAndHash(t *testing.T) {
	contract := snapshotTestContract()
	first := ExecutionInputValues{
		PathParams:  map[string]string{"employee_id": "10001"},
		QueryParams: map[string][]string{"tags": {"south", "east"}, "page": {"1"}},
		Headers:     map[string][]string{"X-Correlation-ID": {"business-correlation"}},
		JSONBody:    json.RawMessage(`{"active":true,"items":[2,1],"name":"张三"}`),
	}
	second := ExecutionInputValues{
		PathParams:  map[string]string{"employee_id": "10001"},
		QueryParams: map[string][]string{"page": {"1"}, "tags": {"east", "south"}},
		Headers:     map[string][]string{"x-correlation-id": {"business-correlation"}},
		JSONBody:    json.RawMessage(` { "name":"张三", "items":[2,1], "active":true } `),
	}
	firstSnapshot, firstBytes, firstHash, err := BuildExecutionInputSnapshot(contract, "POST", "/api/employees/{employee_id}", 3, first)
	if err != nil {
		t.Fatalf("build first snapshot: %v", err)
	}
	secondSnapshot, secondBytes, secondHash, err := BuildExecutionInputSnapshot(contract, "POST", "/api/employees/{employee_id}", 3, second)
	if err != nil {
		t.Fatalf("build second snapshot: %v", err)
	}
	if firstHash != secondHash || string(firstBytes) != string(secondBytes) {
		t.Fatalf("semantic equivalents diverged: %s/%s\n%s\n%s", firstHash, secondHash, firstBytes, secondBytes)
	}
	if got := firstSnapshot.QueryParams["tags"]; len(got) != 2 || got[0] != "east" || got[1] != "south" {
		t.Fatalf("query values were not normalized: %#v", got)
	}
	if _, exists := secondSnapshot.Headers["x-correlation-id"]; !exists {
		t.Fatalf("header name was not normalized: %#v", secondSnapshot.Headers)
	}

	arrayChanged := second
	arrayChanged.JSONBody = json.RawMessage(`{"name":"张三","items":[1,2],"active":true}`)
	_, _, changedHash, err := BuildExecutionInputSnapshot(contract, "POST", "/api/employees/{employee_id}", 3, arrayChanged)
	if err != nil {
		t.Fatalf("build changed snapshot: %v", err)
	}
	if changedHash == firstHash {
		t.Fatal("JSON array order must remain semantic")
	}
	_, _, otherVersionHash, err := BuildExecutionInputSnapshot(contract, "POST", "/api/employees/{employee_id}", 4, first)
	if err != nil || otherVersionHash == firstHash {
		t.Fatalf("interface version must participate in hash: hash=%s err=%v", otherVersionHash, err)
	}
}

func TestExecutionInputSnapshotRejectsContractAndInputViolations(t *testing.T) {
	contract := snapshotTestContract()
	valid := ExecutionInputValues{
		PathParams:  map[string]string{"employee_id": "10001"},
		QueryParams: map[string][]string{"page": {"1"}},
		Headers:     map[string][]string{"x-correlation-id": {"business-correlation"}},
		JSONBody:    json.RawMessage(`{"name":"张三","active":true,"items":[]}`),
	}
	tests := []struct {
		name     string
		contract []byte
		method   string
		path     string
		input    ExecutionInputValues
		want     error
	}{
		{name: "missing path", contract: contract, method: "POST", path: "/api/employees/{employee_id}", input: mutateInput(valid, func(v *ExecutionInputValues) { v.PathParams = nil }), want: myerrors.ErrIntegrationExecutionPathParameterMissing},
		{name: "unknown path", contract: contract, method: "POST", path: "/api/employees/{employee_id}", input: mutateInput(valid, func(v *ExecutionInputValues) { v.PathParams["other"] = "x" }), want: myerrors.ErrIntegrationExecutionPathParameterUnknown},
		{name: "path escape", contract: contract, method: "POST", path: "/api/employees/{employee_id}", input: mutateInput(valid, func(v *ExecutionInputValues) { v.PathParams["employee_id"] = "../admin" }), want: myerrors.ErrIntegrationExecutionInputInvalid},
		{name: "unknown query", contract: contract, method: "POST", path: "/api/employees/{employee_id}", input: mutateInput(valid, func(v *ExecutionInputValues) { v.QueryParams["unknown"] = []string{"x"} }), want: myerrors.ErrIntegrationExecutionQueryParameterInvalid},
		{name: "query type", contract: contract, method: "POST", path: "/api/employees/{employee_id}", input: mutateInput(valid, func(v *ExecutionInputValues) { v.QueryParams["page"] = []string{"one"} }), want: myerrors.ErrIntegrationExecutionQueryParameterInvalid},
		{name: "authorization", contract: contract, method: "POST", path: "/api/employees/{employee_id}", input: mutateInput(valid, func(v *ExecutionInputValues) { v.Headers["Authorization"] = []string{"Bearer bad"} }), want: myerrors.ErrIntegrationExecutionHeaderNotAllowed},
		{name: "header newline", contract: contract, method: "POST", path: "/api/employees/{employee_id}", input: mutateInput(valid, func(v *ExecutionInputValues) { v.Headers["x-correlation-id"] = []string{"ok\r\nbad"} }), want: myerrors.ErrIntegrationExecutionHeaderNotAllowed},
		{name: "get body", contract: contract, method: "GET", path: "/api/employees/{employee_id}", input: valid, want: myerrors.ErrIntegrationExecutionBodyInvalid},
		{name: "invalid JSON", contract: contract, method: "POST", path: "/api/employees/{employee_id}", input: mutateInput(valid, func(v *ExecutionInputValues) { v.JSONBody = json.RawMessage(`{"name":`) }), want: myerrors.ErrIntegrationExecutionBodyInvalid},
		{name: "body type", contract: contract, method: "POST", path: "/api/employees/{employee_id}", input: mutateInput(valid, func(v *ExecutionInputValues) { v.JSONBody = json.RawMessage(`{"name":1,"active":true,"items":[]}`) }), want: myerrors.ErrIntegrationExecutionBodyInvalid},
		{name: "body unknown", contract: contract, method: "POST", path: "/api/employees/{employee_id}", input: mutateInput(valid, func(v *ExecutionInputValues) {
			v.JSONBody = json.RawMessage(`{"name":"张三","active":true,"items":[],"token":"bad"}`)
		}), want: myerrors.ErrIntegrationExecutionSensitiveInputRejected},
		{name: "camel case sensitive body", contract: contract, method: "POST", path: "/api/employees/{employee_id}", input: mutateInput(valid, func(v *ExecutionInputValues) {
			v.JSONBody = json.RawMessage(`{"name":"张三","active":true,"items":[],"apiKey":"bad"}`)
		}), want: myerrors.ErrIntegrationExecutionSensitiveInputRejected},
		{name: "body too large", contract: contract, method: "POST", path: "/api/employees/{employee_id}", input: mutateInput(valid, func(v *ExecutionInputValues) {
			v.JSONBody = json.RawMessage(`{"name":"` + strings.Repeat("a", MaxInputJSONBodyBytes) + `"}`)
		}), want: myerrors.ErrIntegrationExecutionInputSemanticTooLarge},
		{name: "sensitive contract", contract: []byte(`{"version":1,"parameters":[{"code":"token","location":"query","data_type":"string","required":false,"max_length":10,"allow_multiple":false,"sensitive":true}]}`), method: "POST", path: "/api/static", input: ExecutionInputValues{}, want: myerrors.ErrIntegrationExecutionSensitiveInputRejected},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, _, err := BuildExecutionInputSnapshot(test.contract, test.method, test.path, 1, test.input)
			if !errors.Is(err, test.want) {
				t.Fatalf("error=%v want=%v", err, test.want)
			}
		})
	}
}

func TestExecutionInputSnapshotComplexityAndIntegrity(t *testing.T) {
	contract := []byte(`{"version":1,"parameters":[{"code":"payload","location":"body","data_type":"object","required":true,"max_length":4096,"allow_multiple":false,"sensitive":false}]}`)
	nested := `"ok"`
	for index := 0; index < MaxInputJSONDepth+1; index++ {
		nested = `{"level":` + nested + `}`
	}
	_, _, _, err := BuildExecutionInputSnapshot(contract, "POST", "/api/static", 1, ExecutionInputValues{JSONBody: json.RawMessage(`{"payload":` + nested + `}`)})
	if !errors.Is(err, myerrors.ErrIntegrationExecutionInputSemanticTooLarge) {
		t.Fatalf("depth error=%v", err)
	}

	_, encoded, hash, err := BuildExecutionInputSnapshot(nil, "GET", "/api/static", 1, ExecutionInputValues{})
	if err != nil {
		t.Fatalf("build empty input: %v", err)
	}
	if _, err := LoadExecutionInputSnapshot(nil, "GET", "/api/static", 1, encoded, len(encoded), hash); err != nil {
		t.Fatalf("load valid snapshot: %v", err)
	}
	jsonbStyle := []byte(`{"headers": {}, "path_params": {}, "query_params": {}, "version": 1}`)
	if _, err := LoadExecutionInputSnapshot(nil, "GET", "/api/static", 1, jsonbStyle, len(encoded), hash); err != nil {
		t.Fatalf("load semantically equal JSONB representation: %v", err)
	}
	if _, err := LoadExecutionInputSnapshot(nil, "GET", "/api/static", 1, jsonbStyle, len(encoded)+1, hash); !errors.Is(err, myerrors.ErrIntegrationExecutionInputSizeMismatch) {
		t.Fatalf("size mismatch error=%v", err)
	}
	tampered := append([]byte(nil), encoded...)
	tampered = append(tampered[:len(tampered)-1], []byte(`,"json_body":null}`)...)
	if _, err := LoadExecutionInputSnapshot(nil, "GET", "/api/static", 1, tampered, len(tampered), hash); err == nil {
		t.Fatal("tampered snapshot must fail")
	}
	if _, err := LoadExecutionInputSnapshot(nil, "GET", "/api/static", 1, nil, 0, hash); !errors.Is(err, myerrors.ErrIntegrationExecutionInputMissing) {
		t.Fatalf("missing snapshot error=%v", err)
	}
	if _, err := LoadExecutionInputSnapshot(nil, "GET", "/api/static", 1, encoded, len(encoded), strings.Repeat("0", 64)); !errors.Is(err, myerrors.ErrIntegrationExecutionInputHashMismatch) {
		t.Fatalf("hash mismatch error=%v", err)
	}
}

func TestExecutionInputSnapshotConcurrentNormalization(t *testing.T) {
	contract := snapshotTestContract()
	input := ExecutionInputValues{
		PathParams: map[string]string{"employee_id": "10001"}, QueryParams: map[string][]string{"page": {"1"}},
		Headers:  map[string][]string{"x-correlation-id": {"business-correlation"}},
		JSONBody: json.RawMessage(`{"name":"张三","active":true,"items":[]}`),
	}
	var wait sync.WaitGroup
	errorsChannel := make(chan error, 32)
	for index := 0; index < 32; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, _, _, err := BuildExecutionInputSnapshot(contract, "POST", "/api/employees/{employee_id}", 1, input)
			errorsChannel <- err
		}()
	}
	wait.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		if err != nil {
			t.Fatalf("concurrent normalize: %v", err)
		}
	}
}

func snapshotTestContract() []byte {
	return []byte(`{
		"version":1,
		"parameters":[
			{"code":"employee_id","location":"path","data_type":"string","required":true,"max_length":64,"allow_multiple":false,"sensitive":false},
			{"code":"page","location":"query","data_type":"integer","required":true,"max_length":8,"allow_multiple":false,"sensitive":false},
			{"code":"tags","location":"query","data_type":"string","required":false,"max_length":32,"allow_multiple":true,"sensitive":false},
			{"code":"X-Correlation-ID","location":"header","data_type":"string","required":true,"max_length":64,"allow_multiple":false,"sensitive":false},
			{"code":"name","location":"body","data_type":"string","required":true,"max_length":128,"allow_multiple":false,"sensitive":false},
			{"code":"active","location":"body","data_type":"boolean","required":true,"max_length":8,"allow_multiple":false,"sensitive":false},
			{"code":"items","location":"body","data_type":"array","required":true,"max_length":32,"allow_multiple":false,"sensitive":false}
		]
	}`)
}

func mutateInput(source ExecutionInputValues, mutate func(*ExecutionInputValues)) ExecutionInputValues {
	value := ExecutionInputValues{
		PathParams: cloneStringMap(source.PathParams), QueryParams: cloneStringSliceMap(source.QueryParams),
		Headers: cloneStringSliceMap(source.Headers), JSONBody: append(json.RawMessage(nil), source.JSONBody...),
	}
	mutate(&value)
	return value
}
