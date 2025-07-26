package services

import (
	"strings"
	"time"

	"github.com/RITWIZSINGH/DoodleDash-backend/internal/config"
	"github.com/RITWIZSINGH/DoodleDash-backend/internal/models"
	"github.com/RITWIZSINGH/DoodleDash-backend/pkg/websocket"
)

// GameEngine manages game logic and point calculations
type GameEngine struct {
	wordBank *WordBank
	config   *config.Config
}

// NewGameEngine creates a new game engine
func NewGameEngine(wordBank *WordBank, config *config.Config) *GameEngine {
	return &GameEngine{
		wordBank: wordBank,
		config:   config,
	}
}

// StartGame initializes a new game
func (ge *GameEngine) StartGame(room *models.Room) {
	room.StartGame()
}

// ValidateGuess checks if a guess is correct and calculates points
func (ge *GameEngine) ValidateGuess(room *models.Room, userID string, guess string) websocket.GuessResultData {
	user, exists := room.GetPlayer(userID)
	if !exists || user.HasGuessedThisRound || userID == room.CurrentDrawer {
		return websocket.GuessResultData{Correct: false}
	}

	guess = strings.ToLower(strings.TrimSpace(guess))
	correctWord := strings.ToLower(strings.TrimSpace(room.CurrentWord))

	if guess != correctWord {
		user.RecordGuess(false, 0)
		return websocket.GuessResultData{Correct: false}
	}

	// Correct guess
	guessOrder := len(room.GuessedPlayers) + 1
	guessTime := int(time.Since(room.RoundStartTime).Seconds())
	points := ge.CalculateGuesserPoints(
		guessOrder,
		len(room.GuessedPlayers)+1,
		guessTime,
		room.RoundTime,
		len(room.Players),
		string(room.Difficulty),
	)

	user.RecordGuess(true, guessOrder)
	user.AddScore(points)
	room.AddGuess(userID)

	// Calculate bonuses
	orderBonus := int(float64(ge.config.Points.MaxOrderBonus) * (1.0 - float64(guessOrder-1)/float64(len(room.Players))))
	difficultyBonus := int(float64(ge.config.Points.MaxDifficultyBonus) * (1.0 - float64(len(room.GuessedPlayers))/float64(len(room.Players))))
	timeBonus := int(float64(ge.config.Points.MaxTimeBonus) * (1.0 - float64(guessTime)/float64(room.RoundTime)))

	roundEnding := len(room.GuessedPlayers) == len(room.Players)-1 || room.GetTimeLeft() <= 0

	return websocket.GuessResultData{
		Correct:     true,
		Word:        room.CurrentWord,
		Points:      points,
		TotalScore:  user.Score,
		GuessOrder:  guessOrder,
		Bonus:       difficultyBonus,
		TimeBonus:   timeBonus,
		OrderBonus:  orderBonus,
		RoundEnding: roundEnding,
	}
}

// CalculateGuesserPoints calculates points for a correct guess
func (ge *GameEngine) CalculateGuesserPoints(guessOrder, totalGuessers, guessTime, roundDuration, totalPlayers int, difficulty string) int {
	points := ge.config.Points.BaseGuessPoints

	// Order bonus: First to guess gets more points
	orderBonus := float64(ge.config.Points.MaxOrderBonus) * (1.0 - float64(guessOrder-1)/float64(totalPlayers))
	points += int(orderBonus)

	// Difficulty bonus: Fewer correct guesses = more points
	difficultyBonus := float64(ge.config.Points.MaxDifficultyBonus) * (1.0 - float64(totalGuessers)/float64(totalPlayers))
	points += int(difficultyBonus)

	// Time bonus: Faster guesses = more points
	timeBonus := float64(ge.config.Points.MaxTimeBonus) * (1.0 - float64(guessTime)/float64(roundDuration))
	points += int(timeBonus)

	// Difficulty multiplier
	switch difficulty {
	case "medium":
		points = int(float64(points) * 1.25)
	case "hard":
		points = int(float64(points) * 1.5)
	}

	return points
}

// CalculateDrawerPoints calculates points for the drawer
func (ge *GameEngine) CalculateDrawerPoints(totalGuessers, totalPlayers, roundDuration int) int {
	points := ge.config.Points.DrawerBasePoints
	points += totalGuessers * ge.config.Points.DrawerBonusPerGuesser
	return points
}

// EndRound ends the current round
func (ge *GameEngine) EndRound(room *models.Room) {
	room.EndRound()
}

// GetRandomWord selects a random word and hint
func (ge *GameEngine) GetRandomWord(difficulty string) (string, string) {
	word := ge.wordBank.GetRandomWord(difficulty)
	hint := ge.GetWordHint(word, difficulty)
	return word, hint
}

// GetWordHint creates a hint for the word
func (ge *GameEngine) GetWordHint(word, difficulty string) string {
	switch difficulty {
	case "easy":
		return strings.Repeat("_ ", len(word))
	case "medium":
		if len(word) <= 2 {
			return strings.Repeat("_ ", len(word))
		}
		hint := make([]rune, len(word))
		for i := range hint {
			if i == 0 || i == len(word)-1 {
				hint[i] = rune(word[i])
			} else {
				hint[i] = '_'
			}
		}
		return strings.Join(strings.Split(string(hint), ""), " ")
	case "hard":
		// Simple category-based hints
		if strings.Contains(word, "cat") || strings.Contains(word, "dog") || strings.Contains(word, "bird") {
			return "Animal"
		}
		return "Object"
	default:
		return strings.Repeat("_ ", len(word))
	}
}