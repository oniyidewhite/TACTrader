package expert

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_RoundToDecimalPoint(t *testing.T) {
	tests := []struct {
		name      string
		expected  float64
		input     float64
		precision uint8
	}{
		{
			name:      "case",
			expected:  0.00000100,
			input:     0.00000100,
			precision: 6,
		},
		{
			name:      "case",
			expected:  922327.00000100,
			input:     922327.00000100,
			precision: 6,
		},
		{
			name:      "case",
			expected:  922327,
			input:     922327,
			precision: 6,
		},
		{
			name:      "case",
			expected:  922327.111111,
			input:     922327.111111111111111111111111,
			precision: 6,
		},
		{
			name:      "case",
			expected:  1,
			input:     1.0000000000000009,
			precision: 6,
		},
		{
			name:      "case",
			expected:  0.000000,
			input:     0.0000000000000009,
			precision: 6,
		},
		{
			name:      "case",
			expected:  0.100001,
			input:     0.100001,
			precision: 6,
		},
		{
			name:      "case",
			expected:  6,
			input:     6.100001,
			precision: 0,
		},
		{
			name:      "case",
			expected:  6.1,
			input:     6.100001,
			precision: 1,
		},
		{
			name:      "case",
			expected:  6.1,
			input:     6.100001,
			precision: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := RoundToDecimalPoint(tt.input, tt.precision)
			assert.Equal(t, tt.expected, res)
		})
	}
}
