package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDictionaryService_Lookup(t *testing.T) {
	tests := []struct {
		name           string
		word           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
		wantWord       string
	}{
		{
			name: "successful lookup",
			word: "hello",
			mockResponse: `[{
				"word": "hello",
				"phonetic": "/həˈloʊ/",
				"phonetics": [{"text": "/həˈloʊ/", "audio": "https://example.com/hello.mp3"}],
				"meanings": [{
					"partOfSpeech": "exclamation",
					"definitions": [{"definition": "used as a greeting"}]
				}],
				"sourceUrl": "https://example.com"
			}]`,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			wantWord:       "hello",
		},
		{
			name:           "word not found",
			word:           "xyzabc123",
			mockResponse:   `{"title":"No Definitions Found"}`,
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "empty response",
			word:           "test",
			mockResponse:   `[]`,
			mockStatusCode: http.StatusOK,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			// Create service with mock server
			svc := NewDictionaryServiceWithClient(server.Client(), server.URL)

			got, err := svc.Lookup(context.Background(), tt.word)
			if (err != nil) != tt.wantErr {
				t.Errorf("Lookup() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got.Word != tt.wantWord {
					t.Errorf("Lookup() word = %v, want %v", got.Word, tt.wantWord)
				}
				if len(got.Meanings) == 0 {
					t.Error("Lookup() returned no meanings")
				}
			}
		})
	}
}

func TestDictionaryService_GetFirstPartOfSpeech(t *testing.T) {
	mockResponse := `[{
		"word": "run",
		"meanings": [
			{"partOfSpeech": "verb", "definitions": [{"definition": "to move quickly"}]},
			{"partOfSpeech": "noun", "definitions": [{"definition": "an act of running"}]}
		]
	}]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	svc := NewDictionaryServiceWithClient(server.Client(), server.URL)

	got, err := svc.GetFirstPartOfSpeech(context.Background(), "run")
	if err != nil {
		t.Errorf("GetFirstPartOfSpeech() error = %v", err)
		return
	}

	if got != "verb" {
		t.Errorf("GetFirstPartOfSpeech() = %v, want verb", got)
	}
}

func TestDictionaryService_NewService(t *testing.T) {
	svc := NewDictionaryService()

	if svc == nil {
		t.Error("NewDictionaryService() returned nil")
	}

	if svc.baseURL != dictionaryAPIBaseURL {
		t.Errorf("NewDictionaryService() baseURL = %v, want %v", svc.baseURL, dictionaryAPIBaseURL)
	}
}
