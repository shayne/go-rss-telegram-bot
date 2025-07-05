build:
	go build -o rssbot ./cmd/rssbot

build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -o rssbot ./cmd/rssbot

test:
	go test ./...

check:
	go build ./...
	@echo "Running staticcheck..."
	@go run honnef.co/go/tools/cmd/staticcheck@latest -- $(shell go list ./...)

goimports:
	@echo "Running goimports..."
	@find . -name "*.go" -not -path "./.devenv/*" -exec golang.org/x/tools/cmd/goimports@latest -w {} \;

llm: check goimports test
	@echo "success"

.PHONY: build build-linux-amd64 test check goimports llm