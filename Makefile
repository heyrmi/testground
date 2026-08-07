BINARY := playground
PKG    := ./cmd/playground

.PHONY: all web media build run test fmt vet tidy clean

all: web build

web:
	cd web/app && npm ci && npm run build

# The fixture media is committed, so this is normally a no-op that proves the
# committed bytes still match the generator. It needs ffmpeg; nothing else does.
media:
	go run ./scripts/genmedia

build:
	go build -o $(BINARY) $(PKG)

run: build
	./$(BINARY) serve

test:
	go test ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -f $(BINARY)
	rm -rf web/app/dist dist
