# Copyright 2018 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

SHELL := /bin/bash
TARGET = kubefed
GOTARGET = sigs.k8s.io/$(TARGET)
REGISTRY ?= docker.io/mesosphere
IMAGE = $(REGISTRY)/$(TARGET)
DIR := ${CURDIR}
BIN_DIR := bin
DOCKER ?= docker
HOST_ARCH ?= $(shell go env GOARCH)
HOST_PLATFORM ?= $(shell uname -s | tr A-Z a-z)-$(HOST_ARCH)

GIT_VERSION ?= $(shell git describe --always --dirty)
GIT_TAG ?= $(shell git describe --tags --exact-match 2>/dev/null)
GIT_HASH ?= $(shell git rev-parse HEAD)
GIT_BRANCH ?= $(filter-out HEAD,$(shell git rev-parse --abbrev-ref HEAD 2>/dev/null))
BUILDDATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

# Note: this is allowed to be overridden for scripts/deploy-kubefed.sh
IMAGE_NAME = $(REGISTRY)/$(TARGET):$(GIT_VERSION)

GIT_TREESTATE = "clean"
DIFF = $(shell git diff --quiet >/dev/null 2>&1; if [ $$? -eq 1 ]; then echo "1"; fi)
ifeq ($(DIFF), 1)
    GIT_TREESTATE = "dirty"
endif

ifneq ($(VERBOSE),)
VERBOSE_FLAG = -v
endif
BUILDMNT = /go/src/$(GOTARGET)
# The version here should match the version of go configured in
# .github/workflows files.
BUILD_IMAGE ?= golang:1.16.6

HYPERFED_TARGET = bin/hyperfed
CONTROLLER_TARGET = bin/controller-manager
KUBEFEDCTL_TARGET = bin/kubefedctl
WEBHOOK_TARGET = bin/webhook
E2E_BINARY_TARGET = bin/e2e

LDFLAG_OPTIONS = -ldflags "-X sigs.k8s.io/kubefed/pkg/version.version=$(GIT_VERSION) \
                      -X sigs.k8s.io/kubefed/pkg/version.gitCommit=$(GIT_HASH) \
                      -X sigs.k8s.io/kubefed/pkg/version.gitTreeState=$(GIT_TREESTATE) \
                      -X sigs.k8s.io/kubefed/pkg/version.buildDate=$(BUILDDATE)"

export GOPATH ?= $(shell go env GOPATH)
GO_BUILDCMD = CGO_ENABLED=0 go build $(VERBOSE_FLAG) $(LDFLAG_OPTIONS)

TESTARGS ?= $(VERBOSE_FLAG) -timeout 60s
TEST_PKGS ?= $(GOTARGET)/cmd/... $(GOTARGET)/pkg/...
TEST_CMD = go test $(TESTARGS)
TEST = $(TEST_CMD) $(TEST_PKGS)

ISTTY := $(shell [ -t 0 ] && echo 1)

DOCKER_BUILD ?= $(DOCKER) run --rm $(if $(ISTTY),-it) -u $(shell id -u):$(shell id -g) -e GOCACHE=/tmp/gocache -v $(DIR):$(BUILDMNT) -w $(BUILDMNT) $(BUILD_IMAGE)

# TODO (irfanurrehman): can add local compile, and auto-generate targets also if needed
.PHONY: all container push clean hyperfed controller kubefedctl test local-test vet lint build bindir generate webhook e2e deploy.kind

all: container hyperfed controller kubefedctl webhook e2e

# Unit tests
test:
	go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
	source <(setup-envtest use -p env 1.24.x) && \
		go test $(TEST_PKGS)

build: hyperfed controller kubefedctl webhook

lint:
	golangci-lint run -c .golangci.yml --fix

container: $(HYPERFED_TARGET)-linux-$(HOST_ARCH)
	cp -f $(HYPERFED_TARGET)-linux-$(HOST_ARCH) images/kubefed/hyperfed
	$(DOCKER) build images/kubefed -t $(IMAGE_NAME)
	rm -f images/kubefed/hyperfed

bindir:
	mkdir -p $(BIN_DIR)

COMMANDS := $(HYPERFED_TARGET) $(CONTROLLER_TARGET) $(KUBEFEDCTL_TARGET) $(WEBHOOK_TARGET)
PLATFORMS := linux-amd64 linux-arm64 linux-ppc64le linux-s390x darwin-amd64
ALL_BINS :=

define PLATFORM_template
$(1)-$(2): bindir
	$(DOCKER_BUILD) env GOARCH=$(word 2,$(subst -, ,$(2))) GOOS=$(word 1,$(subst -, ,$(2))) $(GO_BUILDCMD) -o $(1)-$(2) cmd/$(3)/main.go
ALL_BINS := $(ALL_BINS) $(1)-$(2)
endef
$(foreach cmd, $(COMMANDS), $(foreach platform, $(PLATFORMS), $(eval $(call PLATFORM_template, $(cmd),$(platform),$(notdir $(cmd))))))

define E2E_PLATFORM_template
$(1)-$(2): bindir
	$(DOCKER_BUILD) env GOARCH=$(word 2,$(subst -, ,$(2))) GOOS=$(word 1,$(subst -, ,$(2))) go test -c $(LDFLAG_OPTIONS) -o $(1)-$(2) ./test/$(3)
ALL_BINS := $(ALL_BINS) $(1)-$(2)
endef
$(foreach platform, $(PLATFORMS), $(eval $(call E2E_PLATFORM_template, $(E2E_BINARY_TARGET),$(platform),$(notdir $(E2E_BINARY_TARGET)))))

define CMD_template
$(1): $(1)-$(HOST_PLATFORM)
	ln -sf $(notdir $(1)-$(HOST_PLATFORM)) $(1)
ALL_BINS := $(ALL_BINS) $(1)
endef
$(foreach cmd, $(COMMANDS), $(eval $(call CMD_template,$(cmd))))
$(eval $(call CMD_template,$(E2E_BINARY_TARGET)))

hyperfed: $(HYPERFED_TARGET)

controller: $(CONTROLLER_TARGET)

kubefedctl: $(KUBEFEDCTL_TARGET)

webhook: $(WEBHOOK_TARGET)

e2e: $(E2E_BINARY_TARGET)

# Generate code
generate-code: controller-gen
	controller-gen object:headerFile=./hack/boilerplate.go.txt paths="./..."

generate: generate-code kubefedctl
	./scripts/sync-up-helm-chart.sh
	./scripts/update-bindata.sh

push: container
	$(DOCKER) push $(IMAGE):$(GIT_VERSION)
ifeq ($(GIT_BRANCH),master)
	$(DOCKER) tag $(IMAGE):$(GIT_VERSION) $(IMAGE):canary
	$(DOCKER) push $(IMAGE):canary
endif
ifneq ($(GIT_TAG),)
	$(DOCKER) tag $(IMAGE):$(GIT_VERSION) $(IMAGE):$(GIT_TAG)
	$(DOCKER) push $(IMAGE):$(GIT_TAG)
	$(DOCKER) tag $(IMAGE):$(GIT_VERSION) $(IMAGE):latest
	$(DOCKER) push $(IMAGE):latest
endif

clean:
	rm -f $(ALL_BINS)
	$(DOCKER) rmi $(IMAGE):$(GIT_VERSION) || true

controller-gen:
	command -v controller-gen &> /dev/null || (cd tools && go install sigs.k8s.io/controller-tools/cmd/controller-gen)

deploy.kind: generate
	KIND_LOAD_IMAGE=y FORCE_REDEPLOY=y ./scripts/deploy-kubefed.sh $(IMAGE_NAME)
