package repository

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/lehmann314159/vocabulator/internal/models"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	// Create the words table
	_, err = db.Exec(`
		CREATE TABLE words (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			word TEXT NOT NULL UNIQUE,
			source TEXT NOT NULL,
			date_learned TEXT NOT NULL,
			part_of_speech TEXT,
			example_sentence TEXT,
			tags TEXT DEFAULT '[]',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	return db
}

func TestSQLiteRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	tests := []struct {
		name    string
		word    *models.Word
		wantErr bool
	}{
		{
			name: "create word with all fields",
			word: &models.Word{
				Word:            "ephemeral",
				Source:          "Book: The Road",
				DateLearned:     "2024-01-15",
				PartOfSpeech:    strPtr("adjective"),
				ExampleSentence: strPtr("The ephemeral beauty of cherry blossoms"),
				Tags:            []string{"literature", "nature"},
			},
			wantErr: false,
		},
		{
			name: "create word with minimal fields",
			word: &models.Word{
				Word:        "ubiquitous",
				Source:      "Article",
				DateLearned: "2024-02-20",
				Tags:        []string{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			created, err := repo.Create(ctx, tt.word)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if created.ID == 0 {
					t.Error("Create() returned word with ID = 0")
				}
				if created.CreatedAt.IsZero() {
					t.Error("Create() returned word with zero CreatedAt")
				}
			}
		})
	}
}

func TestSQLiteRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	// Create a test word
	word := &models.Word{
		Word:        "serendipity",
		Source:      "Conversation",
		DateLearned: "2024-03-10",
		Tags:        []string{"positive"},
	}
	created, _ := repo.Create(ctx, word)

	tests := []struct {
		name    string
		id      int64
		want    string
		wantErr bool
	}{
		{
			name:    "get existing word",
			id:      created.ID,
			want:    "serendipity",
			wantErr: false,
		},
		{
			name:    "get non-existent word",
			id:      9999,
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetByID(ctx, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.Word != tt.want {
				t.Errorf("GetByID() = %v, want %v", got.Word, tt.want)
			}
		})
	}
}

func TestSQLiteRepository_GetByWord(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	// Create a test word
	word := &models.Word{
		Word:        "eloquent",
		Source:      "Speech",
		DateLearned: "2024-04-05",
		Tags:        []string{},
	}
	repo.Create(ctx, word)

	tests := []struct {
		name     string
		wordText string
		wantErr  bool
	}{
		{
			name:     "get existing word by text",
			wordText: "eloquent",
			wantErr:  false,
		},
		{
			name:     "get non-existent word by text",
			wordText: "nonexistent",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetByWord(ctx, tt.wordText)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByWord() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.Word != tt.wordText {
				t.Errorf("GetByWord() = %v, want %v", got.Word, tt.wordText)
			}
		})
	}
}

func TestSQLiteRepository_List(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	// Create test words
	words := []*models.Word{
		{Word: "ephemeral", Source: "Book", DateLearned: "2024-01-15", Tags: []string{"literature"}},
		{Word: "ubiquitous", Source: "Article", DateLearned: "2024-02-20", Tags: []string{"technology"}},
		{Word: "eloquent", Source: "Book", DateLearned: "2024-03-10", Tags: []string{"literature"}},
	}
	for _, w := range words {
		repo.Create(ctx, w)
	}

	tests := []struct {
		name      string
		filter    models.WordFilter
		wantCount int
	}{
		{
			name:      "list all words",
			filter:    models.WordFilter{},
			wantCount: 3,
		},
		{
			name:      "filter by source",
			filter:    models.WordFilter{Source: "Book"},
			wantCount: 2,
		},
		{
			name:      "filter by tag",
			filter:    models.WordFilter{Tag: "literature"},
			wantCount: 2,
		},
		{
			name:      "filter by search",
			filter:    models.WordFilter{Search: "eph"},
			wantCount: 1,
		},
		{
			name:      "filter by date range",
			filter:    models.WordFilter{FromDate: "2024-02-01", ToDate: "2024-02-28"},
			wantCount: 1,
		},
		{
			name:      "with limit",
			filter:    models.WordFilter{Limit: 2},
			wantCount: 2,
		},
		{
			name:      "with offset",
			filter:    models.WordFilter{Limit: 2, Offset: 2},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.List(ctx, tt.filter)
			if err != nil {
				t.Errorf("List() error = %v", err)
				return
			}
			if len(got) != tt.wantCount {
				t.Errorf("List() returned %d words, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestSQLiteRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	// Create a test word
	word := &models.Word{
		Word:        "ephemeral",
		Source:      "Book",
		DateLearned: "2024-01-15",
		Tags:        []string{"literature"},
	}
	created, _ := repo.Create(ctx, word)

	// Update the word
	created.Source = "Updated Book"
	created.Tags = []string{"literature", "updated"}
	updated, err := repo.Update(ctx, created)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify the update
	got, _ := repo.GetByID(ctx, created.ID)
	if got.Source != "Updated Book" {
		t.Errorf("Update() source = %v, want %v", got.Source, "Updated Book")
	}
	if len(got.Tags) != 2 {
		t.Errorf("Update() tags count = %v, want %v", len(got.Tags), 2)
	}
	if updated.UpdatedAt.Before(created.CreatedAt) {
		t.Error("Update() UpdatedAt should be after CreatedAt")
	}
}

func TestSQLiteRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	// Create a test word
	word := &models.Word{
		Word:        "ephemeral",
		Source:      "Book",
		DateLearned: "2024-01-15",
		Tags:        []string{},
	}
	created, _ := repo.Create(ctx, word)

	tests := []struct {
		name    string
		id      int64
		wantErr bool
	}{
		{
			name:    "delete existing word",
			id:      created.ID,
			wantErr: false,
		},
		{
			name:    "delete non-existent word",
			id:      9999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Delete(ctx, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSQLiteRepository_GetRandom(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	// Test with no words
	_, err := repo.GetRandom(ctx)
	if err == nil {
		t.Error("GetRandom() should return error when no words exist")
	}

	// Create test words
	words := []*models.Word{
		{Word: "ephemeral", Source: "Book", DateLearned: "2024-01-15", Tags: []string{}},
		{Word: "ubiquitous", Source: "Article", DateLearned: "2024-02-20", Tags: []string{}},
	}
	for _, w := range words {
		repo.Create(ctx, w)
	}

	// Test with words
	got, err := repo.GetRandom(ctx)
	if err != nil {
		t.Errorf("GetRandom() error = %v", err)
	}
	if got == nil || got.Word == "" {
		t.Error("GetRandom() returned empty word")
	}
}

func TestSQLiteRepository_Count(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	// Create test words
	words := []*models.Word{
		{Word: "ephemeral", Source: "Book", DateLearned: "2024-01-15", Tags: []string{"literature"}},
		{Word: "ubiquitous", Source: "Article", DateLearned: "2024-02-20", Tags: []string{"technology"}},
		{Word: "eloquent", Source: "Book", DateLearned: "2024-03-10", Tags: []string{"literature"}},
	}
	for _, w := range words {
		repo.Create(ctx, w)
	}

	tests := []struct {
		name   string
		filter models.WordFilter
		want   int64
	}{
		{
			name:   "count all",
			filter: models.WordFilter{},
			want:   3,
		},
		{
			name:   "count filtered",
			filter: models.WordFilter{Source: "Book"},
			want:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.Count(ctx, tt.filter)
			if err != nil {
				t.Errorf("Count() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Count() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to create string pointer
func strPtr(s string) *string {
	return &s
}
