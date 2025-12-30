package services

import (
	"bytes"
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/lehmann314159/vocabulator/internal/models"
	"github.com/lehmann314159/vocabulator/internal/repository"
)

func setupTestService(t *testing.T) (*WordService, func()) {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

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

	repo := repository.NewSQLiteRepository(db)
	dictSvc := NewDictionaryService()
	svc := NewWordService(repo, dictSvc)

	cleanup := func() {
		db.Close()
	}

	return svc, cleanup
}

func TestWordService_Create(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	tests := []struct {
		name    string
		req     *models.CreateWordRequest
		wantErr bool
	}{
		{
			name: "valid word",
			req: &models.CreateWordRequest{
				Word:        "ephemeral",
				Source:      "Book",
				DateLearned: "2024-01-15",
				Tags:        []string{"literature"},
			},
			wantErr: false,
		},
		{
			name: "missing word",
			req: &models.CreateWordRequest{
				Source:      "Book",
				DateLearned: "2024-01-15",
			},
			wantErr: true,
		},
		{
			name: "missing source",
			req: &models.CreateWordRequest{
				Word:        "test",
				DateLearned: "2024-01-15",
			},
			wantErr: true,
		},
		{
			name: "missing date",
			req: &models.CreateWordRequest{
				Word:   "test",
				Source: "Book",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.Create(ctx, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.ID == 0 {
				t.Error("Create() returned word with ID = 0")
			}
		})
	}
}

func TestWordService_Create_Duplicate(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	req := &models.CreateWordRequest{
		Word:        "ephemeral",
		Source:      "Book",
		DateLearned: "2024-01-15",
	}

	// First creation should succeed
	_, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatalf("first Create() failed: %v", err)
	}

	// Second creation should fail
	_, err = svc.Create(ctx, req)
	if err == nil {
		t.Error("duplicate Create() should have failed")
	}
}

func TestWordService_Update(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Create a word
	created, _ := svc.Create(ctx, &models.CreateWordRequest{
		Word:        "ephemeral",
		Source:      "Book",
		DateLearned: "2024-01-15",
		Tags:        []string{"literature"},
	})

	// Update the word
	newSource := "Updated Book"
	updated, err := svc.Update(ctx, created.ID, &models.UpdateWordRequest{
		Source: &newSource,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if updated.Source != newSource {
		t.Errorf("Update() source = %v, want %v", updated.Source, newSource)
	}

	// Original fields should be preserved
	if updated.Word != "ephemeral" {
		t.Errorf("Update() word = %v, want ephemeral", updated.Word)
	}
}

func TestWordService_ImportCSV(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		csv          string
		wantImported int
		wantSkipped  int
		wantErr      bool
	}{
		{
			name: "valid CSV",
			csv: `word,source,date_learned,part_of_speech,example_sentence,tags
ephemeral,Book,2024-01-15,adjective,"The ephemeral beauty","literature,nature"
ubiquitous,Article,2024-02-20,adjective,,"technology"`,
			wantImported: 2,
			wantSkipped:  0,
			wantErr:      false,
		},
		{
			name: "missing required field",
			csv: `word,source,date_learned
ephemeral,,2024-01-15`,
			wantImported: 0,
			wantSkipped:  1,
			wantErr:      false,
		},
		{
			name:    "missing required column",
			csv:     `word,source`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh service for each test
			svc, cleanup := setupTestService(t)
			defer cleanup()

			result, err := svc.ImportCSV(ctx, strings.NewReader(tt.csv))
			if (err != nil) != tt.wantErr {
				t.Errorf("ImportCSV() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result.Imported != tt.wantImported {
					t.Errorf("ImportCSV() imported = %v, want %v", result.Imported, tt.wantImported)
				}
				if result.Skipped != tt.wantSkipped {
					t.Errorf("ImportCSV() skipped = %v, want %v", result.Skipped, tt.wantSkipped)
				}
			}
		})
	}
}

func TestWordService_ExportCSV(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Create test words
	pos := "adjective"
	example := "The ephemeral beauty"
	svc.Create(ctx, &models.CreateWordRequest{
		Word:            "ephemeral",
		Source:          "Book",
		DateLearned:     "2024-01-15",
		PartOfSpeech:    &pos,
		ExampleSentence: &example,
		Tags:            []string{"literature", "nature"},
	})

	svc.Create(ctx, &models.CreateWordRequest{
		Word:        "ubiquitous",
		Source:      "Article",
		DateLearned: "2024-02-20",
	})

	var buf bytes.Buffer
	err := svc.ExportCSV(ctx, &buf)
	if err != nil {
		t.Fatalf("ExportCSV() error = %v", err)
	}

	output := buf.String()

	// Check header
	if !strings.HasPrefix(output, "word,source,date_learned,part_of_speech,example_sentence,tags") {
		t.Error("ExportCSV() header is incorrect")
	}

	// Check content
	if !strings.Contains(output, "ephemeral") {
		t.Error("ExportCSV() missing ephemeral")
	}
	if !strings.Contains(output, "ubiquitous") {
		t.Error("ExportCSV() missing ubiquitous")
	}
	if !strings.Contains(output, "literature,nature") {
		t.Error("ExportCSV() missing tags")
	}
}

func TestWordService_GetRandom(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Test with no words
	_, err := svc.GetRandom(ctx)
	if err == nil {
		t.Error("GetRandom() should error when no words exist")
	}

	// Add words
	svc.Create(ctx, &models.CreateWordRequest{
		Word:        "ephemeral",
		Source:      "Book",
		DateLearned: "2024-01-15",
	})

	// Test with words
	got, err := svc.GetRandom(ctx)
	if err != nil {
		t.Errorf("GetRandom() error = %v", err)
	}
	if got == nil || got.Word == "" {
		t.Error("GetRandom() returned empty word")
	}
}
