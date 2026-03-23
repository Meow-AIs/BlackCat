BINARY=blackcat
VERSION?=0.1.0
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)"

.PHONY: build build-all test test-integration test-e2e bench lint clean fmt vet

build:
	CGO_ENABLED=1 go build $(LDFLAGS) -o $(BINARY) ./cmd/blackcat/

build-all:
	# Linux amd64
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 CC="zig cc -target x86_64-linux-gnu" \
		go build $(LDFLAGS) -o $(BINARY)-linux-amd64 ./cmd/blackcat/
	# Linux arm64
	GOOS=linux GOARCH=arm64 CGO_ENABLED=1 CC="zig cc -target aarch64-linux-gnu" \
		go build $(LDFLAGS) -o $(BINARY)-linux-arm64 ./cmd/blackcat/
	# macOS arm64
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 CC="zig cc -target aarch64-macos" \
		go build $(LDFLAGS) -o $(BINARY)-darwin-arm64 ./cmd/blackcat/
	# macOS amd64
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 CC="zig cc -target x86_64-macos" \
		go build $(LDFLAGS) -o $(BINARY)-darwin-amd64 ./cmd/blackcat/
	# Windows amd64
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
		go build $(LDFLAGS) -o $(BINARY)-windows-amd64.exe ./cmd/blackcat/

test:
	CGO_ENABLED=1 go test ./... -count=1 -short

test-integration:
	CGO_ENABLED=1 go test ./tests/integration/... -count=1 -v

test-e2e:
	CGO_ENABLED=1 go test ./tests/e2e/... -count=1 -v

bench:
	CGO_ENABLED=1 go test ./... -bench=. -benchmem -run=^$$

lint: vet fmt
	@echo "Lint passed."

vet:
	go vet ./...

fmt:
	@test -z "$$(gofmt -l .)" || (echo "Files need formatting:" && gofmt -l . && exit 1)

clean:
	rm -f $(BINARY) $(BINARY)-* $(BINARY).exe
	rm -f cover.out cover_new.out coverage.out
