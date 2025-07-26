package services

import (
	"sort"

	"github.com/RITWIZSINGH/DoodleDash-backend/internal/models"
)

// FindBestPublicRoom finds the best public room to join
func (rm *RoomManager) FindBestPublicRoom(maxPlayers int, difficulty string) *models.PublicRoomInfo {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	var candidates []*models.PublicRoomInfo
	for _, room := range rm.rooms {
		if room.Type == models.RoomTypePublic && room.State == models.GameStateLobby && !room.IsFull() && room.IsActive(rm.config.Game.InactiveRoomTimeout) {
			if room.MaxPlayers <= maxPlayers && (difficulty == "" || string(room.Difficulty) == difficulty) {
				candidates = append(candidates, room.GetPublicRoomInfo())
			}
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	// Sort by player count (prefer nearly full rooms) and activity
	sort.Slice(candidates, func(i, j int) bool {
		// Higher player count first
		if candidates[i].PlayerCount != candidates[j].PlayerCount {
			return candidates[i].PlayerCount > candidates[j].PlayerCount
		}
		// More recent activity
		return rm.rooms[candidates[i].ID].LastActivity.After(rm.rooms[candidates[j].ID].LastActivity)
	})

	return candidates[0]
}

// AutoJoinPublicRoom automatically joins a player to a suitable public room
func (rm *RoomManager) AutoJoinPublicRoom(userID string, user *models.User) *models.PublicRoomInfo {
	roomInfo := rm.FindBestPublicRoom(rm.config.Game.MaxPlayersPerRoom, "")
	if roomInfo == nil {
		// Create new room if none suitable
		room := rm.CreateRoom(userID, models.RoomTypePublic, "Auto-Created Room", models.CreateRoomData{
			RoomName:   "Auto-Created Room",
			RoomType:   "public",
			MaxPlayers: rm.config.Game.MaxPlayersPerRoom,
			RoundTime:  int(rm.config.Game.RoundDuration.Seconds()),
			MaxRounds:  rm.config.Game.MaxRounds,
			Difficulty: "easy",
		})
		roomInfo = room.GetPublicRoomInfo()
	}

	if rm.JoinRoom(roomInfo.ID, userID, user) {
		return roomInfo
	}
	return nil
}