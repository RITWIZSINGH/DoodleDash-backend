package websocket

import (
	"github.com/RITWIZSINGH/DoodleDash-backend/internal/models"
)

// Message wraps the models.Message with WebSocket-specific functionality
type Message struct {
	*models.Message
}

// NewMessage creates a new WebSocket message
func NewMessage(msgType models.MessageType, data interface{}) (*Message, error) {
	msg, err := models.NewMessage(msgType, data)
	if err != nil {
		return nil, err
	}

	return &Message{Message: msg}, nil
}

// NewErrorMessage creates an error message
func NewErrorMessage(errorMsg, code string) (*Message, error) {
	errorData := models.ErrorData{
		Message: errorMsg,
		Code:    code,
	}

	return NewMessage(models.MessageTypeError, errorData)
}

// NewChatMessage creates a chat message
func NewChatMessage(username, message string, isSystem bool) (*Message, error) {
	chatData := models.ChatMessageData{
		Username: username,
		Message:  message,
		IsSystem: isSystem,
	}

	return NewMessage(models.MessageTypeChatMessage, chatData)
}

// NewPointsMessage creates a points awarded message
func NewPointsMessage(userID, username string, points, totalScore int, reason string) (*Message, error) {
	pointsData := models.PointsAwardedData{
		UserID:     userID,
		Username:   username,
		Points:     points,
		TotalScore: totalScore,
		Reason:     reason,
	}

	return NewMessage(models.MessageTypePointsAwarded, pointsData)
}

// NewTimerMessage creates a timer message
func NewTimerMessage(timeLeft int, phase string) (*Message, error) {
	timerData := models.TimerData{
		TimeLeft: timeLeft,
		Phase:    phase,
	}

	return NewMessage(models.MessageTypeTimer, timerData)
}

// NewRoomCreatedMessage creates a room created message
func NewRoomCreatedMessage(room *models.PublicRoomInfo) (*Message, error) {
	return NewMessage(models.MessageTypeRoomCreated, room)
}

// NewRoomJoinedMessage creates a room joined message
func NewRoomJoinedMessage(room *models.PublicRoomInfo) (*Message, error) {
	return NewMessage(models.MessageTypeRoomJoined, room)
}

// NewPlayerJoinedMessage creates a player joined message
func NewPlayerJoinedMessage(user *models.PublicUser) (*Message, error) {
	return NewMessage(models.MessageTypePlayerJoined, user)
}

// NewPlayerLeftMessage creates a player left message
func NewPlayerLeftMessage(user *models.PublicUser) (*Message, error) {
	return NewMessage(models.MessageTypePlayerLeft, user)
}

// NewGameStartedMessage creates a game started message
func NewGameStartedMessage(room *models.PublicRoomInfo) (*Message, error) {
	return NewMessage(models.MessageTypeGameStarted, room)
}

// NewRoundData represents data for a new round
type NewRoundData struct {
	Round      int    `json:"round"`
	MaxRounds  int    `json:"max_rounds"`
	DrawerID   string `json:"drawer_id"`
	DrawerName string `json:"drawer_name"`
	WordHint   string `json:"word_hint"`
	TimeLimit  int    `json:"time_limit"`
	Word       string `json:"word,omitempty"` // Only sent to drawer
}

// NewNewRoundMessage creates a new round message
func NewNewRoundMessage(data NewRoundData) (*Message, error) {
	return NewMessage(models.MessageTypeNewRound, data)
}

// RoundEndData represents data for round end
type RoundEndData struct {
	Word         string               `json:"word"`
	DrawerID     string               `json:"drawer_id"`
	DrawerName   string               `json:"drawer_name"`
	DrawerPoints int                  `json:"drawer_points"`
	Guessers     []GuesserResult      `json:"guessers"`
	Leaderboard  []*models.PublicUser `json:"leaderboard"`
	NextRound    int                  `json:"next_round,omitempty"`
}

// GuesserResult represents a guesser's performance in the round
type GuesserResult struct {
	UserID     string `json:"user_id"`
	Username   string `json:"username"`
	Guessed    bool   `json:"guessed"`
	Points     int    `json:"points"`
	GuessOrder int    `json:"guess_order,omitempty"`
	GuessTime  int    `json:"guess_time,omitempty"` // seconds from round start
}

