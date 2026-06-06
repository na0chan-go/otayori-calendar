package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/na0chan-go/otayori-calendar/backend/internal/model"
	"github.com/na0chan-go/otayori-calendar/backend/internal/service"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type extractionRequest struct {
	OCRText       string `json:"ocr_text"`
	AIOutput      string `json:"ai_output"`
	ReferenceYear int    `json:"reference_year"`
}

type extractionOutput struct {
	Events []extractionEvent `json:"events"`
}

type extractionEvent struct {
	Title       string  `json:"title"`
	Date        string  `json:"date"`
	StartTime   *string `json:"start_time"`
	EndTime     *string `json:"end_time"`
	IsAllDay    bool    `json:"is_all_day"`
	Location    string  `json:"location"`
	Description string  `json:"description"`
	Belongings  string  `json:"belongings"`
	Deadline    *string `json:"submission_deadline"`
	Confidence  float64 `json:"confidence"`
	SourceText  string  `json:"source_text"`
}

var japaneseDatePattern = regexp.MustCompile(`(?m)(\d{1,2})月(\d{1,2})日[^\n]*(?:\n[^\n]*)?`)

func (s *Server) extractLetterEvents(c echo.Context) error {
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

	var req extractionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	sourceOCRText := strings.TrimSpace(req.OCRText)
	if sourceOCRText == "" {
		sourceOCRText = strings.TrimSpace(letter.OCRText)
	}

	output, usedExternalAI, err := s.buildLetterExtractionOutput(c, letter, req, sourceOCRText)
	if err != nil {
		if usedExternalAI {
			return externalAIHTTPError(err)
		}
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	events, err := validateExtractionOutput(letter.ID, output)
	if err != nil {
		if usedExternalAI {
			return echo.NewHTTPError(http.StatusBadGateway, "external AI returned invalid output")
		}
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := s.db.WithContext(c.Request().Context()).Transaction(func(tx *gorm.DB) error {
		if sourceOCRText != "" && sourceOCRText != letter.OCRText {
			if err := tx.Model(&letter).Update("ocr_text", sourceOCRText).Error; err != nil {
				return err
			}
		}
		return tx.Create(&events).Error
	}); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to save extracted events")
	}

	return c.JSON(http.StatusCreated, map[string]any{"events": events})
}

func externalAIHTTPError(err error) *echo.HTTPError {
	switch {
	case errors.Is(err, service.ErrGeminiQuotaExceeded):
		return echo.NewHTTPError(http.StatusTooManyRequests, "Gemini APIの利用枠またはクレジットが不足しています")
	case errors.Is(err, service.ErrGeminiAuthentication):
		return echo.NewHTTPError(http.StatusBadGateway, "Gemini APIキーを確認してください")
	default:
		return echo.NewHTTPError(http.StatusBadGateway, "failed to extract events with external AI")
	}
}

func (s *Server) buildLetterExtractionOutput(c echo.Context, letter model.Letter, req extractionRequest, sourceOCRText string) (extractionOutput, bool, error) {
	if strings.TrimSpace(req.AIOutput) != "" || strings.TrimSpace(s.cfg.GeminiAPIKey) == "" {
		output, err := buildExtractionOutput(req, sourceOCRText)
		return output, false, err
	}

	referenceYear := req.ReferenceYear
	if referenceYear == 0 {
		referenceYear = time.Now().Year()
	}

	var image []byte
	if sourceOCRText == "" {
		var err error
		image, err = os.ReadFile(letter.ImagePath)
		if err != nil {
			return extractionOutput{}, true, errors.New("failed to read letter image")
		}
	}

	rawOutput, err := s.extractor.Extract(c.Request().Context(), service.GeminiExtractionRequest{
		OCRText:       sourceOCRText,
		Image:         image,
		MimeType:      letter.MimeType,
		ReferenceYear: referenceYear,
	})
	if err != nil {
		return extractionOutput{}, true, err
	}

	output, err := buildExtractionOutput(extractionRequest{AIOutput: rawOutput}, "")
	return output, true, err
}

func buildExtractionOutput(req extractionRequest, ocrText string) (extractionOutput, error) {
	if strings.TrimSpace(req.AIOutput) != "" {
		var output extractionOutput
		if err := json.Unmarshal([]byte(req.AIOutput), &output); err != nil {
			return extractionOutput{}, errors.New("ai_output must be valid JSON")
		}
		return output, nil
	}

	if strings.TrimSpace(ocrText) == "" {
		return extractionOutput{}, errors.New("ocr_text or ai_output is required")
	}

	year := req.ReferenceYear
	if year == 0 {
		year = time.Now().Year()
	}

	return extractEventsFromOCRText(ocrText, year), nil
}

func extractEventsFromOCRText(ocrText string, year int) extractionOutput {
	matches := japaneseDatePattern.FindAllStringSubmatch(ocrText, -1)
	events := make([]extractionEvent, 0, len(matches))

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		month, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}
		day, err := strconv.Atoi(match[2])
		if err != nil {
			continue
		}

		eventDate := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
		if eventDate.Month() != time.Month(month) || eventDate.Day() != day {
			continue
		}

		sourceText := strings.TrimSpace(match[0])
		title := guessTitleFromSource(sourceText)
		if title == "" {
			title = fmt.Sprintf("%d月%d日の予定", month, day)
		}

		events = append(events, extractionEvent{
			Title:       title,
			Date:        eventDate.Format("2006-01-02"),
			StartTime:   nil,
			EndTime:     nil,
			IsAllDay:    true,
			Location:    "",
			Description: sourceText,
			Confidence:  0.55,
			SourceText:  sourceText,
		})
	}

	return extractionOutput{Events: events}
}

