package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	_ "github.com/mattn/go-sqlite3"

	"github.com/lehmann314159/vocabulator/internal/models"
	"github.com/lehmann314159/vocabulator/internal/repository"
	"github.com/lehmann314159/vocabulator/internal/services"
)

func setupTestHandler(t *testing.T) (*Handler, *chi.Mux, func()) {
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
	dictSvc := services.NewDictionaryService()
	wordSvc := services.NewWordService(repo, dictSvc)
	handler := NewHandler(wordSvc)
	router := NewRouter(handler, "")

	cleanup := func() {
		db.Close()
	}

	return handler, router, cleanup
}

func TestHandler_CreateWord(t *testing.T) {
	_, router, cleanup := setupTestHandler(t)
	defer cleanup()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "valid word",
			body:       `{"word":"ephemeral","source":"Book","date_learned":"2024-01-15","tags":["literature"]}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "missing word",
			body:       `{"source":"Book","date_learned":"2024-01-15"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid JSON",
			body:       `{invalid}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/words", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("CreateWord() status = %v, want %v", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestHandler_GetWord(t *testing.T) {
	_, router, cleanup := setupTestHandler(t)
	defer cleanup()

	// Create a word first
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/words",
		bytes.NewBufferString(`{"word":"ephemeral","source":"Book","date_learned":"2024-01-15"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)

	var created models.Word
	json.NewDecoder(createRec.Body).Decode(&created)

	tests := []struct {
		name       string
		id         string
		wantStatus int
	}{
		{
			name:       "existing word",
			id:         "1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "non-existent word",
			id:         "9999",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid ID",
			id:         "abc",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/words/"+tt.id, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("GetWord() status = %v, want %v", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestHandler_ListWords(t *testing.T) {
	_, router, cleanup := setupTestHandler(t)
	defer cleanup()

	// Create test words
	words := []string{
		`{"word":"ephemeral","source":"Book","date_learned":"2024-01-15","tags":["literature"]}`,
		`{"word":"ubiquitous","source":"Article","date_learned":"2024-02-20","tags":["technology"]}`,
	}

	for _, w := range words {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/words", bytes.NewBufferString(w))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
	}

	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantCount  int
	}{
		{
			name:       "list all",
			query:      "",
			wantStatus: http.StatusOK,
			wantCount:  2,
		},
		{
			name:       "filter by source",
			query:      "?source=Book",
			wantStatus: http.StatusOK,
			wantCount:  1,
		},
		{
			name:       "filter by tag",
			query:      "?tag=literature",
			wantStatus: http.StatusOK,
			wantCount:  1,
		},
		{
			name:       "search",
			query:      "?search=eph",
			wantStatus: http.StatusOK,
			wantCount:  1,
		},
		{
			name:       "with limit",
			query:      "?limit=1",
			wantStatus: http.StatusOK,
			wantCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/words"+tt.query, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("ListWords() status = %v, want %v", rec.Code, tt.wantStatus)
			}

			var response map[string]interface{}
			json.NewDecoder(rec.Body).Decode(&response)

			words := response["words"].([]interface{})
			if len(words) != tt.wantCount {
				t.Errorf("ListWords() count = %v, want %v", len(words), tt.wantCount)
			}
		})
	}
}

func TestHandler_UpdateWord(t *testing.T) {
	_, router, cleanup := setupTestHandler(t)
	defer cleanup()

	// Create a word first
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/words",
		bytes.NewBufferString(`{"word":"ephemeral","source":"Book","date_learned":"2024-01-15"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)

	tests := []struct {
		name       string
		id         string
		body       string
		wantStatus int
	}{
		{
			name:       "valid update",
			id:         "1",
			body:       `{"source":"Updated Book"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "non-existent word",
			id:         "9999",
			body:       `{"source":"Updated"}`,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid ID",
			id:         "abc",
			body:       `{"source":"Updated"}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/api/v1/words/"+tt.id, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("UpdateWord() status = %v, want %v", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestHandler_DeleteWord(t *testing.T) {
	_, router, cleanup := setupTestHandler(t)
	defer cleanup()

	// Create a word first
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/words",
		bytes.NewBufferString(`{"word":"ephemeral","source":"Book","date_learned":"2024-01-15"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)

	tests := []struct {
		name       string
		id         string
		wantStatus int
	}{
		{
			name:       "delete existing",
			id:         "1",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "delete non-existent",
			id:         "9999",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/words/"+tt.id, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("DeleteWord() status = %v, want %v", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestHandler_GetRandomWord(t *testing.T) {
	_, router, cleanup := setupTestHandler(t)
	defer cleanup()

	// Test with no words
	req := httptest.NewRequest(http.MethodGet, "/api/v1/words/random", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("GetRandomWord() with no words status = %v, want %v", rec.Code, http.StatusNotFound)
	}

	// Create a word
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/words",
		bytes.NewBufferString(`{"word":"ephemeral","source":"Book","date_learned":"2024-01-15"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)

	// Test with words
	req = httptest.NewRequest(http.MethodGet, "/api/v1/words/random", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GetRandomWord() with words status = %v, want %v", rec.Code, http.StatusOK)
	}
}

func TestHandler_ImportWords(t *testing.T) {
	_, router, cleanup := setupTestHandler(t)
	defer cleanup()

	csvContent := `word,source,date_learned,part_of_speech,example_sentence,tags
ephemeral,Book,2024-01-15,adjective,"test example","literature,nature"
ubiquitous,Article,2024-02-20,,,"technology"`

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "words.csv")
	part.Write([]byte(csvContent))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/words/import", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("ImportWords() status = %v, want %v, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var result services.ImportResult
	json.NewDecoder(rec.Body).Decode(&result)

	if result.Imported != 2 {
		t.Errorf("ImportWords() imported = %v, want 2", result.Imported)
	}
}

func TestHandler_ExportWords(t *testing.T) {
	_, router, cleanup := setupTestHandler(t)
	defer cleanup()

	// Create a word first
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/words",
		bytes.NewBufferString(`{"word":"ephemeral","source":"Book","date_learned":"2024-01-15","tags":["literature"]}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)

	// Export
	req := httptest.NewRequest(http.MethodGet, "/api/v1/words/export", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("ExportWords() status = %v, want %v", rec.Code, http.StatusOK)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/csv" {
		t.Errorf("ExportWords() Content-Type = %v, want text/csv", contentType)
	}

	body := rec.Body.String()
	if !bytes.Contains([]byte(body), []byte("ephemeral")) {
		t.Error("ExportWords() body should contain 'ephemeral'")
	}
}

func TestHandler_HealthCheck(t *testing.T) {
	_, router, cleanup := setupTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("HealthCheck() status = %v, want %v", rec.Code, http.StatusOK)
	}
}
