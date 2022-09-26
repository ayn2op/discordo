FLAGS=-ldflags "-s -w"

.PHONY: all
all: clean fmt build

.PHONY: clean
clean:
	go clean

.PHONY: fmt
fmt:
	gofmt -d -e -s .

.PHONY: build
build:
	go build $(FLAGS)
