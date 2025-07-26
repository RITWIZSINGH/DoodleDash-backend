package websocket

import (
	"log"
	"sync"
	"time"
)

// Hub maintains the set of active clients and broadcasts messages to the clients
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Clients by user ID for quick lookup
	clientsByUserID map[string]*Client

	// Clients by room ID
	clientsByRoom map[string]map[*Client]bool

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Message handler channel
	messageHandler chan *MessageWithClient

	// Broadcast message to all clients
	broadcast chan []byte

	// Broadcast message to specific room
	roomBroadcast chan *RoomMessage

	// Send message to specific client
	clientMessage chan *ClientMessage

	// Statistics
	stats *HubStats

	// Mutex for thread safety
	mutex sync.RWMutex

	// Shutdown channel
	shutdown chan struct{}

	// Message processor function (injected dependency)
	ProcessMessage func(*MessageWithClient)
}

// RoomMessage represents a message to be sent to a specific room
type RoomMessage struct {
	RoomID  string
	Message []byte
	Exclude *Client // Optional client to exclude from broadcast
}

// ClientMessage represents a message to be sent to a specific client
type ClientMessage struct {
	UserID  string
	Message []byte
}

// HubStats contains statistics about the hub
type HubStats struct {
	ConnectedClients int            `json:"connected_clients"`
	TotalRooms      int            `json:"total_rooms"`
	MessagesHandled int64          `json:"messages_handled"`
	ClientsByRoom   map[string]int `json:"clients_by_room"`
	Uptime          time.Duration  `json:"uptime"`
	StartTime       time.Time      `json:"start_time"`
	
	mutex sync.RWMutex
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:         make(map[*Client]bool),
		clientsByUserID: make(map[string]*Client),
		clientsByRoom:   make(map[string]map[*Client]bool),
		register:        make(chan *Client, 100),
		unregister:      make(chan *Client, 100),
		messageHandler:  make(chan *MessageWithClient, 1000),
		broadcast:       make(chan []byte, 100),
		roomBroadcast:   make(chan *RoomMessage, 500),
		clientMessage:   make(chan *ClientMessage, 500),
		shutdown:        make(chan struct{}),
		stats: &HubStats{
			ClientsByRoom: make(map[string]int),
			StartTime:     time.Now(),
		},
	}
}

// SetMessageProcessor sets the function to process incoming messages
func (h *Hub) SetMessageProcessor(processor func(*MessageWithClient)) {
	h.ProcessMessage = processor
}

// Run starts the hub and handles all incoming requests
func (h *Hub) Run() {
	log.Println("WebSocket Hub starting...")
	
	// Start cleanup routine
	go h.cleanupRoutine()
	
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case messageWithClient := <-h.messageHandler:
			h.handleMessage(messageWithClient)

		case message := <-h.broadcast:
			h.broadcastToAll(message)

		case roomMsg := <-h.roomBroadcast:
			h.broadcastToRoom(roomMsg)

		case clientMsg := <-h.clientMessage:
			h.sendToClient(clientMsg)

		case <-h.shutdown:
			log.Println("WebSocket Hub shutting down...")
			h.shutdownAllClients()
			return
		}
	}
}

// RegisterClient registers a new client with the hub
func (h *Hub) RegisterClient(client *Client) {
	select {
	case h.register <- client:
	default:
		log.Println("Register channel is full, dropping client registration")
		client.Disconnect()
	}
}

// UnregisterClient unregisters a client from the hub
func (h *Hub) UnregisterClient(client *Client) {
	select {
	case h.unregister <- client:
	default:
		log.Println("Unregister channel is full")
	}
}

// BroadcastToAll sends a message to all connected clients
func (h *Hub) BroadcastToAll(message []byte) {
	select {
	case h.broadcast <- message:
	default:
		log.Println("Broadcast channel is full, dropping message")
	}
}

// BroadcastToRoom sends a message to all clients in a specific room
func (h *Hub) BroadcastToRoom(roomID string, message []byte, exclude *Client) {
	roomMsg := &RoomMessage{
		RoomID:  roomID,
		Message: message,
		Exclude: exclude,
	}

	select {
	case h.roomBroadcast <- roomMsg:
	default:
		log.Println("Room broadcast channel is full, dropping message")
	}
}

// SendToClient sends a message to a specific client by user ID
func (h *Hub) SendToClient(userID string, message []byte) {
	clientMsg := &ClientMessage{
		UserID:  userID,
		Message: message,
	}

	select {
	case h.clientMessage <- clientMsg:
	default:
		log.Println("Client message channel is full, dropping message")
	}
}

// GetRoomClients returns all clients in a specific room
func (h *Hub) GetRoomClients(roomID string) []*Client {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	roomClients, exists := h.clientsByRoom[roomID]
	if !exists {
		return []*Client{}
	}

	clients := make([]*Client, 0, len(roomClients))
	for client := range roomClients {
		if client.IsConnected() {
			clients = append(clients, client)
		}
	}

	return clients
}

// GetClientByUserID returns a client by user ID
func (h *Hub) GetClientByUserID(userID string) (*Client, bool) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	client, exists := h.clientsByUserID[userID]
	return client, exists && client.IsConnected()
}

// GetConnectedClients returns the number of connected clients
func (h *Hub) GetConnectedClients() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}

