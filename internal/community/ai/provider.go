// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sessiondb/internal/engine"
	"time"
)

// OpenAICompatibleProvider calls an OpenAI-compatible chat API to generate SQL or explain queries.
type OpenAICompatibleProvider struct {
	BaseURL string
	APIKey  string
	Model   string
	Client  *http.Client
}

// NewOpenAICompatibleProvider returns a provider for the given endpoint and key.
func NewOpenAICompatibleProvider(baseURL, apiKey, model string) *OpenAICompatibleProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &OpenAICompatibleProvider{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
		Client:  &http.Client{Timeout: 60 * time.Second},
	}
}

// openAIReq is the request body for chat completions.
type openAIReq struct {
	Model    string    `json:"model"`
	Messages []openAIMsg `json:"messages"`
}

type openAIMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResp struct {
	Choices []struct {
		Message openAIMsg `json:"message"`
	} `json:"choices"`
}

// GenerateSQL builds a system message with schema context and asks the API for SQL.
func (p *OpenAICompatibleProvider) GenerateSQL(ctx context.Context, prompt string, schema *engine.SchemaContext) (string, error) {
	systemContent := "You are a SQL expert. Generate only valid SQL, no explanation. Use the following schema:\n"
	if schema != nil {
		for _, t := range schema.Tables {
			systemContent += fmt.Sprintf("Table %s.%s: columns %v\n", t.Schema, t.Table, t.Columns)
		}
	}
	body := openAIReq{
		Model: p.Model,
		Messages: []openAIMsg{
			{Role: "system", Content: systemContent},
			{Role: "user", Content: prompt},
		},
	}
	return p.chat(ctx, body)
}

// ExplainQuery asks the API to explain the given SQL.
func (p *OpenAICompatibleProvider) ExplainQuery(ctx context.Context, query string) (string, error) {
	body := openAIReq{
		Model: p.Model,
		Messages: []openAIMsg{
			{Role: "system", Content: "Explain this SQL query briefly in one or two sentences."},
			{Role: "user", Content: query},
		},
	}
	return p.chat(ctx, body)
}

func (p *OpenAICompatibleProvider) chat(ctx context.Context, body openAIReq) (string, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.BaseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	resp, err := p.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("api returned %d: %s", resp.StatusCode, string(b))
	}
	var out openAIResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}
	return out.Choices[0].Message.Content, nil
}
