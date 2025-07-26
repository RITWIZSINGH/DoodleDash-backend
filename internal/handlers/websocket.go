package handlers

import (
	"log"
	"net/http"
	"sort"

	"github.com/gorilla/websocket"
	"github.com/RITWIZSINGH/DoodleDash-backend/internal/models"
	"github.com/RITWIZSINGH/DoodleDash-backend/internal/services"
	"github.com/RITWIZSINGH/DoodleDash-backend/pkg/utils"
	wsocket "github.com/RITWIZSINGH/DoodleDash-backend/pkg/websocket"
)

// ServeWS upgrades an HTTP connection to WebSocket
func ServeWS(hub *wsocket.Hub, w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // In production, implement proper origin checking
		},
	}
	
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		http.Error(w, "WebSocket upgrade failed", http.StatusBadRequest)
		return
	}

	// Create guest user
	user := models.NewGuestUser()
	client := wsocket.NewClient(hub, conn, user)
	hub.RegisterClient(client)

	// Start read and write pumps
	go client.WritePump()
	go client.ReadPump()
}

// HandleWebSocketMessage processes incoming WebSocket messages
func HandleWebSocketMessage(hub *wsocket.Hub, roomManager *services.RoomManager, gameEngine *services.GameEngine, client *wsocket.Client, message *wsocket.Message) {
	switch message.Type {
	case models.MessageTypeConnect:
		handleConnect(hub, client, message)
	case models.MessageTypeCreateRoom:
		handleCreateRoom(hub, roomManager, client, message)
	case models.MessageTypeJoinRoom:
		handleJoinRoom(hub, roomManager, client, message)
	case models.MessageTypeLeaveRoom:
		handleLeaveRoom(hub, roomManager, client, message)
	case models.MessageTypeStartGame:
		handleStartGame(hub, roomManager, gameEngine, client, message)
	case models.MessageTypeDrawStart:
		handleDrawStart(hub, roomManager, client, message)
	case models.MessageTypeDrawMove:
		handleDrawMove(hub, roomManager, client, message)
	case models.MessageTypeDrawEnd:
		handleDrawEnd(hub, roomManager, client, message)
	case models.MessageTypeSendGuess:
		handleSendGuess(hub, roomManager, gameEngine, client, message)
	case models.MessageTypeListPublicRooms:
		handleListPublicRooms(hub, roomManager, client, message)
	default:
		// Note: Using a helper function to send error since sendError is not exported
		sendClientError(client, "Unknown message type", "UNKNOWN_MESSAGE_TYPE")
	}
}

// Helper function to send error messages
func sendClientError(client *wsocket.Client, message, code string) {
	// Create an error message and send it
	errorMsg, err := wsocket.NewErrorMessage(message, code)
	if err != nil {
		log.Printf("Error creating error message: %v", err)
		return
	}
	client.SendMessage(errorMsg)
}

// handleConnect processes a connection message
func handleConnect(hub *wsocket.Hub, client *wsocket.Client, message *wsocket.Message) {
	var data models.ConnectData
	if err := message.UnmarshalData(&data); err != nil {
		sendClientError(client, "Invalid connect data", "INVALID_DATA")
		return
	}

	// Validate and sanitize username
	if !utils.ValidateUserName(data.Username) {
		sendClientError(client, "Invalid username", "INVALID_USERNAME")
		return
	}
	data.Username = utils.SanitizeInput(data.Username)

	// Update user information
	user := client.GetUser()
	user.Username = data.Username
	user.Avatar = data.Avatar
	user.GuestUser = false
	user.UpdateActivity()

	client.SendSystemMessage("Successfully connected to the server")
}

