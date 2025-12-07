.SILENT: build run

format:
	golangci-lint run --fix

lint:
	golangci-lint run

build:
	go build -o bin/as cmd/as/main.go

run: build
	./bin/as
