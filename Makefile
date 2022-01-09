build:
	go build -trimpath -ldflags "-s -w" .

clean:
	go clean