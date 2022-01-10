build:
	go build -trimpath -ldflags "-s -w" .

test:
	go test -v ./...

clean:
	go clean