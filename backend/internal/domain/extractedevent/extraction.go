package extractedevent

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type ExtractionOutput struct {
	Events []ExtractionCandidate `json:"events"`
}

type ExtractionCandidate struct {
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

func ValidateExtractionOutput(output ExtractionOutput) ([]ExtractionCandidate, error) {
	if output.Events == nil {
		return nil, errors.New("events must be an array")
	}
	if len(output.Events) == 0 {
		return nil, errors.New("events must not be empty")
	}

	candidates := make([]ExtractionCandidate, 0, len(output.Events))
	for index, candidate := range output.Events {
		validated, err := ValidateExtractionCandidate(candidate)
		if err != nil {
			return nil, fmt.Errorf("events[%d]: %w", index, err)
		}
		candidates = append(candidates, validated)
	}
	return candidates, nil
}

func ValidateExtractionCandidate(candidate ExtractionCandidate) (ExtractionCandidate, error) {
	candidate.Title = strings.TrimSpace(candidate.Title)
	if candidate.Title == "" {
		return candidate, errors.New("title is required")
	}
	if _, err := time.Parse("2006-01-02", candidate.Date); err != nil {
		return candidate, errors.New("date must be YYYY-MM-DD")
	}
	if candidate.Confidence < 0 || candidate.Confidence > 1 {
		return candidate, errors.New("confidence must be between 0 and 1")
	}

	var err error
	candidate.Deadline, err = validateOptionalDate(candidate.Deadline, "submission_deadline")
	if err != nil {
		return candidate, err
	}
	candidate.StartTime, err = validateOptionalClock(candidate.StartTime, "start_time")
	if err != nil {
		return candidate, err
	}
	candidate.EndTime, err = validateOptionalClock(candidate.EndTime, "end_time")
	if err != nil {
		return candidate, err
	}
	if candidate.StartTime != nil && candidate.EndTime != nil && *candidate.EndTime <= *candidate.StartTime {
		return candidate, errors.New("end_time must be after start_time")
	}
	if candidate.StartTime == nil && candidate.EndTime == nil {
		candidate.IsAllDay = true
	}

	candidate.Location = strings.TrimSpace(candidate.Location)
	candidate.Description = strings.TrimSpace(candidate.Description)
	candidate.Belongings = strings.TrimSpace(candidate.Belongings)
	candidate.SourceText = strings.TrimSpace(candidate.SourceText)
	return candidate, nil
}

func ExtractFromOCRText(ocrText string, year int) ExtractionOutput {
	matches := japaneseDatePattern.FindAllStringSubmatch(ocrText, -1)
	events := make([]ExtractionCandidate, 0, len(matches))
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		month, monthErr := strconv.Atoi(match[1])
		day, dayErr := strconv.Atoi(match[2])
		if monthErr != nil || dayErr != nil {
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
		events = append(events, ExtractionCandidate{
			Title: title, Date: eventDate.Format("2006-01-02"), IsAllDay: true,
			Description: sourceText, Confidence: 0.55, SourceText: sourceText,
		})
	}
	return ExtractionOutput{Events: events}
}

func guessTitleFromSource(sourceText string) string {
	cleaned := regexp.MustCompile(`^\d{1,2}月\d{1,2}日(?:（[^）]+）|\([^)]*\))?`).ReplaceAllString(sourceText, "")
	cleaned = strings.Trim(strings.TrimSpace(cleaned), "。.")
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

func validateOptionalDate(value *string, fieldName string) (*string, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*value)
	if _, err := time.Parse("2006-01-02", trimmed); err != nil {
		return nil, fmt.Errorf("%s must be YYYY-MM-DD", fieldName)
	}
	return &trimmed, nil
}

func validateOptionalClock(value *string, fieldName string) (*string, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*value)
	if _, err := time.Parse("15:04", trimmed); err != nil {
		return nil, fmt.Errorf("%s must be HH:MM", fieldName)
	}
	return &trimmed, nil
}
