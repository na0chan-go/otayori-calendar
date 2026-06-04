package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/na0chan-go/otayori-calendar/backend/internal/model"
	"golang.org/x/oauth2"
	calendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
	"gorm.io/gorm"
)

type manualEventRequest struct {
	Title       string `json:"title"`
	EventDate   string `json:"event_date"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	IsAllDay    *bool  `json:"is_all_day"`
	Location    string `json:"location"`
	Description string `json:"description"`
	TimeZone    string `json:"time_zone"`
}

func (s *Server) createManualEvent(c echo.Context) error {
	userID, err := s.currentUserID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not logged in")
	}

	var req manualEventRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	manualEvent, googleEvent, err := s.buildManualEvent(userID, req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	googleToken, err := s.loadGoogleToken(c.Request().Context(), userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusUnauthorized, "google token not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load google token")
	}

	createdEvent, err := s.insertGoogleCalendarEvent(c.Request().Context(), userID, googleToken, googleEvent)
	if err != nil {
		manualEvent.Status = model.ManualEventStatusFailed
		_ = s.db.WithContext(c.Request().Context()).Create(&manualEvent).Error
		return echo.NewHTTPError(http.StatusBadGateway, "failed to create google calendar event")
	}
	if createdEvent.Id == "" {
		manualEvent.Status = model.ManualEventStatusFailed
		_ = s.db.WithContext(c.Request().Context()).Create(&manualEvent).Error
		return echo.NewHTTPError(http.StatusBadGateway, "google calendar event id is missing")
	}

	manualEvent.GoogleCalendarEventID = createdEvent.Id
	manualEvent.Status = model.ManualEventStatusRegistered
	if err := s.db.WithContext(c.Request().Context()).Create(&manualEvent).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to save manual event")
	}

	return c.JSON(http.StatusCreated, map[string]any{"event": manualEvent})
}

func (s *Server) buildManualEvent(userID uuid.UUID, req manualEventRequest) (model.ManualEvent, *calendar.Event, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return model.ManualEvent{}, nil, errors.New("title is required")
	}

	timeZone := strings.TrimSpace(req.TimeZone)
	if timeZone == "" {
		timeZone = s.cfg.DefaultTimeZone
	}

	location, err := time.LoadLocation(timeZone)
	if err != nil {
		return model.ManualEvent{}, nil, errors.New("time_zone is invalid")
	}

	eventDate, err := time.ParseInLocation("2006-01-02", req.EventDate, location)
	if err != nil {
		return model.ManualEvent{}, nil, errors.New("event_date must be YYYY-MM-DD")
	}

	manualEvent := model.ManualEvent{
		UserID:      userID,
		Title:       title,
		EventDate:   eventDate,
		IsAllDay:    true,
		Location:    strings.TrimSpace(req.Location),
		Description: strings.TrimSpace(req.Description),
		TimeZone:    timeZone,
	}

	googleEvent := &calendar.Event{
		Summary:     manualEvent.Title,
		Location:    manualEvent.Location,
		Description: manualEvent.Description,
	}

	if req.IsAllDay != nil {
		manualEvent.IsAllDay = *req.IsAllDay
	}

	if manualEvent.IsAllDay {
		date := eventDate.Format("2006-01-02")
		endDate := eventDate.AddDate(0, 0, 1).Format("2006-01-02")
		googleEvent.Start = &calendar.EventDateTime{Date: date, TimeZone: timeZone}
		googleEvent.End = &calendar.EventDateTime{Date: endDate, TimeZone: timeZone}
		return manualEvent, googleEvent, nil
	}

	startAt, err := parseClockTime(eventDate, req.StartTime, location)
	if err != nil {
		return model.ManualEvent{}, nil, errors.New("start_time must be HH:MM")
	}
	endAt, err := parseClockTime(eventDate, req.EndTime, location)
	if err != nil {
		return model.ManualEvent{}, nil, errors.New("end_time must be HH:MM")
	}
	if !endAt.After(startAt) {
		return model.ManualEvent{}, nil, errors.New("end_time must be after start_time")
	}

	manualEvent.StartAt = &startAt
	manualEvent.EndAt = &endAt
	googleEvent.Start = &calendar.EventDateTime{DateTime: startAt.Format(time.RFC3339), TimeZone: timeZone}
	googleEvent.End = &calendar.EventDateTime{DateTime: endAt.Format(time.RFC3339), TimeZone: timeZone}

	return manualEvent, googleEvent, nil
}

func (s *Server) loadGoogleToken(ctx context.Context, userID uuid.UUID) (*oauth2.Token, error) {
	var stored model.GoogleToken
	if err := s.db.WithContext(ctx).First(&stored, "user_id = ?", userID).Error; err != nil {
		return nil, err
	}

	accessToken, err := s.tokens.Decrypt(stored.AccessToken)
	if err != nil {
		return nil, err
	}
	refreshToken, err := s.tokens.Decrypt(stored.RefreshToken)
	if err != nil {
		return nil, err
	}

	return &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Expiry:       stored.Expiry,
		TokenType:    "Bearer",
	}, nil
}

func (s *Server) insertGoogleCalendarEvent(ctx context.Context, userID uuid.UUID, token *oauth2.Token, event *calendar.Event) (*calendar.Event, error) {
	oauthConfig, err := s.cfg.GoogleOAuthConfig()
	if err != nil {
		return nil, err
	}

	tokenSource := oauthConfig.TokenSource(ctx, token)
	freshToken, err := tokenSource.Token()
	if err != nil {
		return nil, err
	}
	if err := s.saveGoogleToken(ctx, userID, freshToken, token.RefreshToken); err != nil {
		return nil, err
	}

	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(freshToken))
	service, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	return service.Events.Insert(s.cfg.GoogleCalendarID, event).Do()
}

func (s *Server) saveGoogleToken(ctx context.Context, userID uuid.UUID, token *oauth2.Token, fallbackRefreshToken string) error {
	refreshToken := token.RefreshToken
	if refreshToken == "" {
		refreshToken = fallbackRefreshToken
	}

	accessToken, err := s.tokens.Encrypt(token.AccessToken)
	if err != nil {
		return err
	}
	encryptedRefreshToken, err := s.tokens.Encrypt(refreshToken)
	if err != nil {
		return err
	}

	return s.db.WithContext(ctx).
		Model(&model.GoogleToken{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"access_token":  accessToken,
			"refresh_token": encryptedRefreshToken,
			"expiry":        token.Expiry,
			"updated_at":    time.Now(),
		}).Error
}

func parseClockTime(eventDate time.Time, clock string, location *time.Location) (time.Time, error) {
	parsed, err := time.ParseInLocation("15:04", clock, location)
	if err != nil {
		return time.Time{}, err
	}

	return time.Date(
		eventDate.Year(),
		eventDate.Month(),
		eventDate.Day(),
		parsed.Hour(),
		parsed.Minute(),
		0,
		0,
		location,
	), nil
}
