package services

import (
	"encoding/json"
	"io/ioutil"
	"log"

	"github.com/RITWIZSINGH/DoodleDash-backend/internal/config"
)

// WordBank manages word lists
type WordBank struct {
	easyWords   []string
	mediumWords []string
	hardWords   []string
	usedWords   map[string]bool
}

// NewWordBank creates a new word bank
func NewWordBank(config *config.Config) (*WordBank, error) {
	wb := &WordBank{
		usedWords: make(map[string]bool),
	}

	// Load words
	if err := wb.LoadWords(config.WordBank.EasyWordsFile); err != nil {
		return nil, err
	}
	if err := wb.LoadWords(config.WordBank.MediumWordsFile); err != nil {
		return nil, err
	}
	if err := wb.LoadWords(config.WordBank.HardWordsFile); err != nil {
		return nil, err
	}

	return wb, nil
}

// LoadWords loads words from a file
func (wb *WordBank) LoadWords(filePath string) error {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	var words struct {
		Easy   []string `json:"easy"`
		Medium []string `json:"medium"`
		Hard   []string `json:"hard"`
	}
	if err := json.Unmarshal(data, &words); err != nil {
		return err
	}

	wb.easyWords = append(wb.easyWords, words.Easy...)
	wb.mediumWords = append(wb.mediumWords, words.Medium...)
	wb.hardWords = append(wb.hardWords, words.Hard...)

	log.Printf("Loaded %d easy, %d medium, %d hard words", len(wb.easyWords), len(wb.mediumWords), len(wb.hardWords))
	return nil
}

// GetRandomWord returns a random word for the given difficulty
func (wb *WordBank) GetRandomWord(difficulty string) string {
	var words []string
	switch difficulty {
	case "easy":
		words = wb.easyWords
	case "medium":
		words = wb.mediumWords
	case "hard":
		words = wb.hardWords
	default:
		words = wb.easyWords
	}

	if len(words) == 0 {
		return "default"
	}

	// Simple random selection (in reality, use crypto/rand)
	for i := 0; i < len(words); i++ {
		word := words[i]
		if !wb.usedWords[word] {
			wb.usedWords[word] = true
			return word
		}
	}

	// Reset used words if all have been used
	wb.usedWords = make(map[string]bool)
	word := words[0]
	wb.usedWords[word] = true
	return word
}

// AddCustomWords adds custom words to a room
func (wb *WordBank) AddCustomWords(roomID string, words []string) {
	// In a real implementation, store custom words per room
	wb.easyWords = append(wb.easyWords, words...) // Add to easy for simplicity
	log.Printf("Added %d custom words for room %s", len(words), roomID)
}