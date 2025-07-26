package models

import (
	"strings"
	"sync"
	"time"
)

// RoomType represents the type of room
type RoomType string

const (
	RoomTypePublic  RoomType = "public"
	RoomTypePrivate RoomType = "private"
)

// GameState represents the current state of the game
type GameState string

const (
	GameStateLobby    GameState = "lobby"
	GameStateStarting GameState = "starting"
	GameStatePlaying  GameState = "playing"
	GameStateEnded    GameState = "ended"
)

// GamePhase represents the current phase within a round
type GamePhase string

const (
	GamePhaseWaiting  GamePhase = "waiting"
	GamePhaseDrawing  GamePhase = "drawing"
	GamePhaseGuessing GamePhase = "guessing"
	GamePhaseResults  GamePhase = "results"
)

// Difficulty represents the word difficulty level
type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
)

// Room represents a game room
type Room struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Type        RoomType  `json:"type"`
	HostID      string    `json:"host_id"`
	CreatedAt   time.Time `json:"created_at"`
	LastActivity time.Time `json:"last_activity"`
	
	// Room settings
	MaxPlayers   int        `json:"max_players"`
	RoundTime    int        `json:"round_time"`    // seconds
	MaxRounds    int        `json:"max_rounds"`
	Difficulty   Difficulty `json:"difficulty"`
	CustomWords  []string   `json:"custom_words,omitempty"`
	
	// Current game state
	State        GameState `json:"state"`
	Phase        GamePhase `json:"phase"`
	CurrentRound int       `json:"current_round"`
	RoundStartTime time.Time `json:"round_start_time,omitempty"`
	
	// Players
	Players      map[string]*User `json:"players"`
	PlayerOrder  []string         `json:"player_order"` // For drawer rotation
	
	// Current round data
	CurrentDrawer   string    `json:"current_drawer,omitempty"`
	CurrentWord     string    `json:"current_word,omitempty"`
	WordHint        string    `json:"word_hint,omitempty"`
	GuessedPlayers  []string  `json:"guessed_players,omitempty"`
	RoundEndTime    time.Time `json:"round_end_time,omitempty"`
	
	// Drawing data
	DrawingData []DrawCommand `json:"drawing_data,omitempty"`
	
	mutex sync.RWMutex
}

