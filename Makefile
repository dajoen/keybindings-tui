BINARY := keybindings-tui
BINDIR := bin
PREFIX ?= $(HOME)
INSTALLDIR := $(PREFIX)/.local/bin
GOBIN ?= $(shell go env GOBIN)
ifeq ($(GOBIN),)
GOBIN := $(HOME)/go/bin
endif

.PHONY: build install install-go clean test-local-cache setup

build:
	mkdir -p $(BINDIR)
	go build -o $(BINDIR)/$(BINARY)

install: build
	mkdir -p $(INSTALLDIR)
	cp $(BINDIR)/$(BINARY) $(INSTALLDIR)/$(BINARY)

# Go-native install to your Go bin (e.g. ~/go/bin or $GOBIN)
install-go:
	GOBIN=$(GOBIN) go install ./...

clean:
	rm -f $(BINDIR)/$(BINARY)

test-local-cache:
	mkdir -p .gocache
	GOCACHE=$(CURDIR)/.gocache go test ./...

setup:
	mkdir -p .gocache
	pre-commit install
