package api

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/lehmann314159/vocabulator/internal/models"
	"github.com/lehmann314159/vocabulator/internal/services"
)

// WebHandler handles HTML template rendering
type WebHandler struct {
	wordSvc   *services.WordService
	templates map[string]*template.Template
	partials  *template.Template
}

// NewWebHandler creates a new WebHandler with parsed templates
func NewWebHandler(wordSvc *services.WordService, templatesPath string) (*WebHandler, error) {
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"subtract": func(a, b int) int {
			return a - b
		},
		"deref": func(s *string) string {
			if s == nil {
				return ""
			}
			return *s
		},
	}

	// Parse layout template first
	layoutPath := templatesPath + "/layout.html"
	layoutTmpl, err := template.New("layout.html").Funcs(funcMap).ParseFiles(layoutPath)
	if err != nil {
		return nil, err
	}

	// Define page templates that use the layout
	pageTemplates := []string{
		"index.html",
		"word_form.html",
		"word_detail.html",
		"random.html",
		"import.html",
		"settings.html",
	}

	templates := make(map[string]*template.Template)

	for _, page := range pageTemplates {
		// Clone the layout template for each page
		tmpl, err := layoutTmpl.Clone()
		if err != nil {
			return nil, err
		}
		// Parse the page template into the cloned layout
		tmpl, err = tmpl.ParseFiles(templatesPath + "/" + page)
		if err != nil {
			return nil, err
		}
		templates[page] = tmpl
	}

	// Parse partials (templates without layout)
	partials, err := template.New("").Funcs(funcMap).ParseFiles(
		templatesPath+"/definition.html",
		templatesPath+"/import_result.html",
	)
	if err != nil {
		return nil, err
	}

	return &WebHandler{
		wordSvc:   wordSvc,
		templates: templates,
		partials:  partials,
	}, nil
}

// IndexData contains data for the index page
type IndexData struct {
	Title      string
	Words      []*models.Word
	TotalWords int64
	Page       int
	TotalPages int
	Search     string
}

