BINARY ?= azctx
PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
GO     ?= go

.DEFAULT_GOAL := help

.PHONY: help all build install uninstall test lint clean

help:
	@echo "azctx — Azure-CLI context switcher"
	@echo ""
	@echo "Targets:"
	@echo "  build       Build the $(BINARY) binary in the current directory"
	@echo "  install     Install to \$$BINDIR (currently: $(BINDIR))"
	@echo "  uninstall   Remove the installed binary"
	@echo "  test        Run go test ./..."
	@echo "  lint        Run golangci-lint"
	@echo "  clean       Remove build artifacts"
	@echo ""
	@echo "Examples:"
	@echo "  make install                       # /usr/local/bin (may need sudo)"
	@echo "  make install PREFIX=\$$HOME          # ~/bin"
	@echo "  make install BINDIR=~/.local/bin   # any custom bin dir"

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
