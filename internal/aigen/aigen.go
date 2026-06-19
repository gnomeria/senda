// Package aigen generates suggested test assertions using an LLM.
// It calls the API configured via environment variables:
//
//	SENDA_AI_BASE_URL  — base URL (default: https://api.anthropic.com/v1)
//	SENDA_AI_MODEL     — model ID (default: claude-haiku-4-5-20251001)
//	SENDA_AI_API_KEY   — API key (fallback: ANTHROPIC_API_KEY)
//
// When SENDA_AI_BASE_URL is set to an OpenAI-compatible endpoint (e.g. Ollama
// at http://localhost:11434/v1), the package uses the OpenAI /chat/completions
// format. Otherwise the Anthropic /messages format is used.
package aigen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"senda/internal/model"
)

const (
	defaultBaseURL = "https://api.anthropic.com/v1"
	defaultModel   = "claude-haiku-4-5-20251001"
)

// Config holds resolved AI settings.
type Config struct {
	BaseURL string
	Model   string
	APIKey  string
}

// ResolveConfig reads configuration from environment variables.
func ResolveConfig() Config {
	base := os.Getenv("SENDA_AI_BASE_URL")
	if base == "" {
		base = defaultBaseURL
	}
	model := os.Getenv("SENDA_AI_MODEL")
	if model == "" {
		model = defaultModel
	}
	key := os.Getenv("SENDA_AI_API_KEY")
	if key == "" {
		key = os.Getenv("ANTHROPIC_API_KEY")
	}
	return Config{BaseURL: base, Model: model, APIKey: key}
}

// GenerateAssertions sends the response data to the configured LLM and returns
// suggested Assert rows. Returns an empty slice (not an error) when the LLM
// returns nothing useful.
func GenerateAssertions(ctx context.Context, cfg Config, resp model.Response) ([]model.Assert, error) {
	prompt := buildPrompt(resp)
	reply, err := callLLM(ctx, cfg, prompt)
	if err != nil {
		return nil, err
	}
	return parseAssertions(reply), nil
}

func buildPrompt(resp model.Response) string {
	var b strings.Builder
	b.WriteString("You are a test engineer. Given this HTTP response, suggest 3-6 assertion rules.\n\n")
	fmt.Fprintf(&b, "Status: %d %s\n", resp.Status, resp.StatusText)
	fmt.Fprintf(&b, "Duration: %dms\n", resp.DurationMs)
	// Include relevant response headers
	for k, vals := range resp.Headers {
		kl := strings.ToLower(k)
		if kl == "content-type" || kl == "x-request-id" || strings.Contains(kl, "rate") {
			fmt.Fprintf(&b, "Header %s: %s\n", k, strings.Join(vals, ", "))
		}
	}
	body := resp.Body
	if len(body) > 2000 {
		body = body[:2000] + "...(truncated)"
	}
	fmt.Fprintf(&b, "\nResponse body:\n%s\n\n", body)
	b.WriteString(`Output ONLY a JSON array of assertion objects, no other text. Each object has:
- "target": one of "status", "duration", "json.<path>", "header.<Name>", "body"
- "op": one of "eq", "lt", "lte", "gt", "gte", "contains", "matches", "exists"
- "value": expected value as a string (omit for "exists")
- "enabled": true

Example:
[
  {"target":"status","op":"eq","value":"200","enabled":true},
  {"target":"json.id","op":"exists","enabled":true}
]`)
	return b.String()
}

func callLLM(ctx context.Context, cfg Config, prompt string) (string, error) {
	if cfg.APIKey == "" {
		return "", fmt.Errorf("no AI API key configured (set SENDA_AI_API_KEY or ANTHROPIC_API_KEY)")
	}

	useOpenAI := !strings.Contains(cfg.BaseURL, "anthropic.com")

	var (
		reqBody []byte
		err     error
		url     string
	)

	if useOpenAI {
		url = strings.TrimRight(cfg.BaseURL, "/") + "/chat/completions"
		payload := map[string]any{
			"model": cfg.Model,
			"messages": []map[string]string{
				{"role": "user", "content": prompt},
			},
			"max_tokens": 1024,
		}
		reqBody, err = json.Marshal(payload)
	} else {
		url = strings.TrimRight(cfg.BaseURL, "/") + "/messages"
		payload := map[string]any{
			"model": cfg.Model,
			"messages": []map[string]string{
				{"role": "user", "content": prompt},
			},
			"max_tokens": 1024,
		}
		reqBody, err = json.Marshal(payload)
	}
	if err != nil {
		return "", fmt.Errorf("aigen: marshal: %w", err)
	}

	httpCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(httpCtx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("aigen: request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if useOpenAI {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	} else {
		req.Header.Set("x-api-key", cfg.APIKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("aigen: http: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("aigen: read: %w", err)
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("aigen: API error %d: %s", resp.StatusCode, string(data))
	}

	return extractContent(data, useOpenAI)
}

func extractContent(data []byte, openAI bool) (string, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return "", fmt.Errorf("aigen: parse response: %w", err)
	}
	if openAI {
		// OpenAI: choices[0].message.content
		choices, _ := raw["choices"].([]any)
		if len(choices) == 0 {
			return "", fmt.Errorf("aigen: no choices in response")
		}
		msg, _ := choices[0].(map[string]any)["message"].(map[string]any)
		if msg == nil {
			return "", fmt.Errorf("aigen: no message in choice")
		}
		return fmt.Sprintf("%v", msg["content"]), nil
	}
	// Anthropic: content[0].text
	content, _ := raw["content"].([]any)
	if len(content) == 0 {
		return "", fmt.Errorf("aigen: no content in response")
	}
	block, _ := content[0].(map[string]any)
	return fmt.Sprintf("%v", block["text"]), nil
}

// parseAssertions extracts an Assert slice from the LLM's text output, which
// should be a JSON array. Handles responses that include prose around the JSON.
func parseAssertions(text string) []model.Assert {
	// Find the JSON array in the response
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start < 0 || end <= start {
		return nil
	}
	jsonStr := text[start : end+1]

	var rows []struct {
		Target  string `json:"target"`
		Op      string `json:"op"`
		Value   string `json:"value"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &rows); err != nil {
		return nil
	}

	out := make([]model.Assert, 0, len(rows))
	for _, r := range rows {
		if r.Target == "" || r.Op == "" {
			continue
		}
		out = append(out, model.Assert{
			Target:  r.Target,
			Op:      r.Op,
			Value:   r.Value,
			Enabled: r.Enabled,
		})
	}
	return out
}
