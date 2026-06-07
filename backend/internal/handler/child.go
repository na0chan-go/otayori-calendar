package handler

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/na0chan-go/otayori-calendar/backend/internal/model"
	"gorm.io/gorm"
)

var childColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

type childRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

func (s *Server) listChildren(c echo.Context) error {
	userID, err := s.currentUserID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not logged in")
	}
	var children []model.Child
	if err := s.db.WithContext(c.Request().Context()).Where("user_id = ?", userID).Order("created_at ASC").Find(&children).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load children")
	}
	return c.JSON(http.StatusOK, map[string]any{"children": children})
}

func (s *Server) createChild(c echo.Context) error {
	userID, err := s.currentUserID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not logged in")
	}
	var req childRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	child, err := validatedChild(userID, req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := s.db.WithContext(c.Request().Context()).Create(&child).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to save child")
	}
	return c.JSON(http.StatusCreated, map[string]any{"child": child})
}

func (s *Server) updateChild(c echo.Context) error {
	userID, err := s.currentUserID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not logged in")
	}
	childID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "child not found")
	}
	var req childRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	next, err := validatedChild(userID, req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	var child model.Child
	if err := s.db.WithContext(c.Request().Context()).First(&child, "id = ? AND user_id = ?", childID, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "child not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load child")
	}
	child.Name = next.Name
	child.Color = next.Color
	if err := s.db.WithContext(c.Request().Context()).Save(&child).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to save child")
	}
	return c.JSON(http.StatusOK, map[string]any{"child": child})
}

func validatedChild(userID uuid.UUID, req childRequest) (model.Child, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return model.Child{}, errors.New("name is required")
	}
	color := strings.TrimSpace(req.Color)
	if color == "" {
		color = "#8fcfb0"
	}
	if !childColorPattern.MatchString(color) {
		return model.Child{}, errors.New("color must be #RRGGBB")
	}
	return model.Child{UserID: userID, Name: name, Color: strings.ToLower(color)}, nil
}

func (s *Server) loadOptionalOwnedChild(ctx context.Context, userID uuid.UUID, rawID string) (*model.Child, error) {
	if strings.TrimSpace(rawID) == "" {
		return nil, nil
	}
	childID, err := uuid.Parse(rawID)
	if err != nil {
		return nil, errors.New("child not found")
	}
	var child model.Child
	if err := s.db.WithContext(ctx).First(&child, "id = ? AND user_id = ?", childID, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("child not found")
		}
		return nil, err
	}
	return &child, nil
}

func childDescription(child *model.Child, description string) string {
	if child == nil {
		return description
	}
	if strings.TrimSpace(description) == "" {
		return "対象: " + child.Name
	}
	return "対象: " + child.Name + "\n\n" + description
}
