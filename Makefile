.PHONY: build test vet

build:
	go build ./...

test:
	go test -p 1 ./...

vet:
	go vet ./...
