package services

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/lehmann314159/vocabulator/internal/models"
	"github.com/lehmann314159/vocabulator/internal/repository"
)

// WordService provides business logic for word operations
type WordService struct {
	repo       repository.WordRepository
	dictionary *DictionaryService
}

// NewWordService creates a new word service
func NewWordService(repo repository.WordRepository, dictionary *DictionaryService) *WordService {
	return &WordService{
		repo:       repo,
		dictionary: dictionary,
	}
}

// Create creates a new word
func (s *WordService) Create(ctx context.Context, req *models.CreateWordRequest) (*models.Word, error) {
	if req.Word == "" {
		return nil, fmt.Errorf("word is required")
	}
	if req.Source == "" {
		return nil, fmt.Errorf("source is required")
	}
	if req.DateLearned == "" {
		return nil, fmt.Errorf("date_learned is required")
	}

	// Check for duplicate
	existing, err := s.repo.GetByWord(ctx, req.Word)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("word '%s' already exists", req.Word)
	}

	word := &models.Word{
		Word:            req.Word,
		Source:          req.Source,
		DateLearned:     req.DateLearned,
		PartOfSpeech:    req.PartOfSpeech,
		ExampleSentence: req.ExampleSentence,
		Tags:            req.Tags,
	}

	if word.Tags == nil {
		word.Tags = []string{}
	}

	return s.repo.Create(ctx, word)
}

// GetByID retrieves a word by ID
func (s *WordService) GetByID(ctx context.Context, id int64) (*models.Word, error) {
	return s.repo.GetByID(ctx, id)
}

// List retrieves words with optional filtering
func (s *WordService) List(ctx context.Context, filter models.WordFilter) ([]*models.Word, error) {
	return s.repo.List(ctx, filter)
}

// Update updates an existing word
func (s *WordService) Update(ctx context.Context, id int64, req *models.UpdateWordRequest) (*models.Word, error) {
	word, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Word != nil {
		// Check for duplicate if word is being changed
		if *req.Word != word.Word {
			existing, err := s.repo.GetByWord(ctx, *req.Word)
			if err == nil && existing != nil {
				return nil, fmt.Errorf("word '%s' already exists", *req.Word)
			}
		}
		word.Word = *req.Word
	}
	if req.Source != nil {
		word.Source = *req.Source
	}
	if req.DateLearned != nil {
		word.DateLearned = *req.DateLearned
	}
	if req.PartOfSpeech != nil {
		word.PartOfSpeech = req.PartOfSpeech
	}
	if req.ExampleSentence != nil {
		word.ExampleSentence = req.ExampleSentence
	}
	if req.Tags != nil {
		word.Tags = req.Tags
	}

	return s.repo.Update(ctx, word)
}

// Delete deletes a word by ID
func (s *WordService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// GetRandom retrieves a random word
func (s *WordService) GetRandom(ctx context.Context) (*models.Word, error) {
	return s.repo.GetRandom(ctx)
}

// GetDefinition fetches the definition of a word from the dictionary
func (s *WordService) GetDefinition(ctx context.Context, id int64) (*models.DictionaryResponse, error) {
	word, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.dictionary.Lookup(ctx, word.Word)
}

// ImportResult contains the results of a CSV import operation
type ImportResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors,omitempty"`
}

// ImportCSV imports words from a CSV reader
func (s *WordService) ImportCSV(ctx context.Context, r io.Reader) (*ImportResult, error) {
	reader := csv.NewReader(r)

	// Read header
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Map column names to indices
	colIndex := make(map[string]int)
	for i, col := range header {
		colIndex[strings.ToLower(strings.TrimSpace(col))] = i
	}

	// Validate required columns
	requiredCols := []string{"word", "source", "date_learned"}
	for _, col := range requiredCols {
		if _, ok := colIndex[col]; !ok {
			return nil, fmt.Errorf("missing required column: %s", col)
		}
	}

	result := &ImportResult{}
	lineNum := 1 // Header is line 1

	for {
		lineNum++
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("line %d: %v", lineNum, err))
			result.Skipped++
			continue
		}

		word := &models.Word{
			Word:        strings.TrimSpace(record[colIndex["word"]]),
			Source:      strings.TrimSpace(record[colIndex["source"]]),
			DateLearned: strings.TrimSpace(record[colIndex["date_learned"]]),
			Tags:        []string{},
		}

		if word.Word == "" || word.Source == "" || word.DateLearned == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("line %d: missing required field", lineNum))
			result.Skipped++
			continue
		}

		// Optional fields
		if idx, ok := colIndex["part_of_speech"]; ok && idx < len(record) {
			if val := strings.TrimSpace(record[idx]); val != "" {
				word.PartOfSpeech = &val
			}
		}

		if idx, ok := colIndex["example_sentence"]; ok && idx < len(record) {
			if val := strings.TrimSpace(record[idx]); val != "" {
				word.ExampleSentence = &val
			}
		}

		if idx, ok := colIndex["tags"]; ok && idx < len(record) {
			if val := strings.TrimSpace(record[idx]); val != "" {
				// Split comma-separated tags
				tags := strings.Split(val, ",")
				for i, tag := range tags {
					tags[i] = strings.TrimSpace(tag)
				}
				word.Tags = tags
			}
		}

		// Check for duplicate
		existing, _ := s.repo.GetByWord(ctx, word.Word)
		if existing != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("line %d: word '%s' already exists", lineNum, word.Word))
			result.Skipped++
			continue
		}

		_, err = s.repo.Create(ctx, word)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("line %d: %v", lineNum, err))
			result.Skipped++
			continue
		}

		result.Imported++
	}

	return result, nil
}

// ExportCSV exports all words to CSV format
func (s *WordService) ExportCSV(ctx context.Context, w io.Writer) error {
	words, err := s.repo.List(ctx, models.WordFilter{})
	if err != nil {
		return fmt.Errorf("failed to fetch words: %w", err)
	}

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	header := []string{"word", "source", "date_learned", "part_of_speech", "example_sentence", "tags"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write records
	for _, word := range words {
		partOfSpeech := ""
		if word.PartOfSpeech != nil {
			partOfSpeech = *word.PartOfSpeech
		}

		exampleSentence := ""
		if word.ExampleSentence != nil {
			exampleSentence = *word.ExampleSentence
		}

		tags := strings.Join(word.Tags, ",")

		record := []string{
			word.Word,
			word.Source,
			word.DateLearned,
			partOfSpeech,
			exampleSentence,
			tags,
		}

		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write record: %w", err)
		}
	}

	return nil
}

// Count returns the total number of words
func (s *WordService) Count(ctx context.Context, filter models.WordFilter) (int64, error) {
	return s.repo.Count(ctx, filter)
}
