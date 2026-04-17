// Example: weather agent using agent-runtime-go with mock provider.
//
// Run:
//   go run ./examples/weather_agent
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/KIM3310/agent-runtime-go/providers/mock"
	"github.com/KIM3310/agent-runtime-go/runtime"
)

func main() {
	provider := mock.NewStateMachine()

	tools := []runtime.Tool{
		{
			Name:        "query_data",
			Description: "Query department salary data",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Natural-language query",
					},
				},
				"required": []string{"query"},
			},
			Handler: func(ctx context.Context, args map[string]any) (any, error) {
				return map[string]any{
					"departments": []map[string]any{
						{"name": "Engineering", "avg_salary": 145000},
						{"name": "Sales", "avg_salary": 110000},
						{"name": "Marketing", "avg_salary": 95000},
					},
				}, nil
			},
		},
		{
			Name:        "summarize",
			Description: "Summarize data into a narrative",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"data": map[string]any{"type": "string"},
				},
				"required": []string{"data"},
			},
			Handler: func(ctx context.Context, args map[string]any) (any, error) {
				return "Engineering has the highest average salary.", nil
			},
		},
	}

	r := runtime.NewRunner(
		provider,
		runtime.WithTools(tools),
		runtime.WithMaxSteps(5),
	)

	result, err := r.Run(context.Background(), "What are the top paid departments?")
	if err != nil {
		log.Fatalf("Run failed: %v", err)
	}

	fmt.Println("=== RESULT ===")
	fmt.Printf("Prompt:       %s\n", result.Prompt)
	fmt.Printf("Final answer: %s\n", result.FinalAnswer)
	fmt.Printf("Steps:        %d\n", result.StepCount)
	fmt.Printf("Tool calls:   %d\n", len(result.ToolCalls))
	fmt.Printf("Tokens in:    %d\n", result.TokensIn)
	fmt.Printf("Tokens out:   %d\n", result.TokensOut)
	fmt.Printf("Duration:     %v\n", result.Duration)

	fmt.Println("\n=== TOOL CALLS ===")
	for i, tc := range result.ToolCalls {
		fmt.Printf("%d. %s(%v)\n", i+1, tc.ToolName, tc.Arguments)
		fmt.Printf("   result: %v\n", tc.Result)
	}
}
