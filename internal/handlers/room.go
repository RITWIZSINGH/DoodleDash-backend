package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/RITWIZSINGH/DoodleDash-backend/internal/models"
	"github.com/RITWIZSINGH/DoodleDash-backend/internal/services"
	"github.com/RITWIZSINGH/DoodleDash-backend/pkg/utils"
	"github.com/RITWIZSINGH/DoodleDash-backend/pkg/websocket"
)

// GetPublicRooms returns a list of public rooms
func GetPublicRooms(roomManager *services.RoomManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rooms := roomManager.GetPublicRooms()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rooms)
	}
}

// CreateRoom creates a new room via HTTP
func CreateRoom(hub *websocket.Hub, roomManager *services.RoomManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var data models.CreateRoomData
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate input
		if !utils.ValidateUserName(data.RoomName) {
			http.Error(w, "Invalid room name", http.StatusBadRequest)
			return
		}
		data.RoomName = utils.SanitizeInput(data.RoomName)

		// Create guest user as host
		user := models.NewGuestUser()
		roomType := models.RoomTypePublic
		if data.RoomType == "private" {
			roomType = models.RoomTypePrivate
		}

		// Create room
		room := roomManager.CreateRoom(user.ID, roomType, data.RoomName, data)
		if room == nil {
			http.Error(w, "Failed to create room", http.StatusInternalServerError)
			return
		}

		roomInfo := room.GetPublicRoomInfo()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(roomInfo)
	}
}

// GetRoomDetails returns details about a specific room
func GetRoomDetails(roomManager *services.RoomManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		roomID := vars["roomID"]

		room := roomManager.GetRoom(roomID)
		if room == nil {
			http.Error(w, "Room not found", http.StatusNotFound)
			return
		}

		roomInfo := room.GetPublicRoomInfo()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(roomInfo)
	}
}