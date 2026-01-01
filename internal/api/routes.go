package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates and configures the Chi router
func NewRouter(h *Handler, apiToken string) *chi.Mux {
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
		r.Use(BearerAuth(apiToken))

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

// NewWebRouter creates a router with both API and web routes
func NewWebRouter(h *Handler, wh *WebHandler, apiToken string, staticPath string) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(Recoverer)
	r.Use(Logger)
	r.Use(CORS)

	// Health check endpoint
	r.Get("/health", h.HealthCheck)

	// Static files
	fileServer := http.FileServer(http.Dir(staticPath))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Web routes (HTML pages)
	r.Get("/", wh.Index)
	r.Get("/words/new", wh.NewWordForm)
	r.Post("/words", wh.CreateWord)
	r.Get("/words/{id}", wh.ShowWord)
	r.Get("/words/{id}/edit", wh.EditWordForm)
	r.Put("/words/{id}", wh.UpdateWord)
	r.Delete("/words/{id}", wh.DeleteWord)
	r.Get("/words/{id}/definition", wh.GetDefinition)
	r.Get("/random", wh.Random)
	r.Get("/import", wh.ImportPage)
	r.Post("/import", wh.HandleImport)
	r.Get("/settings", wh.Settings)

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(JSONContentType)
		r.Use(BearerAuth(apiToken))

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
