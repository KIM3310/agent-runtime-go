// Package anthropic implements the Provider interface for Anthropic's Claude API.
package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/KIM3310/agent-runtime-go/runtime"
)

const (
	DefaultBaseURL   = "https://api.anthropic.com/v1"
	DefaultAPIVersion = "2023-06-01"
)

// Provider is an Anthropic Claude provider.
type Provider struct {
	apiKey     string
	baseURL    string
	apiVersion string
	model      string
	httpClient *http.Client
}

// Option configures the Provider.
type Option func(*Provider)

// WithModel sets the model (e.g., "claude-sonnet-4-20250514").
func WithModel(model string) Option {
	return func(p *Provider) { p.model = model }
}

// WithBaseURL overrides the API base URL (for proxies or test endpoints).
func WithBaseURL(url string) Option {
	return func(p *Provider) { p.baseURL = url }
}

// WithHTTPClient supplies a custom http.Client (useful for custom TLS, proxies).
func WithHTTPClient(c *http.Client) Option {
	return func(p *Provider) { p.httpClient = c }
}

// New creates an Anthropic provider.
func New(apiKey string, opts ...Option) *Provider {
	p := &Provider{
		apiKey:     apiKey,
		baseURL:    DefaultBaseURL,
		apiVersion: DefaultAPIVersion,
		model:      "claude-sonnet-4-20250514",
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Name returns the provider identifier.
func (p *Provider) Name() string { return "anthropic" }

// Generate calls the Anthropic Messages API.
func (p *Provider) Generate(ctx context.Context, req runtime.Request) (runtime.Response, error) {
	anthropicReq := p.buildRequest(req)
	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return runtime.Response{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx, "POST", p.baseURL+"/messages", bytes.NewReader(body),
	)
	if err != nil {
		return runtime.Response{}, err
	}
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", p.apiVersion)
	httpReq.Header.Set("content-type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return runtime.Response{}, fmt.Errorf("http call: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 429 {
		return runtime.Response{}, &runtime.RateLimitError{Msg: string(respBody)}
	}
	if resp.StatusCode >= 400 {
		return runtime.Response{}, &runtime.APIStatusError{
			StatusCode: resp.StatusCode,
			Msg:        fmt.Sprintf("anthropic: status %d: %s", resp.StatusCode, string(respBody)),
		}
	}

	var anthropicResp messageResponse
	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return runtime.Response{}, fmt.Errorf("unmarshal response: %w", err)
	}

	return p.parseResponse(anthropicResp), nil
}

// messageRequest is the payload for Anthropic Messages API.
type messageRequest struct {
	Model     string               `json:"model"`
	MaxTokens int                  `json:"max_tokens"`
	Messages  []anthropicMessage   `json:"messages"`
	System    string               `json:"system,omitempty"`
	Tools     []anthropicTool      `json:"tools,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content []anthropicContent `json:"content"`
}

type anthropicContent struct {
	Type       string          `json:"type"`
	Text       string          `json:"text,omitempty"`
	ID         string          `json:"id,omitempty"`
	Name       string          `json:"name,omitempty"`
	Input      map[string]any  `json:"input,omitempty"`
	ToolUseID  string          `json:"tool_use_id,omitempty"`
	Content    string          `json:"content,omitempty"`
}

type anthropicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

type messageResponse struct {
	ID         string             `json:"id"`
	Type       string             `json:"type"`
	Role       string             `json:"role"`
	Content    []anthropicContent `json:"content"`
	Model      string             `json:"model"`
	StopReason string             `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (p *Provider) buildRequest(req runtime.Request) messageRequest {
	messages := make([]anthropicMessage, 0, len(req.Messages))
	var system string

	for _, m := range req.Messages {
		switch m.Role {
		case "system":
			system = m.Content
		case "user":
			messages = append(messages, anthropicMessage{
				Role:    "user",
				Content: []anthropicContent{{Type: "text", Text: m.Content}},
			})
		case "assistant":
			content := []anthropicContent{}
			if m.Content != "" {
				content = append(content, anthropicContent{Type: "text", Text: m.Content})
			}
			for _, tc := range m.ToolCalls {
				content = append(content, anthropicContent{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Name,
					Input: tc.Arguments,
				})
			}
			messages = append(messages, anthropicMessage{
				Role:    "assistant",
				Content: content,
			})
		case "tool":
			messages = append(messages, anthropicMessage{
				Role: "user",
				Content: []anthropicContent{{
					Type:      "tool_result",
					ToolUseID: m.ToolCallID,
					Content:   m.Content,
				}},
			})
		}
	}

	tools := make([]anthropicTool, 0, len(req.Tools))
	for _, t := range req.Tools {
		tools = append(tools, anthropicTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	return messageRequest{
		Model:     p.model,
		MaxTokens: maxTokens,
		Messages:  messages,
		System:    system,
		Tools:     tools,
	}
}

func (p *Provider) parseResponse(resp messageResponse) runtime.Response {
	out := runtime.Response{
		InputTokens:  resp.Usage.InputTokens,
		OutputTokens: resp.Usage.OutputTokens,
		StopReason:   resp.StopReason,
		ModelVersion: resp.Model,
	}

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			out.Text += block.Text
		case "tool_use":
			out.ToolCalls = append(out.ToolCalls, runtime.ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: block.Input,
			})
		}
	}

	return out
}
