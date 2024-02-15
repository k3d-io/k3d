SHELL := /bin/bash

# get git tag
GIT_TAG   ?= $(shell git describe --tags)
ifeq ($(GIT_TAG),)
GIT_TAG   := $(shell git describe --always)
endif

# Go options
GO        ?= go
PKG       := $(shell GOWORK=off go mod vendor)
TAGS      :=
TESTS     := .
TESTFLAGS :=
LDFLAGS   := -w -s -X github.com/k3d-io/k3d/tools/version.Version=${GIT_TAG}
GOFLAGS   := -mod readonly
BINDIR    := $(CURDIR)/bin
BINARIES  := k3d-tools

# Go Package required
PKG_GOX := github.com/mitchellh/gox
PKG_GOLANGCI_LINT := github.com/golangci/golangci-lint/cmd/golangci-lint

export GO111MODULE=on
export CGO_ENABLED=0

# go source directories.
# DIRS defines a single level directly, we only look at *.go in this directory.
# REC_DIRS defines a source code tree. All go files are analyzed recursively.
DIRS :=  .
REC_DIRS := cmd

# Rules for finding all go source files using 'DIRS' and 'REC_DIRS'
GO_SRC := $(foreach dir,$(DIRS),$(wildcard $(dir)/*.go))
GO_SRC += $(foreach dir,$(REC_DIRS),$(shell find $(dir) -name "*.go"))

# Rules for directory list as input for the golangci-lint program
LINT_DIRS := $(DIRS) $(foreach dir,$(REC_DIRS),$(dir)/...)

.PHONY: all

all: clean fmt check build

build:
	$(GO) build $(GOFLAGS) -tags '$(TAGS)' -ldflags '$(LDFLAGS)' -o '$(BINDIR)/$(BINARIES)'

clean:
	@rm -rf $(BINDIR) _dist/

extra-clean: clean
	go clean -i $(PKG_GOX)
	go clean -i $(PKG_GOLANGCI_LINT)

# fmt will fix the golang source style in place.
fmt:
	@gofmt -s -l -w $(GO_SRC)

# check-fmt returns an error code if any source code contains format error.
check-fmt:
	@test -z $(shell gofmt -s -l $(GO_SRC) | tee /dev/stderr) || echo "[WARN] Fix formatting issues with 'make fmt'"

lint:
	@golangci-lint run $(LINT_DIRS)

check: check-fmt lint

# Check for required executables
HAS_GOX := $(shell command -v gox 2> /dev/null)
HAS_GOLANGCI  := $(shell command -v golangci-lint 2> /dev/null)

install-tools:
ifndef HAS_GOX
	(export GO111MODULE=off; go get -u $(PKG_GOX))
endif
ifndef HAS_GOLANGCI
	(export GO111MODULE=off; go get -u $(PKG_GOLANGCI_LINT))
endif
