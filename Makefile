# SyntropyAgent-GO build script

FULL_VERSION = $(shell git describe --tags --dirty --candidates=1)
VERSION = $(shell echo $(FULL_VERSION) | cut -d "-" -f1)
ifeq ($(FULL_VERSION), $(VERSION))
SUBVERSION := ""
else
SUBVERSION = $(shell echo $(FULL_VERSION) | cut -d "-" -f2-4)
endif
ifeq ($(VERSION), "")
VERSION := "0.0.0"
endif

all: agent-go

agent-go:
	@echo Building $`sag$`  $(VERSION) - $(SUBVERSION)
	go build -o sag -ldflags \
		"-X github.com/SyntropyNet/syntropy-agent-go/internal/config.version=$(VERSION) \
		-X github.com/SyntropyNet/syntropy-agent-go/internal/config.subversion=$(SUBVERSION) -s -w" \
		./cmd/main.go

test:
	go test ./...

clean:
	go clean
	rm -f sag