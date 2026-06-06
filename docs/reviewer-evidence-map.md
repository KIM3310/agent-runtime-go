# Review Guide - agent-runtime-go

Updated: 2026-05-30

Use this page as the short path through the repository. It keeps the review grounded in the code, docs, commands, and boundaries that are already present.

## Summary

| Field | Notes |
|---|---|
| Lane | B2B developer infrastructure |
| Core idea | Compact Go-native agent runtime with bounded tool execution and provider adapters. |
| Primary reader | Go backend/platform teams embedding agent execution inside services. |
| Stack | Go |

## Open First

1. Start with the README fast path and architecture section.
2. Open `docs/service-launch-playbook.md` only when reviewing the product or service angle.
3. Check the commands below before making claims about quality.
4. Skim the CI workflows and fixture data before deeper implementation review.
5. Read the boundaries section before presenting the project externally.

## Checks

| Purpose | Command |
|---|---|
| Test suite | `go test ./...` |

## CI

- .github/workflows/architecture-blueprint.yml
- .github/workflows/ci.yml
- .github/workflows/dependency-review.yml
- .github/workflows/repository-health.yml
- .github/workflows/repository-surface.yml
- .github/workflows/secret-scan.yml

## Evidence

- Go testable runtime surface
- go test ./... passes
- Technical review pack is current
- Benchmark path is documented

## Commercial Notes

| Possible offer | Working scope assumption |
|---|---|
| Embedded runtime support | Scope after buyer intake |
| Provider adapter implementation | Scope after buyer intake |
| Reliability audit for Go agent services | Scope after buyer intake |

## Boundaries

- Tool execution must remain bounded
- Secrets stay in customer env
- Provider behavior needs evals

## Useful Metrics

- Tool-call success
- Timeout/circuit-breaker coverage
- Integration time
