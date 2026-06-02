.PHONY: run-server run-cli build clean

CONFIG_FILE_NAME ?=

run-server:
	CONFIG_FILE_NAME=$(CONFIG_FILE_NAME) go run cmd/server/main.go

run-cli:
	go run cmd/cli/main.go

build:
	go build -o bin/spider-server cmd/server/main.go
	go build -o bin/spider-cli cmd/cli/main.go

clean:
	rm -rf bin/
