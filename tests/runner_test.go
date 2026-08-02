package tests

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/KIM3310/agent-runtime-go/providers/mock"
	"github.com/KIM3310/agent-runtime-go/runtime"
)

type requestCapturingProvider struct {
	requests []runtime.Request
}

type failingProvider struct {
	calls int
	err   error
}

func (p *failingProvider) Name() string { return "failing-provider" }

func (p *failingProvider) Generate(_ context.Context, _ runtime.Request) (runtime.Response, error) {
	p.calls++
	return runtime.Response{}, p.err
}

type delayedToolProvider struct {
	calls int
}

func (p *delayedToolProvider) Name() string { return "delayed-tool-provider" }

func (p *delayedToolProvider) Generate(_ context.Context, _ runtime.Request) (runtime.Response, error) {
	p.calls++
	if p.calls == 1 {
		time.Sleep(60 * time.Millisecond)
		return runtime.Response{
			Text: "use tool",
			ToolCalls: []runtime.ToolCall{
				{ID: "call-1", Name: "quick_tool", Arguments: map[string]any{}},
			},
			StopReason: "tool_use",
		}, nil
	}
	return runtime.Response{Text: "done", StopReason: "end_turn"}, nil
}

func (p *requestCapturingProvider) Name() string {
	return "request-capturing-provider"
}

func (p *requestCapturingProvider) Generate(_ context.Context, req runtime.Request) (runtime.Response, error) {
	p.requests = append(p.requests, req)
	return runtime.Response{Text: "done", StopReason: "end_turn"}, nil
}

func TestToolSpecsAreDeterministicallySorted(t *testing.T) {
	provider := &requestCapturingProvider{}
	tools := []runtime.Tool{
		{Name: "zeta", InputSchema: map[string]any{"type": "object"}},
		{Name: "alpha", InputSchema: map[string]any{"type": "object"}},
		{Name: "middle", InputSchema: map[string]any{"type": "object"}},
	}
	runner := runtime.NewRunner(provider, runtime.WithTools(tools))

	for range 20 {
		if _, err := runner.Run(context.Background(), "sort tools"); err != nil {
			t.Fatalf("Run failed: %v", err)
		}
	}

	want := []string{"alpha", "middle", "zeta"}
	for i, req := range provider.requests {
		got := make([]string, 0, len(req.Tools))
		for _, spec := range req.Tools {
			got = append(got, spec.Name)
		}
		if !slices.Equal(got, want) {
			t.Fatalf("request %d tool order = %v, want %v", i, got, want)
		}
	}
}

func TestBasicRun(t *testing.T) {
	responses := []runtime.Response{
		{
			Text:         "The answer is 42.",
			ToolCalls:    nil,
			InputTokens:  10,
			OutputTokens: 5,
			StopReason:   "end_turn",
		},
	}
	provider := mock.New("test", responses)

	runner := runtime.NewRunner(provider)

	result, err := runner.Run(context.Background(), "What is the meaning of life?")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.FinalAnswer != "The answer is 42." {
		t.Errorf("unexpected final answer: %q", result.FinalAnswer)
	}
	if result.StepCount != 1 {
		t.Errorf("expected 1 step, got %d", result.StepCount)
	}
	if result.TokensIn != 10 {
		t.Errorf("expected 10 input tokens, got %d", result.TokensIn)
	}
}

func TestRetryPolicyMaxAttemptsCountsTotalProviderCalls(t *testing.T) {
	provider := &failingProvider{err: &runtime.APIStatusError{StatusCode: 503, Msg: "unavailable"}}
	policy := runtime.RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   0,
		MaxDelay:    0,
		IsRetryable: func(error) bool { return true },
	}
	runner := runtime.NewRunner(provider, runtime.WithRetryPolicy(policy))

	_, err := runner.Run(context.Background(), "retry")
	if err == nil {
		t.Fatal("expected provider error")
	}
	if provider.calls != 3 {
		t.Fatalf("provider calls = %d, want 3 total attempts", provider.calls)
	}
}

