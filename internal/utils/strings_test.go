// Copyright 2025 James D Elliot
// Licensed under the Apache License, Version 2.0
// Originally from: https://github.com/authelia/authelia
package utils

import (
	"fmt"
	"testing"
)

func TestIsStringInSlice(t *testing.T) {
	tests := []struct {
		name     string
		needle   string
		haystack []string
		expected bool
	}{
		{
			name:     "string found in slice",
			needle:   "apple",
			haystack: []string{"apple", "banana", "cherry"},
			expected: true,
		},
		{
			name:     "string not found in slice",
			needle:   "orange",
			haystack: []string{"apple", "banana", "cherry"},
			expected: false,
		},
		{
			name:     "empty slice",
			needle:   "apple",
			haystack: []string{},
			expected: false,
		},
		{
			name:     "empty string in non-empty slice",
			needle:   "",
			haystack: []string{"apple", "banana", "cherry"},
			expected: false,
		},
		{
			name:     "empty string in slice containing empty string",
			needle:   "",
			haystack: []string{"apple", "", "cherry"},
			expected: true,
		},
		{
			name:     "case sensitive - different case",
			needle:   "Apple",
			haystack: []string{"apple", "banana", "cherry"},
			expected: false,
		},
		{
			name:     "duplicate strings in slice",
			needle:   "apple",
			haystack: []string{"apple", "banana", "apple", "cherry"},
			expected: true,
		},
		{
			name:     "single element slice - match",
			needle:   "apple",
			haystack: []string{"apple"},
			expected: true,
		},
		{
			name:     "single element slice - no match",
			needle:   "banana",
			haystack: []string{"apple"},
			expected: false,
		},
		{
			name:     "whitespace string",
			needle:   " ",
			haystack: []string{"apple", " ", "cherry"},
			expected: true,
		},
		{
			name:     "string with leading/trailing spaces",
			needle:   " apple ",
			haystack: []string{"apple", "banana", "cherry"},
			expected: false,
		},
		{
			name:     "nil slice",
			needle:   "apple",
			haystack: nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsStringInSlice(tt.needle, tt.haystack)
			if result != tt.expected {
				t.Errorf("IsStringInSlice(%q, %v) = %v, expected %v",
					tt.needle, tt.haystack, result, tt.expected)
			}
		})
	}
}

// Benchmark test to measure performance
func BenchmarkIsStringInSlice(b *testing.B) {
	haystack := []string{"apple", "banana", "cherry", "date", "elderberry", "fig", "grape"}
	needle := "grape" // worst case - at the end

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsStringInSlice(needle, haystack)
	}
}

// Test with a larger slice
func BenchmarkIsStringInSliceLarge(b *testing.B) {
	// CreateUser a larger haystack
	haystack := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		haystack[i] = fmt.Sprintf("item_%d", i)
	}
	needle := "item_999" // worst case - at the end

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsStringInSlice(needle, haystack)
	}
}
