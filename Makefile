SHELL := /bin/bash

# Build targets
TARGETS ?= darwin/amd64 linux/amd64 linux/386 linux/arm linux/arm64 windows/amd64
TARGET_OBJS ?= darwin-amd64.tar.gz darwin-amd64.tar.gz.sha256 linux-amd64.tar.gz linux-amd64.tar.gz.sha256 linux-386.tar.gz linux-386.tar.gz.sha256 linux-arm.tar.gz linux-arm.tar.gz.sha256 linux-arm64.tar.gz linux-arm64.tar.gz.sha256 windows-amd64.zip windows-amd64.zip.sha256

# get git tag
GIT_TAG   := $(shell git describe --tags)
ifeq ($(GIT_TAG),)
GIT_TAG   := $(shell git describe --always)
endif

# get latest k3s version
K3S_TAG		:= $(shell curl --silent "https://api.github.com/repos/rancher/k3s/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

# Go options
GO        ?= go
PKG       := $(shell go mod vendor)
TAGS      :=
TESTS     := .
TESTFLAGS :=
LDFLAGS   := -w -s -X github.com/rancher/k3d/version.Version=${GIT_TAG} -X github.com/rancher/k3d/version.K3sVersion=${K3S_TAG}
GOFLAGS   :=
BINDIR    := $(CURDIR)/bin
BINARIES  := k3d


# Go Package required
PKG_GOX := github.com/mitchellh/gox
PKG_GOLANGCI_LINT := github.com/golangci/golangci-lint/cmd/golangci-lint

export GO111MODULE=on

# go source files, ignore vendor directory
SRC = $(shell find . -type f -name '*.go' -not -path "./*/*")

.PHONY: all build build-cross clean fmt simplify check extra-clean

all: clean fmt check build

build:
	$(GO) build -i $(GOFLAGS) -tags '$(TAGS)' -ldflags '$(LDFLAGS)' -o '$(BINDIR)/$(BINARIES)'

build-cross: LDFLAGS += -extldflags "-static"
build-cross:
	CGO_ENABLED=0 gox -parallel=3 -output="_dist/$(BINARIES)-{{.OS}}-{{.Arch}}" -osarch='$(TARGETS)' $(GOFLAGS) $(if $(TAGS),-tags '$(TAGS)',) -ldflags '$(LDFLAGS)'

clean:
	@rm -rf $(BINDIR) _dist/

extra-clean: clean
	go clean -i $(PKG_GOX)
	go clean -i $(PKG_GOLANGCI_LINT)

fmt:
	@gofmt -l -w $(SRC)

simplify:
	@gofmt -s -l -w $(SRC)

check:
	@test -z $(shell gofmt -l main.go | tee /dev/stderr) || echo "[WARN] Fix formatting issues with 'make fmt'"
	@golangci-lint run
	@go vet ${SRC}

# Check for required executables
HAS_GOX := $(shell command -v gox 2> /dev/null)
HAS_GOLANGCI  := $(shell command -v golangci-lint 2> /dev/null)

ifndef HAS_GOX
	go get -u $(PKG_GOX)
endif
ifndef HAS_GOLANGCI
	go get -u $(PKG_GOLANGCI_LINT)
endif
