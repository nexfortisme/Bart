package classifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type LMStudioEmbedder struct {
	BaseURL string
	Model string
	client *http.Client
}

type embedRequest struct {
	Model string `json:"model,omitempty"`
	Input string `json:"input"`
}

type embedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func NewLMStudioEmbedder(baseURL, model string) *LMStudioEmbedder {
	return &LMStudioEmbedder{
		BaseURL: baseURL,
		Model:   model,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (e *LMStudioEmbedder) Embed(text string) ([]float32, error) {
	body, err := json.Marshal(embedRequest{
		Model: e.Model,
		Input: text,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal failed: %w", err)
	}

	resp, err := e.client.Post(
		e.BaseURL + "/embeddings",
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var result embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("embedding API error: %s", result.Error.Message)
	}

	if len(result.Data) == 0 || len(result.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return result.Data[0].Embedding, nil
}