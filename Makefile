APP        := gscan
MODULE     := github.com/adikhw/gscan
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS    := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(BUILD_DATE)
PREFIX     ?= $(HOME)/.local
BINDIR     := $(PREFIX)/bin

.PHONY: all build test race vet fmt check bench install clean

all: check build

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(APP) .

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

check:
	@test -z "$$(gofmt -l .)" || { echo "gofmt needed:"; gofmt -l .; exit 1; }
	go vet ./...
	go test -race ./...

bench:
	go test -run '^$$' -bench . -benchmem ./...

install: build
	mkdir -p $(BINDIR)
	install -m 0755 $(APP) $(BINDIR)/$(APP)
	@echo "Installed $(BINDIR)/$(APP)"

clean:
	rm -f $(APP)
	rm -rf dist coverage.out
