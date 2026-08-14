package datapermission

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	myerrors "backend/internal/errors"
)

func TestDimensionValuesNormalizesAndIsolatesValues(t *testing.T) {
	input := []any{int64(9), 3, 9, int32(5)}
	values, err := NewDimensionValues(
		DimensionCodeManagementOrg,
		DataScopeValueTypeBigint,
		input,
	)
	if err != nil {
		t.Fatalf("create dimension values: %v", err)
	}
	input[0] = 100

	want := []int64{3, 5, 9}
	if got := values.BigintValues(); !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected values: got %v want %v", got, want)
	}
	copyValues := values.BigintValues()
	copyValues[0] = 200
	if got := values.BigintValues(); !reflect.DeepEqual(got, want) {
		t.Fatalf("external mutation changed dimension values: %v", got)
	}
}

func TestDimensionValuesSupportsEmptyFactsAndWhitelistJSON(t *testing.T) {
	values, err := NewDimensionValues(
		DimensionCodeLegalEntity,
		DataScopeValueTypeBigint,
		nil,
	)
	if err != nil {
		t.Fatalf("create empty dimension values: %v", err)
	}
	if len(values.Values()) != 0 {
		t.Fatalf("empty facts must stay empty: %v", values.Values())
	}

	payload, err := json.Marshal(values)
	if err != nil {
		t.Fatalf("marshal dimension values: %v", err)
	}
	var decoded map[string]any
	if err = json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode dimension values: %v", err)
	}
	if len(decoded) != 3 || decoded["dimension_code"] != DimensionCodeLegalEntity ||
		decoded["value_type"] != string(DataScopeValueTypeBigint) {
		t.Fatalf("unexpected JSON boundary: %s", payload)
	}
}

func TestDimensionValuesRejectsTypeMismatch(t *testing.T) {
	_, err := NewDimensionValues(
		DimensionCodeEmployee,
		DataScopeValueTypeBigint,
		[]any{"employee-1"},
	)
	assertDimensionValuesErrorCode(
		t,
		err,
		myerrors.ErrorCodeDataPermissionDimensionTypeMismatch,
	)
}

func assertDimensionValuesErrorCode(t *testing.T, err error, code int) {
	t.Helper()
	var adminError *myerrors.ApplicationError
	if !errors.As(err, &adminError) {
		t.Fatalf("expected AdminError, got %T: %v", err, err)
	}
	if adminError.Code != code {
		t.Fatalf("unexpected error code: got %d want %d", adminError.Code, code)
	}
}
