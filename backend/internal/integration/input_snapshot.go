package integration

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	myerrors "backend/internal/errors"
	"backend/model"
)

const (
	ExecutionInputSnapshotVersion = model.IntegrationExecutionInputSnapshotVersion
	InputContractVersion          = 1

	MaxInputParameterDefinitions  = 128
	MaxInputPathParameters        = 32
	MaxInputPathBytes             = 4 * 1024
	MaxInputQueryParameters       = 64
	MaxInputQueryBytes            = 16 * 1024
	MaxInputHeaderParameters      = 16
	MaxInputHeaderBytes           = 8 * 1024
	MaxInputJSONBodyBytes         = 256 * 1024
	MaxInputSnapshotSemanticBytes = 384 * 1024
	MaxInputSnapshotStorageBytes  = 512 * 1024
	MaxInputJSONDepth             = 16
	MaxInputJSONArrayElements     = 256
	MaxInputJSONFields            = 256
	MaxInputJSONStringBytes       = 4 * 1024
)

const (
	InputLocationPath   = "path"
	InputLocationQuery  = "query"
	InputLocationHeader = "header"
	InputLocationBody   = "body"

	InputTypeString  = "string"
	InputTypeInteger = "integer"
	InputTypeNumber  = "number"
	InputTypeBoolean = "boolean"
	InputTypeObject  = "object"
	InputTypeArray   = "array"
	InputTypeNull    = "null"
)

