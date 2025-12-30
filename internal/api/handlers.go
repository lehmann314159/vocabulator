package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/lehmann314159/vocabulator/internal/models"
	"github.com/lehmann314159/vocabulator/internal/services"
)

// Handler contains all HTTP handlers
type Handler struct {
	wordService *services.WordService
}

// NewHandler creates a new handler
func NewHandler(wordService *services.WordService) *Handler {
	return &Handler{
		wordService: wordService,
	}
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}

// ListWords handles GET /api/words
func (h *Handler) ListWords(w http.ResponseWriter, r *http.Request) {
	filter := models.WordFilter{
		Search:   r.URL.Query().Get("search"),
		Source:   r.URL.Query().Get("source"),
		Tag:      r.URL.Query().Get("tag"),
		FromDate: r.URL.Query().Get("from_date"),
		ToDate:   r.URL.Query().Get("to_date"),
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}

	words, err := h.wordService.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list words")
		return
	}

	// Get total count for pagination
	count, _ := h.wordService.Count(r.Context(), filter)

	response := map[string]interface{}{
		"words": words,
		"total": count,
	}

	if words == nil {
		response["words"] = []interface{}{}
	}

	writeJSON(w, http.StatusOK, response)
}

// GetWord handles GET /api/words/{id}
func (h *Handler) GetWord(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid word ID")
		return
	}

	word, err := h.wordService.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "word not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get word")
		return
	}

	writeJSON(w, http.StatusOK, word)
}

// CreateWord handles POST /api/words
func (h *Handler) CreateWord(w http.ResponseWriter, r *http.Request) {
	var req models.CreateWordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	word, err := h.wordService.Create(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, word)
}

// UpdateWord handles PUT /api/words/{id}
func (h *Handler) UpdateWord(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid word ID")
		return
	}

	var req models.UpdateWordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	word, err := h.wordService.Update(r.Context(), id, &req)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "word not found")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, word)
}

// DeleteWord handles DELETE /api/words/{id}
func (h *Handler) DeleteWord(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid word ID")
		return
	}

	err = h.wordService.Delete(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "word not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete word")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetRandomWord handles GET /api/words/random
func (h *Handler) GetRandomWord(w http.ResponseWriter, r *http.Request) {
	word, err := h.wordService.GetRandom(r.Context())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "no words found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get random word")
		return
	}

	writeJSON(w, http.StatusOK, word)
}

// GetWordDefinition handles GET /api/words/{id}/definition
func (h *Handler) GetWordDefinition(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid word ID")
		return
	}

	definition, err := h.wordService.GetDefinition(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "word not found")
			return
		}
		if errors.Is(err, services.ErrWordNotFound) {
			writeError(w, http.StatusNotFound, "definition not found in dictionary")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get definition")
		return
	}

	writeJSON(w, http.StatusOK, definition)
}

// ImportWords handles POST /api/words/import
func (h *Handler) ImportWords(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to parse form")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	result, err := h.wordService.ImportCSV(r.Context(), file)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// ExportWords handles GET /api/words/export
func (h *Handler) ExportWords(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=words.csv")

	err := h.wordService.ExportCSV(r.Context(), w)
	if err != nil {
		// Reset headers since we already set them
		w.Header().Set("Content-Type", "application/json")
		writeError(w, http.StatusInternalServerError, "failed to export words")
		return
	}
}

// HealthCheck handles GET /health
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
