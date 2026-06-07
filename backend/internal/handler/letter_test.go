package handler

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/na0chan-go/otayori-calendar/backend/internal/auth"
	"github.com/na0chan-go/otayori-calendar/backend/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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

func TestDeleteLetterRequiresLogin(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodDelete, "/api/letters/"+uuid.NewString(), nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestDeleteLetterDoesNotDeleteAnotherUsersLetter(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE letters (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		child_id TEXT,
		title TEXT,
		image_path TEXT NOT NULL,
		mime_type TEXT NOT NULL,
		file_size INTEGER NOT NULL,
		ocr_text TEXT,
		created_at DATETIME NOT NULL
	)`).Error; err != nil {
		t.Fatal(err)
	}

	ownerID := uuid.New()
	otherUserID := uuid.New()
	letter := model.Letter{
		ID:        uuid.New(),
		UserID:    ownerID,
		ImagePath: filepath.Join(t.TempDir(), "letter.png"),
		MimeType:  "image/png",
		FileSize:  5,
	}
	if err := os.WriteFile(letter.ImagePath, []byte("image"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&letter).Error; err != nil {
		t.Fatal(err)
	}

	server := newTestServer()
	server.db = db
	req := httptest.NewRequest(http.MethodDelete, "/api/letters/"+letter.ID.String(), nil)
	req.AddCookie(&http.Cookie{
		Name:  auth.SessionCookieName,
		Value: server.sessions.Sign(otherUserID, time.Now()),
	})
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, rec.Code)
	}
	if _, err := os.Stat(letter.ImagePath); err != nil {
		t.Fatalf("another user's image must remain: %v", err)
	}
	var count int64
	if err := db.Model(&model.Letter{}).Where("id = ?", letter.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("another user's letter must remain, got count %d", count)
	}
}

func TestQuarantineRestoreAndRemoveLetterImage(t *testing.T) {
	imagePath := filepath.Join(t.TempDir(), "letter.png")
	if err := os.WriteFile(imagePath, []byte("image"), 0o600); err != nil {
		t.Fatal(err)
	}

	quarantinePath, err := quarantineLetterImage(imagePath)
	if err != nil {
		t.Fatal(err)
	}
	if quarantinePath == "" {
		t.Fatal("expected quarantine path")
	}
	if _, err := os.Stat(imagePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected original image to be moved, got %v", err)
	}

	if err := restoreQuarantinedLetterImage(quarantinePath, imagePath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(imagePath); err != nil {
		t.Fatalf("expected restored image, got %v", err)
	}

	quarantinePath, err = quarantineLetterImage(imagePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := removeQuarantinedLetterImage(quarantinePath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(quarantinePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected quarantined image to be deleted, got %v", err)
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
