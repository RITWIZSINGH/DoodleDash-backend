package utils

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"
)

const (
	// Characters allowed in room codes (avoiding confusing characters like 0, O, I, 1)
	roomCodeChars = "ABCDEFGHIJKLMNPQRSTUVWXYZ23456789"
	roomCodeLength = 6
)

// GenerateRoomCode generates a unique 6-character room code
func GenerateRoomCode() string {
	code := make([]byte, roomCodeLength)
	
	for i := range code {
		// Use crypto/rand for secure random generation
		randomByte := make([]byte, 1)
		_, err := rand.Read(randomByte)
		if err != nil {
			// Fallback to time-based generation if crypto/rand fails
			return generateTimeBasedRoomCode()
		}
		
		code[i] = roomCodeChars[int(randomByte[0])%len(roomCodeChars)]
	}
	
	return string(code)
}

// generateTimeBasedRoomCode generates a room code based on current time (fallback)
func generateTimeBasedRoomCode() string {
	now := time.Now()
	timestamp := now.UnixNano()
	
	code := make([]byte, roomCodeLength)
	for i := range code {
		code[i] = roomCodeChars[int(timestamp>>(i*5))%len(roomCodeChars)]
	}
	
	return string(code)
}

// ValidateRoomCode validates if a room code has the correct format
func ValidateRoomCode(code string) bool {
	// Check length
	if len(code) != roomCodeLength {
		return false
	}
	
	// Convert to uppercase for consistency
	code = strings.ToUpper(code)
	
	// Check if all characters are valid
	for _, char := range code {
		if !strings.ContainsRune(roomCodeChars, char) {
			return false
		}
	}
	
	return true
}

// NormalizeRoomCode normalizes a room code to uppercase
func NormalizeRoomCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

// GenerateUserID generates a unique user ID
func GenerateUserID() string {
	timestamp := time.Now().UnixNano()
	randomPart := GenerateRandomString(8)
	return fmt.Sprintf("user_%d_%s", timestamp, randomPart)
}

// GenerateRoomID generates a unique room ID
func GenerateRoomID() string {
	timestamp := time.Now().UnixNano()
	randomPart := GenerateRandomString(12)
	return fmt.Sprintf("room_%d_%s", timestamp, randomPart)
}

// GenerateRandomString generates a random string of specified length
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to time-based seed
		seed := time.Now().UnixNano()
		for i := range b {
			b[i] = charset[seed%int64(len(charset))]
			seed /= int64(len(charset))
		}
		return string(b)
	}
	
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}