// Example: real-API agent using agent-runtime-go with Anthropic or OpenAI.
//
// Run:
//
//	ANTHROPIC_API_KEY=sk-... go run ./examples/real_api_agent -provider anthropic
//	OPENAI_API_KEY=sk-...    go run ./examples/real_api_agent -provider openai
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/KIM3310/agent-runtime-go/providers/anthropic"
	"github.com/KIM3310/agent-runtime-go/providers/openai"
	"github.com/KIM3310/agent-runtime-go/runtime"
)

func main() {
	providerName := flag.String("provider", "anthropic", "anthropic or openai")
	prompt := flag.String("prompt", "What is the weather in Seoul and Tokyo?", "user prompt")
	flag.Parse()

	var provider runtime.Provider
	switch *providerName {
	case "anthropic":
		key := os.Getenv("ANTHROPIC_API_KEY")
		if key == "" {
			log.Fatal("ANTHROPIC_API_KEY required")
		}
		provider = anthropic.New(key)
	case "openai":
		key := os.Getenv("OPENAI_API_KEY")
		if key == "" {
			log.Fatal("OPENAI_API_KEY required")
		}
		provider = openai.New(key)
	default:
		log.Fatalf("unknown provider: %s", *providerName)
	}

	tools := []runtime.Tool{
		{
			Name:        "get_weather",
			Description: "Get current weather for a city. Returns temperature and condition.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"city": map[string]any{
						"type":        "string",
						"description": "City name, e.g., 'Seoul' or 'Tokyo'",
					},
				},
				"required": []string{"city"},
			},
			Handler: func(ctx context.Context, args map[string]any) (any, error) {
				city, _ := args["city"].(string)
				// Mock: return deterministic data
				weather := map[string]map[string]any{
					"Seoul":    {"temp_c": 18, "condition": "partly cloudy"},
					"Tokyo":    {"temp_c": 20, "condition": "clear"},
					"New York": {"temp_c": 12, "condition": "rain"},
				}
				if w, ok := weather[city]; ok {
					w["city"] = city
					return w, nil
				}
				return map[string]any{"city": city, "temp_c": 15, "condition": "unknown"}, nil
			},
		},
	}

	r := runtime.NewRunner(
		provider,
		runtime.WithTools(tools),
		runtime.WithMaxSteps(5),
	)

	result, err := r.Run(context.Background(), *prompt)
	if err != nil {
		log.Fatalf("Run failed: %v", err)
	}

	fmt.Println("=== AGENT RUN ===")
	fmt.Printf("Provider:     %s\n", provider.Name())
	fmt.Printf("Prompt:       %s\n", result.Prompt)
	fmt.Printf("Final answer: %s\n", result.FinalAnswer)
	fmt.Printf("Steps:        %d\n", result.StepCount)
	fmt.Printf("Tool calls:   %d\n", len(result.ToolCalls))
	fmt.Printf("Tokens in:    %d\n", result.TokensIn)
	fmt.Printf("Tokens out:   %d\n", result.TokensOut)
	fmt.Printf("Duration:     %v\n", result.Duration)

	fmt.Println("\n=== TOOL CALLS ===")
	for i, tc := range result.ToolCalls {
		argsJSON, _ := json.Marshal(tc.Arguments)
		fmt.Printf("%d. %s(%s)\n", i+1, tc.ToolName, argsJSON)
		if tc.Error != nil {
			fmt.Printf("   ERROR: %v\n", tc.Error)
		} else {
			resultJSON, _ := json.Marshal(tc.Result)
			fmt.Printf("   → %s\n", resultJSON)
		}
	}
}
