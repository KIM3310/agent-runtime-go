package tests

import (
	"context"
	"testing"

	"github.com/KIM3310/agent-runtime-go/providers/mock"
	"github.com/KIM3310/agent-runtime-go/runtime"
)

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

func TestMultiStepToolUse(t *testing.T) {
	provider := mock.NewStateMachine()

	tools := []runtime.Tool{
		{
			Name: "query_data",
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
			Name: "summarize",
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
