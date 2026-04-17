package runtime

import (
	"context"
	"time"
)

// Request is what the Runner sends to the Provider.
type Request struct {
	Messages []Message
	Tools    []ToolSpec
	// Additional parameters
	Temperature float32
	MaxTokens   int
}

// Response is what the Provider returns.
type Response struct {
	Text         string
	ToolCalls    []ToolCall
	InputTokens  int
	OutputTokens int
	StopReason   string
	ModelVersion string
}

// Message is a conversation message.
type Message struct {
	Role       string     // "user", "assistant", "tool", "system"
	Content    string     // text content
	ToolCalls  []ToolCall // assistant-role: tool calls made in this message
	ToolCallID string     // tool-role: the ID of the tool call being responded to
}

// Tool is a callable tool registered with the runtime.
type Tool struct {
	Name        string
	Description string
	InputSchema map[string]any
	Handler     func(ctx context.Context, args map[string]any) (any, error)
}

// ToolSpec is the public description of a tool sent to the LLM.
type ToolSpec struct {
	Name        string
	Description string
	InputSchema map[string]any
}

// ToolCall is a request from the LLM to invoke a tool.
type ToolCall struct {
	ID        string
	Name      string
	Arguments map[string]any
}

// ToolCallRecord captures a tool call + its outcome for the result.
type ToolCallRecord struct {
	ToolName  string
	Arguments map[string]any
	Result    any
	Error     error
	Duration  time.Duration
}

// RunResult is what Runner.Run returns.
type RunResult struct {
	Prompt       string
	FinalAnswer  string
	ToolCalls    []ToolCallRecord
	StepCount    int
	TokensIn     int
	TokensOut    int
	Duration     time.Duration
	StartTime    time.Time
	ModelVersion string
	Error        error
}

// Provider is implemented by backends (Anthropic, OpenAI, Bedrock, mock).
type Provider interface {
	Name() string
	Generate(ctx context.Context, req Request) (Response, error)
}

// StreamingProvider is optional — providers that can stream.
type StreamingProvider interface {
	Provider
	GenerateStream(ctx context.Context, req Request) (<-chan Event, error)
}

// Event is a streaming event from a provider.
type Event struct {
	Type     string // "content-delta", "tool-use-delta", "message-end"
	Content  string
	ToolCall *ToolCall
	Err      error
}

// Logger is the minimal logging interface used by the runtime.
type Logger interface {
	Debug(msg string, keysAndValues ...any)
	Info(msg string, keysAndValues ...any)
	Warn(msg string, keysAndValues ...any)
	Error(msg string, keysAndValues ...any)
}

// NopLogger discards all log messages.
type NopLogger struct{}

func (NopLogger) Debug(string, ...any) {}
func (NopLogger) Info(string, ...any)  {}
func (NopLogger) Warn(string, ...any)  {}
func (NopLogger) Error(string, ...any) {}
