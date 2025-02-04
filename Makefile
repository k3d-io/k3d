###################################
#                                 #
#          CONFIGURATION          #
#                                 #
###################################

########## Shell/Terminal Settings ##########
SHELL := /bin/bash

# determine if make is being executed from interactive terminal
INTERACTIVE:=$(shell [ -t 0 ] && echo 1)

# Use Go Modules for everything
export GO111MODULE=on

########## Tags ##########

# get git tag
ifneq ($(GIT_TAG_OVERRIDE),)
$(info GIT_TAG set from env override!)
GIT_TAG := $(GIT_TAG_OVERRIDE)
endif

GIT_TAG   ?= $(shell git describe --tags)
ifeq ($(GIT_TAG),)
GIT_TAG   := $(shell git describe --always)
endif

# Docker image tag derived from Git tag (with prefix "v" stripped off)
K3D_IMAGE_TAG := $(GIT_TAG:v%=%)

# get latest k3s version: grep the tag and replace + with - (difference between git and dockerhub tags)
K3S_TAG := $(shell curl --silent --retry 3 "https://update.k3s.io/v1-release/channels/stable" | egrep -o '/v[^ ]+"' | sed -E 's/\/|\"//g' | sed -E 's/\+/\-/')

ifeq ($(K3S_TAG),)
$(warning K3S_TAG undefined: couldn't get latest k3s image tag!)
$(warning Output of curl: $(shell curl "https://update.k3s.io/v1-release/channels/stable"))
$(error exiting)
endif

########## Source Options ##########
# DIRS defines a single level directly, we only look at *.go in this directory.
# REC_DIRS defines a source code tree. All go files are analyzed recursively.
DIRS :=  .
REC_DIRS := cmd

########## Test Settings ##########
E2E_LOG_LEVEL ?= WARN
E2E_INCLUDE ?=
E2E_EXCLUDE ?=
E2E_EXTRA ?=
E2E_RUNNER_START_TIMEOUT ?= 10
E2E_HELPER_IMAGE_TAG ?=
E2E_KEEP ?=
E2E_PARALLEL ?=
E2E_DIND_VERSION ?=
E2E_K3S_VERSION ?=
E2E_FAIL_FAST ?=

########## Go Build Options ##########
# Build targets
TARGETS ?= darwin/amd64 darwin/arm64 linux/amd64 linux/386 linux/arm linux/arm64 windows/amd64
TARGET_OBJS ?= darwin-amd64.tar.gz darwin-amd64.tar.gz.sha256 darwin-arm64.tar.gz darwin-arm64.tar.gz.sha256 linux-amd64.tar.gz linux-amd64.tar.gz.sha256 linux-386.tar.gz linux-386.tar.gz.sha256 linux-arm.tar.gz linux-arm.tar.gz.sha256 linux-arm64.tar.gz linux-arm64.tar.gz.sha256 windows-amd64.zip windows-amd64.zip.sha256
K3D_HELPER_VERSION ?=

# Go options
GO        ?= go
GOENVPATH := $(shell go env GOPATH)
TAGS      :=
TESTS     := ./...
TESTFLAGS :=
LDFLAGS   := -w -s -X github.com/k3d-io/k3d/v5/version.Version=${GIT_TAG} -X github.com/k3d-io/k3d/v5/version.K3sVersion=${K3S_TAG}
GCFLAGS   :=
GOFLAGS   := -mod=vendor
BINDIR    := $(CURDIR)/bin
BINARIES  := k3d

# Set version of the k3d helper images for build
ifneq ($(K3D_HELPER_VERSION),)
$(info [INFO] Helper Image version set to ${K3D_HELPER_VERSION})
LDFLAGS += -X github.com/k3d-io/k3d/v5/version.HelperVersionOverride=${K3D_HELPER_VERSION}
endif

# Rules for finding all go source files using 'DIRS' and 'REC_DIRS'
GO_SRC := $(foreach dir,$(DIRS),$(wildcard $(dir)/*.go))
GO_SRC += $(foreach dir,$(REC_DIRS),$(shell find $(dir) -name "*.go"))

########## Required Tools ##########
# Go Package required
PKG_GOX := github.com/iwilltry42/gox@v0.1.0
PKG_GOLANGCI_LINT_VERSION := 1.57.2
PKG_GOLANGCI_LINT_SCRIPT := https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh
PKG_GOLANGCI_LINT := github.com/golangci/golangci-lint/cmd/golangci-lint@v${PKG_GOLANGCI_LINT_VERSION}

########## Linting Options ##########
# configuration adjustments for golangci-lint
GOLANGCI_LINT_DISABLED_LINTERS := "" # disabling typecheck, because it currently (06.09.2019) fails with Go 1.13

# Rules for directory list as input for the golangci-lint program
LINT_DIRS := $(DIRS) $(foreach dir,$(REC_DIRS),$(dir)/...)

#############################
#                           #
#          TARGETS          #
#                           #
#############################

.PHONY: all build build-cross clean fmt check-fmt lint check extra-clean install-tools

all: clean fmt check test build

############################
########## Builds ##########
############################

# debug builds
build-debug: GCFLAGS+="all=-N -l"
build-debug: build

# default build target for the local platform
build:
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) -tags '$(TAGS)' -ldflags '$(LDFLAGS)' -gcflags '$(GCFLAGS)' -o '$(BINDIR)/$(BINARIES)'

# cross-compilation for all targets
build-cross: LDFLAGS += -extldflags "-static"
build-cross:
	CGO_ENABLED=0 gox -parallel=3 -output="_dist/$(BINARIES)-{{.OS}}-{{.Arch}}" -osarch='$(TARGETS)' $(GOFLAGS) $(if $(TAGS),-tags '$(TAGS)',) -ldflags '$(LDFLAGS)'
gen-checksum:	build-cross
	$(eval ARTIFACTS_TO_PUBLISH := $(shell ls _dist/*))
	$$(sha256sum $(ARTIFACTS_TO_PUBLISH) > _dist/checksums.txt)

# build a specific docker target ( '%' matches the target as specified in the Dockerfile)
build-docker-%:
	@echo "Building Docker image k3d:$(K3D_IMAGE_TAG)-$*"
	DOCKER_BUILDKIT=1 docker build . --quiet --no-cache -t k3d:$(K3D_IMAGE_TAG)-$* --target $*

# build helper images
build-helper-images: build-proxy-image build-tools-image

build-proxy-image:
	@echo "Building docker image ghcr.io/k3d-io/k3d-proxy:$(K3D_IMAGE_TAG)"
	DOCKER_BUILDKIT=1 docker build --quiet --no-cache proxy/ -f proxy/Dockerfile -t ghcr.io/k3d-io/k3d-proxy:$(K3D_IMAGE_TAG)

build-tools-image:
	@echo "Building docker image ghcr.io/k3d-io/k3d-tools:$(K3D_IMAGE_TAG)"
	DOCKER_BUILDKIT=1 docker build --quiet --no-cache tools/ -f tools/Dockerfile -t ghcr.io/k3d-io/k3d-tools:$(K3D_IMAGE_TAG) --build-arg GIT_TAG=$(GIT_TAG)

##############################
########## Cleaning ##########
##############################

clean:
	@rm -rf $(BINDIR) _dist/

extra-clean: clean
	$(GO) clean -i $(PKG_GOX)
	$(GO) clean -i $(PKG_GOLANGCI_LINT)

##########################################
########## Formatting & Linting ##########
##########################################

# fmt will fix the golang source style in place.
fmt:
	@gofmt -s -l -w $(GO_SRC)

# check-fmt returns an error code if any source code contains format error.
check-fmt:
	@test -z $(shell gofmt -s -l $(GO_SRC) | tee /dev/stderr) || echo "[WARN] Fix formatting issues with 'make fmt'"

lint:
	@golangci-lint run -D $(GOLANGCI_LINT_DISABLED_LINTERS) $(LINT_DIRS)

ci-lint:
	golangci-lint run --timeout 5m0s --out-format=github-actions -D $(GOLANGCI_LINT_DISABLED_LINTERS) $(LINT_DIRS)

check: check-fmt lint

###########################
########## Tests ##########
###########################

test:
	$(GO) test $(TESTS) $(TESTFLAGS)

e2e:
	@echo "Running e2e tests"
	LOG_LEVEL="$(E2E_LOG_LEVEL)" E2E_INCLUDE="$(E2E_INCLUDE)" E2E_EXCLUDE="$(E2E_EXCLUDE)" E2E_EXTRA="$(E2E_EXTRA)" E2E_RUNNER_START_TIMEOUT=$(E2E_RUNNER_START_TIMEOUT) E2E_HELPER_IMAGE_TAG="$(E2E_HELPER_IMAGE_TAG)" E2E_KEEP="$(E2E_KEEP)" E2E_PARALLEL="$(E2E_PARALLEL)" E2E_DIND_VERSION="$(E2E_DIND_VERSION)" E2E_K3S_VERSION="$(E2E_K3S_VERSION)" E2E_FAIL_FAST="$(E2E_FAIL_FAST)" tests/dind.sh "${K3D_IMAGE_TAG}"

ci-tests: fmt check e2e

##########################
########## Misc ##########
##########################


#########################################
########## Setup & Preparation ##########
#########################################

# Check for required executables
HAS_GOX := $(shell command -v gox 2> /dev/null)
HAS_GOLANGCI  := $(shell command -v golangci-lint)
HAS_GOLANGCI_VERSION := $(shell golangci-lint --version | grep "version $(PKG_GOLANGCI_LINT_VERSION) " 2>&1)

install-tools:
ifndef HAS_GOX
	($(GO) install $(PKG_GOX))
endif
ifndef HAS_GOLANGCI
	(curl -sfL $(PKG_GOLANGCI_LINT_SCRIPT) | sh -s -- -b $(GOENVPATH)/bin v${PKG_GOLANGCI_LINT_VERSION})
endif
ifdef HAS_GOLANGCI
ifeq ($(HAS_GOLANGCI_VERSION),)
ifdef INTERACTIVE
	@echo "Warning: Your installed version of golangci-lint (interactive: ${INTERACTIVE}) differs from what we'd like to use. Switch to v${PKG_GOLANGCI_LINT_VERSION}? [Y/n]"
	@read line; if [ $$line == "y" ]; then (curl -sfL $(PKG_GOLANGCI_LINT_SCRIPT) | sh -s -- -b $(GOENVPATH)/bin v${PKG_GOLANGCI_LINT_VERSION}); fi
else
	@echo "Warning: you're not using the same version of golangci-lint as us (v${PKG_GOLANGCI_LINT_VERSION})"
endif
endif
endif

# In the CI system, we need...
# - golangci-lint for linting (lint)
# - gox for cross-compilation (build-cross)
# - kubectl for E2E-tests (e2e)
ci-setup:
	@echo "### Installing Go tools..."
	@echo "### -> Installing golangci-lint..."
	curl -sfL $(PKG_GOLANGCI_LINT_SCRIPT) | sh -s -- -b $(GOENVPATH)/bin v$(PKG_GOLANGCI_LINT_VERSION)

	@echo "### -> Installing gox..."
	./scripts/install-tools.sh gox

	@echo "### Installing kubectl..."
	./scripts/install-tools.sh kubectl
