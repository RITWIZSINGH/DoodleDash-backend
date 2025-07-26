package handlers

import (
	"log"

	"time"

	"github.com/RITWIZSINGH/DoodleDash-backend/internal/models"
	"github.com/RITWIZSINGH/DoodleDash-backend/internal/services"
	"github.com/RITWIZSINGH/DoodleDash-backend/pkg/websocket"
)

// HandleGameStart starts a new game
func HandleGameStart(hub *websocket.Hub, roomManager *services.RoomManager, gameEngine *services.GameEngine, roomID string) {
	room := roomManager.GetRoom(roomID)
	if room == nil {
		return
	}

	room.StartGame()
	gameEngine.StartGame(room)

	// Notify players
	roomInfo := room.GetPublicRoomInfo()
	msg, err := websocket.NewGameStartedMessage(roomInfo)
	if err != nil {
		log.Printf("Error creating game started message: %v", err)
		return
	}
	msgData, err := msg.ToJSON()
	if err != nil {
		log.Printf("Error converting message to JSON: %v", err)
		return
	}
	hub.BroadcastToRoom(roomID, msgData, nil)

	// Start first round
	HandleNewRound(hub, roomManager, gameEngine, roomID)
}

// HandleNewRound starts a new round
func HandleNewRound(hub *websocket.Hub, roomManager *services.RoomManager, gameEngine *services.GameEngine, roomID string) {
	room := roomManager.GetRoom(roomID)
	if room == nil {
		return
	}

	word, hint := gameEngine.GetRandomWord(string(room.Difficulty))
	room.StartNewRound(word, hint)

	drawer, exists := room.GetPlayer(room.CurrentDrawer)
	if !exists {
		log.Printf("Drawer not found for room %s", roomID)
		HandleRoundEnd(hub, roomManager, gameEngine, roomID)
		return
	}

	// Send new round message to drawer (with actual word)
	drawerData := websocket.NewRoundData{
		Round:      room.CurrentRound,
		MaxRounds:  room.MaxRounds,
		DrawerID:   room.CurrentDrawer,
		DrawerName: drawer.Username,
		WordHint:   room.WordHint,
		TimeLimit:  room.RoundTime,
		Word:       room.CurrentWord,
	}
	drawerMsg, err := websocket.NewNewRoundMessage(drawerData)
	if err != nil {
		log.Printf("Error creating new round message: %v", err)
		return
	}
	if client, exists := hub.GetClientByUserID(room.CurrentDrawer); exists {
		client.SendMessage(drawerMsg)
	}

	// Send new round message to others (with hint only)
	othersData := websocket.NewRoundData{
		Round:      room.CurrentRound,
		MaxRounds:  room.MaxRounds,
		DrawerID:   room.CurrentDrawer,
		DrawerName: drawer.Username,
		WordHint:   room.WordHint,
		TimeLimit:  room.RoundTime,
	}
	othersMsg, err := websocket.NewNewRoundMessage(othersData)
	if err != nil {
		log.Printf("Error creating new round message: %v", err)
		return
	}
	othersMsgData, err := othersMsg.ToJSON()
	if err != nil {
		log.Printf("Error converting others message to JSON: %v", err)
		return
	}

	drawerClient, drawerExists := hub.GetClientByUserID(room.CurrentDrawer)
	var excludeClient *websocket.Client
	if drawerExists {
		excludeClient = drawerClient
	}
	hub.BroadcastToRoom(roomID, othersMsgData, excludeClient)

	// Start timer
	go runRoundTimer(hub, roomManager, gameEngine, roomID)
}

