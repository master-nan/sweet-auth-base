package model

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCustomTimeStoresUTCAndDisplaysAsiaShanghai(t *testing.T) {
	var value CustomTime
	if err := json.Unmarshal([]byte(`"2026-08-28 10:30:00"`), &value); err != nil {
		t.Fatalf("unmarshal local time: %v", err)
	}
	instant := time.Time(value)
	if want := time.Date(2026, 8, 28, 2, 30, 0, 0, time.UTC); !instant.Equal(want) {
		t.Fatalf("stored instant = %s, want %s", instant, want)
	}
	databaseValue, err := value.Value()
	if err != nil {
		t.Fatalf("database value: %v", err)
	}
	stored, ok := databaseValue.(time.Time)
	if !ok || stored.Location() != time.UTC || !stored.Equal(instant) {
		t.Fatalf("database value = %#v", databaseValue)
	}
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal local time: %v", err)
	}
	if string(payload) != `"2026-08-28 10:30:00"` {
		t.Fatalf("payload = %s", payload)
	}
}

func TestZeroCustomTimeDoesNotProduceHistoricalTimezoneOffset(t *testing.T) {
	value := CustomTime(time.Time{})
	payload, err := value.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal zero time: %v", err)
	}
	if string(payload) != "null" {
		t.Fatalf("zero time JSON = %s, want null", payload)
	}
	if value.String() != "" || value.Format(time.DateTime) != "" {
		t.Fatalf("zero time should have an empty display value")
	}
	var decoded CustomTime
	if err := json.Unmarshal([]byte("null"), &decoded); err != nil {
		t.Fatalf("unmarshal null custom time: %v", err)
	}
	if !decoded.IsZero() {
		t.Fatalf("expected null to restore zero time, got %v", decoded)
	}
}

func TestNowUsesUTCInstant(t *testing.T) {
	if location := Now().Location(); location != time.UTC {
		t.Fatalf("Now location = %s, want UTC", location)
	}
}
