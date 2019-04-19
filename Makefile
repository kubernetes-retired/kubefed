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

TARGET = federation-v2
GOTARGET = github.com/kubernetes-sigs/$(TARGET)
REGISTRY ?= quay.io/kubernetes-multicluster
IMAGE = $(REGISTRY)/$(TARGET)
DIR := ${CURDIR}
BIN_DIR := bin
DOCKER ?= docker
HOST_OS ?= $(shell uname -s | tr A-Z a-z)

GIT_VERSION ?= $(shell git describe --always --dirty)
GIT_TAG ?= $(shell git describe --tags --exact-match 2>/dev/null)
GIT_HASH ?= $(shell git rev-parse HEAD)
BUILDDATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

# Note: this is allowed to be overridden for scripts/deploy-federation.sh
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
BUILD_IMAGE ?= golang:1.11.2

HYPERFED_TARGET = bin/hyperfed
CONTROLLER_TARGET = bin/controller-manager
KUBEFED2_TARGET = bin/kubefed2

LDFLAG_OPTIONS = -ldflags "-X github.com/kubernetes-sigs/federation-v2/pkg/version.version=$(GIT_VERSION) \
                      -X github.com/kubernetes-sigs/federation-v2/pkg/version.gitCommit=$(GIT_HASH) \
                      -X github.com/kubernetes-sigs/federation-v2/pkg/version.gitTreeState=$(GIT_TREESTATE) \
                      -X github.com/kubernetes-sigs/federation-v2/pkg/version.buildDate=$(BUILDDATE)"

GO_BUILDCMD = CGO_ENABLED=0 go build $(VERBOSE_FLAG) $(LDFLAG_OPTIONS)

TESTARGS ?= $(VERBOSE_FLAG) -timeout 60s
TEST_PKGS ?= $(GOTARGET)/cmd/... $(GOTARGET)/pkg/...
TEST_CMD = go test $(TESTARGS)
TEST = $(TEST_CMD) $(TEST_PKGS)

DOCKER_BUILD ?= $(DOCKER) run --rm -v $(DIR):$(BUILDMNT) -w $(BUILDMNT) $(BUILD_IMAGE) /bin/sh -c

# TODO (irfanurrehman): can add local compile, and auto-generate targets also if needed
.PHONY: all container push clean hyperfed controller kubefed2 test local-test vet fmt build bindir generate

all: container hyperfed controller kubefed2

# Unit tests
test: vet
	go test $(TEST_PKGS)

build: hyperfed controller kubefed2

vet:
	go vet $(TEST_PKGS)

fmt:
	$(shell ./hack/update-gofmt.sh)

container: $(HYPERFED_TARGET)-linux
	cp -f $(HYPERFED_TARGET)-linux images/federation-v2/hyperfed
	$(DOCKER) build images/federation-v2 -t $(IMAGE_NAME)
	rm -f images/federation-v2/hyperfed

bindir:
	mkdir -p $(BIN_DIR)

COMMANDS := $(HYPERFED_TARGET) $(CONTROLLER_TARGET) $(KUBEFED2_TARGET)
OSES := linux darwin
ALL_BINS :=

define OS_template
$(1)-$(2): bindir
	$(DOCKER_BUILD) 'GOARCH=amd64 GOOS=$(2) $(GO_BUILDCMD) -o $(1)-$(2) cmd/$(3)/main.go'
ALL_BINS := $(ALL_BINS) $(1)-$(2)
endef
$(foreach cmd, $(COMMANDS), $(foreach os, $(OSES), $(eval $(call OS_template, $(cmd),$(os),$(notdir $(cmd))))))

define CMD_template
$(1): $(1)-$(HOST_OS)
	ln -sf $(notdir $(1)-$(HOST_OS)) $(1)
ALL_BINS := $(ALL_BINS) $(1)
endef
$(foreach cmd, $(COMMANDS), $(eval $(call CMD_template,$(cmd))))

hyperfed: $(HYPERFED_TARGET)

controller: $(CONTROLLER_TARGET)

kubefed2: $(KUBEFED2_TARGET)

# Generate code
generate: kubefed2
ifndef GOPATH
	$(error GOPATH not defined, please define GOPATH. Run "go help gopath" to learn more about GOPATH)
endif
	go generate ./pkg/... ./cmd/...
	./scripts/sync-up-helm-chart.sh

push:
	$(DOCKER) push $(REGISTRY)/$(TARGET):$(GIT_VERSION)
	if git describe --tags --exact-match >/dev/null 2>&1; \
	then \
		$(DOCKER) tag $(REGISTRY)/$(TARGET):$(GIT_VERSION) $(REGISTRY)/$(TARGET):$(GIT_TAG); \
		$(DOCKER) push $(REGISTRY)/$(TARGET):$(GIT_TAG); \
		$(DOCKER) tag $(REGISTRY)/$(TARGET):$(GIT_VERSION) $(REGISTRY)/$(TARGET):latest; \
		$(DOCKER) push $(REGISTRY)/$(TARGET):latest; \
	fi

clean:
	rm -f $(ALL_BINS)
	$(DOCKER) rmi $(REGISTRY)/$(TARGET):$(GIT_VERSION) || true
