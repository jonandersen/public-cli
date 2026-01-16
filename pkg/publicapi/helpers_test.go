package publicapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatGainLoss(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "positive value",
			input:    "250.00",
			expected: "+$250.00",
		},
		{
			name:     "negative value",
			input:    "-50.00",
			expected: "-$50.00",
		},
		{
			name:     "zero string",
			input:    "0",
			expected: "$0.00",
		},
		{
			name:     "zero decimal",
			input:    "0.00",
			expected: "$0.00",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "$0.00",
		},
		{
			name:     "large positive",
			input:    "12345.67",
			expected: "+$12345.67",
		},
		{
			name:     "large negative",
			input:    "-98765.43",
			expected: "-$98765.43",
		},
		{
			name:     "small positive",
			input:    "0.01",
			expected: "+$0.01",
		},
		{
			name:     "small negative",
			input:    "-0.01",
			expected: "-$0.01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatGainLoss(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatVolume(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected string
	}{
		{
			name:     "zero",
			input:    0,
			expected: "-",
		},
		{
			name:     "small number",
			input:    123,
			expected: "123",
		},
		{
			name:     "thousand",
			input:    1000,
			expected: "1,000",
		},
		{
			name:     "millions",
			input:    1234567,
			expected: "1,234,567",
		},
		{
			name:     "billions",
			input:    1234567890,
			expected: "1,234,567,890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatVolume(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