var (
	inputParameterCodePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,63}$`)
	inputHeaderCodePattern    = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9-]{0,63}$`)
	pathPlaceholderPattern    = regexp.MustCompile(`\{([A-Za-z][A-Za-z0-9_]{0,63})\}`)
	jsonNumberPattern         = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?$`)
)

// InputParameterDefinition 是 InterfaceDefinition 中冻结的最小参数契约。
type InputParameterDefinition struct {
	Code          string `json:"code"`
	Location      string `json:"location"`
	DataType      string `json:"data_type"`
	Required      bool   `json:"required"`
	MaxLength     int    `json:"max_length"`
	AllowMultiple bool   `json:"allow_multiple"`
	Sensitive     bool   `json:"sensitive"`
}

// InterfaceInputContract 不包含脚本、模板、表达式或字段映射能力。
type InterfaceInputContract struct {
	Version    int                        `json:"version"`
	Parameters []InputParameterDefinition `json:"parameters"`
}

// ExecutionInputValues 是创建 Execution 时唯一允许提交的结构化输入。
type ExecutionInputValues struct {
	PathParams  map[string]string   `json:"path_params,omitempty"`
	QueryParams map[string][]string `json:"query_params,omitempty"`
	Headers     map[string][]string `json:"headers,omitempty"`
	JSONBody    json.RawMessage     `json:"json_body,omitempty"`
}

// ExecutionInputSnapshot 是持久化的规范化输入，不包含认证、目标地址或 TLS 配置。
type ExecutionInputSnapshot struct {
	Version     int                 `json:"version"`
	PathParams  map[string]string   `json:"path_params"`
	QueryParams map[string][]string `json:"query_params"`
	Headers     map[string][]string `json:"headers"`
	JSONBody    json.RawMessage     `json:"json_body,omitempty"`
}

// ExecutionInputSummary 是可安全返回给管理端的快照摘要。
type ExecutionInputSummary struct {
	SnapshotVersion int  `json:"snapshot_version"`
	SizeBytes       int  `json:"size_bytes"`
	PathCount       int  `json:"path_count"`
	QueryCount      int  `json:"query_count"`
	HeaderCount     int  `json:"header_count"`
	HasBody         bool `json:"has_body"`
}

// NormalizeInputContract 规范化并验证接口参数契约和路径占位符关系。
func NormalizeInputContract(raw []byte, method, relativePath string) ([]byte, error) {
	contract, err := parseInputContract(raw)
	if err != nil {
		return nil, err
	}
	if err := validateInputContract(&contract, method, relativePath); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(contract)
	if err != nil {
		return nil, myerrors.ErrIntegrationExecutionInputContractMismatch
	}
	return encoded, nil
}

// BuildExecutionInputSnapshot 根据已冻结契约生成规范化快照和服务端 Hash。
func BuildExecutionInputSnapshot(
	contractRaw []byte,
	method string,
	relativePath string,
	interfaceVersion int,
	input ExecutionInputValues,
) (ExecutionInputSnapshot, []byte, string, error) {
	snapshot, encoded, err := normalizeExecutionInputSnapshot(contractRaw, method, relativePath, input)
	if err != nil {
		return ExecutionInputSnapshot{}, nil, "", err
	}
	hash, err := executionInputHash(interfaceVersion, encoded)
	if err != nil {
		return ExecutionInputSnapshot{}, nil, "", err
	}
	return snapshot, encoded, hash, nil
}

// LoadExecutionInputSnapshot 在 Worker 执行前按 JSON 语义重新规范化并校验大小和 Hash。
func LoadExecutionInputSnapshot(
	contractRaw []byte,
	method string,
	relativePath string,
	interfaceVersion int,
	snapshotRaw []byte,
	expectedSize int,
	expectedHash string,
) (ExecutionInputSnapshot, error) {
	if len(snapshotRaw) == 0 || expectedSize <= 0 {
		return ExecutionInputSnapshot{}, myerrors.ErrIntegrationExecutionInputMissing
	}
	if len(snapshotRaw) > MaxInputSnapshotStorageBytes {
		return ExecutionInputSnapshot{}, myerrors.ErrIntegrationExecutionInputStorageTooLarge
	}
	var stored ExecutionInputSnapshot
	if err := decodeStrictJSON(snapshotRaw, &stored); err != nil {
		return ExecutionInputSnapshot{}, myerrors.ErrIntegrationExecutionInputInvalid
	}
	if stored.Version != ExecutionInputSnapshotVersion {
		return ExecutionInputSnapshot{}, myerrors.ErrIntegrationExecutionInputVersionUnsupported
	}
	rebuilt, canonical, err := normalizeExecutionInputSnapshot(
		contractRaw,
		method,
		relativePath,
		ExecutionInputValues{
			PathParams: stored.PathParams, QueryParams: stored.QueryParams,
			Headers: stored.Headers, JSONBody: stored.JSONBody,
		},
	)
	if err != nil {
		return ExecutionInputSnapshot{}, err
	}
	if len(canonical) != expectedSize {
		return ExecutionInputSnapshot{}, myerrors.ErrIntegrationExecutionInputSizeMismatch
	}
	hash, err := executionInputHash(interfaceVersion, canonical)
	if err != nil {
		return ExecutionInputSnapshot{}, err
	}
	if !strings.EqualFold(hash, strings.TrimSpace(expectedHash)) {
		return ExecutionInputSnapshot{}, myerrors.ErrIntegrationExecutionInputHashMismatch
	}
	return rebuilt, nil
}

func normalizeExecutionInputSnapshot(
	contractRaw []byte,
	method string,
	relativePath string,
	input ExecutionInputValues,
) (ExecutionInputSnapshot, []byte, error) {
	contractBytes, err := NormalizeInputContract(contractRaw, method, relativePath)
	if err != nil {
		return ExecutionInputSnapshot{}, nil, err
	}
	var contract InterfaceInputContract
	if err := decodeStrictJSON(contractBytes, &contract); err != nil {
		return ExecutionInputSnapshot{}, nil, myerrors.ErrIntegrationExecutionInputContractMismatch
	}
	snapshot, err := normalizeExecutionInput(contract, method, input)
	if err != nil {
		return ExecutionInputSnapshot{}, nil, err
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return ExecutionInputSnapshot{}, nil, myerrors.ErrIntegrationExecutionInputInvalid
	}
	if len(encoded) > MaxInputSnapshotSemanticBytes {
		return ExecutionInputSnapshot{}, nil, myerrors.ErrIntegrationExecutionInputSemanticTooLarge
	}
	return snapshot, encoded, nil
}

func SummarizeExecutionInput(snapshotRaw []byte, version, size int) ExecutionInputSummary {
	summary := ExecutionInputSummary{SnapshotVersion: version, SizeBytes: size}
	if version != ExecutionInputSnapshotVersion || len(snapshotRaw) == 0 {
		return summary
	}
	var snapshot ExecutionInputSnapshot
	if decodeStrictJSON(snapshotRaw, &snapshot) != nil {
		return summary
	}
	summary.PathCount = len(snapshot.PathParams)
	summary.QueryCount = len(snapshot.QueryParams)
	summary.HeaderCount = len(snapshot.Headers)
	summary.HasBody = len(snapshot.JSONBody) > 0
	return summary
}

func parseInputContract(raw []byte) (InterfaceInputContract, error) {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("{}")) {
		return InterfaceInputContract{Version: InputContractVersion, Parameters: []InputParameterDefinition{}}, nil
	}
	var contract InterfaceInputContract
	if err := decodeStrictJSON(raw, &contract); err != nil {
		return InterfaceInputContract{}, myerrors.ErrIntegrationExecutionInputContractMismatch
	}
	if contract.Parameters == nil {
		contract.Parameters = []InputParameterDefinition{}
	}
	return contract, nil
}

func validateInputContract(contract *InterfaceInputContract, method, relativePath string) error {
	if contract == nil || contract.Version != InputContractVersion || len(contract.Parameters) > MaxInputParameterDefinitions {
		return myerrors.ErrIntegrationExecutionInputContractMismatch
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	seen := make(map[string]struct{}, len(contract.Parameters))
	pathDefinitions := make(map[string]struct{})
	for index := range contract.Parameters {
		parameter := &contract.Parameters[index]
		parameter.Code = strings.TrimSpace(parameter.Code)
		parameter.Location = strings.ToLower(strings.TrimSpace(parameter.Location))
		parameter.DataType = strings.ToLower(strings.TrimSpace(parameter.DataType))
		if parameter.Location == InputLocationHeader {
			parameter.Code = strings.ToLower(parameter.Code)
		}
		validCode := inputParameterCodePattern.MatchString(parameter.Code)
		if parameter.Location == InputLocationHeader {
			validCode = inputHeaderCodePattern.MatchString(parameter.Code)
		}
		if parameter.Sensitive {
			return myerrors.ErrIntegrationExecutionSensitiveInputRejected
		}
		if !validCode || forbiddenInputName(parameter.Code) ||
			!validInputLocation(parameter.Location) || !validInputType(parameter.Location, parameter.DataType) {
			return myerrors.ErrIntegrationExecutionInputContractMismatch
		}
		if parameter.MaxLength == 0 {
			parameter.MaxLength = defaultParameterMaxLength(parameter.Location)
		}
		if parameter.MaxLength < 1 || parameter.MaxLength > maxParameterLength(parameter.Location) {
			return myerrors.ErrIntegrationExecutionInputContractMismatch
		}
		if parameter.AllowMultiple && parameter.Location != InputLocationQuery {
			return myerrors.ErrIntegrationExecutionInputContractMismatch
		}
		if parameter.Location == InputLocationHeader && !allowedSnapshotHeader(parameter.Code) {
			return myerrors.ErrIntegrationExecutionHeaderNotAllowed
		}
		if parameter.Location == InputLocationBody && method == "GET" {
			return myerrors.ErrIntegrationExecutionBodyInvalid
		}
		key := parameter.Location + "\x00" + parameter.Code
		if _, exists := seen[key]; exists {
			return myerrors.ErrIntegrationExecutionInputContractMismatch
		}
		seen[key] = struct{}{}
		if parameter.Location == InputLocationPath {
			if parameter.DataType != InputTypeString || parameter.AllowMultiple || !parameter.Required {
				return myerrors.ErrIntegrationExecutionInputContractMismatch
			}
			pathDefinitions[parameter.Code] = struct{}{}
		}
	}
	placeholders := pathPlaceholderPattern.FindAllStringSubmatch(relativePath, -1)
	pathPlaceholders := make(map[string]struct{}, len(placeholders))
	for _, placeholder := range placeholders {
		pathPlaceholders[placeholder[1]] = struct{}{}
	}
	if len(pathPlaceholders) != len(pathDefinitions) {
		return myerrors.ErrIntegrationExecutionInputContractMismatch
	}
	for name := range pathPlaceholders {
		if _, exists := pathDefinitions[name]; !exists {
			return myerrors.ErrIntegrationExecutionInputContractMismatch
		}
	}
	if strings.ContainsAny(pathPlaceholderPattern.ReplaceAllString(relativePath, ""), "{}") {
		return myerrors.ErrIntegrationExecutionInputContractMismatch
	}
	return nil
}

func normalizeExecutionInput(contract InterfaceInputContract, method string, input ExecutionInputValues) (ExecutionInputSnapshot, error) {
	definitions := make(map[string]InputParameterDefinition, len(contract.Parameters))
	for _, definition := range contract.Parameters {
		definitions[definition.Location+"\x00"+definition.Code] = definition
	}
	path, err := normalizePathInput(input.PathParams, definitions)
	if err != nil {
		return ExecutionInputSnapshot{}, err
	}
	query, err := normalizeQueryInput(input.QueryParams, definitions)
	if err != nil {
		return ExecutionInputSnapshot{}, err
	}
	headers, err := normalizeHeaderInput(input.Headers, definitions)
	if err != nil {
		return ExecutionInputSnapshot{}, err
	}
	body, err := normalizeBodyInput(input.JSONBody, strings.ToUpper(strings.TrimSpace(method)), definitions)
	if err != nil {
		return ExecutionInputSnapshot{}, err
	}
	return ExecutionInputSnapshot{
		Version: ExecutionInputSnapshotVersion, PathParams: path,
		QueryParams: query, Headers: headers, JSONBody: body,
	}, nil
}

func normalizePathInput(input map[string]string, definitions map[string]InputParameterDefinition) (map[string]string, error) {
	if len(input) > MaxInputPathParameters {
		return nil, myerrors.ErrIntegrationExecutionInputSemanticTooLarge
	}
	result := make(map[string]string, len(input))
	total := 0
	for name, value := range input {
		definition, exists := definitions[InputLocationPath+"\x00"+name]
		if !exists {
			return nil, myerrors.ErrIntegrationExecutionPathParameterUnknown
		}
		if value == "" || len(value) > definition.MaxLength || strings.ContainsAny(value, "/\\?&#%") ||
			strings.Contains(value, "..") || containsControlCharacter(value) {
			return nil, myerrors.ErrIntegrationExecutionInputInvalid
		}
		total += len(name) + len(value)
		result[name] = value
	}
	for _, definition := range definitions {
		if definition.Location == InputLocationPath && definition.Required {
			if _, exists := result[definition.Code]; !exists {
				return nil, myerrors.ErrIntegrationExecutionPathParameterMissing
			}
		}
	}
	if total > MaxInputPathBytes {
		return nil, myerrors.ErrIntegrationExecutionInputSemanticTooLarge
	}
	return result, nil
}

func normalizeQueryInput(input map[string][]string, definitions map[string]InputParameterDefinition) (map[string][]string, error) {
	if len(input) > MaxInputQueryParameters {
		return nil, myerrors.ErrIntegrationExecutionInputSemanticTooLarge
	}
	result := make(map[string][]string, len(input))
	total := 0
	for name, values := range input {
		definition, exists := definitions[InputLocationQuery+"\x00"+name]
		if !exists || len(values) == 0 || !definition.AllowMultiple && len(values) != 1 {
			return nil, myerrors.ErrIntegrationExecutionQueryParameterInvalid
		}
		cloned := make([]string, len(values))
		for index, value := range values {
			if len(value) > definition.MaxLength || containsControlCharacter(value) || !validScalarString(value, definition.DataType) {
				return nil, myerrors.ErrIntegrationExecutionQueryParameterInvalid
			}
			total += len(name) + len(value)
			cloned[index] = value
		}
		if definition.AllowMultiple {
			sort.Strings(cloned)
		}
		result[name] = cloned
	}
	for _, definition := range definitions {
		if definition.Location == InputLocationQuery && definition.Required {
			if _, exists := result[definition.Code]; !exists {
				return nil, myerrors.ErrIntegrationExecutionQueryParameterInvalid
			}
		}
	}
	if total > MaxInputQueryBytes {
		return nil, myerrors.ErrIntegrationExecutionInputSemanticTooLarge
	}
	return result, nil
}

func normalizeHeaderInput(input map[string][]string, definitions map[string]InputParameterDefinition) (map[string][]string, error) {
	if len(input) > MaxInputHeaderParameters {
		return nil, myerrors.ErrIntegrationExecutionInputSemanticTooLarge
	}
	result := make(map[string][]string, len(input))
	total := 0
	for rawName, values := range input {
		name := strings.ToLower(strings.TrimSpace(rawName))
		definition, exists := definitions[InputLocationHeader+"\x00"+name]
		if !exists || !allowedSnapshotHeader(name) || len(values) != 1 {
			return nil, myerrors.ErrIntegrationExecutionHeaderNotAllowed
		}
		if _, duplicate := result[name]; duplicate {
			return nil, myerrors.ErrIntegrationExecutionHeaderNotAllowed
		}
		value := strings.TrimSpace(values[0])
		if value == "" || len(value) > definition.MaxLength || containsControlCharacter(value) {
			return nil, myerrors.ErrIntegrationExecutionHeaderNotAllowed
		}
		total += len(name) + len(value)
		result[name] = []string{value}
	}
	for _, definition := range definitions {
		if definition.Location == InputLocationHeader && definition.Required {
			if _, exists := result[definition.Code]; !exists {
				return nil, myerrors.ErrIntegrationExecutionHeaderNotAllowed
			}
		}
	}
	if total > MaxInputHeaderBytes {
		return nil, myerrors.ErrIntegrationExecutionInputSemanticTooLarge
	}
	return result, nil
}

func normalizeBodyInput(raw json.RawMessage, method string, definitions map[string]InputParameterDefinition) (json.RawMessage, error) {
	bodyDefinitions := make(map[string]InputParameterDefinition)
	for _, definition := range definitions {
		if definition.Location == InputLocationBody {
			bodyDefinitions[definition.Code] = definition
		}
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		for _, definition := range bodyDefinitions {
			if definition.Required {
				return nil, myerrors.ErrIntegrationExecutionBodyInvalid
			}
		}
		return nil, nil
	}
	if method == "GET" || len(raw) > MaxInputJSONBodyBytes || len(bodyDefinitions) == 0 {
		if len(raw) > MaxInputJSONBodyBytes {
			return nil, myerrors.ErrIntegrationExecutionInputSemanticTooLarge
		}
		return nil, myerrors.ErrIntegrationExecutionBodyInvalid
	}
	var value any
	if err := decodeStrictJSONWithNumber(raw, &value); err != nil {
		return nil, myerrors.ErrIntegrationExecutionBodyInvalid
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, myerrors.ErrIntegrationExecutionBodyInvalid
	}
	fields := 0
	if err := validateJSONComplexity(object, 1, &fields); err != nil {
		return nil, err
	}
	for name, item := range object {
		definition, exists := bodyDefinitions[name]
		if !exists || forbiddenInputName(name) || !validJSONValueType(item, definition.DataType) || !validJSONValueLength(item, definition.MaxLength) {
			return nil, myerrors.ErrIntegrationExecutionBodyInvalid
		}
	}
	for name, definition := range bodyDefinitions {
		if definition.Required {
			if _, exists := object[name]; !exists {
				return nil, myerrors.ErrIntegrationExecutionBodyInvalid
			}
		}
	}
	encoded, err := json.Marshal(object)
	if err != nil || len(encoded) > MaxInputJSONBodyBytes {
		return nil, myerrors.ErrIntegrationExecutionInputSemanticTooLarge
	}
	return encoded, nil
}

func validateJSONComplexity(value any, depth int, fields *int) error {
	if depth > MaxInputJSONDepth {
		return myerrors.ErrIntegrationExecutionInputSemanticTooLarge
	}
	switch item := value.(type) {
	case map[string]any:
		*fields += len(item)
		if *fields > MaxInputJSONFields {
			return myerrors.ErrIntegrationExecutionInputSemanticTooLarge
		}
		for name, child := range item {
			if len(name) > 64 || containsControlCharacter(name) || forbiddenInputName(name) {
				return myerrors.ErrIntegrationExecutionSensitiveInputRejected
			}
			if err := validateJSONComplexity(child, depth+1, fields); err != nil {
				return err
			}
		}
	case []any:
		if len(item) > MaxInputJSONArrayElements {
			return myerrors.ErrIntegrationExecutionInputSemanticTooLarge
		}
		for _, child := range item {
			if err := validateJSONComplexity(child, depth+1, fields); err != nil {
				return err
			}
		}
	case string:
		if len(item) > MaxInputJSONStringBytes || containsControlCharacter(item) {
			return myerrors.ErrIntegrationExecutionInputSemanticTooLarge
		}
	}
	return nil
}

func executionInputHash(interfaceVersion int, snapshot []byte) (string, error) {
	if interfaceVersion <= 0 {
		return "", myerrors.ErrIntegrationExecutionInputInvalid
	}
	envelope, err := json.Marshal(struct {
		InterfaceVersion int             `json:"interface_version"`
		Input            json.RawMessage `json:"input"`
	}{InterfaceVersion: interfaceVersion, Input: snapshot})
	if err != nil {
		return "", myerrors.ErrIntegrationExecutionInputInvalid
	}
	digest := sha256.Sum256(envelope)
	return hex.EncodeToString(digest[:]), nil
}

func decodeStrictJSON(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("multiple JSON values")
	}
	return nil
}

func decodeStrictJSONWithNumber(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("multiple JSON values")
	}
	return nil
}

func validScalarString(value, dataType string) bool {
	switch dataType {
	case InputTypeString:
		return true
	case InputTypeInteger:
		_, err := strconv.ParseInt(value, 10, 64)
		return err == nil
	case InputTypeNumber:
		return jsonNumberPattern.MatchString(value)
	case InputTypeBoolean:
		return value == "true" || value == "false"
	default:
		return false
	}
}

func validJSONValueType(value any, dataType string) bool {
	if value == nil {
		return dataType == InputTypeNull
	}
	switch dataType {
	case InputTypeString:
		_, ok := value.(string)
		return ok
	case InputTypeInteger:
		number, ok := value.(json.Number)
		if !ok {
			return false
		}
		_, err := number.Int64()
		return err == nil
	case InputTypeNumber:
		number, ok := value.(json.Number)
		return ok && jsonNumberPattern.MatchString(number.String())
	case InputTypeBoolean:
		_, ok := value.(bool)
		return ok
	case InputTypeObject:
		_, ok := value.(map[string]any)
		return ok
	case InputTypeArray:
		_, ok := value.([]any)
		return ok
	case InputTypeNull:
		return value == nil
	default:
		return false
	}
}

func validJSONValueLength(value any, maxLength int) bool {
	switch item := value.(type) {
	case string:
		return len(item) <= maxLength
	case []any:
		return len(item) <= maxLength
	default:
		return true
	}
}

func validInputLocation(value string) bool {
	return value == InputLocationPath || value == InputLocationQuery || value == InputLocationHeader || value == InputLocationBody
}

func validInputType(location, value string) bool {
	switch location {
	case InputLocationPath, InputLocationHeader:
		return value == InputTypeString
	case InputLocationQuery:
		return value == InputTypeString || value == InputTypeInteger || value == InputTypeNumber || value == InputTypeBoolean
	case InputLocationBody:
		return value == InputTypeString || value == InputTypeInteger || value == InputTypeNumber || value == InputTypeBoolean ||
			value == InputTypeObject || value == InputTypeArray || value == InputTypeNull
	default:
		return false
	}
}

func defaultParameterMaxLength(location string) int {
	switch location {
	case InputLocationPath:
		return maxPathParameterLength
	case InputLocationQuery:
		return maxQueryValueLength
	case InputLocationHeader:
		return maxHeaderValueLength
	default:
		return MaxInputJSONStringBytes
	}
}

func maxParameterLength(location string) int {
	if location == InputLocationBody {
		return MaxInputJSONStringBytes
	}
	return defaultParameterMaxLength(location)
}

func allowedSnapshotHeader(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "accept", "accept-language", "user-agent", "x-correlation-id", "idempotency-key", "x-idempotency-key":
		return true
	default:
		return false
	}
}

func forbiddenInputName(name string) bool {
	value := strings.ToLower(strings.TrimSpace(name))
	if strings.HasPrefix(value, "x-forwarded-") {
		return true
	}
	for _, forbidden := range []string{
		"scheme", "host", "port", "base_url", "full_url", "proxy", "dns", "tls", "certificate",
		"credential_id", "credential_code", "credential_secret", "authorization", "proxy-authorization",
		"cookie", "set-cookie", "api_key", "token", "password", "client_secret", "connection",
		"content-length", "transfer-encoding", "upgrade", "sql", "script", "template", "expression",
	} {
		if value == forbidden || strings.Contains(value, forbidden+"_") || strings.Contains(value, "_"+forbidden) {
			return true
		}
	}
	compact := strings.NewReplacer("_", "", "-", "", ".", "").Replace(value)
	for _, sensitive := range []string{
		"credentialid", "credentialcode", "credentialsecret", "authorization", "proxyauthorization",
		"apikey", "token", "password", "clientsecret", "cookie", "setcookie",
	} {
		if strings.Contains(compact, sensitive) {
			return true
		}
	}
	return false
}
