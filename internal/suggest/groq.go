package suggest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	groqEndpoint   = "https://api.groq.com/openai/v1/chat/completions"
	defaultModel   = "openai/gpt-oss-20b-128k"
	systemPrompt   = "You are a shell autocomplete engine. Given a partial command, return ONLY the full completed command. No explanation, no markdown, no quotes."
	requestTimeout = 2 * time.Second
)

type Client struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewClient(apiKey, model string) *Client {
	if model == "" {
		model = defaultModel
	}
	return &Client{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
	}
}

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
	Stream      bool      `json:"stream"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (c *Client) Complete(ctx context.Context, buffer, cwd, shell, historyStr string) (string, error) {
	if strings.TrimSpace(buffer) == "" {
		return "", nil
	}

	userPrompt := buildUserPrompt(buffer, cwd, shell, historyStr)

	body := chatRequest{
		Model: c.model,
		Messages: []message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens:   60,
		Temperature: 0,
		Stream:      false,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, groqEndpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("groq api returned %d", resp.StatusCode)
	}

	var result chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Choices) == 0 {
		return "", nil
	}

	suggestion := strings.TrimSpace(result.Choices[0].Message.Content)
	suggestion = cleanSuggestion(suggestion, buffer)
	return suggestion, nil
}

func buildUserPrompt(buffer, cwd, shell, historyStr string) string {
	var b strings.Builder
	b.WriteString("cwd: ")
	b.WriteString(cwd)
	if historyStr != "" && historyStr != "No shell history available." {
		b.WriteString("\nrecent:\n")
		lines := strings.Split(historyStr, "\n")
		if len(lines) > 3 {
			lines = lines[len(lines)-3:]
		}
		for _, l := range lines {
			b.WriteString(l)
			b.WriteByte('\n')
		}
	}
	b.WriteString("> ")
	b.WriteString(buffer)
	return b.String()
}

func cleanSuggestion(suggestion, buffer string) string {
	suggestion = strings.Trim(suggestion, "`\"'")
	suggestion = strings.TrimPrefix(suggestion, "$ ")

	if strings.HasPrefix(suggestion, buffer) {
		return suggestion
	}

	bufferPrefix := strings.Fields(buffer)
	suggestionFields := strings.Fields(suggestion)
	if len(bufferPrefix) > 0 && len(suggestionFields) > 0 && bufferPrefix[0] == suggestionFields[0] {
		return suggestion
	}

	return ""
}
