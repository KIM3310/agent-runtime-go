.SHELLFLAGS := -eu -o pipefail -c

GO ?= go
GO_MIN_VERSION := 1.26.5

.PHONY: check-go test verify deploy-pages

check-go:
	@if ! command -v "$(GO)" >/dev/null 2>&1; then \
		echo "Go $(GO_MIN_VERSION)+ is required. Install Go or run: make GO=/path/to/go <target>" >&2; \
		exit 1; \
	fi
	@version="$$("$(GO)" env GOVERSION)"; \
	stable="$${version#go}"; \
	case "$$stable" in \
		*[!0-9.]*|*.*.*.*) echo "Stable Go $(GO_MIN_VERSION)+ is required; found $$version at $(GO)." >&2; exit 1 ;; \
	esac; \
	major="$${stable%%.*}"; \
	minor_rest="$${stable#*.}"; \
	minor="$${minor_rest%%.*}"; \
	patch="$${minor_rest#*.}"; \
	if [ "$$minor_rest" = "$$stable" ] || [ "$$patch" = "$$minor_rest" ] || [ -z "$$major" ] || [ -z "$$minor" ] || [ -z "$$patch" ]; then \
		echo "Stable Go $(GO_MIN_VERSION)+ is required; found $$version at $(GO)." >&2; \
		exit 1; \
	fi; \
	if [ "$$major" -lt 1 ] || { [ "$$major" -eq 1 ] && { [ "$$minor" -lt 26 ] || { [ "$$minor" -eq 26 ] && [ "$$patch" -lt 5 ]; }; }; }; then \
		echo "Go $(GO_MIN_VERSION)+ is required; found $$version at $(GO)." >&2; \
		exit 1; \
	fi

test: check-go
	$(GO) test ./...

verify: test

deploy-pages:
	npx --yes wrangler@latest pages deploy site --project-name agent-runtime-go
