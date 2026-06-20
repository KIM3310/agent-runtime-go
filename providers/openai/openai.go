// Package openai implements the Provider interface for OpenAI's Chat Completions API.
package openai

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

const DefaultBaseURL = "https://api.openai.com/v1"

type Provider struct {
	apiKey     string
	baseURL    string
	model      string
	headers    map[string]string
	httpClient *http.Client
}

type Option func(*Provider)

func WithModel(model string) Option {
	return func(p *Provider) { p.model = model }
}

func WithBaseURL(url string) Option {
	return func(p *Provider) { p.baseURL = url }
}

func WithHeader(key string, value string) Option {
	return func(p *Provider) {
		if p.headers == nil {
			p.headers = make(map[string]string)
		}
		p.headers[key] = value
	}
}

func New(apiKey string, opts ...Option) *Provider {
	p := &Provider{
		apiKey:     apiKey,
		baseURL:    DefaultBaseURL,
		model:      "gpt-4o",
		headers:    map[string]string{},
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *Provider) Name() string { return "openai" }

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Tools       []openAITool  `json:"tools,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float32       `json:"temperature"`
}

type chatMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openAITool struct {
	Type     string             `json:"type"`
	Function openAIToolFunction `json:"function"`
}

type openAIToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type openAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type chatResponse struct {
	Choices []struct {
		Index        int         `json:"index"`
		Message      chatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Model string `json:"model"`
}

func (p *Provider) Generate(ctx context.Context, req runtime.Request) (runtime.Response, error) {
	oaiReq := p.buildRequest(req)
	body, err := json.Marshal(oaiReq)
	if err != nil {
		return runtime.Response{}, fmt.Errorf("marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body),
	)
	if err != nil {
		return runtime.Response{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	for key, value := range p.headers {
		if key != "" && value != "" {
			httpReq.Header.Set(key, value)
		}
	}

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
			Msg:        fmt.Sprintf("openai: status %d: %s", resp.StatusCode, string(respBody)),
		}
	}

	var oaiResp chatResponse
	if err := json.Unmarshal(respBody, &oaiResp); err != nil {
		return runtime.Response{}, fmt.Errorf("unmarshal: %w", err)
	}

	return p.parseResponse(oaiResp), nil
}

func (p *Provider) buildRequest(req runtime.Request) chatRequest {
	messages := make([]chatMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		msg := chatMessage{
			Role:       m.Role,
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
		}
		for _, tc := range m.ToolCalls {
			argsJSON, _ := json.Marshal(tc.Arguments)
			oaiTC := openAIToolCall{
				ID:   tc.ID,
				Type: "function",
			}
			oaiTC.Function.Name = tc.Name
			oaiTC.Function.Arguments = string(argsJSON)
			msg.ToolCalls = append(msg.ToolCalls, oaiTC)
		}
		messages = append(messages, msg)
	}

	tools := make([]openAITool, 0, len(req.Tools))
	for _, t := range req.Tools {
		tools = append(tools, openAITool{
			Type: "function",
			Function: openAIToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	return chatRequest{
		Model:       p.model,
		Messages:    messages,
		Tools:       tools,
		MaxTokens:   maxTokens,
		Temperature: req.Temperature,
	}
}

func (p *Provider) parseResponse(resp chatResponse) runtime.Response {
	if len(resp.Choices) == 0 {
		return runtime.Response{}
	}

	choice := resp.Choices[0]
	out := runtime.Response{
		Text:         choice.Message.Content,
		InputTokens:  resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
		StopReason:   choice.FinishReason,
		ModelVersion: resp.Model,
	}

	for _, tc := range choice.Message.ToolCalls {
		var args map[string]any
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			args = map[string]any{"_parse_error": err.Error(), "_raw": tc.Function.Arguments}
		}
		out.ToolCalls = append(out.ToolCalls, runtime.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: args,
		})
	}

	return out
}
