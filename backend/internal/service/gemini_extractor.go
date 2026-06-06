package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultGeminiBaseURL = "https://generativelanguage.googleapis.com/v1beta"

var (
	ErrGeminiQuotaExceeded  = errors.New("gemini api quota exceeded")
	ErrGeminiAuthentication = errors.New("gemini api authentication failed")
	ErrGeminiUnavailable    = errors.New("gemini api unavailable")
)

type GeminiExtractor struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

type GeminiExtractionRequest struct {
	OCRText       string
	Image         []byte
	MimeType      string
	ReferenceYear int
}

type geminiGenerateContentRequest struct {
	Contents         []geminiContent        `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *geminiInlineData `json:"inlineData,omitempty"`
}

type geminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type geminiGenerationConfig struct {
	ResponseMimeType string         `json:"responseMimeType"`
	ResponseSchema   map[string]any `json:"responseSchema"`
}

type geminiGenerateContentResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
}

func NewGeminiExtractor(apiKey, model string, client *http.Client) *GeminiExtractor {
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	return &GeminiExtractor{
		apiKey:  strings.TrimSpace(apiKey),
		model:   strings.TrimSpace(model),
		baseURL: defaultGeminiBaseURL,
		client:  client,
	}
}

func (g *GeminiExtractor) Extract(ctx context.Context, input GeminiExtractionRequest) (string, error) {
	if g.apiKey == "" || g.model == "" {
		return "", errors.New("gemini api is not configured")
	}

	parts := []geminiPart{{Text: extractionPrompt(input.ReferenceYear)}}
	if text := strings.TrimSpace(input.OCRText); text != "" {
		parts = append(parts, geminiPart{Text: text})
	} else {
		if len(input.Image) == 0 || strings.TrimSpace(input.MimeType) == "" {
			return "", errors.New("ocr text or image is required")
		}
		parts = append(parts, geminiPart{InlineData: &geminiInlineData{
			MimeType: input.MimeType,
			Data:     base64.StdEncoding.EncodeToString(input.Image),
		}})
	}

	requestBody, err := json.Marshal(geminiGenerateContentRequest{
		Contents: []geminiContent{{Parts: parts}},
		GenerationConfig: geminiGenerationConfig{
			ResponseMimeType: "application/json",
			ResponseSchema:   extractionResponseSchema(),
		},
	})
	if err != nil {
		return "", fmt.Errorf("encode gemini request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent", strings.TrimRight(g.baseURL, "/"), g.model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(requestBody))
	if err != nil {
		return "", fmt.Errorf("create gemini request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", g.apiKey)

	response, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("call gemini api: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, response.Body)
		switch response.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return "", ErrGeminiAuthentication
		case http.StatusTooManyRequests:
			return "", ErrGeminiQuotaExceeded
		default:
			return "", ErrGeminiUnavailable
		}
	}

	var result geminiGenerateContentResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&result); err != nil {
		return "", fmt.Errorf("decode gemini response: %w", err)
	}
	for _, candidate := range result.Candidates {
		for _, part := range candidate.Content.Parts {
			if text := strings.TrimSpace(part.Text); text != "" {
				return text, nil
			}
		}
	}
	return "", errors.New("gemini response did not contain output")
}

func extractionPrompt(referenceYear int) string {
	return fmt.Sprintf(`保育園のおたよりから、日付がある予定候補だけを抽出してください。
持ち物や注意事項は description に含め、抽出根拠の原文は source_text に含めてください。
不明な値は推測しすぎず、空文字または null にしてください。
年が書かれていない場合は %d 年として解釈してください。
個人名など、予定に不要な個人情報は出力しないでください。`, referenceYear)
}

func extractionResponseSchema() map[string]any {
	nullableString := map[string]any{
		"type":     "string",
		"nullable": true,
	}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"events": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"title":       map[string]any{"type": "string"},
						"date":        map[string]any{"type": "string", "format": "date"},
						"start_time":  nullableString,
						"end_time":    nullableString,
						"is_all_day":  map[string]any{"type": "boolean"},
						"location":    map[string]any{"type": "string"},
						"description": map[string]any{"type": "string"},
						"confidence":  map[string]any{"type": "number", "minimum": 0, "maximum": 1},
						"source_text": map[string]any{"type": "string"},
					},
					"required": []string{
						"title", "date", "start_time", "end_time", "is_all_day",
						"location", "description", "confidence", "source_text",
					},
				},
			},
		},
		"required": []string{"events"},
	}
}