// GetStats returns current hub statistics
func (h *Hub) GetStats() *HubStats {
	h.stats.mutex.Lock()
	defer h.stats.mutex.Unlock()

	h.mutex.RLock()
	defer h.mutex.RUnlock()

	// Update current stats
	h.stats.ConnectedClients = len(h.clients)
	h.stats.TotalRooms = len(h.clientsByRoom)
	h.stats.Uptime = time.Since(h.stats.StartTime)

	// Update clients by room
	h.stats.ClientsByRoom = make(map[string]int)
	for roomID, clients := range h.clientsByRoom {
		h.stats.ClientsByRoom[roomID] = len(clients)
	}

	// Return a copy
	return &HubStats{
		ConnectedClients: h.stats.ConnectedClients,
		TotalRooms:      h.stats.TotalRooms,
		MessagesHandled: h.stats.MessagesHandled,
		ClientsByRoom:   h.stats.ClientsByRoom,
		Uptime:         h.stats.Uptime,
		StartTime:      h.stats.StartTime,
	}
}

// Shutdown gracefully shuts down the hub
func (h *Hub) Shutdown() {
	close(h.shutdown)
}

// Internal methods

func (h *Hub) registerClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Add to main clients map
	h.clients[client] = true

	// Add to user ID lookup
	user := client.GetUser()
	if user != nil {
		// If user already has a connection, disconnect the old one
		if oldClient, exists := h.clientsByUserID[user.ID]; exists {
			log.Printf("User %s reconnecting, disconnecting old connection", user.Username)
			oldClient.Disconnect()
		}
		h.clientsByUserID[user.ID] = client
	}

	log.Printf("Client registered: %s (Total: %d)", client.getUserDisplayName(), len(h.clients))
}

func (h *Hub) unregisterClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Remove from main clients map
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)

		// Remove from user ID lookup
		user := client.GetUser()
		if user != nil {
			if h.clientsByUserID[user.ID] == client {
				delete(h.clientsByUserID, user.ID)
			}
		}

		// Remove from room
		roomID := client.GetRoomID()
		if roomID != "" {
			h.removeClientFromRoom(client, roomID)
		}

		log.Printf("Client unregistered: %s (Total: %d)", client.getUserDisplayName(), len(h.clients))
	}
}

func (h *Hub) addClientToRoom(client *Client, roomID string) {
	// Add client to room
	if h.clientsByRoom[roomID] == nil {
		h.clientsByRoom[roomID] = make(map[*Client]bool)
	}
	h.clientsByRoom[roomID][client] = true

	// Set client's room
	client.SetRoomID(roomID)

	log.Printf("Client %s joined room %s", client.getUserDisplayName(), roomID)
}

func (h *Hub) removeClientFromRoom(client *Client, roomID string) {
	if roomClients, exists := h.clientsByRoom[roomID]; exists {
		delete(roomClients, client)

		// If room is empty, remove it
		if len(roomClients) == 0 {
			delete(h.clientsByRoom, roomID)
			log.Printf("Room %s removed (empty)", roomID)
		}
	}

	// Clear client's room
	client.SetRoomID("")

	log.Printf("Client %s left room %s", client.getUserDisplayName(), roomID)
}

func (h *Hub) handleMessage(messageWithClient *MessageWithClient) {
	// Update stats
	h.stats.mutex.Lock()
	h.stats.MessagesHandled++
	h.stats.mutex.Unlock()

	// Process the message using the injected processor
	if h.ProcessMessage != nil {
		h.ProcessMessage(messageWithClient)
	} else {
		log.Println("No message processor set, dropping message")
	}
}

func (h *Hub) broadcastToAll(message []byte) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for client := range h.clients {
		if client.IsConnected() {
			select {
			case client.send <- message:
			default:
				// Client's send channel is full, disconnect them
				go client.Disconnect()
			}
		}
	}
}

func (h *Hub) broadcastToRoom(roomMsg *RoomMessage) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	roomClients, exists := h.clientsByRoom[roomMsg.RoomID]
	if !exists {
		return
	}

	for client := range roomClients {
		// Skip excluded client
		if roomMsg.Exclude != nil && client == roomMsg.Exclude {
			continue
		}

		if client.IsConnected() {
			select {
			case client.send <- roomMsg.Message:
			default:
				// Client's send channel is full, disconnect them
				go client.Disconnect()
			}
		}
	}
}

func (h *Hub) sendToClient(clientMsg *ClientMessage) {
	h.mutex.RLock()
	client, exists := h.clientsByUserID[clientMsg.UserID]
	h.mutex.RUnlock()

	if exists && client.IsConnected() {
		select {
		case client.send <- clientMsg.Message:
		default:
			// Client's send channel is full, disconnect them
			go client.Disconnect()
		}
	}
}

func (h *Hub) cleanupRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.cleanupDisconnectedClients()
		case <-h.shutdown:
			return
		}
	}
}

func (h *Hub) cleanupDisconnectedClients() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	var disconnectedClients []*Client

	// Find disconnected clients
	for client := range h.clients {
		if !client.IsConnected() {
			disconnectedClients = append(disconnectedClients, client)
		}
	}

	// Remove disconnected clients
	for _, client := range disconnectedClients {
		delete(h.clients, client)

		user := client.GetUser()
		if user != nil && h.clientsByUserID[user.ID] == client {
			delete(h.clientsByUserID, user.ID)
		}

		roomID := client.GetRoomID()
		if roomID != "" {
			h.removeClientFromRoom(client, roomID)
		}
	}

	if len(disconnectedClients) > 0 {
		log.Printf("Cleaned up %d disconnected clients", len(disconnectedClients))
	}
}

func (h *Hub) shutdownAllClients() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	for client := range h.clients {
		client.Disconnect()
	}
}

// AddClientToRoom adds a client to a room (public method)
func (h *Hub) AddClientToRoom(client *Client, roomID string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.addClientToRoom(client, roomID)
}

// RemoveClientFromRoom removes a client from a room (public method)  
func (h *Hub) RemoveClientFromRoom(client *Client, roomID string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.removeClientFromRoom(client, roomID)
}