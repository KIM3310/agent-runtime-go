package main

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFixturesUsesBenchmarkSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixtures.jsonl")
	line := `{"prompt_id":"p-001","user_message":"Compare revenue.","expected_tool_sequence":["query_sales_data","summarize_trend"],"answer_keywords":["revenue","change"]}` + "\n"
	if err := os.WriteFile(path, []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}

	fixtures, err := loadFixtures(path)
	if err != nil {
		t.Fatalf("loadFixtures failed: %v", err)
	}
	if len(fixtures) != 1 {
		t.Fatalf("fixture count = %d, want 1", len(fixtures))
	}

	got := fixtures[0]
	if got.ID != "p-001" || got.Prompt != "Compare revenue." {
		t.Fatalf("fixture identity parsed incorrectly: %#v", got)
	}
	if len(got.ExpectedTools) != 2 || got.ExpectedTools[1] != "summarize_trend" {
		t.Fatalf("expected tool sequence parsed incorrectly: %#v", got.ExpectedTools)
	}
	if len(got.Keywords) != 2 || got.Keywords[0] != "revenue" {
		t.Fatalf("answer keywords parsed incorrectly: %#v", got.Keywords)
	}
}

func TestSummarizeRequiresExactToolSequence(t *testing.T) {
	results := []Result{
		{
			ID:                "exact",
			ToolNames:         []string{"query_sales_data", "summarize_trend"},
			ExpectedToolNames: []string{"query_sales_data", "summarize_trend"},
		},
		{
			ID:                "extra",
			ToolNames:         []string{"query_sales_data", "summarize_trend", "send_email"},
			ExpectedToolNames: []string{"query_sales_data", "summarize_trend"},
		},
	}

	summary := summarize(results)
	if math.Abs(summary.ToolCallSuccessRate-0.5) > 0.0001 {
		t.Fatalf("tool-call success rate = %f, want 0.5", summary.ToolCallSuccessRate)
	}
}
