BINARY ?= azctx
PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
GO     ?= go

.DEFAULT_GOAL := help

MAKEPKG_TMP := $(or $(TMPDIR),/tmp)/azctx-makepkg

.PHONY: help all build install uninstall test lint clean arch-build arch-install updatesums clean-arch

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
	@echo "  arch-build     Build an Arch Linux package (.pkg.tar.zst)"
	@echo "  arch-install   Build and install the Arch package with pacman"
	@echo "  updatesums     Refresh sha256sums in PKGBUILD (needs pacman-contrib)"
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

arch-build:
	BUILDDIR=$(MAKEPKG_TMP) SRCDEST=$(MAKEPKG_TMP) PKGDEST=$(CURDIR) makepkg -f

arch-install:
	BUILDDIR=$(MAKEPKG_TMP) SRCDEST=$(MAKEPKG_TMP) PKGDEST=$(CURDIR) makepkg -sif

updatesums:
	updpkgsums

clean-arch:
	rm -f *.pkg.tar.zst azctx-*.tar.gz
	rm -rf $(MAKEPKG_TMP)

clean: clean-arch
	rm -f $(BINARY)
	rm -rf dist
