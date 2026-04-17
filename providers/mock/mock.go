// Package mock provides a deterministic mock provider for tests.
package mock

import (
	"context"
	"fmt"

	"github.com/KIM3310/agent-runtime-go/runtime"
)

// Provider is a configurable mock for testing.
// Given a sequence of canned responses, it returns them in order.
type Provider struct {
	name      string
	responses []runtime.Response
	index     int
}

// New creates a Mock provider with canned responses.
func New(name string, responses []runtime.Response) *Provider {
	return &Provider{
		name:      name,
		responses: responses,
	}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return p.name
}

// Generate returns the next canned response.
func (p *Provider) Generate(ctx context.Context, req runtime.Request) (runtime.Response, error) {
	if p.index >= len(p.responses) {
		return runtime.Response{}, fmt.Errorf("mock: no more responses (index=%d, len=%d)", p.index, len(p.responses))
	}
	resp := p.responses[p.index]
	p.index++
	return resp, nil
}

// Reset rewinds to the beginning.
func (p *Provider) Reset() {
	p.index = 0
}

// StateMachineProvider is a more realistic mock that simulates a 3-step agent:
// 1. Respond with a "query" tool call
// 2. Respond with a "summarize" tool call
// 3. Respond with a final answer
type StateMachineProvider struct {
	step int
}

// NewStateMachine creates a state-machine mock.
func NewStateMachine() *StateMachineProvider {
	return &StateMachineProvider{}
}

func (p *StateMachineProvider) Name() string { return "state-machine-mock" }

func (p *StateMachineProvider) Generate(ctx context.Context, req runtime.Request) (runtime.Response, error) {
	defer func() { p.step++ }()

	switch p.step {
	case 0:
		return runtime.Response{
			Text: "I'll look that up.",
			ToolCalls: []runtime.ToolCall{
				{
					ID:        "call_001",
					Name:      "query_data",
					Arguments: map[string]any{"query": "top departments"},
				},
			},
			InputTokens:  50,
			OutputTokens: 20,
			StopReason:   "tool_use",
		}, nil
	case 1:
		return runtime.Response{
			Text: "Now let me summarize.",
			ToolCalls: []runtime.ToolCall{
				{
					ID:        "call_002",
					Name:      "summarize",
					Arguments: map[string]any{"data": "from previous call"},
				},
			},
			InputTokens:  80,
			OutputTokens: 25,
			StopReason:   "tool_use",
		}, nil
	default:
		return runtime.Response{
			Text:         "Final answer: based on the data, Engineering is the highest-paid department.",
			ToolCalls:    nil,
			InputTokens:  100,
			OutputTokens: 40,
			StopReason:   "end_turn",
		}, nil
	}
}

// Reset rewinds the state machine.
func (p *StateMachineProvider) Reset() {
	p.step = 0
}
