package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateType(t *testing.T) {
	tests := []struct {
		name     string
		tp       string
		expected bool
	}{
		{
			name:     "valid counter type",
			tp:       "counter",
			expected: true,
		},
		{
			name:     "valid gauge type",
			tp:       "gauge",
			expected: true,
		},
		{
			name:     "invalid type",
			tp:       "invalid",
			expected: false,
		},
		{
			name:     "empty type",
			tp:       "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateType(tt.tp)
			assert.Equal(t, tt.expected, result)
		})
	}
}
