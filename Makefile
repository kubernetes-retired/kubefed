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
REGISTRY ?= quay.io/kubernetes-multicluster
IMAGE = $(REGISTRY)/$(TARGET)
DIR := ${CURDIR}
BIN_DIR := bin
DOCKER ?= docker
HOST_ARCH ?= $(shell go env GOARCH)
HOST_PLATFORM ?= $(shell uname -s | tr A-Z a-z)-$(HOST_ARCH)

GIT_VERSION ?= $(shell git describe --always --dirty)
GIT_TAG ?= $(shell git describe --tags --exact-match 2>/dev/null)
GIT_HASH ?= $(shell git rev-parse HEAD)
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
# .travis.yml
BUILD_IMAGE ?= golang:1.12.5

HYPERFED_TARGET = bin/hyperfed
CONTROLLER_TARGET = bin/controller-manager
KUBEFEDCTL_TARGET = bin/kubefedctl
WEBHOOK_TARGET = bin/webhook

LDFLAG_OPTIONS = -ldflags "-X sigs.k8s.io/kubefed/pkg/version.version=$(GIT_VERSION) \
                      -X sigs.k8s.io/kubefed/pkg/version.gitCommit=$(GIT_HASH) \
                      -X sigs.k8s.io/kubefed/pkg/version.gitTreeState=$(GIT_TREESTATE) \
                      -X sigs.k8s.io/kubefed/pkg/version.buildDate=$(BUILDDATE)"

GO_BUILDCMD = CGO_ENABLED=0 go build $(VERBOSE_FLAG) $(LDFLAG_OPTIONS)

TESTARGS ?= $(VERBOSE_FLAG) -timeout 60s
TEST_PKGS ?= $(GOTARGET)/cmd/... $(GOTARGET)/pkg/...
TEST_CMD = go test $(TESTARGS)
TEST = $(TEST_CMD) $(TEST_PKGS)

DOCKER_BUILD ?= $(DOCKER) run --rm -v $(DIR):$(BUILDMNT) -w $(BUILDMNT) $(BUILD_IMAGE) /bin/sh -c

# TODO (irfanurrehman): can add local compile, and auto-generate targets also if needed
.PHONY: all container push clean hyperfed controller kubefedctl test local-test vet fmt build bindir generate webhook

all: container hyperfed controller kubefedctl webhook

# Unit tests
test: vet
	go test $(TEST_PKGS)

build: hyperfed controller kubefedctl webhook

vet:
	go vet $(TEST_PKGS)

fmt:
	$(shell ./hack/update-gofmt.sh)

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
	$(DOCKER_BUILD) 'GOARCH=$(word 2,$(subst -, ,$(2))) GOOS=$(word 1,$(subst -, ,$(2))) $(GO_BUILDCMD) -o $(1)-$(2) cmd/$(3)/main.go'
ALL_BINS := $(ALL_BINS) $(1)-$(2)
endef
$(foreach cmd, $(COMMANDS), $(foreach platform, $(PLATFORMS), $(eval $(call PLATFORM_template, $(cmd),$(platform),$(notdir $(cmd))))))

define CMD_template
$(1): $(1)-$(HOST_PLATFORM)
	ln -sf $(notdir $(1)-$(HOST_PLATFORM)) $(1)
ALL_BINS := $(ALL_BINS) $(1)
endef
$(foreach cmd, $(COMMANDS), $(eval $(call CMD_template,$(cmd))))

hyperfed: $(HYPERFED_TARGET)

controller: $(CONTROLLER_TARGET)

kubefedctl: $(KUBEFEDCTL_TARGET)

webhook: $(WEBHOOK_TARGET)

# Generate code
generate-code:
ifndef GOPATH
	$(error GOPATH not defined, please define GOPATH. Run "go help gopath" to learn more about GOPATH)
endif
	go generate ./pkg/... ./cmd/...

generate: generate-code kubefedctl
	./scripts/sync-up-helm-chart.sh

push: container

	if [[ -z "$(TRAVIS_PULL_REQUEST)" ]]; \
	then \
		$(DOCKER) push $(IMAGE):$(GIT_VERSION); \
	elif [[ "$(TRAVIS_PULL_REQUEST)" == "false" && "$(TRAVIS_SECURE_ENV_VARS)" == "true" ]]; \
	then \
		$(DOCKER) login -u "$(QUAY_USERNAME)" -p "$(QUAY_PASSWORD)" quay.io; \
		if [[ "$(TRAVIS_BRANCH)" == "master" ]]; \
		then \
			$(DOCKER) tag $(IMAGE):$(GIT_VERSION) $(IMAGE):canary; \
			$(DOCKER) push $(IMAGE):canary; \
		fi; \
		\
		if git describe --tags --exact-match >/dev/null 2>&1; \
		then \
			$(DOCKER) tag $(IMAGE):$(GIT_VERSION) $(IMAGE):$(GIT_TAG); \
			$(DOCKER) push $(IMAGE):$(GIT_TAG); \
			$(DOCKER) tag $(IMAGE):$(GIT_VERSION) $(IMAGE):latest; \
			$(DOCKER) push $(IMAGE):latest; \
		fi \
	fi

clean:
	rm -f $(ALL_BINS)
	$(DOCKER) rmi $(IMAGE):$(GIT_VERSION) || true
