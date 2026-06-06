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
	"gorm.io/gorm"
)

type updateExtractedEventRequest struct {
	Title              *string `json:"title"`
	EventDate          *string `json:"event_date"`
	StartTime          *string `json:"start_time"`
	EndTime            *string `json:"end_time"`
	IsAllDay           *bool   `json:"is_all_day"`
	Location           *string `json:"location"`
	Description        *string `json:"description"`
	Belongings         *string `json:"belongings"`
	SubmissionDeadline *string `json:"submission_deadline"`
}

type bulkExtractedEventsRequest struct {
	IDs []string `json:"ids"`
}

type restoreExtractedEventStatusRequest struct {
	ID             string `json:"id"`
	ExpectedStatus string `json:"expected_status"`
	Status         string `json:"status"`
}

type restoreExtractedEventStatusesRequest struct {
	Events []restoreExtractedEventStatusRequest `json:"events"`
}

type bulkExtractedEventsResponse struct {
	Events  []model.ExtractedEvent     `json:"events"`
	Results []bulkExtractedEventResult `json:"results"`
	Summary bulkExtractedEventsSummary `json:"summary"`
}

type bulkExtractedEventResult struct {
	ID      string                `json:"id"`
	Status  string                `json:"status"`
	Message string                `json:"message,omitempty"`
	Event   *model.ExtractedEvent `json:"event,omitempty"`
}

type bulkExtractedEventsSummary struct {
	Success int `json:"success"`
	Failed  int `json:"failed"`
}

var (
	errExtractedEventAlreadyRegistered = errors.New("event is already registered")
	errExtractedEventNotEditable       = errors.New("registered events cannot be edited")
	errExtractedEventNotRegisterable   = errors.New("only confirmed, failed, or deleted events can be registered")
	errGoogleCalendarEventIDMissing    = errors.New("google calendar event id is missing")
	errGoogleCalendarEventCreateFailed = errors.New("failed to create google calendar event")
	errSaveExtractedEventFailed        = errors.New("failed to save extracted event")
)

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

	_ = s.syncCalendarEventExistence(c.Request().Context(), userID, nil, events)

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
	if err := validateExtractedEventEditable(event); err != nil {
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}

	if err := applyExtractedEventUpdate(&event, req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := s.db.WithContext(c.Request().Context()).Save(&event).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update extracted event")
	}

	return c.JSON(http.StatusOK, map[string]any{"event": event})
}

func (s *Server) bulkConfirmExtractedEvents(c echo.Context) error {
	userID, err := s.currentUserID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not logged in")
	}

	eventIDs, err := parseBulkExtractedEventIDs(c)
	if err != nil {
		return err
	}

	eventsByID, err := s.loadOwnedExtractedEventsByID(c.Request().Context(), userID, eventIDs)
	if err != nil {
		return err
	}

	response := newBulkExtractedEventsResponse()
	for _, eventID := range eventIDs {
		event, ok := eventsByID[eventID]
		if !ok {
			response.addFailure(eventID.String(), "extracted event not found")
			continue
		}
		if event.Status == model.ExtractedEventStatusRegistered {
			response.addFailure(event.ID.String(), "registered events cannot be confirmed")
			continue
		}

		event.Status = model.ExtractedEventStatusConfirmed
		event.GoogleCalendarEventID = ""
		if err := s.db.WithContext(c.Request().Context()).Save(&event).Error; err != nil {
			response.addFailure(event.ID.String(), "failed to confirm extracted event")
			continue
		}
		response.addSuccess(event, "confirmed")
	}

	return c.JSON(http.StatusOK, response)
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
	if event.Status == model.ExtractedEventStatusRegistered {
		return echo.NewHTTPError(http.StatusConflict, "registered events cannot be ignored")
	}

	event.Status = model.ExtractedEventStatusIgnored
	event.GoogleCalendarEventID = ""
	if err := s.db.WithContext(c.Request().Context()).Save(&event).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to ignore extracted event")
	}

	return c.JSON(http.StatusOK, map[string]any{"event": event})
}

func (s *Server) bulkIgnoreExtractedEvents(c echo.Context) error {
	userID, err := s.currentUserID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not logged in")
	}

	eventIDs, err := parseBulkExtractedEventIDs(c)
	if err != nil {
		return err
	}

	eventsByID, err := s.loadOwnedExtractedEventsByID(c.Request().Context(), userID, eventIDs)
	if err != nil {
		return err
	}

	response := newBulkExtractedEventsResponse()
	for _, eventID := range eventIDs {
		event, ok := eventsByID[eventID]
		if !ok {
			response.addFailure(eventID.String(), "extracted event not found")
			continue
		}
		if event.Status == model.ExtractedEventStatusRegistered {
			response.addFailure(event.ID.String(), "registered events cannot be ignored")
			continue
		}

		event.Status = model.ExtractedEventStatusIgnored
		event.GoogleCalendarEventID = ""
		if err := s.db.WithContext(c.Request().Context()).Save(&event).Error; err != nil {
			response.addFailure(event.ID.String(), "failed to ignore extracted event")
			continue
		}
		response.addSuccess(event, "ignored")
	}

	return c.JSON(http.StatusOK, response)
}

