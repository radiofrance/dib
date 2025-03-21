//nolint:testpackage
package strutil

import (
	"reflect"
	"testing"
)

func TestDedupeStrSlice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "No duplicates",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "Duplicates in input",
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "All elements are duplicates",
			input:    []string{"a", "a", "a"},
			expected: []string{"a"},
		},
		{
			name:     "Mixed empty and non-empty strings",
			input:    []string{"", "a", "", "b", "a"},
			expected: []string{"", "a", "b"},
		},
		{
			name:     "Case-sensitive duplicates",
			input:    []string{"a", "A", "b", "B", "a"},
			expected: []string{"a", "A", "b", "B"},
		},
		{
			name:     "Special characters and spaces",
			input:    []string{" ", "!", "@", "!", " "},
			expected: []string{" ", "!", "@"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := DedupeStrSlice(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("DedupeStrSlice(%v) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConvertKVStringsToMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []string
		expected map[string]string
	}{
		{
			name:     "Single key-value pair",
			input:    []string{"key=value"},
			expected: map[string]string{"key": "value"},
		},
		{
			name:     "Multiple key-value pairs",
			input:    []string{"name=John", "age=30", "city=Paris"},
			expected: map[string]string{"name": "John", "age": "30", "city": "Paris"},
		},
		{
			name:     "No equals sign in string",
			input:    []string{"keyWithoutValue"},
			expected: map[string]string{"keyWithoutValue": ""},
		},
		{
			name:     "Empty string in input",
			input:    []string{""},
			expected: map[string]string{"": ""},
		},
		{
			name:     "Value containing equals sign",
			input:    []string{"data=this=is=value"},
			expected: map[string]string{"data": "this=is=value"},
		},
		{
			name:     "Empty input slice",
			input:    []string{},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := ConvertKVStringsToMap(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ConvertKVStringsToMap(%v) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}
