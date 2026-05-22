.PHONY: run build tidy test test-unit test-integration test-vm

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

# MikroTik VM tests (VirtualBox; default admin@192.168.56.11:8729). See docs/testing.md
test-vm:
	go test -tags=vm ./internal/vmtest/... -count=1 -timeout=15m
