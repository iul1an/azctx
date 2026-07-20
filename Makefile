BINARY  := aztx
PREFIX  := $(HOME)/bin
GO      := go

.PHONY: all build install test lint clean

all: build

build:
	$(GO) build -o $(BINARY) .

install: build
	install -d $(PREFIX)
	install -m 0755 $(BINARY) $(PREFIX)/$(BINARY)

test:
	$(GO) test ./...

lint:
	golangci-lint run

clean:
	rm -f $(BINARY)
