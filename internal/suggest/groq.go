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
	systemPrompt   = `You are a shell autocomplete engine embedded in a terminal. You will receive the user's current working directory, their shell, recent command history, and their partially typed command.

Your job is to predict and return the SINGLE most likely full command the user is trying to type. Think about:
- What command they are starting to type (even from just 2-3 characters)
- Their recent history for patterns and context
- Their current directory for relevant files/paths
- Common shell commands, flags, and arguments

Rules:
- Return ONLY the completed command, nothing else
- No explanation, no markdown, no backticks, no quotes around the command
- The completion must start with exactly what the user has typed so far
- Prefer practical, real commands over generic ones
- Include flags and arguments when they are clearly implied
- If unsure, complete just the command name`
	requestTimeout = 2 * time.Second
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
