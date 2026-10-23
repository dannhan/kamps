#!/usr/bin/env bash
set -euo pipefail

unformatted="$(gofmt -l .)"
if [[ -n "$unformatted" ]]; then
	echo "gofmt needed:"
	echo "$unformatted"
	exit 1
fi

go vet ./...
go test ./...
