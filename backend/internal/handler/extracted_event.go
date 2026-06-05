package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/na0chan-go/otayori-calendar/backend/internal/model"
	calendar "google.golang.org/api/calendar/v3"
	"gorm.io/gorm"
)

type updateExtractedEventRequest struct {
	Title       *string `json:"title"`
	EventDate   *string `json:"event_date"`
	StartTime   *string `json:"start_time"`
	EndTime     *string `json:"end_time"`
	IsAllDay    *bool   `json:"is_all_day"`
	Location    *string `json:"location"`
	Description *string `json:"description"`
}

func (s *Server) listExtractedEvents(c echo.Context) error {
	userID, err := s.currentUserID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not logged in")
	}

	var events []model.ExtractedEvent
	if err := s.db.WithContext(c.Request().Context()).
		Joins("JOIN letters ON letters.id = extracted_events.letter_id").
		Where("letters.user_id = ?", userID).
		Order("extracted_events.event_date ASC, extracted_events.created_at DESC").
		Find(&events).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load extracted events")
	}

	return c.JSON(http.StatusOK, map[string]any{"events": events})
}

func (s *Server) updateExtractedEvent(c echo.Context) error {
	userID, err := s.currentUserID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not logged in")
	}

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "extracted event not found")
	}

	var req updateExtractedEventRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	event, err := s.loadOwnedExtractedEvent(c, userID, eventID)
	if err != nil {
		return err
	}

	if err := applyExtractedEventUpdate(&event, req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := s.db.WithContext(c.Request().Context()).Save(&event).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update extracted event")
	}

	return c.JSON(http.StatusOK, map[string]any{"event": event})
}

func (s *Server) ignoreExtractedEvent(c echo.Context) error {
	userID, err := s.currentUserID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not logged in")
	}

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "extracted event not found")
	}

	event, err := s.loadOwnedExtractedEvent(c, userID, eventID)
	if err != nil {
		return err
	}

	event.Status = model.ExtractedEventStatusIgnored
	if err := s.db.WithContext(c.Request().Context()).Save(&event).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to ignore extracted event")
	}

	return c.JSON(http.StatusOK, map[string]any{"event": event})
}

func (s *Server) registerExtractedEvent(c echo.Context) error {
	userID, err := s.currentUserID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not logged in")
	}

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "extracted event not found")
	}

	event, err := s.loadOwnedExtractedEvent(c, userID, eventID)
	if err != nil {
		return err
	}

	if event.GoogleCalendarEventID != "" && event.Status != model.ExtractedEventStatusDeleted {
		return echo.NewHTTPError(http.StatusConflict, "event is already registered")
	}
	if event.Status != model.ExtractedEventStatusConfirmed &&
		event.Status != model.ExtractedEventStatusFailed &&
		event.Status != model.ExtractedEventStatusDeleted {
		return echo.NewHTTPError(http.StatusConflict, "only confirmed, failed, or deleted events can be registered")
	}

	googleEvent, err := s.buildGoogleEventFromExtractedEvent(event)
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
		event.Status = model.ExtractedEventStatusFailed
		_ = s.db.WithContext(c.Request().Context()).Save(&event).Error
		return echo.NewHTTPError(http.StatusBadGateway, "failed to create google calendar event")
	}
	if createdEvent.Id == "" {
		event.Status = model.ExtractedEventStatusFailed
		_ = s.db.WithContext(c.Request().Context()).Save(&event).Error
		return echo.NewHTTPError(http.StatusBadGateway, "google calendar event id is missing")
	}

	event.GoogleCalendarEventID = createdEvent.Id
	event.Status = model.ExtractedEventStatusRegistered
	if err := s.db.WithContext(c.Request().Context()).Save(&event).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to save extracted event")
	}

	return c.JSON(http.StatusOK, map[string]any{"event": event})
}

