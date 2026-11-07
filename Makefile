GO ?= go
BIN ?= dc2-client

.PHONY: build test vet fmt check cross-windows envembed dist

build:
	$(GO) build -o $(BIN) .

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

fmt:
	gofmt -w .

check: fmt vet test

cross-windows:
	GOOS=windows GOARCH=amd64 $(GO) build -ldflags "-s -w" -o $(BIN).exe .

# Requires a .env file in the repository root.
envembed:
	$(GO) build -tags envembed -ldflags "-s -w" -o $(BIN) .

# Cross-compiled release artifacts in dist/ with SHA256SUMS.
dist:
	./scripts/build.sh all
