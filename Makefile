GO ?= go

.PHONY: run build test fmt vet check

run:
	$(GO) run ./cmd/kamps

build:
	$(GO) build -o kamps ./cmd/kamps

test:
	$(GO) test ./...

fmt:
	gofmt -w .

vet:
	$(GO) vet ./...

check:
	./scripts/check.sh
