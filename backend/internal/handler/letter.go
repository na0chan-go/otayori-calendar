package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	letterdomain "github.com/na0chan-go/otayori-calendar/backend/internal/domain/letter"
	"github.com/na0chan-go/otayori-calendar/backend/internal/model"
	letterport "github.com/na0chan-go/otayori-calendar/backend/internal/port/letter"
	letterusecase "github.com/na0chan-go/otayori-calendar/backend/internal/usecase/letter"
	"gorm.io/gorm"
)

const (
	maxLetterImageSize      = 10 << 20
	maxLetterUploadBodySize = maxLetterImageSize + (1 << 20)
)

var allowedLetterImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

type letterResponse struct {
	ID         uuid.UUID  `json:"id"`
	ChildID    *uuid.UUID `json:"child_id"`
	ChildName  string     `json:"child_name"`
	ChildColor string     `json:"child_color"`
	Title      string     `json:"title"`
	MimeType   string     `json:"mime_type"`
	FileSize   int64      `json:"file_size"`
	ImageURL   string     `json:"image_url"`
	CreatedAt  string     `json:"created_at"`
}

func (s *Server) uploadLetter(c echo.Context) error {
	userID, err := s.currentUserID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not logged in")
	}
	child, err := s.loadOptionalOwnedChild(c.Request().Context(), userID, c.FormValue("child_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	c.Request().Body = http.MaxBytesReader(c.Response(), c.Request().Body, maxLetterUploadBodySize)
	if err := c.Request().ParseMultipartForm(maxLetterImageSize); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "image must be 10MB or smaller")
		}
		return echo.NewHTTPError(http.StatusBadRequest, "invalid multipart form")
	}

	fileHeader, err := c.FormFile("image")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "image is required")
	}
	if fileHeader.Size <= 0 || fileHeader.Size > maxLetterImageSize {
		return echo.NewHTTPError(http.StatusBadRequest, "image must be 10MB or smaller")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to open image")
	}
	defer file.Close()

	head := make([]byte, 512)
	n, err := io.ReadFull(file, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to read image")
	}
	mimeType := http.DetectContentType(head[:n])
	extension, ok := allowedLetterImageTypes[mimeType]
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "image must be JPEG, PNG, or WebP")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to read image")
	}

	letterID := uuid.New()
	userDir := filepath.Join(s.cfg.LetterStorageDir, userID.String())
	if err := os.MkdirAll(userDir, 0o700); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to prepare storage")
	}

	imagePath := filepath.Join(userDir, letterID.String()+extension)
	destination, err := os.OpenFile(imagePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to save image")
	}
	defer destination.Close()

	written, err := io.Copy(destination, io.LimitReader(file, maxLetterImageSize+1))
	if err != nil {
		_ = os.Remove(imagePath)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to save image")
	}
	if written > maxLetterImageSize {
		_ = os.Remove(imagePath)
		return echo.NewHTTPError(http.StatusBadRequest, "image must be 10MB or smaller")
	}

	letter := model.Letter{
		ID:        letterID,
		UserID:    userID,
		Child:     child,
		Title:     strings.TrimSpace(c.FormValue("title")),
		ImagePath: imagePath,
		MimeType:  mimeType,
		FileSize:  written,
	}
	if child != nil {
		letter.ChildID = &child.ID
	}
	if err := s.db.WithContext(c.Request().Context()).Create(&letter).Error; err != nil {
		_ = os.Remove(imagePath)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to save letter")
	}

	return c.JSON(http.StatusCreated, map[string]any{"letter": newLetterResponse(letter)})
}

func (s *Server) listLetters(c echo.Context) error {
	userID, err := s.currentUserID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not logged in")
	}

	var letters []model.Letter
	if err := s.db.WithContext(c.Request().Context()).
		Preload("Child").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&letters).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load letters")
	}

	items := make([]letterResponse, 0, len(letters))
	for _, letter := range letters {
		items = append(items, newLetterResponse(letter))
	}

	return c.JSON(http.StatusOK, map[string]any{"letters": items})
}

