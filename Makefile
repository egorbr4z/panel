VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X github.com/egorbr4z/panel/internal/api/handler.Version=$(VERSION)
BIN := vpanel

.PHONY: all build frontend backend run test lint tidy docker clean

all: frontend build

## build: compile the Go binary (embeds whatever is in internal/web/dist)
build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/vpanel

## frontend: build the React SPA into internal/web/dist
frontend:
	cd frontend && npm ci && npm run build

## run: run the server locally
run:
	go run ./cmd/vpanel

## test: run Go unit tests
test:
	go test ./...

## lint: vet + gofmt check
lint:
	go vet ./...
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './internal/core/*/pb/*'))" || (echo "gofmt needed"; exit 1)

## tidy: sync go.mod
tidy:
	go mod tidy

## docker: build the deployment image
docker:
	docker build -f deploy/Dockerfile -t vpanel:$(VERSION) .

clean:
	rm -f $(BIN)
	rm -rf frontend/dist internal/web/dist/assets
