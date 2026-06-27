package utils

import (
	"math"
	"testing"
)

func TestIntFromAnyRequiresIntegralNumericValues(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  int
		ok    bool
	}{
		{name: "int", value: 3, want: 3, ok: true},
		{name: "string", value: " 4 ", want: 4, ok: true},
		{name: "integral json number", value: float64(5), want: 5, ok: true},
		{name: "fractional json number", value: float64(5.25), ok: false},
		{name: "nan", value: math.NaN(), ok: false},
		{name: "positive infinity", value: math.Inf(1), ok: false},
		{name: "uint64 overflow", value: ^uint64(0), ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := IntFromAny(tt.value)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Fatalf("value = %d, want %d", got, tt.want)
			}
		})
	}
}
