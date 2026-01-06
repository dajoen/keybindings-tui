BINARY := keybindings-tui
PREFIX ?= $(HOME)
BINDIR := $(PREFIX)/.local/bin
GOBIN ?= $(shell go env GOBIN)
ifeq ($(GOBIN),)
GOBIN := $(HOME)/go/bin
endif

.PHONY: build install install-go clean

build:
	go build -o $(BINARY)

install: build
	mkdir -p $(BINDIR)
	cp $(BINARY) $(BINDIR)/$(BINARY)

# Go-native install to your Go bin (e.g. ~/go/bin or $GOBIN)
install-go:
	GOBIN=$(GOBIN) go install ./...

clean:
	rm -f $(BINARY)
