package runtime

import (
	"math"
	"testing"
)

func TestIntegerValidationMatchesJSONSchemaSemantics(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  bool
	}{
		{name: "decoded whole number", value: 2.0, want: true},
		{name: "fractional number", value: 1.5, want: false},
		{name: "native integer", value: int64(2), want: true},
		{name: "not a number", value: math.NaN(), want: false},
		{name: "infinity", value: math.Inf(1), want: false},
		{name: "string", value: "2", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isInteger(test.value); got != test.want {
				t.Fatalf("isInteger(%T) = %v, want %v", test.value, got, test.want)
			}
		})
	}
}
