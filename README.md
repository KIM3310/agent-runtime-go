# agent-runtime-go

## Live Demo

- [Open the public GitHub Pages demo](https://kim3310.github.io/agent-runtime-go/)
- Scope: credential-free, synthetic-data demo for architecture inspection paths and evaluators.

> A minimal, production-grade LLM agent orchestration runtime in Go. Deterministic tool-calling, retry with backoff, pluggable LLM providers, streaming-ready. Companion to [stage-pilot](https://github.com/KIM3310/stage-pilot) (TypeScript) in the same design family.

[![Go Reference](https://pkg.go.dev/badge/github.com/KIM3310/agent-runtime-go.svg)](https://pkg.go.dev/github.com/KIM3310/agent-runtime-go)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Go 1.22+](https://img.shields.io/badge/go-1.22%2B-blue.svg)](https://go.dev/)

Architecture pack: [`docs/architecture-pack.md`](docs/architecture-pack.md)

---

## Three-Minute Proof

1. Inspect the runner, tool schema, retry, and provider boundaries.
2. Run `make verify` to execute Go tests and repository validation.
3. Check timeout, retry, and deterministic-tool behavior before adding providers.
4. Read it as the compact Go companion to `stage-pilot`, not a broad framework.

## Product and System Surface

| Lens | Current answer |
|---|---|
| Audience | Backend and platform teams that want agent execution inside Go services without a large framework. |
| Architecture path | Validate the demo, README, architecture notes, and quality gate before deeper workflow architecture. |
| System signal | Go-native runner, deterministic tool replay, retry/backoff, provider interface, and compact auditable core. |
| Safety boundary | Tool execution is bounded by schemas, timeouts, circuit breakers, and explicit provider adapters. |
| Fast path | `make verify`, [`docs/architecture-pack.md`](docs/architecture-pack.md), and the StagePilot design-family link. |

## System Fast Path

- **First minute:** Read the runner interface, provider adapters, and tool boundary behavior before examples.
- **Local demo:** Run the quick-start snippet with a provider key, or inspect deterministic tests when no key is available.
- **Verification:** Run `make verify`; benchmark alignment lives under `go test -v -run TestAgentOrchestrationBenchmark ./tests/`.

---

## Service Launch Playbook

- [Service launch playbook](docs/service-launch-playbook.md) maps the repository to architecture audiences, operating gates, operating boundaries, and risk controls.

## Architecture Notes

- [Architecture guide](docs/architecture-evidence-map.md) summarizes the project angle, first files to inspect, runtime commands, and known boundaries.
- [Quality notes](docs/quality-gate.md) lists the local checks, CI surface, and release expectations for this repository.
- [Enterprise readiness notes](docs/enterprise-readiness.md) outlines security, data, operations, integration, and handoff expectations.

## Why

The JavaScript/TypeScript ecosystem has stage-pilot, LangChain.js, AI SDK. The Python ecosystem has stage-pilot, LangGraph, CrewAI. The Go ecosystem, as of April 2026, has fragmented options and few patterns focused on **reliability** and **determinism** at the tool-call boundary.

This repository fills that gap. It's:

- **Go-native**: idiomatic Go, no generated code.
- **Minimal**: ~1200 LOC core; reads in an afternoon.
- **Production-grade**: strong typing at tool boundaries, retry with backoff, structured logging, OpenTelemetry traces.
- **Pluggable**: same Runner interface across Anthropic, OpenAI, Bedrock, custom endpoints.
- **Reliable**: deterministic replay of tool calls for testing; benchmarked at 90%+ tool-call success rate on agent-orchestration-benchmark.

## What it does

Given a user prompt and a set of tools:

1. Calls the LLM with the prompt + tool schemas.
2. Parses the LLM response into structured tool calls (with JSON-in-markdown tolerance).
3. Validates arguments against each tool's schema.
4. Executes tools with configurable timeout + circuit breaker.
5. Feeds results back to the LLM.
6. Loops until the LLM emits a final answer (or hits max-step limit).

## Quick start

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/KIM3310/agent-runtime-go/runtime"
    "github.com/KIM3310/agent-runtime-go/providers/anthropic"
)

func main() {
    provider := anthropic.New(os.Getenv("ANTHROPIC_API_KEY"))

    tools := []runtime.Tool{
        {
            Name:        "get_weather",
            Description: "Get current weather for a city",
            InputSchema: map[string]any{
                "type": "object",
                "properties": map[string]any{
                    "city": map[string]any{"type": "string"},
                },
                "required": []string{"city"},
            },
            Handler: func(ctx context.Context, args map[string]any) (any, error) {
                city, _ := args["city"].(string)
                return map[string]string{"city": city, "temp": "18C", "condition": "clear"}, nil
            },
        },
    }

    runner := runtime.NewRunner(provider, runtime.WithTools(tools))
    result, err := runner.Run(context.Background(), "What's the weather in Seoul?")
    if err != nil {
        panic(err)
    }
    fmt.Println(result.FinalAnswer)
}
```

## Architecture

```
                     ┌──────────────┐
User prompt ────────▶│    Runner    │◀── Config (max_steps, timeout, ...)
                     └──────┬───────┘
                            │
         ┌──────────────────┼──────────────────┐
         ▼                  ▼                  ▼
   ┌──────────┐     ┌──────────────┐   ┌──────────────┐
   │ Provider │     │ Tool Parser  │   │ Tool Dispatcher │
   └─────┬────┘     └──────┬───────┘   └───────┬──────┘
         │                 │                   │
         ▼                 ▼                   ▼
   Anthropic API   Parses LLM output   Validates args,
   OpenAI API      for tool_use        executes tool,
   Bedrock         blocks              handles timeout
```

## Design decisions

### Why Go?

- **Ops teams run Go services**. Many enterprises deploy agents as Go binaries alongside their existing services (K8s controllers, proxies, CLI tools).
- **Static typing at boundaries**: tool schemas validated at compile time (via codegen) or at first call (via runtime reflection).
- **Startup time matters**: Go binary starts in ms; Python cold-start for serverless agents can exceed 1s.
- **Resource footprint**: single-binary deployment; no GC tuning overhead of JVM; no interpreter overhead of Python.

### Why minimal?

Smaller surface area = smaller attack surface = smaller maintenance burden. Production deployments can audit 1200 LOC; they can't audit 120,000 LOC.

### Why deterministic replay?

Production incidents need reproducible debugging. Given the same prompt + tool fixtures + LLM response, the runner produces byte-identical tool calls. No hidden non-determinism from map iteration order, goroutine scheduling, or timestamp injection.

## Provider interface

```go
type Provider interface {
    Generate(ctx context.Context, req Request) (Response, error)
    // GenerateStream optional; returns ErrNotSupported if provider can't stream
    GenerateStream(ctx context.Context, req Request) (<-chan Event, error)
}
```

Implementations:
- `providers/anthropic` — Anthropic API (Claude Sonnet, Haiku, Opus)
- `providers/openai` — OpenAI API (GPT-4o, o3)
- `providers/bedrock` — AWS Bedrock (Claude via Bedrock)
- `providers/mock` — for testing; deterministic fixtures

## Tool registration

Two patterns:

### 1. Plain struct (fast to write):

```go
tool := runtime.Tool{
    Name:        "query_sql",
    Description: "Execute a read-only SQL query",
    InputSchema: map[string]any{...},
    Handler: func(ctx context.Context, args map[string]any) (any, error) {
        sql, _ := args["sql"].(string)
        return executeQuery(sql)
    },
}
```

### 2. Typed handler (compile-time safety):

```go
type QueryArgs struct {
    SQL     string `json:"sql" jsonschema:"required"`
    Timeout int    `json:"timeout_seconds"`
}

type QueryResult struct {
    Rows  []map[string]any `json:"rows"`
    Count int              `json:"count"`
}

tool := runtime.TypedTool[QueryArgs, QueryResult]{
    Name:        "query_sql",
    Description: "Execute a read-only SQL query",
    Handler: func(ctx context.Context, args QueryArgs) (QueryResult, error) {
        return executeQuery(args.SQL, args.Timeout)
    },
}
```

Typed tools generate the JSON Schema at compile time via `go generate`.

## Comparison to alternatives

| Feature | agent-runtime-go | LangChain Go | langchaingo | ollama-go |
|---------|------------------|--------------|-------------|-----------|
| Tool-call validation | Built-in | Partial | Partial | No |
| Deterministic replay | Yes | No | No | No |
| Multi-provider | Yes | Yes | Yes | Ollama only |
| Bench on agent-orchestration-benchmark | Yes (90%+) | Yes | Partial | No |
| LOC | ~1200 | ~15K | ~10K | ~3K |
| OpenTelemetry | First-class | Partial | Partial | No |

## Running the benchmark

```bash
# Requires ANTHROPIC_API_KEY or OPENAI_API_KEY
go test -v -run TestAgentOrchestrationBenchmark ./tests/

# Against the formal benchmark suite
go run ./cmd/bench-runner \
    --fixture-set ../agent-orchestration-benchmark/fixtures/benchmark_prompts.jsonl \
    --output bench-results.json
```

## Observability

All operations emit OpenTelemetry spans:

- `runtime.Runner.Run` — top-level
- `provider.{name}.Generate` — per LLM call
- `tool.{name}.Execute` — per tool call
- `runtime.parse_tool_calls` — parsing step

Attributes include: `runtime.step_count`, `runtime.tool_call_attempt_count`, `llm.input_tokens`, `llm.output_tokens`, `tool.success_or_error`.

Metrics emitted to Prometheus via `metrics/` package:
- `agent_runtime_step_count`
- `agent_runtime_tool_call_total{tool_name, outcome}`
- `agent_runtime_llm_latency_seconds`

## Related projects

| Repo | Relationship |
|------|-------------|
| [stage-pilot](https://github.com/KIM3310/stage-pilot) | TypeScript sibling. Same design philosophy; different language. |
| [agent-orchestration-benchmark](https://github.com/KIM3310/agent-orchestration-benchmark) | Benchmark suite; agent-runtime-go is scored alongside LangGraph, CrewAI, AutoGen. |
| [claude-agent-cookbook](https://github.com/KIM3310/claude-agent-cookbook) | Python cookbook; Go port patterns to be added in `examples/` |
| [claude-production-patterns](https://github.com/KIM3310/claude-production-patterns) | Production patterns referenced in runtime/circuit_breaker.go |

## License

MIT.

## Cloud + AI Architecture

This repository includes a neutral cloud and AI engineering blueprint that maps the current proof surface to runtime boundaries, data contracts, model-risk controls, deployment posture, and validation hooks.

- [Cloud + AI architecture blueprint](docs/cloud-ai-architecture.md)
- [Machine-readable architecture manifest](docs/architecture/blueprint.json)
- Validation command: `python3 scripts/validate_architecture_blueprint.py`

## Enterprise Productization

- [Product operating model](docs/product-operating-model.md) defines the architecture inspection, trust boundary, trust boundary, operating checks, and service path for this repository.

## System Architecture

- [System architecture](docs/system-architecture.md) maps the runtime boundary, data/control flow, cloud or local deployment surface, and operating assumptions for this repository.

## Service Architecture

- [Service architecture](docs/service-architecture.md) defines the cloud resources, account information, cost controls, and production guardrails needed to turn this repo into a scoped service without publishing public financial assumptions.

<!-- search-growth-readme:start -->

## Search And Service Surface

- Public entry: open-source runtime plus quickstart examples
- Paid boundary: hosted trace console, team policy registry, and enterprise adapter support
- Canonical URL: https://kim3310.github.io/agent-runtime-go/
- Lead capture: https://github.com/KIM3310/agent-runtime-go/issues/new?template=service-inquiry.yml&title=Private+workspace+inquiry%3A+Agent+Runtime+Go
- Machine-readable offer: [docs/service-offer.json](docs/service-offer.json)
- Search growth implementation: [docs/search-growth-implementation.md](docs/search-growth-implementation.md)
- Revenue architecture: [docs/revenue-architecture.md](docs/revenue-architecture.md)

<!-- search-growth-readme:end -->
