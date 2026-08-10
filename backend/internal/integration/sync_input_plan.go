package integration

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	myerrors "backend/internal/errors"
)

const SyncExecutionInputPlanVersion = 1

const (
	SyncTimeFormatRFC3339          = "rfc3339"
	SyncTimeFormatUnixSeconds      = "unix_seconds"
	SyncTimeFormatUnixMilliseconds = "unix_milliseconds"
)

type SyncWindowBinding struct {
	Location string `json:"location"`
	Code     string `json:"code"`
	Format   string `json:"format"`
}

type SyncExecutionInputPlan struct {
	Version            int                  `json:"version"`
	StaticInput        ExecutionInputValues `json:"static_input"`
	WindowStartBinding *SyncWindowBinding   `json:"window_start_binding,omitempty"`
	WindowEndBinding   *SyncWindowBinding   `json:"window_end_binding,omitempty"`
}

type SyncInputPlanSummary struct {
	Version              int  `json:"version"`
	StaticParameterCount int  `json:"static_parameter_count"`
	HasWindowBindings    bool `json:"has_window_bindings"`
}

// NormalizeSyncExecutionInputPlan 严格校验计划，并使用正式 ExecutionInputSnapshot 规范化器完成最终契约复核。
func NormalizeSyncExecutionInputPlan(raw, contractRaw []byte, method, relativePath string, interfaceVersion int, checkpointMode string) ([]byte, SyncInputPlanSummary, error) {
	var plan SyncExecutionInputPlan
	if len(bytes.TrimSpace(raw)) == 0 || decodeStrictJSON(raw, &plan) != nil || plan.Version != SyncExecutionInputPlanVersion {
		return nil, SyncInputPlanSummary{}, myerrors.ErrSyncInputPlanInvalid
	}
	contractBytes, err := NormalizeInputContract(contractRaw, method, relativePath)
	if err != nil {
		return nil, SyncInputPlanSummary{}, myerrors.ErrSyncInputPlanContractMismatch
	}
	var contract InterfaceInputContract
	if decodeStrictJSON(contractBytes, &contract) != nil {
		return nil, SyncInputPlanSummary{}, myerrors.ErrSyncInputPlanContractMismatch
	}
	definitions := make(map[string]InputParameterDefinition, len(contract.Parameters))
	for _, definition := range contract.Parameters {
		definitions[definition.Location+"\x00"+definition.Code] = definition
	}
	if checkpointMode == "none" {
		if plan.WindowStartBinding != nil || plan.WindowEndBinding != nil {
			return nil, SyncInputPlanSummary{}, myerrors.ErrSyncInputPlanInvalid
		}
	} else if checkpointMode == "timestamp" {
		if plan.WindowStartBinding == nil || plan.WindowEndBinding == nil {
			return nil, SyncInputPlanSummary{}, myerrors.ErrSyncInputPlanInvalid
		}
	} else {
		return nil, SyncInputPlanSummary{}, myerrors.ErrSyncCheckpointInvalid
	}
	input := cloneExecutionInputValues(plan.StaticInput)
	for _, binding := range []*SyncWindowBinding{plan.WindowStartBinding, plan.WindowEndBinding} {
		if binding == nil {
			continue
		}
		binding.Location = strings.ToLower(strings.TrimSpace(binding.Location))
		binding.Code = strings.TrimSpace(binding.Code)
		if binding.Location == InputLocationHeader {
			binding.Code = strings.ToLower(binding.Code)
		}
		binding.Format = strings.ToLower(strings.TrimSpace(binding.Format))
		definition, ok := definitions[binding.Location+"\x00"+binding.Code]
		if !ok || definition.Sensitive || definition.AllowMultiple || !validSyncTimeBinding(definition, binding.Format) || syncInputContains(input, binding.Location, binding.Code) {
			return nil, SyncInputPlanSummary{}, myerrors.ErrSyncInputPlanContractMismatch
		}
		if err := setSyncWindowValue(&input, *binding, syncBoundaryValue(binding.Format)); err != nil {
			return nil, SyncInputPlanSummary{}, err
		}
	}
	if plan.WindowStartBinding != nil && plan.WindowEndBinding != nil &&
		plan.WindowStartBinding.Location == plan.WindowEndBinding.Location && plan.WindowStartBinding.Code == plan.WindowEndBinding.Code {
		return nil, SyncInputPlanSummary{}, myerrors.ErrSyncInputPlanContractMismatch
	}
	if _, _, _, err := BuildExecutionInputSnapshot(contractRaw, method, relativePath, interfaceVersion, input); err != nil {
		return nil, SyncInputPlanSummary{}, myerrors.ErrSyncInputPlanContractMismatch
	}
	encoded, err := json.Marshal(plan)
	if err != nil || len(encoded) > MaxInputSnapshotSemanticBytes {
		return nil, SyncInputPlanSummary{}, myerrors.ErrSyncInputPlanInvalid
	}
	return encoded, summarizeSyncInputPlan(plan), nil
}

func SummarizeSyncExecutionInputPlan(raw []byte) SyncInputPlanSummary {
	var plan SyncExecutionInputPlan
	if decodeStrictJSON(raw, &plan) != nil {
		return SyncInputPlanSummary{}
	}
	return summarizeSyncInputPlan(plan)
}

func DecodeSyncExecutionInputPlan(raw []byte) (SyncExecutionInputPlan, error) {
	var plan SyncExecutionInputPlan
	if decodeStrictJSON(raw, &plan) != nil || plan.Version != SyncExecutionInputPlanVersion {
		return SyncExecutionInputPlan{}, myerrors.ErrSyncInputPlanInvalid
	}
	return plan, nil
}

