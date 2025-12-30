package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates and configures the Chi router
func NewRouter(h *Handler) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(Recoverer)
	r.Use(Logger)
	r.Use(CORS)

	// Health check endpoint
	r.Get("/health", h.HealthCheck)

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(JSONContentType)

		r.Route("/words", func(r chi.Router) {
			r.Get("/", h.ListWords)
			r.Post("/", h.CreateWord)

			// Special routes before /{id} to avoid conflicts
			r.Get("/random", h.GetRandomWord)
			r.Post("/import", h.ImportWords)
			r.Get("/export", h.ExportWords)

			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", h.GetWord)
				r.Put("/", h.UpdateWord)
				r.Delete("/", h.DeleteWord)
				r.Get("/definition", h.GetWordDefinition)
			})
		})
	})

	return r
}
