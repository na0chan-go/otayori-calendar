package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGeminiExtractorExtractsFromOCRText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-goog-api-key") != "test-api-key" {
			t.Fatal("expected API key header")
		}

		var request geminiGenerateContentRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if len(request.Contents) != 1 || len(request.Contents[0].Parts) != 2 {
			t.Fatalf("unexpected request parts: %#v", request.Contents)
		}
		if !strings.Contains(request.Contents[0].Parts[1].Text, "身体測定") {
			t.Fatal("expected OCR text in request")
		}
		if request.GenerationConfig.ResponseMimeType != "application/json" {
			t.Fatal("expected JSON response MIME type")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"{\"events\":[]}"}]}}]}`))
	}))
	defer server.Close()

	extractor := NewGeminiExtractor("test-api-key", "gemini-test", server.Client())
	extractor.baseURL = server.URL

	output, err := extractor.Extract(context.Background(), GeminiExtractionRequest{
		OCRText:       "6月12日 身体測定",
		ReferenceYear: 2026,
	})
	if err != nil {
		t.Fatal(err)
	}
	if output != `{"events":[]}` {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestGeminiExtractorExtractsFromImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request geminiGenerateContentRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		image := request.Contents[0].Parts[1].InlineData
		if image == nil || image.MimeType != "image/png" || image.Data != "aW1hZ2U=" {
			t.Fatalf("unexpected inline image: %#v", image)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"{\"events\":[]}"}]}}]}`))
	}))
	defer server.Close()

	extractor := NewGeminiExtractor("test-api-key", "gemini-test", server.Client())
	extractor.baseURL = server.URL

	if _, err := extractor.Extract(context.Background(), GeminiExtractionRequest{
		Image:         []byte("image"),
		MimeType:      "image/png",
		ReferenceYear: 2026,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestGeminiExtractorDoesNotExposeErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "sensitive OCR text", http.StatusTooManyRequests)
	}))
	defer server.Close()

	extractor := NewGeminiExtractor("test-api-key", "gemini-test", server.Client())
	extractor.baseURL = server.URL

	_, err := extractor.Extract(context.Background(), GeminiExtractionRequest{
		OCRText:       "private text",
		ReferenceYear: 2026,
	})
	if err == nil {
		t.Fatal("expected Gemini API error")
	}
	if strings.Contains(err.Error(), "sensitive OCR text") || strings.Contains(err.Error(), "private text") {
		t.Fatalf("error must not expose sensitive input or response body: %v", err)
	}
}

func TestGeminiExtractorReturnsQuotaError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "billing details", http.StatusTooManyRequests)
	}))
	defer server.Close()

	extractor := NewGeminiExtractor("test-api-key", "gemini-test", server.Client())
	extractor.baseURL = server.URL

	_, err := extractor.Extract(context.Background(), GeminiExtractionRequest{
		OCRText:       "private text",
		ReferenceYear: 2026,
	})
	if !errors.Is(err, ErrGeminiQuotaExceeded) {
		t.Fatalf("expected quota error, got %v", err)
	}
}
