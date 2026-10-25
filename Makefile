GO ?= go

.PHONY: run build test test-race fmt vet check

run:
	$(GO) run ./cmd/kamps

build:
	$(GO) build -o kamps ./cmd/kamps

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

fmt:
	gofmt -w .

vet:
	$(GO) vet ./...

check:
	./scripts/check.sh
