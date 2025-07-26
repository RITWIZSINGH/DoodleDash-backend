package handlers

import (
	"log"
	"net/http"
	"sort"

	gorillaws "github.com/gorilla/websocket"
	"github.com/RITWIZSINGH/DoodleDash-backend/internal/models"
	"github.com/RITWIZSINGH/DoodleDash-backend/internal/services"
	"github.com/RITWIZSINGH/DoodleDash-backend/pkg/utils"
	"github.com/RITWIZSINGH/DoodleDash-backend/pkg/websocket"
)
// ServeWS upgrades an HTTP connection to WebSocket
func ServeWS(hub *websocket.Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := gorillaws.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // In production, implement proper origin checking
		},
	}.Upgrade(w, r, nil)
	}.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		http.Error(w, "WebSocket upgrade failed", http.StatusBadRequest)
		return
	}

	// Create guest user
	user := models.NewGuestUser()
	client := websocket.NewClient(hub, conn, user)
	hub.RegisterClient(client)

	// Start read and write pumps
	go client.WritePump()
	go client.ReadPump()
}

// HandleWebSocketMessage processes incoming WebSocket messages
func HandleWebSocketMessage(hub *websocket.Hub, roomManager *services.RoomManager, gameEngine *services.GameEngine, client *websocket.Client, message *websocket.Message) {
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
		client.SendError("Unknown message type", "UNKNOWN_MESSAGE_TYPE")
	}
}