func (s *Server) restoreExtractedEventStatuses(c echo.Context) error {
	userID, err := s.currentUserID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not logged in")
	}

	var req restoreExtractedEventStatusesRequest
	if err := c.Bind(&req); err != nil || len(req.Events) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	response := newBulkExtractedEventsResponse()
	for _, restore := range req.Events {
		eventID, err := uuid.Parse(restore.ID)
		if err != nil {
			response.addFailure(restore.ID, "extracted event not found")
			continue
		}
		event, err := s.loadOwnedExtractedEvent(c, userID, eventID)
		if err != nil {
			response.addFailure(restore.ID, "extracted event not found")
			continue
		}
		if err := validateExtractedEventStatusRestore(event.Status, restore.ExpectedStatus, restore.Status); err != nil {
			response.addFailure(restore.ID, err.Error())
			continue
		}

		event.Status = restore.Status
		if err := s.db.WithContext(c.Request().Context()).Save(&event).Error; err != nil {
			response.addFailure(restore.ID, "failed to restore extracted event")
			continue
		}
		response.addSuccess(event, "restored")
	}

	return c.JSON(http.StatusOK, response)
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

	if err := validateExtractedEventRegisterable(event); err != nil {
		return extractedEventRegisterHTTPError(err)
	}
	if _, err := s.buildGoogleEventFromExtractedEvent(event); err != nil {
		return extractedEventRegisterHTTPError(err)
	}

	googleToken, err := s.loadGoogleToken(c.Request().Context(), userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusUnauthorized, "google token not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load google token")
	}

	if err := s.registerExtractedEventRecord(c.Request().Context(), userID, googleToken, &event); err != nil {
		return extractedEventRegisterHTTPError(err)
	}

	return c.JSON(http.StatusOK, map[string]any{"event": event})
}

func (s *Server) bulkRegisterExtractedEvents(c echo.Context) error {
	userID, err := s.currentUserID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not logged in")
	}

	eventIDs, err := parseBulkExtractedEventIDs(c)
	if err != nil {
		return err
	}

	eventsByID, err := s.loadOwnedExtractedEventsByID(c.Request().Context(), userID, eventIDs)
	if err != nil {
		return err
	}

	googleToken, err := s.loadGoogleToken(c.Request().Context(), userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusUnauthorized, "google token not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load google token")
	}

	response := newBulkExtractedEventsResponse()
	for _, eventID := range eventIDs {
		event, ok := eventsByID[eventID]
		if !ok {
			response.addFailure(eventID.String(), "extracted event not found")
			continue
		}

		if err := s.registerExtractedEventRecord(c.Request().Context(), userID, googleToken, &event); err != nil {
			response.addFailure(event.ID.String(), err.Error())
			response.addEvent(event)
			continue
		}
		response.addSuccess(event, "registered")
	}

	return c.JSON(http.StatusOK, response)
}

func parseBulkExtractedEventIDs(c echo.Context) ([]uuid.UUID, error) {
	var req bulkExtractedEventsRequest
	if err := c.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if len(req.IDs) == 0 {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "ids are required")
	}

	eventIDs := make([]uuid.UUID, 0, len(req.IDs))
	seen := make(map[uuid.UUID]struct{}, len(req.IDs))
	for _, rawID := range req.IDs {
		eventID, err := uuid.Parse(strings.TrimSpace(rawID))
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "ids must be valid UUIDs")
		}
		if _, ok := seen[eventID]; ok {
			continue
		}
		seen[eventID] = struct{}{}
		eventIDs = append(eventIDs, eventID)
	}
	return eventIDs, nil
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

func (s *Server) loadOwnedExtractedEventsByID(ctx context.Context, userID uuid.UUID, eventIDs []uuid.UUID) (map[uuid.UUID]model.ExtractedEvent, error) {
	var events []model.ExtractedEvent
	if err := s.db.WithContext(ctx).
		Joins("JOIN letters ON letters.id = extracted_events.letter_id").
		Where("extracted_events.id IN ? AND letters.user_id = ?", eventIDs, userID).
		Find(&events).Error; err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "failed to load extracted events")
	}

	eventsByID := make(map[uuid.UUID]model.ExtractedEvent, len(events))
	for _, event := range events {
		eventsByID[event.ID] = event
	}
	return eventsByID, nil
}

