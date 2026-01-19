.PHONY: run build test migrate docker-up docker-down sqlc

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod

# Binary name
BINARY=agentdm

build:
	$(GOBUILD) -o bin/$(BINARY) ./cmd/server

run: build
	./bin/$(BINARY)

test:
	$(GOTEST) -v ./...

tidy:
	$(GOMOD) tidy

sqlc:
	cd db && sqlc generate

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

migrate-up:
	@echo "Migrations are auto-applied via docker-compose init scripts"

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/
