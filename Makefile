BINARY := playground
PKG    := ./cmd/playground

.PHONY: all web docs build run test fmt vet tidy clean

all: web build

web:
	cd web/app && npm ci && npm run build

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
