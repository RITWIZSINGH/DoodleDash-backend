package services

import (
	"log"
	"sync"
	"time"

	"github.com/RITWIZSINGH/DoodleDash-backend/internal/config"
	"github.com/RITWIZSINGH/DoodleDash-backend/internal/models"
)

// RoomManager manages game rooms
type RoomManager struct {
	rooms       map[string]*models.Room
	roomByCode  map[string]*models.Room
	mutex       sync.RWMutex
	config      *config.Config
	cleanupStop chan struct{}
}

// NewRoomManager creates a new room manager
func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms:      make(map[string]*models.Room),
		roomByCode: make(map[string]*models.Room),
		cleanupStop: make(chan struct{}),
		config:     config.GetConfig(),
	}
}

// CreateRoom creates a new room
func (rm *RoomManager) CreateRoom(hostID string, roomType models.RoomType, roomName string, settings models.CreateRoomData) *models.Room {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	room := models.NewRoom(hostID, roomType, roomName, settings)
	rm.rooms[room.ID] = room
	rm.roomByCode[room.Code] = room

	log.Printf("Created room %s (%s) by host %s", room.ID, room.Code, hostID)
	return room
}

// GetRoom returns a room by ID
func (rm *RoomManager) GetRoom(roomID string) *models.Room {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	return rm.rooms[roomID]
}

// GetRoomByCode returns a room by its code
func (rm *RoomManager) GetRoomByCode(code string) *models.Room {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	return rm.roomByCode[code]
}

// GetPublicRooms returns all public rooms
func (rm *RoomManager) GetPublicRooms() []*models.PublicRoomInfo {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	rooms := make([]*models.PublicRoomInfo, 0)
	for _, room := range rm.rooms {
		if room.Type == models.RoomTypePublic && room.IsActive(rm.config.Game.InactiveRoomTimeout) {
			rooms = append(rooms, room.GetPublicRoomInfo())
		}
	}
	return rooms
}

// JoinRoom adds a player to a room
func (rm *RoomManager) JoinRoom(roomID, userID string, user *models.User) bool {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	room := rm.rooms[roomID]
	if room == nil {
		return false
	}

	if room.IsFull() {
		return false
	}

	return room.AddPlayer(user)
}

// LeaveRoom removes a player from a room
func (rm *RoomManager) LeaveRoom(roomID, userID string) bool {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	room := rm.rooms[roomID]
	if room == nil {
		return false
	}

	if room.RemovePlayer(userID) {
		// If room is empty, remove it
		if room.GetPlayerCount() == 0 {
			delete(rm.rooms, roomID)
			delete(rm.roomByCode, room.Code)
			log.Printf("Removed empty room %s (%s)", room.ID, room.Code)
		}
		return true
	}
	return false
}

// IsRoomFull checks if a room is full
func (rm *RoomManager) IsRoomFull(roomID string) bool {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	room := rm.rooms[roomID]
	if room == nil {
		return true
	}
	return room.IsFull()
}

// UpdateRoomActivity updates a room's last activity timestamp
func (rm *RoomManager) UpdateRoomActivity(roomID string) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if room, exists := rm.rooms[roomID]; exists {
		room.LastActivity = time.Now()
	}
}

// Cleanup removes inactive rooms
func (rm *RoomManager) Cleanup() {
	ticker := time.NewTicker(rm.config.Game.RoomCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rm.cleanupInactiveRooms()
		case <-rm.cleanupStop:
			return
		}
	}
}

// cleanupInactiveRooms removes rooms that have been inactive for too long
func (rm *RoomManager) cleanupInactiveRooms() {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	for roomID, room := range rm.rooms {
		if !room.IsActive(rm.config.Game.InactiveRoomTimeout) {
			delete(rm.rooms, roomID)
			delete(rm.roomByCode, room.Code)
			log.Printf("Cleaned up inactive room %s (%s)", roomID, room.Code)
		}
	}
}

// StopCleanup stops the cleanup routine
func (rm *RoomManager) StopCleanup() {
	close(rm.cleanupStop)
}