.PHONY: build
build:
	go build \
		-trimpath \
		-buildmode=pie \
		-ldflags "-s -w" \
		.

.PHONY: test
test:
	go test -v ./...

.PHONY: fmt
fmt:
	gofmt -d -e -s .

.PHONY: clean
clean:
	go clean
