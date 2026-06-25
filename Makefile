.SHELLFLAGS := -eu -o pipefail -c

GO ?= go
GO_MIN_VERSION := 1.22

.PHONY: check-go test verify

check-go:
	@if ! command -v "$(GO)" >/dev/null 2>&1; then \
		echo "Go $(GO_MIN_VERSION)+ is required. Install Go or run: make GO=/path/to/go <target>" >&2; \
		exit 1; \
	fi
	@version="$$("$(GO)" env GOVERSION)"; \
	major_minor="$${version#go}"; \
	major="$${major_minor%%.*}"; \
	minor_rest="$${major_minor#*.}"; \
	minor="$${minor_rest%%.*}"; \
	if [ "$$major" -lt 1 ] || { [ "$$major" -eq 1 ] && [ "$$minor" -lt 22 ]; }; then \
		echo "Go $(GO_MIN_VERSION)+ is required; found $$version at $(GO)." >&2; \
		exit 1; \
	fi

test: check-go
	$(GO) test ./...

verify: test
