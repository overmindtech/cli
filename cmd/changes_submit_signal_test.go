package cmd

import (
	"testing"
)

func TestValidateValue(t *testing.T) {
	tests := []struct {
		input          float64
		expectedOutput float64
		expectError    bool
	}{
		{input: 5.0, expectedOutput: 5.0, expectError: false},
		{input: 0.0, expectedOutput: 0.0, expectError: false},
		{input: -1.0, expectedOutput: -1.0, expectError: false},
		{input: 11.0, expectedOutput: 0.0, expectError: true},
		{input: -6.0, expectedOutput: 0.0, expectError: true},
	}

	for _, test := range tests {
		output, err := validateValue(test.input)
		if (err != nil) != test.expectError {
			t.Errorf("validateValue(%v) unexpected error status: got %v, want error: %v", test.input, err != nil, test.expectError)
		}
		if output != test.expectedOutput {
			t.Errorf("validateValue(%v) = %v; want %v", test.input, output, test.expectedOutput)
		}
	}
}
