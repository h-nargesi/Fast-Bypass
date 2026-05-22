.PHONY: run build tidy test test-unit test-integration

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

tidy:
	go mod tidy

test:
	go test ./...

test-unit:
	go test ./internal/owner/... ./internal/quota/... ./internal/password/... ./internal/mikrotik/... ./internal/store/... ./internal/auth/...

test-integration:
	go test ./internal/integration/... -count=1

test-acceptance:
	go test ./internal/integration/... -run Acceptance -count=1