// NewRoundEndedMessage creates a round ended message
func NewRoundEndedMessage(data RoundEndData) (*Message, error) {
	return NewMessage(models.MessageTypeRoundEnded, data)
}

// GameEndData represents data for game end
type GameEndData struct {
	Winner      *models.PublicUser   `json:"winner"`
	Leaderboard []*models.PublicUser `json:"leaderboard"`
	GameStats   GameStats            `json:"game_stats"`
}

// GameStats represents statistics for the completed game
type GameStats struct {
	TotalRounds  int                    `json:"total_rounds"`
	TotalPlayers int                    `json:"total_players"`
	AverageScore float64                `json:"average_score"`
	HighestScore int                    `json:"highest_score"`
	PlayerStats  map[string]PlayerStats `json:"player_stats"`
}

// PlayerStats represents individual player statistics
type PlayerStats struct {
	CorrectGuesses int     `json:"correct_guesses"`
	TotalGuesses   int     `json:"total_guesses"`
	Accuracy       float64 `json:"accuracy"`
	TimesDrawer    int     `json:"times_drawer"`
	AveragePoints  float64 `json:"average_points"`
}

// NewGameEndedMessage creates a game ended message
func NewGameEndedMessage(data GameEndData) (*Message, error) {
	return NewMessage(models.MessageTypeGameEnded, data)
}

// GuessResultData represents the result of a guess
type GuessResultData struct {
	Correct     bool   `json:"correct"`
	Word        string `json:"word,omitempty"` // Only if correct
	Points      int    `json:"points"`
	TotalScore  int    `json:"total_score"`
	GuessOrder  int    `json:"guess_order,omitempty"`
	Bonus       int    `json:"bonus,omitempty"`
	TimeBonus   int    `json:"time_bonus,omitempty"`
	OrderBonus  int    `json:"order_bonus,omitempty"`
	RoundEnding bool   `json:"round_ending"`
}

// NewGuessResultMessage creates a guess result message
func NewGuessResultMessage(data GuessResultData) (*Message, error) {
	return NewMessage(models.MessageTypeGuessResult, data)
}

// PublicRoomsListData represents the list of public rooms
type PublicRoomsListData struct {
	Rooms []*models.PublicRoomInfo `json:"rooms"`
	Total int                      `json:"total"`
}

// NewPublicRoomsListMessage creates a public rooms list message
func NewPublicRoomsListMessage(rooms []*models.PublicRoomInfo) (*Message, error) {
	data := PublicRoomsListData{
		Rooms: rooms,
		Total: len(rooms),
	}

	return NewMessage(models.MessageTypePublicRoomsList, data)
}

// DrawDataMessage represents drawing data for broadcasting
type DrawDataMessage struct {
	Type   string  `json:"type"` // "start", "move", "end", "clear"
	X      float64 `json:"x,omitempty"`
	Y      float64 `json:"y,omitempty"`
	Color  string  `json:"color,omitempty"`
	Size   float64 `json:"size,omitempty"`
	UserID string  `json:"user_id"`
}

// NewDrawDataMessage creates a draw data message
func NewDrawDataMessage(drawData DrawDataMessage) (*Message, error) {
	return NewMessage(models.MessageTypeDrawData, drawData)
}

// LeaderboardData represents current game leaderboard
type LeaderboardData struct {
	Players      []*models.PublicUser `json:"players"`
	CurrentRound int                  `json:"current_round"`
	MaxRounds    int                  `json:"max_rounds"`
}

// NewLeaderboardMessage creates a leaderboard message
func NewLeaderboardMessage(players []*models.PublicUser, currentRound, maxRounds int) (*Message, error) {
	data := LeaderboardData{
		Players:      players,
		CurrentRound: currentRound,
		MaxRounds:    maxRounds,
	}

	return NewMessage(models.MessageTypeLeaderboard, data)
}

// ParseMessage parses a JSON message from WebSocket
func ParseMessage(data []byte) (*Message, error) {
	msg, err := models.ParseMessage(data)
	if err != nil {
		return nil, err
	}

	return &Message{Message: msg}, nil
}

// SetRoomAndUser sets the room ID and user ID for the message
func (m *Message) SetRoomAndUser(roomID, userID string) {
	m.RoomID = roomID
	m.UserID = userID
}
