package tests

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

const agentRuntimeInquiryURL = "https://kim3310-doeon-kim-portfolio.pages.dev/?offer=agent-runtime-go&inquiry=agent-reliability-audit#private-inquiry"

func TestServiceOfferUsesCentralAgentReliabilityAuditLane(t *testing.T) {
	for _, path := range []string{"../docs/service-offer.json", "../site/service-offer.json"} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			var offer map[string]any
			if err := json.Unmarshal(raw, &offer); err != nil {
				t.Fatalf("service offer is not valid JSON: %v", err)
			}

			if got := offer["lead_capture_url"]; got != agentRuntimeInquiryURL {
				t.Fatalf("lead_capture_url = %v, want %s", got, agentRuntimeInquiryURL)
			}

			commerce := offer["commerce"].(map[string]any)
			if got := commerce["lane_id"]; got != "agent-reliability-audit" {
				t.Fatalf("lane_id = %v, want agent-reliability-audit", got)
			}
			if got := commerce["lane_name"]; got != "Agent Reliability Audit" {
				t.Fatalf("lane_name = %v, want Agent Reliability Audit", got)
			}

			structuredData := offer["structured_data"].(map[string]any)
			offers := structuredData["offers"].([]any)
			paidOffer := offers[1].(map[string]any)
			if got := paidOffer["name"]; got != "fixed-scope Agent Reliability Audit" {
				t.Fatalf("paid JSON-LD offer name = %v", got)
			}
			if got := paidOffer["url"]; got != agentRuntimeInquiryURL {
				t.Fatalf("paid JSON-LD offer URL = %v, want %s", got, agentRuntimeInquiryURL)
			}
		})
	}
}

func TestPublicSiteStatesSyntheticDemoBoundary(t *testing.T) {
	raw, err := os.ReadFile("../site/index.html")
	if err != nil {
		t.Fatal(err)
	}
	html := string(raw)

	for _, want := range []string{
		"Request private audit",
		"Try synthetic demo",
		"fixed-scope Agent Reliability Audit",
		"credential-free and synthetic",
		agentRuntimeInquiryURL,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("site/index.html missing %q", want)
		}
	}

	for _, disallowed := range []string{
		"hosted trace console",
		"team policy registry",
		"View paid options",
	} {
		if strings.Contains(html, disallowed) {
			t.Fatalf("site/index.html still contains disallowed claim %q", disallowed)
		}
	}
}