// DrawCommand represents a drawing action
type DrawCommand struct {
	Type      string    `json:"type"` // "start", "move", "end", "clear"
	X         float64   `json:"x,omitempty"`
	Y         float64   `json:"y,omitempty"`
	Color     string    `json:"color,omitempty"`
	Size      float64   `json:"size,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// NewRoom creates a new room with the specified parameters
func NewRoom(hostID string, roomType RoomType, roomName string, settings CreateRoomData) *Room {
	now := time.Now()
	
	return &Room{
		ID:           generateRoomID(),
		Code:         generateRoomCode(),
		Name:         roomName,
		Type:         roomType,
		HostID:       hostID,
		CreatedAt:    now,
		LastActivity: now,
		
		MaxPlayers:  settings.MaxPlayers,
		RoundTime:   settings.RoundTime,
		MaxRounds:   settings.MaxRounds,
		Difficulty:  Difficulty(settings.Difficulty),
		CustomWords: settings.CustomWords,
		
		State:        GameStateLobby,
		Phase:        GamePhaseWaiting,
		CurrentRound: 0,
		
		Players:     make(map[string]*User),
		PlayerOrder: make([]string, 0),
		
		GuessedPlayers: make([]string, 0),
		DrawingData:    make([]DrawCommand, 0),
	}
}

// AddPlayer adds a player to the room
func (r *Room) AddPlayer(user *User) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	if len(r.Players) >= r.MaxPlayers {
		return false
	}
	
	if _, exists := r.Players[user.ID]; exists {
		return false
	}
	
	r.Players[user.ID] = user
	r.PlayerOrder = append(r.PlayerOrder, user.ID)
	r.LastActivity = time.Now()
	
	return true
}

// RemovePlayer removes a player from the room
func (r *Room) RemovePlayer(userID string) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	if _, exists := r.Players[userID]; !exists {
		return false
	}
	
	delete(r.Players, userID)
	
	// Remove from player order
	for i, id := range r.PlayerOrder {
		if id == userID {
			r.PlayerOrder = append(r.PlayerOrder[:i], r.PlayerOrder[i+1:]...)
			break
		}
	}
	
	// Remove from guessed players if present
	for i, id := range r.GuessedPlayers {
		if id == userID {
			r.GuessedPlayers = append(r.GuessedPlayers[:i], r.GuessedPlayers[i+1:]...)
			break
		}
	}
	
	// If the host left, assign new host
	if r.HostID == userID && len(r.Players) > 0 {
		r.assignNewHost()
	}
	
	r.LastActivity = time.Now()
	return true
}

// GetPlayer returns a player by ID
func (r *Room) GetPlayer(userID string) (*User, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	player, exists := r.Players[userID]
	return player, exists
}

// GetPlayerCount returns the current number of players
func (r *Room) GetPlayerCount() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return len(r.Players)
}

// IsFull checks if the room is at maximum capacity
func (r *Room) IsFull() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return len(r.Players) >= r.MaxPlayers
}

// CanStart checks if the game can be started
func (r *Room) CanStart() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	if r.State != GameStateLobby {
		return false
	}
	
	if len(r.Players) < 2 {
		return false
	}
	
	// Check if all players are ready
	for _, player := range r.Players {
		if !player.IsReady {
			return false
		}
	}
	
	return true
}

// StartGame initializes the game
func (r *Room) StartGame() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	r.State = GameStatePlaying
	r.Phase = GamePhaseDrawing
	r.CurrentRound = 1
	r.LastActivity = time.Now()
	
	// Reset all players' round data
	for _, player := range r.Players {
		player.ResetRoundData()
	}
	
	// Set first drawer (usually the host)
	if len(r.PlayerOrder) > 0 {
		r.CurrentDrawer = r.PlayerOrder[0]
	}
}

// StartNewRound starts a new round
func (r *Room) StartNewRound(word, hint string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	r.CurrentRound++
	r.Phase = GamePhaseDrawing
	r.CurrentWord = word
	r.WordHint = hint
	r.RoundStartTime = time.Now()
	r.RoundEndTime = r.RoundStartTime.Add(time.Duration(r.RoundTime) * time.Second)
	r.GuessedPlayers = make([]string, 0)
	r.DrawingData = make([]DrawCommand, 0)
	r.LastActivity = time.Now()
	
	// Reset all players' round data
	for _, player := range r.Players {
		player.ResetRoundData()
	}
	
	// Move to next drawer
	r.selectNextDrawer()
}

// EndRound ends the current round
func (r *Room) EndRound() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	r.Phase = GamePhaseResults
	r.LastActivity = time.Now()
}

// EndGame ends the entire game
func (r *Room) EndGame() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	r.State = GameStateLobby
	r.Phase = GamePhaseWaiting
	r.CurrentRound = 0
	r.CurrentDrawer = ""
	r.CurrentWord = ""
	r.WordHint = ""
	r.GuessedPlayers = make([]string, 0)
	r.DrawingData = make([]DrawCommand, 0)
	r.LastActivity = time.Now()
	
	// Reset all players
	for _, player := range r.Players {
		player.ResetRoundData()
		player.SetReady(false)
	}
}

// AddGuess records a player's guess
func (r *Room) AddGuess(userID string) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	// Check if player already guessed
	for _, id := range r.GuessedPlayers {
		if id == userID {
			return false
		}
	}
	
	r.GuessedPlayers = append(r.GuessedPlayers, userID)
	r.LastActivity = time.Now()
	return true
}

// GetTimeLeft returns seconds left in current round
func (r *Room) GetTimeLeft() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	if r.RoundEndTime.IsZero() {
		return 0
	}
	
	timeLeft := time.Until(r.RoundEndTime).Seconds()
	if timeLeft < 0 {
		return 0
	}
	
	return int(timeLeft)
}

// AddDrawCommand adds a drawing command to the room
func (r *Room) AddDrawCommand(cmd DrawCommand) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	cmd.Timestamp = time.Now()
	r.DrawingData = append(r.DrawingData, cmd)
	r.LastActivity = time.Now()
}

// ClearDrawing clears all drawing data
func (r *Room) ClearDrawing() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	r.DrawingData = make([]DrawCommand, 0)
	r.LastActivity = time.Now()
}

// GetPublicRoomInfo returns public information about the room
func (r *Room) GetPublicRoomInfo() *PublicRoomInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	playerList := make([]*PublicUser, 0, len(r.Players))
	for _, player := range r.Players {
		playerList = append(playerList, player.ToPublicUser())
	}
	
	return &PublicRoomInfo{
		ID:           r.ID,
		Code:         r.Code,
		Name:         r.Name,
		Type:         string(r.Type),
		PlayerCount:  len(r.Players),
		MaxPlayers:   r.MaxPlayers,
		State:        string(r.State),
		Phase:        string(r.Phase),
		CurrentRound: r.CurrentRound,
		MaxRounds:    r.MaxRounds,
		RoundTime:    r.RoundTime,
		Difficulty:   string(r.Difficulty),
		Players:      playerList,
		TimeLeft:     r.GetTimeLeft(),
		CanJoin:      r.State == GameStateLobby && !r.IsFull(),
	}
}

// IsActive checks if the room has been active recently
func (r *Room) IsActive(timeout time.Duration) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return time.Since(r.LastActivity) <= timeout
}

// Helper methods

func (r *Room) assignNewHost() {
	if len(r.Players) == 0 {
		r.HostID = ""
		return
	}
	
	// Assign first player in order as new host
	for _, playerID := range r.PlayerOrder {
		if _, exists := r.Players[playerID]; exists {
			r.HostID = playerID
			return
		}
	}
}

func (r *Room) selectNextDrawer() {
	if len(r.PlayerOrder) == 0 {
		return
	}
	
	// Find current drawer index
	currentIndex := -1
	for i, playerID := range r.PlayerOrder {
		if playerID == r.CurrentDrawer {
			currentIndex = i
			break
		}
	}
	
	// Move to next player, wrapping around
	nextIndex := (currentIndex + 1) % len(r.PlayerOrder)
	r.CurrentDrawer = r.PlayerOrder[nextIndex]
}

// PublicRoomInfo represents room information that can be shared publicly
type PublicRoomInfo struct {
	ID           string        `json:"id"`
	Code         string        `json:"code"`
	Name         string        `json:"name"`
	Type         string        `json:"type"`
	PlayerCount  int           `json:"player_count"`
	MaxPlayers   int           `json:"max_players"`
	State        string        `json:"state"`
	Phase        string        `json:"phase"`
	CurrentRound int           `json:"current_round"`
	MaxRounds    int           `json:"max_rounds"`
	RoundTime    int           `json:"round_time"`
	Difficulty   string        `json:"difficulty"`
	Players      []*PublicUser `json:"players"`
	TimeLeft     int           `json:"time_left"`
	CanJoin      bool          `json:"can_join"`
}

// Helper functions for room creation
func generateRoomID() string {
	return "room_" + generateRandomString(12)
}

func generateRoomCode() string {
	// Generate 6-character alphanumeric code
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var code strings.Builder
	for i := 0; i < 6; i++ {
		code.WriteByte(charset[generateRandomInt(len(charset))])
	}
	return code.String()
}