func guessTitleFromSource(sourceText string) string {
	cleaned := regexp.MustCompile(`^\d{1,2}月\d{1,2}日(?:（[^）]+）|\([^)]*\))?`).ReplaceAllString(sourceText, "")
	cleaned = strings.TrimSpace(cleaned)
	cleaned = strings.Trim(cleaned, "。.")
	if cleaned == "" {
		return ""
	}

	parts := regexp.MustCompile(`[。\n、,]`).Split(cleaned, 2)
	title := strings.TrimSpace(parts[0])
	title = strings.TrimSuffix(title, "を行います")
	title = strings.TrimSuffix(title, "です")
	title = strings.TrimSuffix(title, "があります")
	return strings.TrimSpace(title)
}

func validateExtractionOutput(letterID uuid.UUID, output extractionOutput) ([]model.ExtractedEvent, error) {
	if output.Events == nil {
		return nil, errors.New("events must be an array")
	}
	if len(output.Events) == 0 {
		return nil, errors.New("events must not be empty")
	}

	events := make([]model.ExtractedEvent, 0, len(output.Events))
	for index, event := range output.Events {
		extractedEvent, err := validateExtractionEvent(letterID, event)
		if err != nil {
			return nil, fmt.Errorf("events[%d]: %w", index, err)
		}
		events = append(events, extractedEvent)
	}

	return events, nil
}

func validateExtractionEvent(letterID uuid.UUID, event extractionEvent) (model.ExtractedEvent, error) {
	title := strings.TrimSpace(event.Title)
	if title == "" {
		return model.ExtractedEvent{}, errors.New("title is required")
	}

	eventDate, err := time.Parse("2006-01-02", event.Date)
	if err != nil {
		return model.ExtractedEvent{}, errors.New("date must be YYYY-MM-DD")
	}

	if event.Confidence < 0 || event.Confidence > 1 {
		return model.ExtractedEvent{}, errors.New("confidence must be between 0 and 1")
	}

	submissionDeadline, err := normalizeOptionalDate(event.Deadline, "submission_deadline")
	if err != nil {
		return model.ExtractedEvent{}, err
	}

	startTime, err := normalizeOptionalClock(event.StartTime, "start_time")
	if err != nil {
		return model.ExtractedEvent{}, err
	}
	endTime, err := normalizeOptionalClock(event.EndTime, "end_time")
	if err != nil {
		return model.ExtractedEvent{}, err
	}
	if startTime != nil && endTime != nil && *endTime <= *startTime {
		return model.ExtractedEvent{}, errors.New("end_time must be after start_time")
	}
	isAllDay := event.IsAllDay
	if startTime == nil && endTime == nil {
		isAllDay = true
	}

	return model.ExtractedEvent{
		LetterID:           letterID,
		Title:              title,
		EventDate:          eventDate,
		StartTime:          startTime,
		EndTime:            endTime,
		IsAllDay:           isAllDay,
		Location:           strings.TrimSpace(event.Location),
		Description:        strings.TrimSpace(event.Description),
		Belongings:         strings.TrimSpace(event.Belongings),
		SubmissionDeadline: submissionDeadline,
		Confidence:         event.Confidence,
		SourceText:         strings.TrimSpace(event.SourceText),
		Status:             model.ExtractedEventStatusDraft,
	}, nil
}

func normalizeOptionalDate(value *string, fieldName string) (*time.Time, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}

	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(*value))
	if err != nil {
		return nil, fmt.Errorf("%s must be YYYY-MM-DD", fieldName)
	}
	return &parsed, nil
}

func normalizeOptionalClock(value *string, fieldName string) (*datatypes.Time, error) {
	if value == nil {
		return nil, nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := time.Parse("15:04", trimmed)
	if err != nil {
		return nil, fmt.Errorf("%s must be HH:MM", fieldName)
	}

	clock := datatypes.NewTime(parsed.Hour(), parsed.Minute(), 0, 0)
	return &clock, nil
}
