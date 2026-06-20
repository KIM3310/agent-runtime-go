# Enterprise Readiness Notes - agent-runtime-go

Updated: 2026-05-30

This note defines what an enterprise architecture inspection, public-sector operator, serious user, or technical evaluator can safely infer from this repository today. It is intentionally conservative: public proof is separated from production claims.

## Scope

| Field | Notes |
|---|---|
| Repository | `agent-runtime-go` |
| Lane | B2B developer infrastructure |
| Primary reader | Go backend/platform teams embedding agent execution inside services. |
| Core wedge | Compact Go-native agent runtime with bounded tool execution and provider adapters. |
| Stack | Go |
| Readiness posture | Pilot-ready technical surface; production use requires customer-specific identity, monitoring, data, and support controls. |

## Enterprise Controls

| Control | Current expectation |
|---|---|
| Data boundary | Public artifacts should use demo, fixture, or synthetic data until the architecture inspection approves data handling, retention, and access controls. |
| Identity and access | Production pilots should add SSO/OIDC, RBAC, scoped service accounts, secret rotation, and admin-visible access architectures. |
| Auditability | Keep decision logs, generated reports, CI results, eval outputs, and operator handoff artifacts inspectable. |
| Observability | Track health checks, latency, error budget, cost, eval pass rate, audit-log completeness, and handoff/report generation status. |
| Release gate | Test suite: go test ./... |
| Support handoff | Name the owner, escalation path, rollback path, known limits, and architecture cadence before a production testing. |

## Verification Surface

| Purpose | Command |
|---|---|
| Test suite | `go test ./...` |

## CI Surface

- .github/workflows/architecture-blueprint.yml
- .github/workflows/ci.yml
- .github/workflows/dependency-architecture.yml
- .github/workflows/repository-health.yml
- .github/workflows/repository-surface.yml
- .github/workflows/secret-scan.yml

## Acceptance Criteria

- go test ./... can be run or the equivalent CI gate is visible.
- README, architecture guide, quality notes, service model, and this readiness note agree on the same scope.
- Demo, fixture, synthetic, or public-data boundaries are explicit before an architecture inspection sees outputs.
- A architecture inspection can identify the first useful outcome without reading implementation details.
- Production claims stay behind customer-specific validation, access control, monitoring, and support handoff.

## Integration Path

- Run a synthetic-data walkthrough with the architecture inspection and document the acceptance criteria.
- Scope a controlled pilot using approved data, named users, secrets, and rollback paths.
- Convert the pilot into an operating handoff with monitoring, architecture cadence, support owner, and renewal metric.

## Proof Points

- go test ./... passes
- Architecture pack is current
- Benchmark path is documented

## Operating Metrics

- Tool-call success
- Timeout/circuit-breaker coverage
- Integration time

## Open Risks

- Tool execution must remain bounded
- Secrets stay in customer env
- Provider behavior needs evals

## Finish Line

- Keep the public repository honest, runnable, and easy to architecture.
- Keep sensitive data, secrets, private tenant details, and unsupported claims out of public artifacts.
- Treat this repository as a proof surface until an approved pilot defines users, data, access, monitoring, support, and success metrics.
