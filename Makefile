BINARY := playground
PKG    := ./cmd/playground

.PHONY: all web media docs build run test fmt vet tidy clean

all: web build

web:
	cd web/app && npm ci && npm run build

# The fixture media is committed, so this is normally a no-op that proves the
# committed bytes still match the generator. It needs ffmpeg; nothing else does.
media:
	go run ./scripts/genmedia

# The documentation site, generated from the manifest so the catalogue cannot
# drift from what the binary actually serves.
docs:
	go run ./docs/gen

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
