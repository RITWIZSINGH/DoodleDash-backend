package config

import (
	"fmt"
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"
)

// Config represents the application configuration
type Config struct {
	Server     ServerConfig     `yaml:"server"`
	WebSocket  WebSocketConfig  `yaml:"websocket"`
	Game       GameConfig       `yaml:"game"`
	Points     PointsConfig     `yaml:"points"`
	RateLimit  RateLimitConfig  `yaml:"rate_limit"`
	CORS       CORSConfig       `yaml:"cors"`
	WordBank   WordBankConfig   `yaml:"word_bank"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Port         string        `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

// WebSocketConfig contains WebSocket-specific configuration
type WebSocketConfig struct {
	ReadBufferSize  int           `yaml:"read_buffer_size"`
	WriteBufferSize int           `yaml:"write_buffer_size"`
	MaxMessageSize  int64         `yaml:"max_message_size"`
	PongWait        time.Duration `yaml:"pong_wait"`
	PingPeriod      time.Duration `yaml:"ping_period"`
	WriteWait       time.Duration `yaml:"write_wait"`
}

// GameConfig contains game-specific configuration
type GameConfig struct {
	MaxPlayersPerRoom      int           `yaml:"max_players_per_room"`
	MinPlayersToStart      int           `yaml:"min_players_to_start"`
	RoundDuration          time.Duration `yaml:"round_duration"`
	MaxRounds              int           `yaml:"max_rounds"`
	RoomCleanupInterval    time.Duration `yaml:"room_cleanup_interval"`
	InactiveRoomTimeout    time.Duration `yaml:"inactive_room_timeout"`
}

// PointsConfig contains point system configuration
type PointsConfig struct {
	BaseGuessPoints        int `yaml:"base_guess_points"`
	MaxOrderBonus          int `yaml:"max_order_bonus"`
	MaxDifficultyBonus     int `yaml:"max_difficulty_bonus"`
	MaxTimeBonus           int `yaml:"max_time_bonus"`
	DrawerBasePoints       int `yaml:"drawer_base_points"`
	DrawerBonusPerGuesser  int `yaml:"drawer_bonus_per_guesser"`
}

// RateLimitConfig contains rate limiting configuration
type RateLimitConfig struct {
	RequestsPerMinute int `yaml:"requests_per_minute"`
	BurstSize         int `yaml:"burst_size"`
}

// CORSConfig contains CORS configuration
type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
	AllowedMethods []string `yaml:"allowed_methods"`
	AllowedHeaders []string `yaml:"allowed_headers"`
}

// WordBankConfig contains word bank file paths
type WordBankConfig struct {
	EasyWordsFile   string `yaml:"easy_words_file"`
	MediumWordsFile string `yaml:"medium_words_file"`
	HardWordsFile   string `yaml:"hard_words_file"`
}

// Global configuration instance
var AppConfig *Config

// LoadConfig loads configuration from a YAML file
func LoadConfig(configPath string) (*Config, error) {
	// First try to load from file
	config, err := loadConfigFromFile(configPath)
	if err != nil {
		// If file loading fails, use default config
		fmt.Printf("Warning: Could not load config from %s, using defaults: %v\n", configPath, err)
		config = GetDefaultConfig()
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Set global config
	AppConfig = config

	return config, nil
}

// loadConfigFromFile loads configuration from a YAML file
func loadConfigFromFile(configPath string) (*Config, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// GetDefaultConfig returns the default configuration
func GetDefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         ":8080",
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		WebSocket: WebSocketConfig{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			MaxMessageSize:  512,
			PongWait:        60 * time.Second,
			PingPeriod:      54 * time.Second,
			WriteWait:       10 * time.Second,
		},
		Game: GameConfig{
			MaxPlayersPerRoom:   8,
			MinPlayersToStart:   2,
			RoundDuration:       60 * time.Second,
			MaxRounds:           5,
			RoomCleanupInterval: 5 * time.Minute,
			InactiveRoomTimeout: 30 * time.Minute,
		},
		Points: PointsConfig{
			BaseGuessPoints:       100,
			MaxOrderBonus:         50,
			MaxDifficultyBonus:    100,
			MaxTimeBonus:          25,
			DrawerBasePoints:      20,
			DrawerBonusPerGuesser: 15,
		},
		RateLimit: RateLimitConfig{
			RequestsPerMinute: 60,
			BurstSize:         10,
		},
		CORS: CORSConfig{
			AllowedOrigins: []string{
				"http://localhost:3000",
				"http://localhost:8080",
				"https://yourdomain.com",
			},
			AllowedMethods: []string{
				"GET", "POST", "PUT", "DELETE", "OPTIONS",
			},
			AllowedHeaders: []string{
				"Origin", "Content-Type", "Accept", "Authorization",
			},
		},
		WordBank: WordBankConfig{
			EasyWordsFile:   "data/words.json",
			MediumWordsFile: "data/words.json",
			HardWordsFile:   "data/words.json",
		},
	}
}

