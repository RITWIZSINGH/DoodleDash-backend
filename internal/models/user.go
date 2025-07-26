package models

import (
	"sync"
	"time"
)

// User represents a player in the game
type User struct {
	ID             string    `json:"id"`
	Username       string    `json:"username"`
	Avatar         string    `json:"avatar"`
	Score          int       `json:"score"`
	IsReady        bool      `json:"is_ready"`
	IsConnected    bool      `json:"is_connected"`
	LastActivity   time.Time `json:"last_activity"`
	JoinedAt       time.Time `json:"joined_at"`
	GuestUser      bool      `json:"guest_user"`
	
	// Game-specific data
	HasGuessedThisRound bool      `json:"has_guessed_this_round"`
	GuessTime          time.Time `json:"guess_time,omitempty"`
	GuessOrder         int       `json:"guess_order,omitempty"`
	
	// Statistics
	RoundsWon        int `json:"rounds_won"`
	TotalGuesses     int `json:"total_guesses"`
	CorrectGuesses   int `json:"correct_guesses"`
	TimesDrawer      int `json:"times_drawer"`
	
	mutex sync.RWMutex
}

// NewUser creates a new user with the given username and avatar
func NewUser(username, avatar string) *User {
	now := time.Now()
	return &User{
		ID:           generateUserID(),
		Username:     username,
		Avatar:       avatar,
		Score:        0,
		IsReady:      false,
		IsConnected:  true,
		LastActivity: now,
		JoinedAt:     now,
		GuestUser:    true, // Default to guest user
		
		HasGuessedThisRound: false,
		RoundsWon:          0,
		TotalGuesses:       0,
		CorrectGuesses:     0,
		TimesDrawer:        0,
	}
}

// NewGuestUser creates a new guest user with a generated name
func NewGuestUser() *User {
	username := generateGuestUsername()
	avatar := generateRandomAvatar()
	return NewUser(username, avatar)
}

// UpdateActivity updates the user's last activity timestamp
func (u *User) UpdateActivity() {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	u.LastActivity = time.Now()
}

// AddScore adds points to the user's total score
func (u *User) AddScore(points int) {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	u.Score += points
}

// SetReady sets the user's ready status
func (u *User) SetReady(ready bool) {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	u.IsReady = ready
}

// SetConnected sets the user's connection status
func (u *User) SetConnected(connected bool) {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	u.IsConnected = connected
	if connected {
		u.LastActivity = time.Now()
	}
}

// RecordGuess records that the user made a guess in this round
func (u *User) RecordGuess(correct bool, guessOrder int) {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	
	u.HasGuessedThisRound = true
	u.GuessTime = time.Now()
	u.TotalGuesses++
	
	if correct {
		u.CorrectGuesses++
		u.GuessOrder = guessOrder
	}
}

// RecordDrawerTurn records that the user was the drawer
func (u *User) RecordDrawerTurn() {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	u.TimesDrawer++
}

// ResetRoundData resets data specific to the current round
func (u *User) ResetRoundData() {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	
	u.HasGuessedThisRound = false
	u.GuessTime = time.Time{}
	u.GuessOrder = 0
	u.IsReady = false
}

// GetAccuracy returns the user's guess accuracy as a percentage
func (u *User) GetAccuracy() float64 {
	u.mutex.RLock()
	defer u.mutex.RUnlock()
	
	if u.TotalGuesses == 0 {
		return 0.0
	}
	return float64(u.CorrectGuesses) / float64(u.TotalGuesses) * 100.0
}

// IsInactive checks if the user has been inactive for too long
func (u *User) IsInactive(timeout time.Duration) bool {
	u.mutex.RLock()
	defer u.mutex.RUnlock()
	return time.Since(u.LastActivity) > timeout
}

// ToPublicUser returns a sanitized version of the user for public consumption
func (u *User) ToPublicUser() *PublicUser {
	u.mutex.RLock()
	defer u.mutex.RUnlock()
	
	return &PublicUser{
		ID:                  u.ID,
		Username:            u.Username,
		Avatar:              u.Avatar,
		Score:               u.Score,
		IsReady:             u.IsReady,
		IsConnected:         u.IsConnected,
		HasGuessedThisRound: u.HasGuessedThisRound,
		RoundsWon:           u.RoundsWon,
		Accuracy:            u.GetAccuracy(),
	}
}

// PublicUser represents user data that can be shared with other players
type PublicUser struct {
	ID                  string  `json:"id"`
	Username            string  `json:"username"`
	Avatar              string  `json:"avatar"`
	Score               int     `json:"score"`
	IsReady             bool    `json:"is_ready"`
	IsConnected         bool    `json:"is_connected"`
	HasGuessedThisRound bool    `json:"has_guessed_this_round"`
	RoundsWon           int     `json:"rounds_won"`
	Accuracy            float64 `json:"accuracy"`
}

// Helper functions for user creation
func generateUserID() string {
	// Generate a unique user ID (timestamp + random string)
	return "user_" + generateRandomString(8) + "_" + generateTimestamp()
}

func generateGuestUsername() string {
	adjectives := []string{"Cool", "Happy", "Clever", "Swift", "Brave", "Lucky", "Smart", "Quick"}
	nouns := []string{"Artist", "Player", "Gamer", "Drawer", "Master", "Pro", "Star", "Ace"}
	
	adj := adjectives[generateRandomInt(len(adjectives))]
	noun := nouns[generateRandomInt(len(nouns))]
	num := generateRandomInt(1000)
	
	return adj + noun + generateIntToString(num)
}

func generateRandomAvatar() string {
	avatars := []string{
		"🎨", "🖌️", "✏️", "🖊️", "🖍️", "✨", "🌟", "⭐", "🎭", "🎪",
		"🎨", "🦄", "🌈", "🎯", "🎲", "🃏", "🎪", "🎭", "🎨", "🖼️",
	}
	return avatars[generateRandomInt(len(avatars))]
}

// Utility functions (these would normally be in a utils package)
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	// Implementation would use crypto/rand for secure random string
	return "abc123de" // Placeholder
}

func generateTimestamp() string {
	return "1234567890" // Placeholder - would use actual timestamp
}

func generateRandomInt(max int) int {
	// Would use crypto/rand for secure random number
	return 0 // Placeholder
}

func generateIntToString(num int) string {
	// Would convert int to string
	return "123" // Placeholder
}