BINARY := playground
PKG    := ./cmd/playground

.PHONY: all web build run test fmt vet tidy clean

all: web build

web:
	cd web/app && npm ci && npm run build

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
