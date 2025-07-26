package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/RITWIZSINGH/DoodleDash-backend/internal/config"
	"github.com/RITWIZSINGH/DoodleDash-backend/internal/handlers"
	"github.com/RITWIZSINGH/DoodleDash-backend/internal/middleware"
	"github.com/RITWIZSINGH/DoodleDash-backend/internal/services"
	"github.com/RITWIZSINGH/DoodleDash-backend/pkg/websocket"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize WebSocket hub
	hub := websocket.NewHub()

	// Initialize services
	roomManager := services.NewRoomManager()
	wordBank, err := services.NewWordBank(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize word bank: %v", err)
	}
	gameEngine := services.NewGameEngine(wordBank, cfg)

	// Set up message processor for WebSocket hub
	hub.SetMessageProcessor(func(msg *websocket.MessageWithClient) {
		handlers.HandleWebSocketMessage(hub, roomManager, gameEngine, msg.Client, msg.Message)
	})

	// Start hub in a goroutine
	go hub.Run()

	// Set up router
	router := mux.NewRouter()
	setupRoutes(router, hub, roomManager)

	// Apply middleware
	srv := &http.Server{
		Addr:         cfg.Server.Port,
		Handler:      middleware.ApplyMiddleware(router, cfg),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server
	go func() {
		log.Printf("Starting server on %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Handle graceful shutdown
	gracefulShutdown(srv, hub, roomManager)
}

// setupRoutes configures the HTTP routes
func setupRoutes(router *mux.Router, hub *websocket.Hub, roomManager *services.RoomManager) {
	// Health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// WebSocket endpoint
	router.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handlers.ServeWS(hub, w, r)
	}).Methods("GET")

	// Room API endpoints
	roomRouter := router.PathPrefix("/api/rooms").Subrouter()
	roomRouter.HandleFunc("/public", handlers.GetPublicRooms(roomManager)).Methods("GET")
	roomRouter.HandleFunc("", handlers.CreateRoom(hub, roomManager)).Methods("POST")
	roomRouter.HandleFunc("/{roomID}", handlers.GetRoomDetails(roomManager)).Methods("GET")
}

// gracefulShutdown handles server shutdown gracefully
func gracefulShutdown(srv *http.Server, hub *websocket.Hub, roomManager *services.RoomManager) {
	// Create channel for OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	log.Println("Shutting down server...")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown WebSocket hub
	hub.Shutdown()

	// Cleanup rooms
	roomManager.Cleanup()

	// Shutdown HTTP server
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server gracefully stopped")
}