BINARY := gwt
PKG := ./...

.PHONY: help
help: ## List available targets with descriptions
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk -F ':.*?## ' '{printf "  \033[1m%-10s\033[0m %s\n", $$1, $$2}'

.PHONY: all
all: build ## Build the binary (default)

.PHONY: build
build: ## Compile the gwt binary
	go build -o $(BINARY) .

.PHONY: test
test: ## Run all Go tests
	go test $(PKG)

.PHONY: vet
vet: ## Run go vet
	go vet $(PKG)

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run $(PKG)

.PHONY: fmt
fmt: ## Format code and tidy modules
	gofmt -w .
	go mod tidy

.PHONY: check
check: fmt vet lint test ## Format, vet, lint, and test

.PHONY: run
run: ## Run the binary (usage: make run ARGS="add foo")
	go run . $(ARGS)

.PHONY: install
install: ## Install gwt to GOPATH/bin
	go install .

.PHONY: clean
clean: ## Remove the built binary
	rm -f $(BINARY)