package handler

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/na0chan-go/otayori-calendar/backend/internal/model"
	"golang.org/x/oauth2"
	calendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"gorm.io/datatypes"
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

type calendarEventResponse struct {
	ID                    uuid.UUID  `json:"id"`
	SourceType            string     `json:"source_type"`
	Title                 string     `json:"title"`
	EventDate             time.Time  `json:"event_date"`
	StartAt               *time.Time `json:"start_at"`
	EndAt                 *time.Time `json:"end_at"`
	IsAllDay              bool       `json:"is_all_day"`
	Location              string     `json:"location"`
	Description           string     `json:"description"`
	TimeZone              string     `json:"time_zone"`
	GoogleCalendarEventID string     `json:"google_calendar_event_id"`
	Status                string     `json:"status"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

type googleCalendarEventGetter interface {
	Get(calendarID string, eventID string) *calendar.EventsGetCall
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

func (s *Server) retryManualEvent(c echo.Context) error {
	userID, err := s.currentUserID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not logged in")
	}

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "manual event not found")
	}

	var manualEvent model.ManualEvent
	if err := s.db.WithContext(c.Request().Context()).
		First(&manualEvent, "id = ? AND user_id = ?", eventID, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "manual event not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load manual event")
	}

	if manualEvent.GoogleCalendarEventID != "" && manualEvent.Status != model.ManualEventStatusDeleted {
		return echo.NewHTTPError(http.StatusConflict, "event is already registered")
	}
	if manualEvent.Status != model.ManualEventStatusFailed && manualEvent.Status != model.ManualEventStatusDeleted {
		return echo.NewHTTPError(http.StatusConflict, "only failed or deleted events can be retried")
	}

	googleEvent, err := s.buildGoogleEventFromManualEvent(manualEvent)
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
		_ = s.db.WithContext(c.Request().Context()).Save(&manualEvent).Error
		return echo.NewHTTPError(http.StatusBadGateway, "failed to create google calendar event")
	}
	if createdEvent.Id == "" {
		manualEvent.Status = model.ManualEventStatusFailed
		_ = s.db.WithContext(c.Request().Context()).Save(&manualEvent).Error
		return echo.NewHTTPError(http.StatusBadGateway, "google calendar event id is missing")
	}

	manualEvent.GoogleCalendarEventID = createdEvent.Id
	manualEvent.Status = model.ManualEventStatusRegistered
	if err := s.db.WithContext(c.Request().Context()).Save(&manualEvent).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to save manual event")
	}

	return c.JSON(http.StatusOK, map[string]any{"event": manualEvent})
}

func (s *Server) listCalendarEvents(c echo.Context) error {
	userID, err := s.currentUserID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not logged in")
	}

	var manualEvents []model.ManualEvent
	if err := s.db.WithContext(c.Request().Context()).
		Where("user_id = ? AND status IN ?", userID, []string{
			model.ManualEventStatusRegistered,
			model.ManualEventStatusFailed,
			model.ManualEventStatusDeleted,
		}).
		Order("event_date ASC, created_at DESC").
		Find(&manualEvents).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load calendar events")
	}

	var extractedEvents []model.ExtractedEvent
	if err := s.db.WithContext(c.Request().Context()).
		Joins("JOIN letters ON letters.id = extracted_events.letter_id").
		Where("letters.user_id = ? AND extracted_events.status IN ?", userID, []string{
			model.ExtractedEventStatusRegistered,
			model.ExtractedEventStatusFailed,
			model.ExtractedEventStatusDeleted,
		}).
		Order("extracted_events.event_date ASC, extracted_events.created_at DESC").
		Find(&extractedEvents).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load calendar events")
	}

	_ = s.syncCalendarEventExistence(c.Request().Context(), userID, manualEvents, extractedEvents)

	events := make([]calendarEventResponse, 0, len(manualEvents)+len(extractedEvents))
	for _, event := range manualEvents {
		events = append(events, newManualCalendarEventResponse(event))
	}
	for _, event := range extractedEvents {
		events = append(events, newExtractedCalendarEventResponse(event, s.cfg.DefaultTimeZone))
	}
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].EventDate.Equal(events[j].EventDate) {
			return events[i].CreatedAt.After(events[j].CreatedAt)
		}
		return events[i].EventDate.Before(events[j].EventDate)
	})

	return c.JSON(http.StatusOK, map[string]any{"events": events})
}

func (s *Server) syncCalendarEventExistence(ctx context.Context, userID uuid.UUID, manualEvents []model.ManualEvent, extractedEvents []model.ExtractedEvent) error {
	if !hasRegisteredCalendarEvent(manualEvents, extractedEvents) {
		return nil
	}

	googleToken, err := s.loadGoogleToken(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	calendarEvents, err := s.googleCalendarEventsGetter(ctx, userID, googleToken)
	if err != nil {
		return err
	}

	for index := range manualEvents {
		if manualEvents[index].Status != model.ManualEventStatusRegistered || manualEvents[index].GoogleCalendarEventID == "" {
			continue
		}
		exists, err := s.googleCalendarEventExists(calendarEvents, manualEvents[index].GoogleCalendarEventID)
		if err != nil {
			return err
		}
		if !exists {
			manualEvents[index].Status = model.ManualEventStatusDeleted
			if err := s.db.WithContext(ctx).Model(&manualEvents[index]).Update("status", model.ManualEventStatusDeleted).Error; err != nil {
				return err
			}
		}
	}

	for index := range extractedEvents {
		if extractedEvents[index].Status == model.ExtractedEventStatusDeleted || extractedEvents[index].GoogleCalendarEventID == "" {
			continue
		}
		exists, err := s.googleCalendarEventExists(calendarEvents, extractedEvents[index].GoogleCalendarEventID)
		if err != nil {
			return err
		}
		nextStatus := syncedExtractedEventStatus(exists)
		if extractedEvents[index].Status != nextStatus {
			extractedEvents[index].Status = nextStatus
			if err := s.db.WithContext(ctx).Model(&extractedEvents[index]).Update("status", nextStatus).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

func hasRegisteredCalendarEvent(manualEvents []model.ManualEvent, extractedEvents []model.ExtractedEvent) bool {
	for _, event := range manualEvents {
		if event.Status == model.ManualEventStatusRegistered && event.GoogleCalendarEventID != "" {
			return true
		}
	}
	for _, event := range extractedEvents {
		if event.Status != model.ExtractedEventStatusDeleted && event.GoogleCalendarEventID != "" {
			return true
		}
	}
	return false
}

func syncedExtractedEventStatus(googleCalendarEventExists bool) string {
	if googleCalendarEventExists {
		return model.ExtractedEventStatusRegistered
	}
	return model.ExtractedEventStatusDeleted
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

func (s *Server) buildGoogleEventFromManualEvent(manualEvent model.ManualEvent) (*calendar.Event, error) {
	googleEvent := &calendar.Event{
		Summary:     manualEvent.Title,
		Location:    manualEvent.Location,
		Description: manualEvent.Description,
	}

	timeZone := strings.TrimSpace(manualEvent.TimeZone)
	if timeZone == "" {
		timeZone = s.cfg.DefaultTimeZone
	}

	if manualEvent.IsAllDay {
		date := manualEvent.EventDate.Format("2006-01-02")
		endDate := manualEvent.EventDate.AddDate(0, 0, 1).Format("2006-01-02")
		googleEvent.Start = &calendar.EventDateTime{Date: date, TimeZone: timeZone}
		googleEvent.End = &calendar.EventDateTime{Date: endDate, TimeZone: timeZone}
		return googleEvent, nil
	}

	if manualEvent.StartAt == nil || manualEvent.EndAt == nil {
		return nil, errors.New("start_at and end_at are required when is_all_day is false")
	}
	if !manualEvent.EndAt.After(*manualEvent.StartAt) {
		return nil, errors.New("end_at must be after start_at")
	}

	googleEvent.Start = &calendar.EventDateTime{DateTime: manualEvent.StartAt.Format(time.RFC3339), TimeZone: timeZone}
	googleEvent.End = &calendar.EventDateTime{DateTime: manualEvent.EndAt.Format(time.RFC3339), TimeZone: timeZone}
	return googleEvent, nil
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

func (s *Server) googleCalendarEventsGetter(ctx context.Context, userID uuid.UUID, token *oauth2.Token) (googleCalendarEventGetter, error) {
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

	return service.Events, nil
}

func (s *Server) googleCalendarEventExists(events googleCalendarEventGetter, eventID string) (bool, error) {
	event, err := events.Get(s.cfg.GoogleCalendarID, eventID).Do()
	return googleCalendarEventExistsFromResult(event, err)
}

func googleCalendarEventExistsFromResult(event *calendar.Event, err error) (bool, error) {
	if err == nil {
		return event == nil || event.Status != "cancelled", nil
	}

	var googleErr *googleapi.Error
	if errors.As(err, &googleErr) && (googleErr.Code == http.StatusNotFound || googleErr.Code == http.StatusGone) {
		return false, nil
	}
	return false, err
}

func newManualCalendarEventResponse(event model.ManualEvent) calendarEventResponse {
	return calendarEventResponse{
		ID:                    event.ID,
		SourceType:            "manual",
		Title:                 event.Title,
		EventDate:             event.EventDate,
		StartAt:               event.StartAt,
		EndAt:                 event.EndAt,
		IsAllDay:              event.IsAllDay,
		Location:              event.Location,
		Description:           event.Description,
		TimeZone:              event.TimeZone,
		GoogleCalendarEventID: event.GoogleCalendarEventID,
		Status:                event.Status,
		CreatedAt:             event.CreatedAt,
		UpdatedAt:             event.UpdatedAt,
	}
}

func newExtractedCalendarEventResponse(event model.ExtractedEvent, timeZone string) calendarEventResponse {
	response := calendarEventResponse{
		ID:                    event.ID,
		SourceType:            "extracted",
		Title:                 event.Title,
		EventDate:             event.EventDate,
		IsAllDay:              event.IsAllDay,
		Location:              event.Location,
		Description:           extractedEventCalendarDescription(event),
		TimeZone:              timeZone,
		GoogleCalendarEventID: event.GoogleCalendarEventID,
		Status:                event.Status,
		CreatedAt:             event.CreatedAt,
		UpdatedAt:             event.UpdatedAt,
	}

	if event.StartTime != nil {
		startHour, startMinute := datatypesClockParts(*event.StartTime)
		startAt := time.Date(
			event.EventDate.Year(),
			event.EventDate.Month(),
			event.EventDate.Day(),
			startHour,
			startMinute,
			0,
			0,
			time.Local,
		)
		response.StartAt = &startAt
	}
	if event.EndTime != nil {
		endHour, endMinute := datatypesClockParts(*event.EndTime)
		endAt := time.Date(
			event.EventDate.Year(),
			event.EventDate.Month(),
			event.EventDate.Day(),
			endHour,
			endMinute,
			0,
			0,
			time.Local,
		)
		response.EndAt = &endAt
	}

	return response
}

func datatypesClockParts(clock datatypes.Time) (int, int) {
	duration := time.Duration(clock)
	return int(duration / time.Hour), int((duration % time.Hour) / time.Minute)
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
