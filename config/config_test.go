package config

import (
	"testing"

	"github.com/carlmjohnson/be"
)

func TestMaskSensitiveValue(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{
			name:     "mask token",
			value:    "abc123def456",
			expected: "abc1********",
		},
		{
			name:     "mask short token",
			value:    "abc",
			expected: "***",
		},
		{
			name:     "empty token",
			value:    "",
			expected: "(not set)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskSensitiveValue(tt.value)
			be.Equal(t, tt.expected, result)
		})
	}
}

func TestSetConfig(t *testing.T) {
	// Test that SetConfig properly sets up the table rows
	m := New()
	testConfig := Config{
		Debug:                   true,
		Token:                   "test-token-123456",
		DebitsAsNegative:        false,
		HidePendingTransactions: true,
	}

	m.SetConfig(testConfig)

	// Basic test to ensure the config was set without panicking
	// More detailed tests would require accessing the internal table state
	if m.configTable.Rows() == nil {
		t.Error("Expected config table to have rows after SetConfig")
	}
}
