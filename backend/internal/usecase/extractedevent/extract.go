package extractedevent

import (
	"context"
	"errors"
	"strings"

	extractedeventport "github.com/na0chan-go/otayori-calendar/backend/internal/port/extractedevent"
)

var ErrSaveExtractionFailed = errors.New("failed to save extracted events")

type ExtractInput struct {
	OCRText       string
	StoredOCRText string
	AIOutput      string
	ReferenceYear int
}

type ExtractResult struct {
	SourceOCRText  string
	UsedExternalAI bool
}

type Extract struct {
	Generator  extractedeventport.ExtractionGenerator
	Repository extractedeventport.ExtractionRepository
}

func (u Extract) Execute(ctx context.Context, input ExtractInput) (ExtractResult, error) {
	sourceOCRText := strings.TrimSpace(input.OCRText)
	if sourceOCRText == "" {
		sourceOCRText = strings.TrimSpace(input.StoredOCRText)
	}

	usedExternalAI, err := u.Generator.Generate(ctx, sourceOCRText, input.AIOutput, input.ReferenceYear)
	result := ExtractResult{SourceOCRText: sourceOCRText, UsedExternalAI: usedExternalAI}
	if err != nil {
		return result, err
	}
	if err := u.Repository.SaveExtraction(ctx, sourceOCRText); err != nil {
		return result, ErrSaveExtractionFailed
	}
	return result, nil
}
