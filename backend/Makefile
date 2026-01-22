.PHONY: run build test docker-up docker-down sqlc dev

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

# Start local development environment
dev: docker-up
	@echo "Waiting for MySQL to be ready..."
	@sleep 5
	$(MAKE) run

# Run with AutoDM enabled (requires OPENAI_API_KEY)
run-autodm: build
	AUTODM_ENABLED=true ./bin/$(BINARY)
