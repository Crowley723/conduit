// Copyright 2025 James D Elliot
// Licensed under the Apache License, Version 2.0
// Originally from: https://github.com/authelia/authelia

package utils

// IsStringInSlice checks if a single string is in a slice of strings.
func IsStringInSlice(needle string, haystack []string) (inSlice bool) {
	for _, b := range haystack {
		if b == needle {
			return true
		}
	}

	return false
}
