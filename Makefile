SHELL := /bin/bash

# Build targets
TARGETS ?= darwin/amd64 linux/amd64 linux/386 linux/arm linux/arm64 windows/amd64
TARGET_OBJS ?= darwin-amd64.tar.gz darwin-amd64.tar.gz.sha256 linux-amd64.tar.gz linux-amd64.tar.gz.sha256 linux-386.tar.gz linux-386.tar.gz.sha256 linux-arm.tar.gz linux-arm.tar.gz.sha256 linux-arm64.tar.gz linux-arm64.tar.gz.sha256 windows-amd64.zip windows-amd64.zip.sha256

# Go options
GO        ?= go
PKG       := $(shell go mod vendor)
TAGS      :=
TESTS     := .
TESTFLAGS :=
LDFLAGS   := -w -s
GOFLAGS   :=
BINDIR    := $(CURDIR)/bin
BINARIES := k3d

# go source files, ignore vendor directory
SRC = $(shell find . -type f -name '*.go' -not -path "./vendor/*")

.PHONY: all build build-cross clean install uninstall fmt simplify check run bootstrap

all: check install

build: 
	$(GO) build -i $(GOFLAGS) -tags '$(TAGS)' -ldflags '$(LDFLAGS)' -o '$(BINDIR)/$(BINARIES)'

build-cross: LDFLAGS += -extldflags "-static"
build-cross:	
	CGO_ENABLED=0 gox -parallel=3 -output="_dist/$(BINARIES)-{{.OS}}-{{.Arch}}" -osarch='$(TARGETS)' $(GOFLAGS) $(if $(TAGS),-tags '$(TAGS)',) -ldflags '$(LDFLAGS)'

clean:
	@rm -rf $(BINDIR) _dist/

fmt:
	@gofmt -l -w $(SRC)

simplify:
	@gofmt -s -l -w $(SRC)

check:
	@test -z $(shell gofmt -l main.go | tee /dev/stderr) || echo "[WARN] Fix formatting issues with 'make fmt'"
	@for d in $$(go list ./... | grep -v /vendor/); do golint $${d}; done
	@go vet ${SRC}

# Check for required executables 
HAS_GOX := $(shell command -v gox;)
HAS_GIT := $(shell command -v git;)

bootstrap:
ifndef HAS_GOX
	go get -u github.com/mitchellh/gox
endif

ifndef HAS_GIT
	$(error You must install Git)
endif