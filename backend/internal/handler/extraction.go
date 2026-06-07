package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	extractedeventdomain "github.com/na0chan-go/otayori-calendar/backend/internal/domain/extractedevent"
	"github.com/na0chan-go/otayori-calendar/backend/internal/model"
	"github.com/na0chan-go/otayori-calendar/backend/internal/service"
	extractedeventusecase "github.com/na0chan-go/otayori-calendar/backend/internal/usecase/extractedevent"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type extractionRequest struct {
	OCRText       string `json:"ocr_text"`
	AIOutput      string `json:"ai_output"`
	ReferenceYear int    `json:"reference_year"`
}

type extractionOutput = extractedeventdomain.ExtractionOutput
type extractionEvent = extractedeventdomain.ExtractionCandidate

var errExternalAIInvalidOutput = errors.New("external AI returned invalid output")

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

	adapter := &letterExtractionAdapter{server: s, letter: &letter}
	extract := extractedeventusecase.Extract{Generator: adapter, Repository: adapter}
	result, err := extract.Execute(c.Request().Context(), extractedeventusecase.ExtractInput{
		OCRText:       req.OCRText,
		StoredOCRText: letter.OCRText,
		AIOutput:      req.AIOutput,
		ReferenceYear: req.ReferenceYear,
	})
	if err != nil {
		if errors.Is(err, errExternalAIInvalidOutput) {
			return echo.NewHTTPError(http.StatusBadGateway, err.Error())
		}
		if errors.Is(err, extractedeventusecase.ErrSaveExtractionFailed) {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		if result.UsedExternalAI {
			return externalAIHTTPError(err)
		}
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusCreated, map[string]any{"events": adapter.events})
}

type letterExtractionAdapter struct {
	server *Server
	letter *model.Letter
	events []model.ExtractedEvent
}

func (a *letterExtractionAdapter) Generate(ctx context.Context, sourceOCRText, aiOutput string, referenceYear int) (bool, error) {
	output, usedExternalAI, err := a.server.buildLetterExtractionOutput(ctx, *a.letter, extractionRequest{
		AIOutput:      aiOutput,
		ReferenceYear: referenceYear,
	}, sourceOCRText)
	if err != nil {
		return usedExternalAI, err
	}

	a.events, err = validateExtractionOutput(a.letter.ID, output)
	if err != nil && usedExternalAI {
		return true, errExternalAIInvalidOutput
	}
	return usedExternalAI, err
}

func (a *letterExtractionAdapter) SaveExtraction(ctx context.Context, sourceOCRText string) error {
	return a.server.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if sourceOCRText != "" && sourceOCRText != a.letter.OCRText {
			if err := tx.Model(a.letter).Update("ocr_text", sourceOCRText).Error; err != nil {
				return err
			}
		}
		return tx.Create(&a.events).Error
	})
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

func (s *Server) buildLetterExtractionOutput(ctx context.Context, letter model.Letter, req extractionRequest, sourceOCRText string) (extractionOutput, bool, error) {
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

	rawOutput, err := s.extractor.Extract(ctx, service.GeminiExtractionRequest{
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

	return extractedeventdomain.ExtractFromOCRText(ocrText, year), nil
}

func extractEventsFromOCRText(ocrText string, year int) extractionOutput {
	return extractedeventdomain.ExtractFromOCRText(ocrText, year)
}

func validateExtractionOutput(letterID uuid.UUID, output extractionOutput) ([]model.ExtractedEvent, error) {
	candidates, err := extractedeventdomain.ValidateExtractionOutput(output)
	if err != nil {
		return nil, err
	}
	events := make([]model.ExtractedEvent, 0, len(candidates))
	for index, candidate := range candidates {
		extractedEvent, err := extractionCandidateModel(letterID, candidate)
		if err != nil {
			return nil, fmt.Errorf("events[%d]: %w", index, err)
		}
		events = append(events, extractedEvent)
	}

	return events, nil
}

func validateExtractionEvent(letterID uuid.UUID, event extractionEvent) (model.ExtractedEvent, error) {
	candidate, err := extractedeventdomain.ValidateExtractionCandidate(event)
	if err != nil {
		return model.ExtractedEvent{}, err
	}
	return extractionCandidateModel(letterID, candidate)
}

func extractionCandidateModel(letterID uuid.UUID, event extractionEvent) (model.ExtractedEvent, error) {
	eventDate, err := time.Parse("2006-01-02", event.Date)
	if err != nil {
		return model.ExtractedEvent{}, err
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

	return model.ExtractedEvent{
		LetterID:           letterID,
		Title:              event.Title,
		EventDate:          eventDate,
		StartTime:          startTime,
		EndTime:            endTime,
		IsAllDay:           event.IsAllDay,
		Location:           event.Location,
		Description:        event.Description,
		Belongings:         event.Belongings,
		SubmissionDeadline: submissionDeadline,
		Confidence:         event.Confidence,
		SourceText:         event.SourceText,
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