// MaterializeSyncExecutionInputPlan 将已校验的计划绑定到当前切片窗口。
// 返回值仍需由 Integration Application Service 通过正式快照规范化器复核。
func MaterializeSyncExecutionInputPlan(raw []byte, windowStart, windowEnd *time.Time) (ExecutionInputValues, error) {
	plan, err := DecodeSyncExecutionInputPlan(raw)
	if err != nil {
		return ExecutionInputValues{}, err
	}
	input := cloneExecutionInputValues(plan.StaticInput)
	bindings := []struct {
		binding *SyncWindowBinding
		value   *time.Time
	}{
		{binding: plan.WindowStartBinding, value: windowStart},
		{binding: plan.WindowEndBinding, value: windowEnd},
	}
	for _, item := range bindings {
		if item.binding == nil && item.value == nil {
			continue
		}
		if item.binding == nil || item.value == nil {
			return ExecutionInputValues{}, myerrors.ErrSyncInputPlanInvalid
		}
		value, err := formatSyncWindowValue(item.value.UTC(), item.binding.Format)
		if err != nil {
			return ExecutionInputValues{}, err
		}
		if err := setSyncWindowValue(&input, *item.binding, value); err != nil {
			return ExecutionInputValues{}, err
		}
	}
	return input, nil
}

func formatSyncWindowValue(value time.Time, format string) (any, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case SyncTimeFormatRFC3339:
		return value.Format(time.RFC3339Nano), nil
	case SyncTimeFormatUnixSeconds:
		return json.Number(strconv.FormatInt(value.Unix(), 10)), nil
	case SyncTimeFormatUnixMilliseconds:
		return json.Number(strconv.FormatInt(value.UnixMilli(), 10)), nil
	default:
		return nil, myerrors.ErrSyncInputPlanInvalid
	}
}

func validSyncTimeBinding(definition InputParameterDefinition, format string) bool {
	if definition.Location != InputLocationPath && definition.Location != InputLocationQuery && definition.Location != InputLocationHeader && definition.Location != InputLocationBody {
		return false
	}
	switch format {
	case SyncTimeFormatRFC3339:
		return definition.DataType == InputTypeString
	case SyncTimeFormatUnixSeconds, SyncTimeFormatUnixMilliseconds:
		return definition.DataType == InputTypeString || definition.DataType == InputTypeInteger || definition.DataType == InputTypeNumber
	default:
		return false
	}
}

func syncBoundaryValue(format string) any {
	boundary := time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC)
	switch format {
	case SyncTimeFormatRFC3339:
		return boundary.Format(time.RFC3339)
	case SyncTimeFormatUnixSeconds:
		return json.Number("946782245")
	default:
		return json.Number("946782245000")
	}
}

func setSyncWindowValue(input *ExecutionInputValues, binding SyncWindowBinding, value any) error {
	text := ""
	switch item := value.(type) {
	case string:
		text = item
	case json.Number:
		text = item.String()
	}
	switch binding.Location {
	case InputLocationPath:
		if input.PathParams == nil {
			input.PathParams = map[string]string{}
		}
		input.PathParams[binding.Code] = text
	case InputLocationQuery:
		if input.QueryParams == nil {
			input.QueryParams = map[string][]string{}
		}
		input.QueryParams[binding.Code] = []string{text}
	case InputLocationHeader:
		if input.Headers == nil {
			input.Headers = map[string][]string{}
		}
		input.Headers[binding.Code] = []string{text}
	case InputLocationBody:
		body := map[string]any{}
		if len(bytes.TrimSpace(input.JSONBody)) > 0 && decodeStrictJSON(input.JSONBody, &body) != nil {
			return myerrors.ErrSyncInputPlanInvalid
		}
		body[binding.Code] = value
		encoded, err := json.Marshal(body)
		if err != nil {
			return myerrors.ErrSyncInputPlanInvalid
		}
		input.JSONBody = encoded
	default:
		return myerrors.ErrSyncInputPlanContractMismatch
	}
	return nil
}

func syncInputContains(input ExecutionInputValues, location, code string) bool {
	switch location {
	case InputLocationPath:
		_, ok := input.PathParams[code]
		return ok
	case InputLocationQuery:
		_, ok := input.QueryParams[code]
		return ok
	case InputLocationHeader:
		for key := range input.Headers {
			if strings.EqualFold(key, code) {
				return true
			}
		}
		return false
	case InputLocationBody:
		body := map[string]any{}
		if len(bytes.TrimSpace(input.JSONBody)) == 0 {
			return false
		}
		if decodeStrictJSON(input.JSONBody, &body) != nil {
			return true
		}
		_, ok := body[code]
		return ok
	default:
		return true
	}
}

func cloneExecutionInputValues(value ExecutionInputValues) ExecutionInputValues {
	result := ExecutionInputValues{PathParams: map[string]string{}, QueryParams: map[string][]string{}, Headers: map[string][]string{}, JSONBody: append(json.RawMessage(nil), value.JSONBody...)}
	for key, item := range value.PathParams {
		result.PathParams[key] = item
	}
	for key, items := range value.QueryParams {
		result.QueryParams[key] = append([]string(nil), items...)
	}
	for key, items := range value.Headers {
		result.Headers[key] = append([]string(nil), items...)
	}
	return result
}

func summarizeSyncInputPlan(plan SyncExecutionInputPlan) SyncInputPlanSummary {
	count := len(plan.StaticInput.PathParams) + len(plan.StaticInput.QueryParams) + len(plan.StaticInput.Headers)
	var body map[string]any
	if json.Unmarshal(plan.StaticInput.JSONBody, &body) == nil {
		count += len(body)
	}
	return SyncInputPlanSummary{Version: plan.Version, StaticParameterCount: count, HasWindowBindings: plan.WindowStartBinding != nil && plan.WindowEndBinding != nil}
}
