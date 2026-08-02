# agent-runtime-go

A small, auditable Go prototype for LLM tool orchestration. It keeps the control loop explicit: provider calls, JSON-schema validation, bounded retries, tool timeouts, and deterministic test fixtures.

[![Go Reference](https://pkg.go.dev/badge/github.com/KIM3310/agent-runtime-go.svg)](https://pkg.go.dev/github.com/KIM3310/agent-runtime-go)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Go 1.26.5+](https://img.shields.io/badge/go-1.26.5%2B-blue.svg)](https://go.dev/)

Public demo: [agent-runtime-go.pages.dev](https://agent-runtime-go.pages.dev/)

## System Overview

| Area | Current implementation |
|---|---|
| Runtime | Sequential agent loop with a configurable step limit and overall timeout |
| Tool boundary | Name lookup, JSON-schema argument validation, and a 30-second execution timeout |
| Provider resilience | Retryable 429/5xx errors with bounded exponential backoff and optional jitter |
| Providers | Anthropic, OpenAI-compatible endpoints, and deterministic mock fixtures |
| Diagnostics | Structured run result, token totals, per-tool duration, and injectable logger |
| Verification | Go tests, deterministic fixtures, repository-surface checks, and CI |

This is an inspectable reference implementation, not a hosted production runtime. The public demo and benchmark path use synthetic data and mock responses unless a caller explicitly supplies a provider key.

## Three-Minute Proof

1. Read [`runtime/runner.go`](runtime/runner.go) for the complete orchestration loop.
2. Read [`runtime/retry.go`](runtime/retry.go) for the retry contract: `MaxAttempts` means total provider calls.
3. Read [`tests/runner_test.go`](tests/runner_test.go) for step-limit, schema, retry, timing, and tool-order coverage.
4. Run `make verify` with Go 1.26.5 or newer.

## Evaluation Path

```bash
git clone https://github.com/KIM3310/agent-runtime-go.git
cd agent-runtime-go
make verify

# If Go is installed outside PATH:
make GO=/path/to/go verify
```

The local gate runs every Go package, including the command-line benchmark harness and public-site contract tests.

## What It Does

Given a prompt and a registered set of tools, the runner:

1. sends the conversation and sorted tool schemas to a provider;
2. retries only errors allowed by the configured retry policy;
3. validates every requested tool and its arguments;
4. executes each tool within a bounded context;
5. returns the tool result to the provider; and
6. stops on a final answer, timeout, provider error, or maximum-step limit.

```text
Prompt
  |
  v
Runner -----> Provider (Anthropic, OpenAI-compatible, or mock)
  ^                    |
  |                    v
  +---- tool result <- schema validation <- tool call
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/KIM3310/agent-runtime-go/providers/anthropic"
    "github.com/KIM3310/agent-runtime-go/runtime"
)

func main() {
    provider := anthropic.New(os.Getenv("ANTHROPIC_API_KEY"))
    tool := runtime.Tool{
        Name: "get_weather",
        InputSchema: map[string]any{
            "type": "object",
            "properties": map[string]any{
                "city": map[string]any{"type": "string"},
            },
            "required": []string{"city"},
        },
        Handler: func(_ context.Context, args map[string]any) (any, error) {
            return map[string]any{
                "city": args["city"], "temp": "18C", "condition": "clear",
            }, nil
        },
    }

    runner := runtime.NewRunner(provider, runtime.WithTool(tool))
    result, err := runner.Run(context.Background(), "Weather in Seoul?")
    if err != nil {
        panic(err)
    }
    fmt.Println(result.FinalAnswer)
}
```

## Runtime Contracts

### Retry semantics

`RetryPolicy.MaxAttempts` is the total number of provider calls, including the first call. Values below one are treated as one attempt. The default policy permits up to five calls for rate-limit and transient server errors.

### Determinism

Registered tools are sorted by name before every provider request. Mock fixtures make the runner loop reproducible in tests. Live model output is inherently provider-dependent, so this repository does not claim byte-identical replay of real API responses.

### Tool timing

`ToolCallRecord.Duration` measures the individual tool execution. `RunResult.Duration` measures the complete run.

### Provider interface

```go
type Provider interface {
    Name() string
    Generate(ctx context.Context, req Request) (Response, error)
}
```

Implemented adapters:

- [`providers/anthropic`](providers/anthropic)
- [`providers/openai`](providers/openai), including OpenAI-compatible endpoints
- [`providers/mock`](providers/mock) for credential-free tests

`StreamingProvider` is an extension interface in the type surface. The current runner uses synchronous `Generate`; streaming orchestration is not implemented yet.

## Benchmark Boundary

[`cmd/bench-runner`](cmd/bench-runner) reads the schema used by `agent-orchestration-benchmark` and exercises the runtime with the deterministic mock state machine. It is useful for fixture compatibility and scoring-pipeline checks. It is not a live-provider quality benchmark and does not support a model-performance claim.

```bash
go run ./cmd/bench-runner \
  -fixtures ../agent-orchestration-benchmark/fixtures/benchmark_prompts.jsonl \
  -output bench-results.json
```

## Current Limits

- No AWS Bedrock adapter.
- No circuit breaker.
- No OpenTelemetry or Prometheus instrumentation.
- No generated typed-tool API.
- No streaming execution path in `Runner`.
- No claim of production readiness without deployment-specific identity, secrets, monitoring, persistence, and recovery controls.

These are extension points, not hidden capabilities.

## Architecture Notes

- [Architecture pack](docs/architecture-pack.md)
- [Architecture evidence map](docs/architecture-evidence-map.md)
- [Quality gate](docs/quality-gate.md)
- [Enterprise readiness boundary](docs/enterprise-readiness.md)
- [Cloud + AI architecture](docs/cloud-ai-architecture.md)
- [Machine-readable architecture manifest](docs/architecture/blueprint.json)
- Validation command: `python3 scripts/validate_architecture_blueprint.py`

## Related Work

- [stage-pilot](https://github.com/KIM3310/stage-pilot): TypeScript reliability lab and attributed parser baseline.
- [agent-orchestration-benchmark](https://github.com/KIM3310/agent-orchestration-benchmark): fixture and scoring schema used by the mock harness.

## License

MIT. See [LICENSE](LICENSE).
