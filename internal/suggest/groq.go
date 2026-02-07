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
	defaultModel   = "llama-3.1-8b-instant"
	systemPrompt   = `You are a shell autocomplete engine. Given a partially typed command, predict the full command.

ALWAYS complete aggressively. Even from 2 characters, predict the full command with flags and arguments.
Use the history and cwd to make smart predictions. If they recently ran a command, predict they'll run something related.

Rules:
- Return ONLY the completed command
- No explanation, no markdown, no backticks, no quotes
- The completion MUST start with exactly what the user has typed so far
- Always include likely flags and arguments, never return just a bare command name
- Be specific: prefer "git push origin main" over "git push"
- If the user typed part of a path or filename, complete it based on context`
	requestTimeout = 3 * time.Second
)

type Client struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

type Option func(*Client)

func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

func NewClient(apiKey, model string, opts ...Option) *Client {
	if model == "" {
		model = defaultModel
	}
	c := &Client{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
	}
	for _, o := range opts {
		o(c)
	}
	return c
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
		MaxTokens:   120,
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
	b.WriteString("shell: ")
	b.WriteString(shell)
	b.WriteString("\ncwd: ")
	b.WriteString(cwd)
	if historyStr != "" && historyStr != "No shell history available." {
		b.WriteString("\nrecent history:\n")
		for _, l := range strings.Split(historyStr, "\n") {
			b.WriteString("  ")
			b.WriteString(l)
			b.WriteByte('\n')
		}
	}
	b.WriteString("\n> ")
	b.WriteString(buffer)
	return b.String()
}

func cleanSuggestion(suggestion, buffer string) string {
	suggestion = strings.TrimSpace(suggestion)
	suggestion = strings.Trim(suggestion, "`\"'")
	suggestion = strings.TrimPrefix(suggestion, "$ ")
	suggestion = strings.TrimSpace(suggestion)

	if idx := strings.IndexByte(suggestion, '\n'); idx != -1 {
		suggestion = suggestion[:idx]
	}

	if strings.HasPrefix(suggestion, buffer) {
		return suggestion
	}

	if strings.HasPrefix(strings.ToLower(suggestion), strings.ToLower(buffer)) {
		return buffer + suggestion[len(buffer):]
	}

	return ""
}
