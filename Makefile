.PHONY: run build test lint clean

run:
	go run ./cmd/paylock

build:
	go build -o bin/paylock ./cmd/paylock

test:
	go test ./... -v -race -cover

lint:
	go vet ./...

clean:
	rm -rf bin/
