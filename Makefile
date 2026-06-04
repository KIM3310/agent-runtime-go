.SHELLFLAGS := -eu -o pipefail -c

.PHONY: test verify

test:
	go test ./...

verify: test