// Index handles the home page / word list
func (h *WebHandler) Index(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit := 20
	offset := (page - 1) * limit

	search := r.URL.Query().Get("search")

	filter := models.WordFilter{
		Limit:  limit,
		Offset: offset,
		Search: search,
	}

	words, err := h.wordSvc.List(r.Context(), filter)
	if err != nil {
		h.renderError(w, "Failed to load words", http.StatusInternalServerError)
		return
	}

	total, err := h.wordSvc.Count(r.Context(), filter)
	if err != nil {
		total = 0
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	data := IndexData{
		Words:      words,
		TotalWords: total,
		Page:       page,
		TotalPages: totalPages,
		Search:     search,
	}

	h.render(w, "index.html", data)
}

// WordFormData contains data for the word form
type WordFormData struct {
	Title      string
	Word       *models.Word
	TagsString string
}

// NewWordForm shows the form to add a new word
func (h *WebHandler) NewWordForm(w http.ResponseWriter, r *http.Request) {
	today := time.Now().Format("2006-01-02")
	data := WordFormData{
		Title: "Add Word",
		Word: &models.Word{
			DateLearned: today,
		},
	}
	h.render(w, "word_form.html", data)
}

// CreateWord handles creating a new word from the form
func (h *WebHandler) CreateWord(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderError(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	tags := parseTags(r.FormValue("tags"))

	req := models.CreateWordRequest{
		Word:        r.FormValue("word"),
		Source:      r.FormValue("source"),
		DateLearned: r.FormValue("date_learned"),
		Tags:        tags,
	}

	if pos := r.FormValue("part_of_speech"); pos != "" {
		req.PartOfSpeech = &pos
	}
	if ex := r.FormValue("example_sentence"); ex != "" {
		req.ExampleSentence = &ex
	}

	_, err := h.wordSvc.Create(r.Context(), &req)
	if err != nil {
		h.renderError(w, "Failed to create word: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// WordDetailData contains data for the word detail page
type WordDetailData struct {
	Title string
	Word  *models.Word
}

// ShowWord displays a single word
func (h *WebHandler) ShowWord(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		h.renderError(w, "Invalid word ID", http.StatusBadRequest)
		return
	}

	word, err := h.wordSvc.GetByID(r.Context(), id)
	if err != nil {
		h.renderError(w, "Word not found", http.StatusNotFound)
		return
	}

	data := WordDetailData{
		Title: word.Word,
		Word:  word,
	}
	h.render(w, "word_detail.html", data)
}

// EditWordForm shows the form to edit a word
func (h *WebHandler) EditWordForm(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		h.renderError(w, "Invalid word ID", http.StatusBadRequest)
		return
	}

	word, err := h.wordSvc.GetByID(r.Context(), id)
	if err != nil {
		h.renderError(w, "Word not found", http.StatusNotFound)
		return
	}

	data := WordFormData{
		Title:      "Edit Word",
		Word:       word,
		TagsString: strings.Join(word.Tags, ", "),
	}
	h.render(w, "word_form.html", data)
}

// UpdateWord handles updating a word from the form
func (h *WebHandler) UpdateWord(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		h.renderError(w, "Invalid word ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	tags := parseTags(r.FormValue("tags"))

	source := r.FormValue("source")
	dateLearned := r.FormValue("date_learned")
	partOfSpeech := r.FormValue("part_of_speech")
	exampleSentence := r.FormValue("example_sentence")

	req := models.UpdateWordRequest{
		Source:          &source,
		DateLearned:     &dateLearned,
		PartOfSpeech:    &partOfSpeech,
		ExampleSentence: &exampleSentence,
		Tags:            tags,
	}

	_, err = h.wordSvc.Update(r.Context(), id, &req)
	if err != nil {
		h.renderError(w, "Failed to update word: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// DeleteWord handles deleting a word
func (h *WebHandler) DeleteWord(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		h.renderError(w, "Invalid word ID", http.StatusBadRequest)
		return
	}

	err = h.wordSvc.Delete(r.Context(), id)
	if err != nil {
		h.renderError(w, "Failed to delete word", http.StatusInternalServerError)
		return
	}

	// Return empty response for HTMX to remove the row
	w.WriteHeader(http.StatusOK)
}

// RandomData contains data for the random word page
type RandomData struct {
	Title string
	Word  *models.Word
}

// Random shows a random word
func (h *WebHandler) Random(w http.ResponseWriter, r *http.Request) {
	word, err := h.wordSvc.GetRandom(r.Context())
	if err != nil {
		// No words available
		data := RandomData{Title: "Random Word", Word: nil}
		h.render(w, "random.html", data)
		return
	}

	data := RandomData{
		Title: "Random Word",
		Word:  word,
	}
	h.render(w, "random.html", data)
}

// DefinitionData contains data for the definition partial
type DefinitionData struct {
	Definition *models.DictionaryResponse
	Error      string
}

// GetDefinition fetches and displays a word definition
func (h *WebHandler) GetDefinition(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		h.renderPartial(w, "definition.html", DefinitionData{Error: "Invalid word ID"})
		return
	}

	def, err := h.wordSvc.GetDefinition(r.Context(), id)
	if err != nil {
		h.renderPartial(w, "definition.html", DefinitionData{Error: "Definition not found"})
		return
	}

	h.renderPartial(w, "definition.html", DefinitionData{Definition: def})
}

// ImportData contains data for the import page
type ImportData struct {
	Title string
}

// ImportPage shows the CSV import form
func (h *WebHandler) ImportPage(w http.ResponseWriter, r *http.Request) {
	data := ImportData{Title: "Import Words"}
	h.render(w, "import.html", data)
}

// ImportResultData contains data for the import result
type ImportResultData struct {
	Imported int
	Skipped  int
	Errors   []string
	Error    string
}

// HandleImport processes a CSV file upload
func (h *WebHandler) HandleImport(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		h.renderPartial(w, "import_result.html", ImportResultData{Error: "No file uploaded"})
		return
	}
	defer file.Close()

	result, err := h.wordSvc.ImportCSV(r.Context(), file)
	if err != nil {
		h.renderPartial(w, "import_result.html", ImportResultData{Error: err.Error()})
		return
	}

	h.renderPartial(w, "import_result.html", ImportResultData{
		Imported: result.Imported,
		Skipped:  result.Skipped,
		Errors:   result.Errors,
	})
}

// SettingsData contains data for the settings page
type SettingsData struct {
	Title string
}

// Settings shows the settings page
func (h *WebHandler) Settings(w http.ResponseWriter, r *http.Request) {
	data := SettingsData{Title: "Settings"}
	h.render(w, "settings.html", data)
}

// render renders a full page with layout
func (h *WebHandler) render(w http.ResponseWriter, content string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	tmpl, ok := h.templates[content]
	if !ok {
		http.Error(w, "Template not found: "+content, http.StatusInternalServerError)
		return
	}

	// Execute the layout template (which includes the content)
	err := tmpl.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// renderPartial renders just a partial template (for HTMX)
func (h *WebHandler) renderPartial(w http.ResponseWriter, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	err := h.partials.ExecuteTemplate(w, name, data)
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// renderError renders an error page
func (h *WebHandler) renderError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte("<html><body><h1>Error</h1><p>" + message + "</p><a href='/'>Back to home</a></body></html>"))
}

// parseTags splits a comma-separated tag string into a slice
func parseTags(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			tags = append(tags, p)
		}
	}
	return tags
}