// handleCreateRoom processes room creation
func handleCreateRoom(hub *wsocket.Hub, roomManager *services.RoomManager, client *wsocket.Client, message *wsocket.Message) {
	var data models.CreateRoomData
	if err := message.UnmarshalData(&data); err != nil {
		sendClientError(client, "Invalid room creation data", "INVALID_DATA")
		return
	}

	// Validate input
	if !utils.ValidateUserName(data.RoomName) {
		sendClientError(client, "Invalid room name", "INVALID_ROOM_NAME")
		return
	}
	data.RoomName = utils.SanitizeInput(data.RoomName)

	roomType := models.RoomTypePublic
	if data.RoomType == "private" {
		roomType = models.RoomTypePrivate
	}

	// Create room
	room := roomManager.CreateRoom(client.GetUser().ID, roomType, data.RoomName, data)
	if room == nil {
		sendClientError(client, "Failed to create room", "ROOM_CREATION_FAILED")
		return
	}

	// Add client to room
	hub.AddClientToRoom(client, room.ID)
	roomManager.JoinRoom(room.ID, client.GetUser().ID, client.GetUser())

	// Send room created message
	roomInfo := room.GetPublicRoomInfo()
	msg, err := wsocket.NewRoomCreatedMessage(roomInfo)
	if err != nil {
		sendClientError(client, "Failed to create room message", "MESSAGE_CREATION_FAILED")
		return
	}
	client.SendMessage(msg)

	// Broadcast to all about new public room
	if room.Type == models.RoomTypePublic {
		publicRooms := roomManager.GetPublicRooms()
		roomsMsg, err := wsocket.NewPublicRoomsListMessage(publicRooms)
		if err != nil {
			log.Printf("Error creating public rooms list message: %v", err)
			return
		}
		jsonData, err := roomsMsg.ToJSON()
		if err != nil {
			log.Printf("Error converting rooms message to JSON: %v", err)
			return
		}
		hub.BroadcastToAll(jsonData)
	}
}

// handleJoinRoom processes joining a room
func handleJoinRoom(hub *wsocket.Hub, roomManager *services.RoomManager, client *wsocket.Client, message *wsocket.Message) {
	var data models.JoinRoomData
	if err := message.UnmarshalData(&data); err != nil {
		sendClientError(client, "Invalid join room data", "INVALID_DATA")
		return
	}

	// Validate room code
	data.RoomCode = utils.NormalizeRoomCode(data.RoomCode)
	if !utils.ValidateRoomCode(data.RoomCode) {
		sendClientError(client, "Invalid room code", "INVALID_ROOM_CODE")
		return
	}

	// Find room by code
	room := roomManager.GetRoomByCode(data.RoomCode)
	if room == nil {
		sendClientError(client, "Room not found", "ROOM_NOT_FOUND")
		return
	}

	// Join room
	if !roomManager.JoinRoom(room.ID, client.GetUser().ID, client.GetUser()) {
		sendClientError(client, "Failed to join room", "JOIN_FAILED")
		return
	}

	// Update client
	hub.AddClientToRoom(client, room.ID)

	// Send room joined message to client
	roomInfo := room.GetPublicRoomInfo()
	roomMsg, err := wsocket.NewRoomJoinedMessage(roomInfo)
	if err != nil {
		sendClientError(client, "Failed to create room joined message", "MESSAGE_CREATION_FAILED")
		return
	}
	client.SendMessage(roomMsg)

	// Notify other players
	playerMsg, err := wsocket.NewPlayerJoinedMessage(client.GetUser().ToPublicUser())
	if err != nil {
		log.Printf("Error creating player joined message: %v", err)
		return
	}
	jsonData, err := playerMsg.ToJSON()
	if err != nil {
		log.Printf("Error converting player joined message to JSON: %v", err)
		return
	}
	hub.BroadcastToRoom(room.ID, jsonData, client)
}

// handleLeaveRoom processes leaving a room
func handleLeaveRoom(hub *wsocket.Hub, roomManager *services.RoomManager, client *wsocket.Client, message *wsocket.Message) {
	roomID := client.GetRoomID()
	if roomID == "" {
		sendClientError(client, "Not in a room", "NOT_IN_ROOM")
		return
	}

	room := roomManager.GetRoom(roomID)
	if room == nil {
		sendClientError(client, "Room not found", "ROOM_NOT_FOUND")
		return
	}

	// Remove from room
	roomManager.LeaveRoom(roomID, client.GetUser().ID)
	hub.RemoveClientFromRoom(client, roomID)

	// Notify other players
	playerMsg, err := wsocket.NewPlayerLeftMessage(client.GetUser().ToPublicUser())
	if err != nil {
		log.Printf("Error creating player left message: %v", err)
		return
	}
	jsonData, err := playerMsg.ToJSON()
	if err != nil {
		log.Printf("Error converting player left message to JSON: %v", err)
		return
	}
	hub.BroadcastToRoom(roomID, jsonData, nil)

	// Send confirmation to client
	client.SendSystemMessage("You have left the room")
}

