.SILENT: build run

format:
	golangci-lint run --fix

lint:
	golangci-lint run

build:
	go build -o bin/asb cmd/asb/*

run: build
	./bin/asb
