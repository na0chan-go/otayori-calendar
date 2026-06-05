package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/na0chan-go/otayori-calendar/backend/internal/model"
)

func TestNewLetterResponseHidesImagePath(t *testing.T) {
	letterID := uuid.New()
	response := newLetterResponse(model.Letter{
		ID:        letterID,
		Title:     "6月のおたより",
		ImagePath: "/private/storage/letter.png",
		MimeType:  "image/png",
		FileSize:  1234,
	})

	if response.ImageURL != "/api/letters/"+letterID.String()+"/image" {
		t.Fatalf("unexpected image url: %q", response.ImageURL)
	}
	if strings.Contains(response.ImageURL, "private") {
		t.Fatal("image url should not expose storage path")
	}
}

func TestUploadLetterRequiresLogin(t *testing.T) {
	server := newTestServer()
	body, contentType := multipartBody(t, "image", "letter.png", []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/letters", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func multipartBody(t *testing.T, fieldName string, fileName string, content []byte) (*bytes.Buffer, string) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	return &body, writer.FormDataContentType()
}
