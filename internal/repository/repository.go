package repository

import (
	"context"

	"github.com/lehmann314159/vocabulator/internal/models"
)

// WordRepository defines the interface for word persistence operations
type WordRepository interface {
	// Create inserts a new word and returns the created word with ID
	Create(ctx context.Context, word *models.Word) (*models.Word, error)

	// GetByID retrieves a word by its ID
	GetByID(ctx context.Context, id int64) (*models.Word, error)

	// GetByWord retrieves a word by the word text itself
	GetByWord(ctx context.Context, word string) (*models.Word, error)

	// List retrieves words with optional filtering
	List(ctx context.Context, filter models.WordFilter) ([]*models.Word, error)

	// Update modifies an existing word
	Update(ctx context.Context, word *models.Word) (*models.Word, error)

	// Delete removes a word by ID
	Delete(ctx context.Context, id int64) error

	// GetRandom retrieves a random word
	GetRandom(ctx context.Context) (*models.Word, error)

	// Count returns the total number of words matching the filter
	Count(ctx context.Context, filter models.WordFilter) (int64, error)
}
