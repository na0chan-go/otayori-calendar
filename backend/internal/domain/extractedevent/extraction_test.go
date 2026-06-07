package extractedevent

import "testing"

func TestValidateExtractionOutput(t *testing.T) {
	start, end, deadline := "15:00", "16:00", "2026-06-10"
	candidates, err := ValidateExtractionOutput(ExtractionOutput{Events: []ExtractionCandidate{{
		Title: " 保護者会 ", Date: "2026-06-12", StartTime: &start, EndTime: &end,
		Deadline: &deadline, Confidence: 0.8,
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if candidates[0].Title != "保護者会" || candidates[0].Deadline == nil {
		t.Fatalf("unexpected candidate: %#v", candidates[0])
	}
}

func TestExtractFromOCRText(t *testing.T) {
	output := ExtractFromOCRText("6月12日（金）身体測定を行います。\n朝は薄着で登園してください。", 2026)
	if len(output.Events) != 1 || output.Events[0].Title != "身体測定" || output.Events[0].Date != "2026-06-12" {
		t.Fatalf("unexpected output: %#v", output)
	}
}
