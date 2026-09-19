MAKEFLAGS += --silent

.PHONY: build vet test test-race lint vulncheck check tidy

build:
	go build ./...

vet:
	go vet ./...

test:
	go test ./...

test-race:
	go test ./... -race

lint:
	golangci-lint run ./...

vulncheck:
	govulncheck ./...

# Everything CI runs, in one place for local use before pushing.
check: build vet test-race lint vulncheck

tidy:
	go mod tidy
