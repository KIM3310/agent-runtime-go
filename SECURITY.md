# Security Policy

## Supported Versions

Security fixes are applied to the default branch. Consumers should run the latest commit or latest tagged release when available.

## Reporting a Vulnerability

Do not open a public issue for suspected vulnerabilities. Use GitHub private vulnerability reporting if it is enabled for this repository, or contact the repository owner through their GitHub profile.

Please include:

- A clear description of the issue and affected package
- Reproduction steps or a minimal proof of concept
- Potential impact and any known mitigations
- Whether API keys, provider responses, tool calls, or logs may be exposed

## Security Expectations

- Never commit provider API keys, production traces, tool credentials, or customer prompts.
- Treat tool-call arguments and model outputs as untrusted input.
- Run local verification before merging:

```bash
go test ./...
```
