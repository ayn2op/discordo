build:
	go build -trimpath -ldflags "-s -w" .

test:
	go test -v ./...

fmt:
	gofmt -d -e -s .

clean:
	go clean