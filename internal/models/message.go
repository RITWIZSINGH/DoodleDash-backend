package models

import (
	"encoding/json"
	"time"
)

// MessageType represents different types of WebSocket messages
type MessageType string

const (
	// Connection messages
	MessageTypeConnect     MessageType = "connect"
	MessageTypeDisconnect  MessageType = "disconnect"
	
	// Room messages
	MessageTypeCreateRoom       MessageType = "create_room"
	MessageTypeJoinRoom        MessageType = "join_room"
	MessageTypeLeaveRoom       MessageType = "leave_room"
	MessageTypeRoomCreated     MessageType = "room_created"
	MessageTypeRoomJoined      MessageType = "room_joined"
	MessageTypeRoomLeft        MessageType = "room_left"
	MessageTypePlayerJoined    MessageType = "player_joined"
	MessageTypePlayerLeft      MessageType = "player_left"
	MessageTypeListPublicRooms MessageType = "list_public_rooms"
	MessageTypePublicRoomsList MessageType = "public_rooms_list"
	
	// Game messages
	MessageTypeStartGame    MessageType = "start_game"
	MessageTypeGameStarted  MessageType = "game_started"
	MessageTypeNewRound     MessageType = "new_round"
	MessageTypeRoundEnded   MessageType = "round_ended"
	MessageTypeGameEnded    MessageType = "game_ended"
	
	// Drawing messages
	MessageTypeDrawStart MessageType = "draw_start"
	MessageTypeDrawMove  MessageType = "draw_move"
	MessageTypeDrawEnd   MessageType = "draw_end"
	MessageTypeDrawData  MessageType = "draw_data"
	MessageTypeClearCanvas MessageType = "clear_canvas"
	
	// Chat and guessing messages
	MessageTypeSendGuess   MessageType = "send_guess"
	MessageTypeGuessResult MessageType = "guess_result"
	MessageTypeChatMessage MessageType = "chat_message"
	MessageTypeCorrectGuess MessageType = "correct_guess"
	
	// System messages
	MessageTypeError        MessageType = "error"
	MessageTypePointsAwarded MessageType = "points_awarded"
	MessageTypeTimer        MessageType = "timer"
	MessageTypeLeaderboard  MessageType = "leaderboard"
)

// Message represents a WebSocket message
type Message struct {
	Type      MessageType     `json:"type"`
	Data      json.RawMessage `json:"data,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	UserID    string          `json:"user_id,omitempty"`
	RoomID    string          `json:"room_id,omitempty"`
}

// NewMessage creates a new message with the current timestamp
func NewMessage(msgType MessageType, data interface{}) (*Message, error) {
	var rawData json.RawMessage
	var err error
	
	if data != nil {
		rawData, err = json.Marshal(data)
		if err != nil {
			return nil, err
		}
	}
	
	return &Message{
		Type:      msgType,
		Data:      rawData,
		Timestamp: time.Now(),
	}, nil
}

// ToJSON converts the message to JSON bytes
func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// ParseMessage parses JSON bytes into a Message
func ParseMessage(data []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// UnmarshalData unmarshals the message data into the provided interface
func (m *Message) UnmarshalData(v interface{}) error {
	if m.Data == nil {
		return nil
	}
	return json.Unmarshal(m.Data, v)
}

// Connect message data
type ConnectData struct {
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
}

// Room creation data
type CreateRoomData struct {
	RoomName    string `json:"room_name"`
	RoomType    string `json:"room_type"` // "public" or "private"
	MaxPlayers  int    `json:"max_players"`
	RoundTime   int    `json:"round_time"`
	MaxRounds   int    `json:"max_rounds"`
	Difficulty  string `json:"difficulty"` // "easy", "medium", "hard"
	CustomWords []string `json:"custom_words,omitempty"`
}

// Room join data
type JoinRoomData struct {
	RoomCode string `json:"room_code"`
}

// Drawing data structures
type DrawStartData struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Color string  `json:"color"`
	Size  float64 `json:"size"`
}

type DrawMoveData struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type DrawEndData struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Guess data
type GuessData struct {
	Guess string `json:"guess"`
}

// Chat message data
type ChatMessageData struct {
	Message  string `json:"message"`
	Username string `json:"username"`
	IsSystem bool   `json:"is_system"`
}

// Points awarded data
type PointsAwardedData struct {
	UserID     string `json:"user_id"`
	Username   string `json:"username"`
	Points     int    `json:"points"`
	TotalScore int    `json:"total_score"`
	Reason     string `json:"reason"`
}

// Timer data
type TimerData struct {
	TimeLeft int    `json:"time_left"`
	Phase    string `json:"phase"` // "drawing", "guessing", "results"
}

// Error data
type ErrorData struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}