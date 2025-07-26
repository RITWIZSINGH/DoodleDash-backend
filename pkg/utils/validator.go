package utils

import (
	"regexp"
	"strings"
)

// ValidateUserName validates a username
func ValidateUserName(name string) bool {
	if len(name) < 3 || len(name) > 20 {
		return false
	}

	// Check for allowed characters
	allowed := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	if !allowed.MatchString(name) {
		return false
	}

	// Check for inappropriate words (simplified)
	badWords := []string{"admin", "system", "root"}
	nameLower := strings.ToLower(name)
	for _, word := range badWords {
		if strings.Contains(nameLower, word) {
			return false
		}
	}

	return true
}

// SanitizeInput cleans user input
func SanitizeInput(input string) string {
	// Remove potentially dangerous characters
	input = strings.ReplaceAll(input, "<", "")
	input = strings.ReplaceAll(input, ">", "")
	input = strings.ReplaceAll(input, "&", "")
	input = strings.ReplaceAll(input, "\"", "")
	input = strings.ReplaceAll(input, "'", "")
	return strings.TrimSpace(input)
}