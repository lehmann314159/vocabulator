package models

import (
	"time"
)

// Word represents a vocabulary word entity
type Word struct {
	ID              int64     `json:"id"`
	Word            string    `json:"word"`
	Source          string    `json:"source"`
	DateLearned     string    `json:"date_learned"` // YYYY-MM-DD format
	PartOfSpeech    *string   `json:"part_of_speech,omitempty"`
	ExampleSentence *string   `json:"example_sentence,omitempty"`
	Tags            []string  `json:"tags"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CreateWordRequest represents the request body for creating a word
type CreateWordRequest struct {
	Word            string   `json:"word"`
	Source          string   `json:"source"`
	DateLearned     string   `json:"date_learned"`
	PartOfSpeech    *string  `json:"part_of_speech,omitempty"`
	ExampleSentence *string  `json:"example_sentence,omitempty"`
	Tags            []string `json:"tags,omitempty"`
}

// UpdateWordRequest represents the request body for updating a word
type UpdateWordRequest struct {
	Word            *string  `json:"word,omitempty"`
	Source          *string  `json:"source,omitempty"`
	DateLearned     *string  `json:"date_learned,omitempty"`
	PartOfSpeech    *string  `json:"part_of_speech,omitempty"`
	ExampleSentence *string  `json:"example_sentence,omitempty"`
	Tags            []string `json:"tags,omitempty"`
}

// WordFilter represents query parameters for filtering words
type WordFilter struct {
	Search   string
	Source   string
	Tag      string
	FromDate string
	ToDate   string
	Limit    int
	Offset   int
}

// DictionaryEntry represents a response from the dictionary API
type DictionaryEntry struct {
	Word      string       `json:"word"`
	Phonetic  string       `json:"phonetic,omitempty"`
	Phonetics []Phonetic   `json:"phonetics,omitempty"`
	Meanings  []Meaning    `json:"meanings"`
	SourceURL string       `json:"sourceUrl,omitempty"`
}

// Phonetic represents pronunciation information
type Phonetic struct {
	Text  string `json:"text,omitempty"`
	Audio string `json:"audio,omitempty"`
}

// Meaning represents a word meaning with definitions
type Meaning struct {
	PartOfSpeech string       `json:"partOfSpeech"`
	Definitions  []Definition `json:"definitions"`
}

// Definition represents a single definition
type Definition struct {
	Definition string   `json:"definition"`
	Example    string   `json:"example,omitempty"`
	Synonyms   []string `json:"synonyms,omitempty"`
	Antonyms   []string `json:"antonyms,omitempty"`
}

// DictionaryResponse is the full response we return to clients
type DictionaryResponse struct {
	Word       string            `json:"word"`
	Phonetic   string            `json:"phonetic,omitempty"`
	AudioURL   string            `json:"audio_url,omitempty"`
	Meanings   []Meaning         `json:"meanings"`
	SourceURLs []string          `json:"source_urls,omitempty"`
}