// handleStartGame processes game start request
func handleStartGame(hub *wsocket.Hub, roomManager *services.RoomManager, gameEngine *services.GameEngine, client *wsocket.Client, message *wsocket.Message) {
	roomID := client.GetRoomID()
	if roomID == "" {
		sendClientError(client, "Not in a room", "NOT_IN_ROOM")
		return
	}

	room := roomManager.GetRoom(roomID)
	if room == nil {
		sendClientError(client, "Room not found", "ROOM_NOT_FOUND")
		return
	}

	if room.HostID != client.GetUser().ID {
		sendClientError(client, "Only host can start game", "NOT_HOST")
		return
	}

	if !room.CanStart() {
		sendClientError(client, "Not enough players or not all ready", "CANNOT_START")
		return
	}

	HandleGameStart(hub, roomManager, gameEngine, roomID)
}

// handleDrawStart processes start of a drawing action
func handleDrawStart(hub *wsocket.Hub, roomManager *services.RoomManager, client *wsocket.Client, message *wsocket.Message) {
	roomID := client.GetRoomID()
	if roomID == "" {
		sendClientError(client, "Not in a room", "NOT_IN_ROOM")
		return
	}

	room := roomManager.GetRoom(roomID)
	if room == nil {
		sendClientError(client, "Room not found", "ROOM_NOT_FOUND")
		return
	}

	if room.CurrentDrawer != client.GetUser().ID {
		sendClientError(client, "Not your turn to draw", "NOT_DRAWER")
		return
	}

	var data models.DrawStartData
	if err := message.UnmarshalData(&data); err != nil {
		sendClientError(client, "Invalid draw data", "INVALID_DATA")
		return
	}

	drawCmd := models.DrawCommand{
		Type:  "start",
		X:     data.X,
		Y:     data.Y,
		Color: data.Color,
		Size:  data.Size,
	}
	room.AddDrawCommand(drawCmd)

	drawMsg, err := wsocket.NewDrawDataMessage(wsocket.DrawDataMessage{
		Type:   "start",
		X:      data.X,
		Y:      data.Y,
		Color:  data.Color,
		Size:   data.Size,
		UserID: client.GetUser().ID,
	})
	if err != nil {
		log.Printf("Error creating draw data message: %v", err)
		return
	}
	jsonData, err := drawMsg.ToJSON()
	if err != nil {
		log.Printf("Error converting draw message to JSON: %v", err)
		return
	}
	hub.BroadcastToRoom(roomID, jsonData, client)
}

// handleDrawMove processes ongoing drawing action
func handleDrawMove(hub *wsocket.Hub, roomManager *services.RoomManager, client *wsocket.Client, message *wsocket.Message) {
	roomID := client.GetRoomID()
	if roomID == "" {
		sendClientError(client, "Not in a room", "NOT_IN_ROOM")
		return
	}

	room := roomManager.GetRoom(roomID)
	if room == nil {
		sendClientError(client, "Room not found", "ROOM_NOT_FOUND")
		return
	}

	if room.CurrentDrawer != client.GetUser().ID {
		sendClientError(client, "Not your turn to draw", "NOT_DRAWER")
		return
	}

	var data models.DrawMoveData
	if err := message.UnmarshalData(&data); err != nil {
		sendClientError(client, "Invalid draw data", "INVALID_DATA")
		return
	}

	drawCmd := models.DrawCommand{
		Type: "move",
		X:    data.X,
		Y:    data.Y,
	}
	room.AddDrawCommand(drawCmd)

	drawMsg, err := wsocket.NewDrawDataMessage(wsocket.DrawDataMessage{
		Type:   "move",
		X:      data.X,
		Y:      data.Y,
		UserID: client.GetUser().ID,
	})
	if err != nil {
		log.Printf("Error creating draw data message: %v", err)
		return
	}
	jsonData, err := drawMsg.ToJSON()
	if err != nil {
		log.Printf("Error converting draw message to JSON: %v", err)
		return
	}
	hub.BroadcastToRoom(roomID, jsonData, client)
}

// handleDrawEnd processes end of drawing action
func handleDrawEnd(hub *wsocket.Hub, roomManager *services.RoomManager, client *wsocket.Client, message *wsocket.Message) {
	roomID := client.GetRoomID()
	if roomID == "" {
		sendClientError(client, "Not in a room", "NOT_IN_ROOM")
		return
	}

	room := roomManager.GetRoom(roomID)
	if room == nil {
		sendClientError(client, "Room not found", "ROOM_NOT_FOUND")
		return
	}

	if room.CurrentDrawer != client.GetUser().ID {
		sendClientError(client, "Not your turn to draw", "NOT_DRAWER")
		return
	}

	var data models.DrawEndData
	if err := message.UnmarshalData(&data); err != nil {
		sendClientError(client, "Invalid draw data", "INVALID_DATA")
		return
	}

	drawCmd := models.DrawCommand{
		Type: "end",
		X:    data.X,
		Y:    data.Y,
	}
	room.AddDrawCommand(drawCmd)

	drawMsg, err := wsocket.NewDrawDataMessage(wsocket.DrawDataMessage{
		Type:   "end",
		X:      data.X,
		Y:      data.Y,
		UserID: client.GetUser().ID,
	})
	if err != nil {
		log.Printf("Error creating draw data message: %v", err)
		return
	}
	jsonData, err := drawMsg.ToJSON()
	if err != nil {
		log.Printf("Error converting draw message to JSON: %v", err)
		return
	}
	hub.BroadcastToRoom(roomID, jsonData, client)
}