func TestNonRetryableProviderErrorIsNotRetried(t *testing.T) {
	provider := &failingProvider{err: errors.New("invalid request")}
	policy := runtime.RetryPolicy{
		MaxAttempts: 5,
		BaseDelay:   0,
		MaxDelay:    0,
		IsRetryable: func(error) bool { return false },
	}
	runner := runtime.NewRunner(provider, runtime.WithRetryPolicy(policy))

	_, err := runner.Run(context.Background(), "do not retry")
	if err == nil {
		t.Fatal("expected provider error")
	}
	if provider.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.calls)
	}
}

func TestZeroRetryPolicyFallsBackToOneAttempt(t *testing.T) {
	provider := &failingProvider{err: errors.New("invalid request")}
	runner := runtime.NewRunner(provider, runtime.WithRetryPolicy(runtime.RetryPolicy{}))

	_, err := runner.Run(context.Background(), "one attempt")
	if err == nil {
		t.Fatal("expected provider error")
	}
	if provider.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.calls)
	}
}

func TestToolCallDurationMeasuresOnlyToolExecution(t *testing.T) {
	provider := &delayedToolProvider{}
	tool := runtime.Tool{
		Name:        "quick_tool",
		InputSchema: map[string]any{"type": "object"},
		Handler: func(context.Context, map[string]any) (any, error) {
			return "ok", nil
		},
	}
	runner := runtime.NewRunner(provider, runtime.WithTool(tool))

	result, err := runner.Run(context.Background(), "measure tool")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if len(result.ToolCalls) != 1 {
		t.Fatalf("tool call count = %d, want 1", len(result.ToolCalls))
	}
	if got := result.ToolCalls[0].Duration; got >= 30*time.Millisecond {
		t.Fatalf("tool duration = %v; appears to include provider latency", got)
	}
}

func TestMultiStepToolUse(t *testing.T) {
	provider := mock.NewStateMachine()

	tools := []runtime.Tool{
		{
			Name: "query_sales_data",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"query": map[string]any{"type": "string"}},
				"required":   []string{"query"},
			},
			Handler: func(ctx context.Context, args map[string]any) (any, error) {
				return map[string]any{"data": "some data"}, nil
			},
		},
		{
			Name: "summarize_trend",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"data": map[string]any{"type": "string"}},
				"required":   []string{"data"},
			},
			Handler: func(ctx context.Context, args map[string]any) (any, error) {
				return "summary text", nil
			},
		},
	}

	runner := runtime.NewRunner(provider, runtime.WithTools(tools), runtime.WithMaxSteps(5))

	result, err := runner.Run(context.Background(), "Analyze data")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.StepCount != 3 {
		t.Errorf("expected 3 steps, got %d", result.StepCount)
	}
	if len(result.ToolCalls) != 2 {
		t.Errorf("expected 2 tool calls, got %d", len(result.ToolCalls))
	}
	for i, tc := range result.ToolCalls {
		if tc.Error != nil {
			t.Errorf("tool call %d returned error: %v", i, tc.Error)
		}
	}
}

func TestStateMachineFinalAnswerIsSubstantiatedByToolOutputs(t *testing.T) {
	provider := mock.NewStateMachine()

	tools := []runtime.Tool{
		{
			Name: "query_sales_data",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"query": map[string]any{"type": "string"}},
				"required":   []string{"query"},
			},
			Handler: func(ctx context.Context, args map[string]any) (any, error) {
				return []map[string]any{
					{"quarter": "Q4 2023", "revenue_k_usd": 2410.0},
					{"quarter": "Q1 2024", "revenue_k_usd": 2680.0},
				}, nil
			},
		},
		{
			Name: "summarize_trend",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"data": map[string]any{"type": "string"}},
				"required":   []string{"data"},
			},
			Handler: func(ctx context.Context, args map[string]any) (any, error) {
				before := 2410.0
				after := 2680.0
				change := after - before
				pct := change / before * 100
				return fmt.Sprintf("Revenue increased by %.1f%%, a change of $%.1fk.", pct, change), nil
			},
		},
	}

	runner := runtime.NewRunner(provider, runtime.WithTools(tools), runtime.WithMaxSteps(5))
	result, err := runner.Run(context.Background(), "Compare revenue")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if len(result.ToolCalls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(result.ToolCalls))
	}

	summary, ok := result.ToolCalls[1].Result.(string)
	if !ok {
		t.Fatalf("summarize_trend result = %#v, want string", result.ToolCalls[1].Result)
	}
	for _, expected := range []string{"revenue increased by 11.2%", "$270.0k"} {
		if !strings.Contains(strings.ToLower(summary), strings.ToLower(expected)) {
			t.Fatalf("summary %q does not contain %q", summary, expected)
		}
		if !strings.Contains(strings.ToLower(result.FinalAnswer), strings.ToLower(expected)) {
			t.Fatalf("final answer %q does not contain %q", result.FinalAnswer, expected)
		}
	}
}

