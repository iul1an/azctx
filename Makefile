BINARY ?= aztx
PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
GO     ?= go

.PHONY: all build install uninstall test lint clean

all: build

build:
	$(GO) build -o $(BINARY) .

install: build
	install -d $(DESTDIR)$(BINDIR)
	install -m 0755 $(BINARY) $(DESTDIR)$(BINDIR)/$(BINARY)

uninstall:
	rm -f $(DESTDIR)$(BINDIR)/$(BINARY)

test:
	$(GO) test ./...

lint:
	golangci-lint run

clean:
	rm -f $(BINARY)
	rm -rf dist