// HandleRoundEnd ends the current round
func HandleRoundEnd(hub *websocket.Hub, roomManager *services.RoomManager, gameEngine *services.GameEngine, roomID string) {
	room := roomManager.GetRoom(roomID)
	if room == nil {
		return
	}

	// Award drawer points
	drawerPoints := gameEngine.CalculateDrawerPoints(len(room.GuessedPlayers), len(room.Players), room.RoundTime)
	if drawer, exists := room.GetPlayer(room.CurrentDrawer); exists {
		drawer.AddScore(drawerPoints)
		drawer.RecordDrawerTurn()
	}

	// Prepare round end data
	guessers := make([]websocket.GuesserResult, 0, len(room.Players))
	for _, userID := range room.GuessedPlayers {
		if player, exists := room.GetPlayer(userID); exists {
			var points int
			if player.GuessOrder > 0 {
				points = gameEngine.CalculateGuesserPoints(player.GuessOrder, len(room.GuessedPlayers), int(player.GuessTime.Sub(room.RoundStartTime).Seconds()), room.RoundTime, len(room.Players), string(room.Difficulty))
			} else {
				points = 0
			}
			guessers = append(guessers, websocket.GuesserResult{
				UserID:     player.ID,
				Username:   player.Username,
				Guessed:    true,
				Points:     points,
				GuessOrder: player.GuessOrder,
				GuessTime:  int(player.GuessTime.Sub(room.RoundStartTime).Seconds()),
			})
		}
	}

	// Include non-guessers
	for userID, player := range room.Players {
		if !contains(room.GuessedPlayers, userID) && userID != room.CurrentDrawer {
			guessers = append(guessers, websocket.GuesserResult{
				UserID:   player.ID,
				Username: player.Username,
				Guessed:  false,
				Points:   0,
			})
		}
	}

	drawer, _ := room.GetPlayer(room.CurrentDrawer)
	roundEndData := websocket.RoundEndData{
		Word:         room.CurrentWord,
		DrawerID:     room.CurrentDrawer,
		DrawerName:   drawer.Username,
		DrawerPoints: drawerPoints,
		Guessers:     guessers,
		Leaderboard:  getLeaderboard(room),
		NextRound:    room.CurrentRound + 1,
	}

	// Check if this is the last round
	if room.CurrentRound >= room.MaxRounds {
		roundEndData.NextRound = 0 // Indicate game is ending
	}

	room.EndRound()
	gameEngine.EndRound(room)

	// Send round end message
	msg, err := websocket.NewRoundEndedMessage(roundEndData)
	if err != nil {
		log.Printf("Error creating round ended message: %v", err)
		return
	}
	msgData, err := msg.ToJSON()
	if err != nil {
		log.Printf("Error converting round ended message to JSON: %v", err)
		return
	}
	hub.BroadcastToRoom(roomID, msgData, nil)

	// Check if game should end
	if room.CurrentRound >= room.MaxRounds {
		HandleGameEnd(hub, roomManager, gameEngine, roomID)
	} else {
		HandleNewRound(hub, roomManager, gameEngine, roomID)
	}
}

// HandleGameEnd ends the game
func HandleGameEnd(hub *websocket.Hub, roomManager *services.RoomManager, gameEngine *services.GameEngine, roomID string) {
	room := roomManager.GetRoom(roomID)
	if room == nil {
		return
	}

	// Calculate game statistics
	playerStats := make(map[string]websocket.PlayerStats)
	totalScore := 0
	highestScore := 0
	var winner *models.PublicUser
	leaderboard := getLeaderboard(room)

	for _, player := range room.Players {
		totalScore += player.Score
		if player.Score > highestScore {
			highestScore = player.Score
			winner = player.ToPublicUser()
		}
		playerStats[player.ID] = websocket.PlayerStats{
			CorrectGuesses: player.CorrectGuesses,
			TotalGuesses:   player.TotalGuesses,
			Accuracy:       player.GetAccuracy(),
			TimesDrawer:    player.TimesDrawer,
			AveragePoints:  float64(player.Score) / float64(room.CurrentRound),
		}
	}

	gameEndData := websocket.GameEndData{
		Winner:      winner,
		Leaderboard: leaderboard,
		GameStats: websocket.GameStats{
			TotalRounds:  room.CurrentRound,
			TotalPlayers: len(room.Players),
			AverageScore: float64(totalScore) / float64(len(room.Players)),
			HighestScore: highestScore,
			PlayerStats:  playerStats,
		},
	}

	room.EndGame()

	// Send game end message
	msg, err := websocket.NewGameEndedMessage(gameEndData)
	if err != nil {
		log.Printf("Error creating game ended message: %v", err)
		return
	}
	msgData, err := msg.ToJSON()
	if err != nil {
		log.Printf("Error converting game ended message to JSON: %v", err)
		return
	}
	hub.BroadcastToRoom(roomID, msgData, nil)
}

// runRoundTimer manages the round timer
func runRoundTimer(hub *websocket.Hub, roomManager *services.RoomManager, gameEngine *services.GameEngine, roomID string) {
	room := roomManager.GetRoom(roomID)
	if room == nil {
		return
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			timeLeft := room.GetTimeLeft()
			timerMsg, err := websocket.NewTimerMessage(timeLeft, string(room.Phase))
			if err != nil {
				log.Printf("Error creating timer message: %v", err)
				continue
			}
			timerData, err := timerMsg.ToJSON()
			if err != nil {
				log.Printf("Error converting timer message to JSON: %v", err)
				continue
			}
			hub.BroadcastToRoom(roomID, timerData, nil)

			if timeLeft <= 0 {
				HandleRoundEnd(hub, roomManager, gameEngine, roomID)
				return
			}
		}
	}
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
