package extractedevent

import (
	"context"
	"errors"
	"testing"
)

type extractionGeneratorStub struct {
	sourceOCRText string
	aiOutput      string
	referenceYear int
	external      bool
	err           error
}

func (g *extractionGeneratorStub) Generate(_ context.Context, sourceOCRText, aiOutput string, referenceYear int) (bool, error) {
	g.sourceOCRText = sourceOCRText
	g.aiOutput = aiOutput
	g.referenceYear = referenceYear
	return g.external, g.err
}

type extractionRepositoryStub struct {
	sourceOCRText string
	err           error
}

func (r *extractionRepositoryStub) SaveExtraction(_ context.Context, sourceOCRText string) error {
	r.sourceOCRText = sourceOCRText
	return r.err
}

func TestExtractExecuteUsesInputOCRAndSaves(t *testing.T) {
	generator := &extractionGeneratorStub{}
	repository := &extractionRepositoryStub{}
	usecase := Extract{Generator: generator, Repository: repository}

	result, err := usecase.Execute(context.Background(), ExtractInput{
		OCRText:       " 6月12日 身体測定 ",
		StoredOCRText: "保存済み",
		ReferenceYear: 2026,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.SourceOCRText != "6月12日 身体測定" || generator.sourceOCRText != result.SourceOCRText {
		t.Fatalf("unexpected source OCR text: %#v", result)
	}
	if repository.sourceOCRText != result.SourceOCRText {
		t.Fatalf("expected saved OCR text, got %q", repository.sourceOCRText)
	}
}

func TestExtractExecuteFallsBackToStoredOCR(t *testing.T) {
	generator := &extractionGeneratorStub{}
	usecase := Extract{Generator: generator, Repository: &extractionRepositoryStub{}}

	result, err := usecase.Execute(context.Background(), ExtractInput{StoredOCRText: " 保存済みOCR "})
	if err != nil {
		t.Fatal(err)
	}
	if result.SourceOCRText != "保存済みOCR" {
		t.Fatalf("unexpected source OCR text: %q", result.SourceOCRText)
	}
}

func TestExtractExecuteDoesNotSaveGenerationFailure(t *testing.T) {
	generator := &extractionGeneratorStub{external: true, err: errors.New("AI unavailable")}
	repository := &extractionRepositoryStub{}
	usecase := Extract{Generator: generator, Repository: repository}

	result, err := usecase.Execute(context.Background(), ExtractInput{OCRText: "保存してはいけないOCR"})
	if err == nil || !result.UsedExternalAI {
		t.Fatalf("expected external generation error, got %#v, %v", result, err)
	}
	if repository.sourceOCRText != "" {
		t.Fatal("failed generation must not be saved")
	}
}

func TestExtractExecuteReturnsSaveError(t *testing.T) {
	generator := &extractionGeneratorStub{external: true}
	usecase := Extract{
		Generator:  generator,
		Repository: &extractionRepositoryStub{err: errors.New("db unavailable")},
	}
	result, err := usecase.Execute(context.Background(), ExtractInput{})
	if !errors.Is(err, ErrSaveExtractionFailed) {
		t.Fatalf("expected save error, got %v", err)
	}
	if !result.UsedExternalAI {
		t.Fatal("expected external AI usage to remain visible on save failure")
	}
}
