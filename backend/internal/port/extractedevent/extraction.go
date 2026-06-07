package extractedevent

import "context"

type ExtractionGenerator interface {
	Generate(ctx context.Context, sourceOCRText string, aiOutput string, referenceYear int) (usedExternalAI bool, err error)
}

type ExtractionRepository interface {
	SaveExtraction(ctx context.Context, sourceOCRText string) error
}
