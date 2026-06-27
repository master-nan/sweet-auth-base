package main

import "testing"

func TestShouldRunStartupStep(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "unset defaults to run", value: "", want: true},
		{name: "true runs", value: "true", want: true},
		{name: "one runs", value: "1", want: true},
		{name: "false skips", value: "false", want: false},
		{name: "zero skips", value: "0", want: false},
		{name: "no skips", value: " no ", want: false},
		{name: "off skips", value: "off", want: false},
		{name: "unknown runs", value: "unexpected", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRunStartupStep(tt.value); got != tt.want {
				t.Fatalf("shouldRunStartupStep(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}