func TestMaxStepsExceeded(t *testing.T) {
	// Provider keeps returning tool calls forever
	provider := mock.New("loop", []runtime.Response{
		{
			Text: "step",
			ToolCalls: []runtime.ToolCall{
				{ID: "1", Name: "noop", Arguments: map[string]any{}},
			},
			StopReason: "tool_use",
		},
		{
			Text: "step",
			ToolCalls: []runtime.ToolCall{
				{ID: "2", Name: "noop", Arguments: map[string]any{}},
			},
			StopReason: "tool_use",
		},
		{
			Text: "step",
			ToolCalls: []runtime.ToolCall{
				{ID: "3", Name: "noop", Arguments: map[string]any{}},
			},
			StopReason: "tool_use",
		},
	})

	tool := runtime.Tool{
		Name:        "noop",
		InputSchema: map[string]any{"type": "object"},
		Handler: func(ctx context.Context, args map[string]any) (any, error) {
			return nil, nil
		},
	}

	runner := runtime.NewRunner(provider, runtime.WithTool(tool), runtime.WithMaxSteps(2))

	_, err := runner.Run(context.Background(), "loop")
	if err != runtime.ErrMaxStepsExceeded {
		t.Errorf("expected ErrMaxStepsExceeded, got %v", err)
	}
}

func TestUnknownTool(t *testing.T) {
	provider := mock.New("unknown_tool", []runtime.Response{
		{
			Text: "calling unknown tool",
			ToolCalls: []runtime.ToolCall{
				{ID: "1", Name: "does_not_exist", Arguments: map[string]any{}},
			},
			StopReason: "tool_use",
		},
		{
			Text:       "Final answer after tool error",
			ToolCalls:  nil,
			StopReason: "end_turn",
		},
	})

	runner := runtime.NewRunner(provider, runtime.WithMaxSteps(5))

	result, err := runner.Run(context.Background(), "unknown")
	if err != nil {
		t.Fatalf("expected graceful handling, got error: %v", err)
	}
	if len(result.ToolCalls) != 1 {
		t.Errorf("expected 1 tool call record, got %d", len(result.ToolCalls))
	}
	if result.ToolCalls[0].Error == nil {
		t.Errorf("expected tool error to be recorded")
	}
}

func TestArgValidation(t *testing.T) {
	provider := mock.New("bad_args", []runtime.Response{
		{
			Text: "calling with bad args",
			ToolCalls: []runtime.ToolCall{
				{ID: "1", Name: "strict_tool", Arguments: map[string]any{"wrong_key": "value"}},
			},
			StopReason: "tool_use",
		},
		{
			Text:       "Final answer after validation error",
			StopReason: "end_turn",
		},
	})

	tool := runtime.Tool{
		Name: "strict_tool",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"required_key": map[string]any{"type": "string"},
			},
			"required": []string{"required_key"},
		},
		Handler: func(ctx context.Context, args map[string]any) (any, error) {
			return "should not reach here", nil
		},
	}

	runner := runtime.NewRunner(provider, runtime.WithTool(tool), runtime.WithMaxSteps(5))
	result, err := runner.Run(context.Background(), "validate")
	if err != nil {
		t.Fatalf("expected graceful handling, got: %v", err)
	}
	if result.ToolCalls[0].Error == nil {
		t.Error("expected validation error to be recorded")
	}
}
