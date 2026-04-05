.PHONY: build lint test

build:
	go build -o lwts-cli .

lint:
	golangci-lint run ./...

test:
	go test ./...