// handleConnect processes a connection message
func handleConnect(hub *websocket.Hub, client *websocket.Client, message *websocket.Message) {
	var data models.ConnectData
	if err := message.UnmarshalData(&data); err != nil {
		client.SendError("Invalid connect data", "INVALID_DATA")
		return
	}

	// Validate and sanitize username
	if !utils.ValidateUserName(data.Username) {
		client.SendError("Invalid username", "INVALID_USERNAME")
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
func handleCreateRoom(hub *websocket.Hub, roomManager *services.RoomManager, client *websocket.Client, message *websocket.Message) {
	var data models.CreateRoomData
	if err := message.UnmarshalData(&data); err != nil {
		client.SendError("Invalid room creation data", "INVALID_DATA")
		return
	}

	// Validate input
	if !utils.ValidateUserName(data.RoomName) {
		client.SendError("Invalid room name", "INVALID_ROOM_NAME")
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
		client.SendError("Failed to create room", "ROOM_CREATION_FAILED")
		return
	}

	// Add client to room
	hub.AddClientToRoom(client, room.ID)
	roomManager.JoinRoom(room.ID, client.GetUser().ID, client.GetUser())

	// Send room created message
	roomInfo := room.GetPublicRoomInfo()
	msg, err := websocket.NewRoomCreatedMessage(roomInfo)
	if err != nil {
		client.SendError("Failed to create room message", "MESSAGE_CREATION_FAILED")
		return
	}
	client.SendMessage(msg)

	// Broadcast to all about new public room
	if room.Type == models.RoomTypePublic {
		publicRooms := roomManager.GetPublicRooms()
		roomsMsg, err := websocket.NewPublicRoomsListMessage(publicRooms)
		if err != nil {
			log.Printf("Error creating public rooms list message: %v", err)
			return
		}
		hub.BroadcastToAll(roomsMsg.ToJSON())
	}
}

// handleJoinRoom processes joining a room
func handleJoinRoom(hub *websocket.Hub, roomManager *services.RoomManager, client *websocket.Client, message *websocket.Message) {
	var data models.JoinRoomData
	if err := message.UnmarshalData(&data); err != nil {
		client.SendError("Invalid join room data", "INVALID_DATA")
		return
	}

	// Validate room code
	data.RoomCode = utils.NormalizeRoomCode(data.RoomCode)
	if !utils.ValidateRoomCode(data.RoomCode) {
		client.SendError("Invalid room code", "INVALID_ROOM_CODE")
		return
	}

	// Find room by code
	room := roomManager.GetRoomByCode(data.RoomCode)
	if room == nil {
		client.SendError("Room not found", "ROOM_NOT_FOUND")
		return
	}

	// Join room
	if !roomManager.JoinRoom(room.ID, client.GetUser().ID, client.GetUser()) {
		client.SendError("Failed to join room", "JOIN_FAILED")
		return
	}

	// Update client
	hub.AddClientToRoom(client, room.ID)

	// Send room joined message to client
	roomInfo := room.GetPublicRoomInfo()
	roomMsg, err := websocket.NewRoomJoinedMessage(roomInfo)
	if err != nil {
		client.SendError("Failed to create room joined message", "MESSAGE_CREATION_FAILED")
		return
	}
	client.SendMessage(roomMsg)

	// Notify other players
	playerMsg, err := websocket.NewPlayerJoinedMessage(client.GetUser().ToPublicUser())
	if err != nil {
		log.Printf("Error creating player joined message: %v", err)
		return
	}
	hub.BroadcastToRoom(room.ID, playerMsg.ToJSON(), client)
}

// handleLeaveRoom processes leaving a room
func handleLeaveRoom(hub *websocket.Hub, roomManager *services.RoomManager, client *websocket.Client, message *websocket.Message) {
	roomID := client.GetRoomID()
	if roomID == "" {
		client.SendError("Not in a room", "NOT_IN_ROOM")
		return
	}

	room := roomManager.GetRoom(roomID)
	if room == nil {
		client.SendError("Room not found", "ROOM_NOT_FOUND")
		return
	}

	// Remove from room
	roomManager.LeaveRoom(roomID, client.GetUser().ID)
	hub.RemoveClientFromRoom(client, roomID)

	// Notify other players
	playerMsg, err := websocket.NewPlayerLeftMessage(client.GetUser().ToPublicUser())
	if err != nil {
		log.Printf("Error creating player left message: %v", err)
		return
	}
	hub.BroadcastToRoom(roomID, playerMsg.ToJSON(), nil)

	// Send confirmation to client
	client.SendSystemMessage("You have left the room")
}

// handleStartGame processes game start request
func handleStartGame(hub *websocket.Hub, roomManager *services.RoomManager, gameEngine *services.GameEngine, client *websocket.Client, message *websocket.Message) {
	roomID := client.GetRoomID()
	if roomID == "" {
		client.SendError("Not in a room", "NOT_IN_ROOM")
		return
	}

	room := roomManager.GetRoom(roomID)
	if room == nil {
		client.SendError("Room not found", "ROOM_NOT_FOUND")
		return
	}

	if room.HostID != client.GetUser().ID {
		client.SendError("Only host can start game", "NOT_HOST")
		return
	}

	if !room.CanStart() {
		client.SendError("Not enough players or not all ready", "CANNOT_START")
		return
	}

	HandleGameStart(hub, roomManager, gameEngine, roomID)
}

// handleDrawStart processes start of a drawing action
func handleDrawStart(hub *websocket.Hub, roomManager *services.RoomManager, client *websocket.Client, message *websocket.Message) {
	roomID := client.GetRoomID()
	if roomID == "" {
		client.SendError("Not in a room", "NOT_IN_ROOM")
		return
	}

	room := roomManager.GetRoom(roomID)
	if room == nil {
		client.SendError("Room not found", "ROOM_NOT_FOUND")
		return
	}

	if room.CurrentDrawer != client.GetUser().ID {
		client.SendError("Not your turn to draw", "NOT_DRAWER")
		return
	}

	var data models.DrawStartData
	if err := message.UnmarshalData(&data); err != nil {
		client.SendError("Invalid draw data", "INVALID_DATA")
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

	drawMsg, err := websocket.NewDrawDataMessage(websocket.DrawDataMessage{
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
	hub.BroadcastToRoom(roomID, drawMsg.ToJSON(), client)
}

// handleDrawMove processes ongoing drawing action
func handleDrawMove(hub *websocket.Hub, roomManager *services.RoomManager, client *websocket.Client, message *websocket.Message) {
	roomID := client.GetRoomID()
	if roomID == "" {
		client.SendError("Not in a room", "NOT_IN_ROOM")
		return
	}

	room := roomManager.GetRoom(roomID)
	if room == nil {
		client.SendError("Room not found", "ROOM_NOT_FOUND")
		return
	}

	if room.CurrentDrawer != client.GetUser().ID {
		client.SendError("Not your turn to draw", "NOT_DRAWER")
		return
	}

	var data models.DrawMoveData
	if err := message.UnmarshalData(&data); err != nil {
		client.SendError("Invalid draw data", "INVALID_DATA")
		return
	}

	drawCmd := models.DrawCommand{
		Type: "move",
		X:    data.X,
		Y:    data.Y,
	}
	room.AddDrawCommand(drawCmd)

	drawMsg, err := websocket.NewDrawDataMessage(websocket.DrawDataMessage{
		Type:   "move",
		X:      data.X,
		Y:      data.Y,
		UserID: client.GetUser().ID,
	})
	if err != nil {
		log.Printf("Error creating draw data message: %v", err)
		return
	}
	hub.BroadcastToRoom(roomID, drawMsg.ToJSON(), client)
}

// handleDrawEnd processes end of drawing action
func handleDrawEnd(hub *websocket.Hub, roomManager *services.RoomManager, client *websocket.Client, message *websocket.Message) {
	roomID := client.GetRoomID()
	if roomID == "" {
		client.SendError("Not in a room", "NOT_IN_ROOM")
		return
	}

	room := roomManager.GetRoom(roomID)
	if room == nil {
		client.SendError("Room not found", "ROOM_NOT_FOUND")
		return
	}

	if room.CurrentDrawer != client.GetUser().ID {
		client.SendError("Not your turn to draw", "NOT_DRAWER")
		return
	}

	var data models.DrawEndData
	if err := message.UnmarshalData(&data); err != nil {
		client.SendError("Invalid draw data", "INVALID_DATA")
		return
	}

	drawCmd := models.DrawCommand{
		Type: "end",
		X:    data.X,
		Y:    data.Y,
	}
	room.AddDrawCommand(drawCmd)

	drawMsg, err := websocket.NewDrawDataMessage(websocket.DrawDataMessage{
		Type:   "end",
		X:      data.X,
		Y:      data.Y,
		UserID: client.GetUser().ID,
	})
	if err != nil {
		log.Printf("Error creating draw data message: %v", err)
		return
	}
	hub.BroadcastToRoom(roomID, drawMsg.ToJSON(), client)
}

// handleSendGuess processes a player's guess
func handleSendGuess(hub *websocket.Hub, roomManager *services.RoomManager, gameEngine *services.GameEngine, client *websocket.Client, message *websocket.Message) {
	roomID := client.GetRoomID()
	if roomID == "" {
		client.SendError("Not in a room", "NOT_IN_ROOM")
		return
	}

	room := roomManager.GetRoom(roomID)
	if room == nil {
		client.SendError("Room not found", "ROOM_NOT_FOUND")
		return
	}

	if room.State != models.GameStatePlaying || room.Phase != models.GamePhaseDrawing {
		client.SendError("Game not in progress", "INVALID_STATE")
		return
	}

	var data models.GuessData
	if err := message.UnmarshalData(&data); err != nil {
		client.SendError("Invalid guess data", "INVALID_DATA")
		return
	}

	data.Guess = utils.SanitizeInput(data.Guess)

	// Broadcast guess as chat message
	chatMsg, err := websocket.NewChatMessage(client.GetUser().Username, data.Guess, false)
	if err != nil {
		log.Printf("Error creating chat message: %v", err)
		return
	}
	hub.BroadcastToRoom(roomID, chatMsg.ToJSON(), nil)

	// Validate guess
	result := gameEngine.ValidateGuess(room, client.GetUser().ID, data.Guess)
	if !result.Correct {
		return
	}

	// Send guess result to player
	resultMsg, err := websocket.NewGuessResultMessage(result)
	if err != nil {
		log.Printf("Error creating guess result message: %v", err)
		return
	}
	client.SendMessage(resultMsg)

	// Broadcast points awarded
	pointsMsg, err := websocket.NewPointsMessage(
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
	hub.BroadcastToRoom(roomID, pointsMsg.ToJSON(), nil)

	// Update leaderboard
	leaderboardMsg, err := websocket.NewLeaderboardMessage(getLeaderboard(room), room.CurrentRound, room.MaxRounds)
	if err != nil {
		log.Printf("Error creating leaderboard message: %v", err)
		return
	}
	hub.BroadcastToRoom(roomID, leaderboardMsg.ToJSON(), nil)

	// Check if round should end
	if result.RoundEnding {
		HandleRoundEnd(hub, roomManager, gameEngine, roomID)
	}
}

// handleListPublicRooms sends list of public rooms
func handleListPublicRooms(hub *websocket.Hub, roomManager *services.RoomManager, client *websocket.Client, message *websocket.Message) {
	rooms := roomManager.GetPublicRooms()
	msg, err := websocket.NewPublicRoomsListMessage(rooms)
	if err != nil {
		client.SendError("Failed to list rooms", "LIST_ROOMS_FAILED")
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