package main

import (
	"testing"
)

func TestCleanClassificationResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no changes needed",
			input:    "deer",
			expected: "deer",
		},
		{
			name:     "trailing period",
			input:    "deer.",
			expected: "deer",
		},
		{
			name:     "double quotes",
			input:    `"deer"`,
			expected: "deer",
		},
		{
			name:     "single quotes",
			input:    "'deer'",
			expected: "deer",
		},
		{
			name:     "curly double quotes",
			input:    "\u201cdeer\u201d",
			expected: "deer",
		},
		{
			name:     "curly single quotes",
			input:    "\u2018deer\u2019",
			expected: "deer",
		},
		{
			name:     "quotes and period",
			input:    `"deer."`,
			expected: "deer",
		},
		{
			name:     "curly quotes and period",
			input:    "\u201cdeer.\u201d",
			expected: "deer",
		},
		{
			name:     "whitespace with quotes",
			input:    `  "deer"  `,
			expected: "deer",
		},
		{
			name:     "whitespace inside quotes",
			input:    `" deer "`,
			expected: "deer",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "just quotes",
			input:    `""`,
			expected: "",
		},
		{
			name:     "mismatched quotes",
			input:    `"deer'`,
			expected: `"deer'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanClassificationResponse(tt.input)
			if result != tt.expected {
				t.Errorf("cleanClassificationResponse(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}