# SyntropyAgent-GO build script

all: agent-go

agent-go:
	go build -o sag -ldflags="-s -w" main.go

clean:
	go clean
	rm -f sag