// validateConfig validates the configuration values
func validateConfig(config *Config) error {
	// Validate server config
	if config.Server.Port == "" {
		return fmt.Errorf("server port cannot be empty")
	}

	// Validate game config
	if config.Game.MaxPlayersPerRoom < 2 {
		return fmt.Errorf("max players per room must be at least 2")
	}
	if config.Game.MinPlayersToStart < 2 {
		return fmt.Errorf("min players to start must be at least 2")
	}
	if config.Game.MinPlayersToStart > config.Game.MaxPlayersPerRoom {
		return fmt.Errorf("min players to start cannot be greater than max players per room")
	}
	if config.Game.RoundDuration <= 0 {
		return fmt.Errorf("round duration must be positive")
	}
	if config.Game.MaxRounds <= 0 {
		return fmt.Errorf("max rounds must be positive")
	}

	// Validate points config
	if config.Points.BaseGuessPoints <= 0 {
		return fmt.Errorf("base guess points must be positive")
	}

	// Validate WebSocket config
	if config.WebSocket.ReadBufferSize <= 0 {
		return fmt.Errorf("WebSocket read buffer size must be positive")
	}
	if config.WebSocket.WriteBufferSize <= 0 {
		return fmt.Errorf("WebSocket write buffer size must be positive")
	}
	if config.WebSocket.MaxMessageSize <= 0 {
		return fmt.Errorf("WebSocket max message size must be positive")
	}

	// Validate rate limit config
	if config.RateLimit.RequestsPerMinute <= 0 {
		return fmt.Errorf("requests per minute must be positive")
	}
	if config.RateLimit.BurstSize <= 0 {
		return fmt.Errorf("burst size must be positive")
	}

	return nil
}

// GetConfig returns the current configuration
func GetConfig() *Config {
	if AppConfig == nil {
		AppConfig = GetDefaultConfig()
	}
	return AppConfig
}

// IsDevelopment returns true if running in development mode
func IsDevelopment() bool {
	// In a real application, this would check environment variables
	return true
}

// IsProduction returns true if running in production mode
func IsProduction() bool {
	return !IsDevelopment()
}

// GetServerAddress returns the full server address
func GetServerAddress() string {
	config := GetConfig()
	if config.Server.Port[0] != ':' {
		return ":" + config.Server.Port
	}
	return config.Server.Port
}

// GetWebSocketURL returns the WebSocket URL for clients
func GetWebSocketURL() string {
	config := GetConfig()
	
	scheme := "ws"
	if IsProduction() {
		scheme = "wss"
	}
	
	host := "localhost"
	if IsProduction() {
		host = "yourdomain.com"
	}
	
	return fmt.Sprintf("%s://%s%s/ws", scheme, host, config.Server.Port)
}

// SaveConfig saves the current configuration to a file
func SaveConfig(configPath string) error {
	if AppConfig == nil {
		return fmt.Errorf("no configuration to save")
	}

	data, err := yaml.Marshal(AppConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := ioutil.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// UpdateGameConfig updates game configuration at runtime
func UpdateGameConfig(updates GameConfig) error {
	config := GetConfig()
	
	// Validate updates
	tempConfig := *config
	tempConfig.Game = updates
	
	if err := validateConfig(&tempConfig); err != nil {
		return err
	}
	
	// Apply updates
	config.Game = updates
	
	return nil
}

// UpdatePointsConfig updates points configuration at runtime
func UpdatePointsConfig(updates PointsConfig) error {
	config := GetConfig()
	
	// Validate updates
	tempConfig := *config
	tempConfig.Points = updates
	
	if err := validateConfig(&tempConfig); err != nil {
		return err
	}
	
	// Apply updates
	config.Points = updates
	
	return nil
}