func (s *Server) loadOwnedExtractedEvent(c echo.Context, userID uuid.UUID, eventID uuid.UUID) (model.ExtractedEvent, error) {
	var event model.ExtractedEvent
	if err := s.db.WithContext(c.Request().Context()).
		Joins("JOIN letters ON letters.id = extracted_events.letter_id").
		Where("extracted_events.id = ? AND letters.user_id = ?", eventID, userID).
		First(&event).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.ExtractedEvent{}, echo.NewHTTPError(http.StatusNotFound, "extracted event not found")
		}
		return model.ExtractedEvent{}, echo.NewHTTPError(http.StatusInternalServerError, "failed to load extracted event")
	}
	return event, nil
}

func (s *Server) buildGoogleEventFromExtractedEvent(event model.ExtractedEvent) (*calendar.Event, error) {
	googleEvent := &calendar.Event{
		Summary:     event.Title,
		Location:    event.Location,
		Description: event.Description,
	}

	timeZone := s.cfg.DefaultTimeZone
	if event.IsAllDay {
		date := event.EventDate.Format("2006-01-02")
		endDate := event.EventDate.AddDate(0, 0, 1).Format("2006-01-02")
		googleEvent.Start = &calendar.EventDateTime{Date: date, TimeZone: timeZone}
		googleEvent.End = &calendar.EventDateTime{Date: endDate, TimeZone: timeZone}
		return googleEvent, nil
	}

	if event.StartTime == nil || event.EndTime == nil {
		return nil, errors.New("start_time and end_time are required when is_all_day is false")
	}

	startHour, startMinute := datatypesClockParts(*event.StartTime)
	endHour, endMinute := datatypesClockParts(*event.EndTime)
	startAt := time.Date(event.EventDate.Year(), event.EventDate.Month(), event.EventDate.Day(), startHour, startMinute, 0, 0, time.Local)
	endAt := time.Date(event.EventDate.Year(), event.EventDate.Month(), event.EventDate.Day(), endHour, endMinute, 0, 0, time.Local)
	if !endAt.After(startAt) {
		return nil, errors.New("end_time must be after start_time")
	}

	googleEvent.Start = &calendar.EventDateTime{DateTime: startAt.Format(time.RFC3339), TimeZone: timeZone}
	googleEvent.End = &calendar.EventDateTime{DateTime: endAt.Format(time.RFC3339), TimeZone: timeZone}
	return googleEvent, nil
}

func applyExtractedEventUpdate(event *model.ExtractedEvent, req updateExtractedEventRequest) error {
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return errors.New("title is required")
		}
		event.Title = title
	}

	if req.EventDate != nil {
		eventDate, err := time.Parse("2006-01-02", strings.TrimSpace(*req.EventDate))
		if err != nil {
			return errors.New("event_date must be YYYY-MM-DD")
		}
		event.EventDate = eventDate
	}

	if req.StartTime != nil {
		startTime, err := normalizeOptionalClock(req.StartTime, "start_time")
		if err != nil {
			return err
		}
		event.StartTime = startTime
	}
	if req.EndTime != nil {
		endTime, err := normalizeOptionalClock(req.EndTime, "end_time")
		if err != nil {
			return err
		}
		event.EndTime = endTime
	}
	if req.IsAllDay != nil {
		event.IsAllDay = *req.IsAllDay
	}

	if event.IsAllDay {
		event.StartTime = nil
		event.EndTime = nil
	} else {
		if event.StartTime == nil || event.EndTime == nil {
			return errors.New("start_time and end_time are required when is_all_day is false")
		}
		if *event.EndTime <= *event.StartTime {
			return errors.New("end_time must be after start_time")
		}
	}

	if req.Location != nil {
		event.Location = strings.TrimSpace(*req.Location)
	}
	if req.Description != nil {
		event.Description = strings.TrimSpace(*req.Description)
	}

	if event.Status != model.ExtractedEventStatusRegistered && event.Status != model.ExtractedEventStatusDeleted {
		event.Status = model.ExtractedEventStatusConfirmed
	}
	return nil
}
