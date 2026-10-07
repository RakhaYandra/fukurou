.PHONY: build run vet test race lint clean

build:
	go build -o bin/fukurou ./cmd/fukurou

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
