.PHONY: run build test lint clean

run:
	go run ./cmd/orca

build:
	go build -o bin/orca ./cmd/orca

test:
	go test ./... -v -race -cover

lint:
	go vet ./...

clean:
	rm -rf bin/ storage/
