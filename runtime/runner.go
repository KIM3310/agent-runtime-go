// Package runtime implements the core agent orchestration loop.
package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Runner is the top-level agent orchestrator.
type Runner struct {
	provider    Provider
	tools       map[string]Tool
	maxSteps    int
	timeout     time.Duration
	retryPolicy RetryPolicy
	logger      Logger
}

// NewRunner creates a Runner. Use options to configure.
func NewRunner(provider Provider, opts ...Option) *Runner {
	r := &Runner{
		provider:    provider,
		tools:       make(map[string]Tool),
		maxSteps:    10,
		timeout:     60 * time.Second,
		retryPolicy: DefaultRetryPolicy(),
		logger:      NopLogger{},
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Run executes an agent interaction with the given prompt.
// Returns the final answer or error.
func (r *Runner) Run(ctx context.Context, prompt string) (*RunResult, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	conversation := []Message{
		{Role: "user", Content: prompt},
	}

	result := &RunResult{
		Prompt:       prompt,
		ToolCalls:    []ToolCallRecord{},
		StepCount:    0,
		StartTime:    time.Now(),
		ModelVersion: r.provider.Name(),
	}

	for step := 0; step < r.maxSteps; step++ {
		result.StepCount++
		r.logger.Debug("runtime.step", "step", step, "messages", len(conversation))

		req := Request{
			Messages: conversation,
			Tools:    r.toolsAsSpecs(),
		}

		resp, err := r.callProviderWithRetry(ctx, req)
		if err != nil {
			result.Error = err
			result.Duration = time.Since(result.StartTime)
			return result, fmt.Errorf("provider call failed at step %d: %w", step, err)
		}

		result.TokensIn += resp.InputTokens
		result.TokensOut += resp.OutputTokens

		// Parse tool calls from response
		toolCalls := resp.ToolCalls

		if len(toolCalls) == 0 {
			// No more tool calls — this is the final answer
			result.FinalAnswer = resp.Text
			result.Duration = time.Since(result.StartTime)
			return result, nil
		}

		// Append assistant's message (with tool calls) to conversation
		conversation = append(conversation, Message{
			Role:      "assistant",
			Content:   resp.Text,
			ToolCalls: toolCalls,
		})

		// Execute each tool call
		for _, tc := range toolCalls {
			toolResult, err := r.executeTool(ctx, tc)
			record := ToolCallRecord{
				ToolName:  tc.Name,
				Arguments: tc.Arguments,
				Result:    toolResult,
				Error:     err,
				Duration:  time.Since(result.StartTime),
			}
			result.ToolCalls = append(result.ToolCalls, record)

			if err != nil {
				// Tool execution failed; feed error back to LLM
				conversation = append(conversation, Message{
					Role:         "tool",
					Content:      fmt.Sprintf(`{"error": %q}`, err.Error()),
					ToolCallID:   tc.ID,
				})
				continue
			}

			resultJSON, _ := json.Marshal(toolResult)
			conversation = append(conversation, Message{
				Role:       "tool",
				Content:    string(resultJSON),
				ToolCallID: tc.ID,
			})
		}
	}

	// Hit max steps without convergence
	result.Error = ErrMaxStepsExceeded
	result.Duration = time.Since(result.StartTime)
	return result, ErrMaxStepsExceeded
}

func (r *Runner) callProviderWithRetry(ctx context.Context, req Request) (Response, error) {
	var resp Response
	var err error

	for attempt := 0; attempt <= r.retryPolicy.MaxAttempts; attempt++ {
		resp, err = r.provider.Generate(ctx, req)
		if err == nil {
			return resp, nil
		}
		if !r.retryPolicy.IsRetryable(err) || attempt == r.retryPolicy.MaxAttempts {
			return resp, err
		}
		delay := r.retryPolicy.Delay(attempt)
		r.logger.Warn("runtime.retry", "attempt", attempt, "delay", delay, "error", err)

		select {
		case <-ctx.Done():
			return resp, ctx.Err()
		case <-time.After(delay):
		}
	}
	return resp, err
}

func (r *Runner) executeTool(ctx context.Context, tc ToolCall) (any, error) {
	tool, ok := r.tools[tc.Name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownTool, tc.Name)
	}

	if err := validateArgs(tc.Arguments, tool.InputSchema); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidArgs, err)
	}

	toolCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	return tool.Handler(toolCtx, tc.Arguments)
}

func (r *Runner) toolsAsSpecs() []ToolSpec {
	specs := make([]ToolSpec, 0, len(r.tools))
	for _, t := range r.tools {
		specs = append(specs, ToolSpec{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}
	return specs
}

// Errors
var (
	ErrMaxStepsExceeded = errors.New("max steps exceeded without convergence")
	ErrUnknownTool      = errors.New("unknown tool")
	ErrInvalidArgs      = errors.New("invalid tool arguments")
)
