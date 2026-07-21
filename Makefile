BINARY ?= azctx
PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
GO     ?= go

.DEFAULT_GOAL := help

MAKEPKG_TMP := $(or $(TMPDIR),/tmp)/azctx-makepkg

.PHONY: help all build install uninstall test lint clean arch-build arch-install arch-bump updatesums clean-arch

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
	@echo "  arch-bump      Point PKGBUILD at the latest release tag and refresh checksums"
	@echo "  updatesums     Refresh sha256sums in PKGBUILD (needs pacman-contrib)"
	@echo ""
	@echo "Examples:"
	@echo "  make install                       # /usr/local/bin (may need sudo)"
	@echo "  make install PREFIX=\$$HOME          # ~/bin"
	@echo "  make install BINDIR=~/.local/bin   # any custom bin dir"

all: build

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

build:
	$(GO) build -ldflags "-X main.version=$(VERSION)" -o $(BINARY) .

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

arch-bump:
	@ver=$$(git describe --tags --abbrev=0 | sed 's/^v//'); \
	sed -i "s/^pkgver=.*/pkgver=$$ver/; s/^pkgrel=.*/pkgrel=1/" PKGBUILD; \
	updpkgsums; \
	echo "PKGBUILD now at $$ver"

updatesums:
	updpkgsums

clean-arch:
	rm -f *.pkg.tar.zst azctx-*.tar.gz
	rm -rf $(MAKEPKG_TMP)

clean: clean-arch
	rm -f $(BINARY)
	rm -rf dist
