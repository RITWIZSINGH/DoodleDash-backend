package websocket

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/RITWIZSINGH/DoodleDash-backend/internal/models"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// WebSocket upgrader with reasonable defaults
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, implement proper origin checking
		return true
	},
}

// Client represents a WebSocket client connection
type Client struct {
	// The websocket connection
	conn *websocket.Conn

	// Hub that manages this client
	hub *Hub

	// Buffered channel of outbound messages
	send chan []byte

	// User information
	user *models.User

	// Current room ID
	roomID string

	// Connection metadata
	connectedAt time.Time
	
	// Mutex for thread safety
	mutex sync.RWMutex
	
	// Connection state
	isConnected bool
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn, user *models.User) *Client {
	return &Client{
		conn:        conn,
		hub:         hub,
		send:        make(chan []byte, 256),
		user:        user,
		connectedAt: time.Now(),
		isConnected: true,
	}
}

// GetUser returns the client's user (thread-safe)
func (c *Client) GetUser() *models.User {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.user
}

// SetUser sets the client's user (thread-safe)
func (c *Client) SetUser(user *models.User) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.user = user
}

// GetRoomID returns the client's current room ID (thread-safe)
func (c *Client) GetRoomID() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.roomID
}

// SetRoomID sets the client's current room ID (thread-safe)
func (c *Client) SetRoomID(roomID string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.roomID = roomID
}

// IsConnected returns the connection status (thread-safe)
func (c *Client) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.isConnected
}

// ReadPump pumps messages from the websocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.disconnect()
	}()

	// Set connection parameters
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Clean up the message
		messageBytes = bytes.TrimSpace(bytes.Replace(messageBytes, newline, space, -1))

		// Parse the message
		message, err := ParseMessage(messageBytes)
		if err != nil {
			log.Printf("Error parsing message: %v", err)
			c.sendError("Invalid message format", "INVALID_MESSAGE")
			continue
		}

		// Set message metadata
		if c.user != nil {
			message.UserID = c.user.ID
		}
		message.RoomID = c.GetRoomID()

		// Update user activity
		if c.user != nil {
			c.user.UpdateActivity()
		}

		// Send to hub for processing
		select {
		case c.hub.messageHandler <- &MessageWithClient{
			Message: message,
			Client:  c,
		}:
		default:
			log.Println("Hub message handler is full, dropping message")
		}
	}
}

// WritePump pumps messages from the hub to the websocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SendMessage sends a message to this specific client
func (c *Client) SendMessage(message *Message) error {
	if !c.IsConnected() {
		return ErrClientDisconnected
	}

	messageBytes, err := message.ToJSON()
	if err != nil {
		return err
	}

	select {
	case c.send <- messageBytes:
		return nil
	default:
		// Channel is full, client is slow or disconnected
		c.disconnect()
		return ErrClientDisconnected
	}
}

// SendError sends an error message to the client
func (c *Client) sendError(message, code string) {
	errorMsg, err := NewErrorMessage(message, code)
	if err != nil {
		log.Printf("Error creating error message: %v", err)
		return
	}

	c.SendMessage(errorMsg)
}

// SendSystemMessage sends a system chat message to the client
func (c *Client) SendSystemMessage(message string) {
	systemMsg, err := NewChatMessage("System", message, true)
	if err != nil {
		log.Printf("Error creating system message: %v", err)
		return
	}

	c.SendMessage(systemMsg)
}

// disconnect handles client disconnection
func (c *Client) disconnect() {
	c.mutex.Lock()
	wasConnected := c.isConnected
	c.isConnected = false
	c.mutex.Unlock()

	if wasConnected {
		// Update user status
		if c.user != nil {
			c.user.SetConnected(false)
		}

		// Notify hub about disconnection
		c.hub.unregister <- c

		// Close the connection
		c.conn.Close()

		// Close the send channel
		close(c.send)

		log.Printf("Client disconnected: %s", c.getUserDisplayName())
	}
}

// Disconnect safely disconnects the client
func (c *Client) Disconnect() {
	c.disconnect()
}

// getUserDisplayName returns a display name for logging
func (c *Client) getUserDisplayName() string {
	if c.user != nil {
		return c.user.Username + " (" + c.user.ID + ")"
	}
	return "unknown"
}

// GetConnectionInfo returns information about the connection
func (c *Client) GetConnectionInfo() ConnectionInfo {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return ConnectionInfo{
		UserID:      c.getUserID(),
		Username:    c.getUserName(),
		RoomID:      c.roomID,
		ConnectedAt: c.connectedAt,
		IsConnected: c.isConnected,
	}
}

func (c *Client) getUserID() string {
	if c.user != nil {
		return c.user.ID
	}
	return ""
}

func (c *Client) getUserName() string {
	if c.user != nil {
		return c.user.Username
	}
	return ""
}

// ConnectionInfo represents information about a client connection
type ConnectionInfo struct {
	UserID      string    `json:"user_id"`
	Username    string    `json:"username"`
	RoomID      string    `json:"room_id"`
	ConnectedAt time.Time `json:"connected_at"`
	IsConnected bool      `json:"is_connected"`
}

// MessageWithClient pairs a message with its originating client
type MessageWithClient struct {
	Message *Message
	Client  *Client
}

// Custom errors
var (
	ErrClientDisconnected = fmt.Errorf("client is disconnected")
)