// handleSendGuess processes a player's guess
func handleSendGuess(hub *wsocket.Hub, roomManager *services.RoomManager, gameEngine *services.GameEngine, client *wsocket.Client, message *wsocket.Message) {
	roomID := client.GetRoomID()
	if roomID == "" {
		sendClientError(client, "Not in a room", "NOT_IN_ROOM")
		return
	}

	room := roomManager.GetRoom(roomID)
	if room == nil {
		sendClientError(client, "Room not found", "ROOM_NOT_FOUND")
		return
	}

	if room.State != models.GameStatePlaying || room.Phase != models.GamePhaseDrawing {
		sendClientError(client, "Game not in progress", "INVALID_STATE")
		return
	}

	var data models.GuessData
	if err := message.UnmarshalData(&data); err != nil {
		sendClientError(client, "Invalid guess data", "INVALID_DATA")
		return
	}

	data.Guess = utils.SanitizeInput(data.Guess)

	// Broadcast guess as chat message
	chatMsg, err := wsocket.NewChatMessage(client.GetUser().Username, data.Guess, false)
	if err != nil {
		log.Printf("Error creating chat message: %v", err)
		return
	}
	chatJsonData, err := chatMsg.ToJSON()
	if err != nil {
		log.Printf("Error converting chat message to JSON: %v", err)
		return
	}
	hub.BroadcastToRoom(roomID, chatJsonData, nil)

	// Validate guess
	result := gameEngine.ValidateGuess(room, client.GetUser().ID, data.Guess)
	if !result.Correct {
		return
	}

	// Send guess result to player
	resultMsg, err := wsocket.NewGuessResultMessage(result)
	if err != nil {
		log.Printf("Error creating guess result message: %v", err)
		return
	}
	client.SendMessage(resultMsg)

	// Broadcast points awarded
	pointsMsg, err := wsocket.NewPointsMessage(
		client.GetUser().ID,
		client.GetUser().Username,
		result.Points,
		client.GetUser().Score,
		"Correct guess",
	)
	if err != nil {
		log.Printf("Error creating points message: %v", err)
		return
	}
	pointsJsonData, err := pointsMsg.ToJSON()
	if err != nil {
		log.Printf("Error converting points message to JSON: %v", err)
		return
	}
	hub.BroadcastToRoom(roomID, pointsJsonData, nil)

	// Update leaderboard
	leaderboardMsg, err := wsocket.NewLeaderboardMessage(getLeaderboard(room), room.CurrentRound, room.MaxRounds)
	if err != nil {
		log.Printf("Error creating leaderboard message: %v", err)
		return
	}
	leaderboardJsonData, err := leaderboardMsg.ToJSON()
	if err != nil {
		log.Printf("Error converting leaderboard message to JSON: %v", err)
		return
	}
	hub.BroadcastToRoom(roomID, leaderboardJsonData, nil)

	// Check if round should end
	if result.RoundEnding {
		HandleRoundEnd(hub, roomManager, gameEngine, roomID)
	}
}

// handleListPublicRooms sends list of public rooms
func handleListPublicRooms(hub *wsocket.Hub, roomManager *services.RoomManager, client *wsocket.Client, message *wsocket.Message) {
	rooms := roomManager.GetPublicRooms()
	msg, err := wsocket.NewPublicRoomsListMessage(rooms)
	if err != nil {
		sendClientError(client, "Failed to list rooms", "LIST_ROOMS_FAILED")
		return
	}
	client.SendMessage(msg)
}

// getLeaderboard generates leaderboard from room players
func getLeaderboard(room *models.Room) []*models.PublicUser {
	players := make([]*models.PublicUser, 0, len(room.Players))
	for _, player := range room.Players {
		players = append(players, player.ToPublicUser())
	}
	// Sort by score descending
	sort.Slice(players, func(i, j int) bool {
		return players[i].Score > players[j].Score
	})
	return players
}