func (s *Server) registerExtractedEventRecord(ctx context.Context, userID uuid.UUID, googleToken *oauth2.Token, event *model.ExtractedEvent) error {
	if err := validateExtractedEventRegisterable(*event); err != nil {
		return err
	}

	googleEvent, err := s.buildGoogleEventFromExtractedEvent(*event)
	if err != nil {
		return err
	}

	createdEvent, err := s.insertGoogleCalendarEvent(ctx, userID, googleToken, googleEvent)
	if err != nil {
		event.Status = model.ExtractedEventStatusFailed
		_ = s.db.WithContext(ctx).Save(event).Error
		return errGoogleCalendarEventCreateFailed
	}
	if createdEvent.Id == "" {
		event.Status = model.ExtractedEventStatusFailed
		_ = s.db.WithContext(ctx).Save(event).Error
		return errGoogleCalendarEventIDMissing
	}

	event.GoogleCalendarEventID = createdEvent.Id
	event.Status = model.ExtractedEventStatusRegistered
	if err := s.db.WithContext(ctx).Save(event).Error; err != nil {
		return errSaveExtractedEventFailed
	}
	return nil
}

func validateExtractedEventRegisterable(event model.ExtractedEvent) error {
	if event.GoogleCalendarEventID != "" && event.Status != model.ExtractedEventStatusDeleted {
		return errExtractedEventAlreadyRegistered
	}
	if event.Status != model.ExtractedEventStatusConfirmed &&
		event.Status != model.ExtractedEventStatusFailed &&
		event.Status != model.ExtractedEventStatusDeleted {
		return errExtractedEventNotRegisterable
	}

	return nil
}

func extractedEventRegisterHTTPError(err error) *echo.HTTPError {
	switch {
	case errors.Is(err, errExtractedEventAlreadyRegistered), errors.Is(err, errExtractedEventNotRegisterable):
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	case errors.Is(err, errGoogleCalendarEventCreateFailed), errors.Is(err, errGoogleCalendarEventIDMissing):
		return echo.NewHTTPError(http.StatusBadGateway, err.Error())
	case errors.Is(err, errSaveExtractedEventFailed):
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	default:
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
}

func newBulkExtractedEventsResponse() bulkExtractedEventsResponse {
	return bulkExtractedEventsResponse{
		Events:  []model.ExtractedEvent{},
		Results: []bulkExtractedEventResult{},
		Summary: bulkExtractedEventsSummary{},
	}
}

func (r *bulkExtractedEventsResponse) addSuccess(event model.ExtractedEvent, message string) {
	r.Summary.Success++
	r.addEvent(event)
	r.Results = append(r.Results, bulkExtractedEventResult{
		ID:      event.ID.String(),
		Status:  "success",
		Message: message,
		Event:   &event,
	})
}

func (r *bulkExtractedEventsResponse) addFailure(id string, message string) {
	r.Summary.Failed++
	r.Results = append(r.Results, bulkExtractedEventResult{
		ID:      id,
		Status:  "failed",
		Message: message,
	})
}

func (r *bulkExtractedEventsResponse) addEvent(event model.ExtractedEvent) {
	r.Events = append(r.Events, event)
}

func (s *Server) buildGoogleEventFromExtractedEvent(event model.ExtractedEvent) (*calendar.Event, error) {
	googleEvent := &calendar.Event{
		Summary:     event.Title,
		Location:    event.Location,
		Description: extractedEventCalendarDescription(event),
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

func extractedEventCalendarDescription(event model.ExtractedEvent) string {
	parts := make([]string, 0, 3)
	if description := strings.TrimSpace(event.Description); description != "" {
		parts = append(parts, description)
	}
	if belongings := strings.TrimSpace(event.Belongings); belongings != "" {
		parts = append(parts, "持ち物: "+belongings)
	}
	if event.SubmissionDeadline != nil {
		parts = append(parts, "提出期限: "+event.SubmissionDeadline.Format("2006年1月2日"))
	}
	return strings.Join(parts, "\n\n")
}

func validateExtractedEventEditable(event model.ExtractedEvent) error {
	if event.Status == model.ExtractedEventStatusRegistered {
		return errExtractedEventNotEditable
	}
	return nil
}

func validateExtractedEventStatusRestore(currentStatus, expectedStatus, targetStatus string) error {
	if currentStatus != expectedStatus {
		return errors.New("event status changed after the original action")
	}
	if !isLocallyRestorableExtractedEventStatus(currentStatus) || !isLocallyRestorableExtractedEventStatus(targetStatus) {
		return errors.New("event status cannot be restored")
	}
	return nil
}

func isLocallyRestorableExtractedEventStatus(status string) bool {
	return status == model.ExtractedEventStatusDraft ||
		status == model.ExtractedEventStatusConfirmed ||
		status == model.ExtractedEventStatusIgnored
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
	if req.Belongings != nil {
		event.Belongings = strings.TrimSpace(*req.Belongings)
	}
	if req.SubmissionDeadline != nil {
		submissionDeadline, err := normalizeOptionalDate(req.SubmissionDeadline, "submission_deadline")
		if err != nil {
			return err
		}
		event.SubmissionDeadline = submissionDeadline
	}

	if event.Status != model.ExtractedEventStatusRegistered && event.Status != model.ExtractedEventStatusDeleted {
		event.Status = model.ExtractedEventStatusConfirmed
	}
	return nil
}
