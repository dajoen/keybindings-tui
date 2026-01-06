BINARY := keybindings-tui
PREFIX ?= $(HOME)
BINDIR := $(PREFIX)/.local/bin

.PHONY: build install clean

build:
	go build -o $(BINARY)

install: build
	mkdir -p $(BINDIR)
	cp $(BINARY) $(BINDIR)/$(BINARY)

clean:
	rm -f $(BINARY)
