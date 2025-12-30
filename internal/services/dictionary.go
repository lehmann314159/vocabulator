package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/lehmann314159/vocabulator/internal/models"
)

const (
	dictionaryAPIBaseURL = "https://api.dictionaryapi.dev/api/v2/entries/en"
	defaultTimeout       = 10 * time.Second
)

// DictionaryService provides dictionary lookup functionality
type DictionaryService struct {
	client  *http.Client
	baseURL string
}

// NewDictionaryService creates a new dictionary service
func NewDictionaryService() *DictionaryService {
	return &DictionaryService{
		client: &http.Client{
			Timeout: defaultTimeout,
		},
		baseURL: dictionaryAPIBaseURL,
	}
}

// NewDictionaryServiceWithClient creates a new dictionary service with a custom HTTP client
func NewDictionaryServiceWithClient(client *http.Client, baseURL string) *DictionaryService {
	return &DictionaryService{
		client:  client,
		baseURL: baseURL,
	}
}

// ErrWordNotFound is returned when the word is not found in the dictionary
var ErrWordNotFound = fmt.Errorf("word not found in dictionary")

// Lookup fetches the definition of a word from the dictionary API
func (s *DictionaryService) Lookup(ctx context.Context, word string) (*models.DictionaryResponse, error) {
	url := fmt.Sprintf("%s/%s", s.baseURL, word)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch definition: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrWordNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dictionary API returned status %d", resp.StatusCode)
	}

	var entries []models.DictionaryEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(entries) == 0 {
		return nil, ErrWordNotFound
	}

	return s.transformResponse(entries), nil
}

// transformResponse converts the API response to our response format
func (s *DictionaryService) transformResponse(entries []models.DictionaryEntry) *models.DictionaryResponse {
	entry := entries[0]

	response := &models.DictionaryResponse{
		Word:     entry.Word,
		Phonetic: entry.Phonetic,
		Meanings: entry.Meanings,
	}

	// Find the first audio URL
	for _, phonetic := range entry.Phonetics {
		if phonetic.Audio != "" {
			response.AudioURL = phonetic.Audio
			break
		}
		// Use phonetic text if main phonetic is empty
		if response.Phonetic == "" && phonetic.Text != "" {
			response.Phonetic = phonetic.Text
		}
	}

	// Collect source URLs from all entries
	seen := make(map[string]bool)
	for _, e := range entries {
		if e.SourceURL != "" && !seen[e.SourceURL] {
			response.SourceURLs = append(response.SourceURLs, e.SourceURL)
			seen[e.SourceURL] = true
		}
	}

	return response
}

// GetFirstPartOfSpeech returns the first part of speech from a dictionary lookup
func (s *DictionaryService) GetFirstPartOfSpeech(ctx context.Context, word string) (string, error) {
	resp, err := s.Lookup(ctx, word)
	if err != nil {
		return "", err
	}

	if len(resp.Meanings) > 0 {
		return resp.Meanings[0].PartOfSpeech, nil
	}

	return "", nil
}
