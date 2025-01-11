LDFLAGS = "-s -w"

build:
	go build -ldflags $(LDFLAGS) ./cli
