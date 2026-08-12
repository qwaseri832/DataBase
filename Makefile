.PHONY: build run-server run-cli test lint fmt clean

CONFIG ?= config.yml

build:
	go build -o bin/spider-server ./cmd/server
	go build -o bin/spider-cli ./cmd/cli

run-server:
	go run ./cmd/server -config $(CONFIG)

run-cli:
	go run ./cmd/cli

test:
	go test -race -count=1 ./...

lint:
	go vet ./...
	gofmt -l .

fmt:
	gofmt -w .

clean:
	rm -rf bin/ data/ spider.log