func (s *Server) showLetterImage(c echo.Context) error {
	userID, err := s.currentUserID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not logged in")
	}

	letterID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "letter not found")
	}

	var letter model.Letter
	if err := s.db.WithContext(c.Request().Context()).
		First(&letter, "id = ? AND user_id = ?", letterID, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "letter not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load letter")
	}

	c.Response().Header().Set(echo.HeaderContentType, letter.MimeType)
	c.Response().Header().Set(echo.HeaderCacheControl, "private, max-age=300")
	return c.File(letter.ImagePath)
}

func (s *Server) deleteLetter(c echo.Context) error {
	userID, err := s.currentUserID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not logged in")
	}

	letterID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "letter not found")
	}

	adapter := &letterDeletionAdapter{server: s}
	err = (letterusecase.Delete{Repository: adapter, Storage: adapter}).
		Execute(c.Request().Context(), letterID, userID)
	switch {
	case errors.Is(err, letterusecase.ErrNotFound):
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	case errors.Is(err, letterusecase.ErrLoadFailed),
		errors.Is(err, letterusecase.ErrPrepareDeletionFailed),
		errors.Is(err, letterusecase.ErrDeleteFailed),
		errors.Is(err, letterusecase.ErrImageDeletionFailed):
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	case err != nil:
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

type letterDeletionAdapter struct {
	server *Server
}

func (a *letterDeletionAdapter) FindOwned(ctx context.Context, letterID, userID uuid.UUID) (letterdomain.Letter, error) {
	var letter model.Letter
	if err := a.server.db.WithContext(ctx).First(&letter, "id = ? AND user_id = ?", letterID, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return letterdomain.Letter{}, letterport.ErrNotFound
		}
		return letterdomain.Letter{}, err
	}
	return letterDomain(letter), nil
}

func (a *letterDeletionAdapter) Delete(ctx context.Context, letter letterdomain.Letter) error {
	return a.server.db.WithContext(ctx).Delete(&model.Letter{ID: letter.ID}).Error
}

func (a *letterDeletionAdapter) Quarantine(_ context.Context, imagePath string) (string, error) {
	return quarantineLetterImage(imagePath)
}

func (a *letterDeletionAdapter) Restore(_ context.Context, quarantinePath, imagePath string) error {
	return restoreQuarantinedLetterImage(quarantinePath, imagePath)
}

func (a *letterDeletionAdapter) Remove(_ context.Context, quarantinePath string) error {
	return removeQuarantinedLetterImage(quarantinePath)
}

func letterDomain(letter model.Letter) letterdomain.Letter {
	return letterdomain.Letter{ID: letter.ID, UserID: letter.UserID, ImagePath: letter.ImagePath}
}

func quarantineLetterImage(imagePath string) (string, error) {
	quarantinePath := imagePath + ".deleting-" + uuid.NewString()
	if err := os.Rename(imagePath, quarantinePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return quarantinePath, nil
}

func restoreQuarantinedLetterImage(quarantinePath, imagePath string) error {
	if quarantinePath == "" {
		return nil
	}
	return os.Rename(quarantinePath, imagePath)
}

func removeQuarantinedLetterImage(quarantinePath string) error {
	if quarantinePath == "" {
		return nil
	}
	return os.Remove(quarantinePath)
}

func newLetterResponse(letter model.Letter) letterResponse {
	response := letterResponse{
		ID:        letter.ID,
		ChildID:   letter.ChildID,
		Title:     letter.Title,
		MimeType:  letter.MimeType,
		FileSize:  letter.FileSize,
		ImageURL:  fmt.Sprintf("/api/letters/%s/image", letter.ID.String()),
		CreatedAt: letter.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if letter.Child != nil {
		response.ChildName = letter.Child.Name
		response.ChildColor = letter.Child.Color
	}
	return response
}
