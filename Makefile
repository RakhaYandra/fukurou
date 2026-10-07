.PHONY: build run vet test race lint clean install

PREFIX ?= $(HOME)/.local
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o bin/fukurou ./cmd/fukurou

release:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/fukurou-linux-amd64 ./cmd/fukurou

install: build
	install -Dm755 bin/fukurou $(PREFIX)/bin/fukurou

run: build
	./bin/fukurou --debug

vet:
	go vet ./...

test:
	go test ./...

race:
	go test -race ./...

clean:
	rm -rf bin/
