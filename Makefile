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
#

TARGET = federation-v2
GOTARGET = github.com/kubernetes-sigs/$(TARGET)
REGISTRY ?= quay.io/kubernetes-multicluster
IMAGE = $(REGISTRY)/$(TARGET)
DIR := ${CURDIR}
BIN_DIR := bin
DOCKER ?= docker

GIT_VERSION ?= $(shell git describe --always --dirty)
GIT_TAG ?= $(shell git describe --tags --exact-match 2>/dev/null)
GIT_HASH ?= $(shell git rev-parse HEAD)
BUILDDATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

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
BUILDCMD_HYPERFED = CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(HYPERFED_TARGET) $(VERBOSE_FLAG) $(LDFLAG_OPTIONS)
BUILDCMD_CONTROLLER = CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(CONTROLLER_TARGET) $(VERBOSE_FLAG) $(LDFLAG_OPTIONS)
BUILDCMD_KUBEFED2 = go build -o $(KUBEFED2_TARGET) $(VERBOSE_FLAG) $(LDFLAG_OPTIONS)

BUILD_HYPERFED = $(BUILDCMD_HYPERFED) cmd/hyperfed/main.go
BUILD_CONTROLLER = $(BUILDCMD_CONTROLLER) cmd/controller-manager/main.go
BUILD_KUBEFED2 = $(BUILDCMD_KUBEFED2) cmd/kubefed2/kubefed2.go

TESTARGS ?= $(VERBOSE_FLAG) -timeout 60s
TEST_PKGS ?= $(GOTARGET)/cmd/... $(GOTARGET)/pkg/...
TEST_CMD = go test $(TESTARGS)
TEST = $(TEST_CMD) $(TEST_PKGS)

DOCKER_BUILD ?= $(DOCKER) run --rm -v $(DIR):$(BUILDMNT) -w $(BUILDMNT) $(BUILD_IMAGE) /bin/sh -c

# TODO (irfanurrehman): can add local compile, and auto-generate targets also if needed
.PHONY: all container push clean hyperfed controller kubefed2 test local-test vet fmt build

all: container hyperfed controller kubefed2

# Unit tests
test: vet
	go test $(TEST_PKGS)

build: hyperfed controller kubefed2

vet:
	go vet $(TEST_PKGS)

fmt:
	go fmt $(TEST_PKGS)

container: hyperfed
	cp -f bin/hyperfed images/federation-v2/
	$(DOCKER) build images/federation-v2 \
		-t $(REGISTRY)/$(TARGET):$(GIT_VERSION)
	rm -f images/federation-v2/hyperfed

$(BIN_DIR):
	mkdir $(BIN_DIR)

hyperfed: $(BIN_DIR)
	$(DOCKER_BUILD) '$(BUILD_HYPERFED)'

controller: $(BIN_DIR)
	$(DOCKER_BUILD) '$(BUILD_CONTROLLER)'

kubefed2: $(BIN_DIR)
	$(DOCKER_BUILD) '$(BUILD_KUBEFED2)'

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
	rm -f $(KUBEFED2_TARGET) $(CONTROLLER_TARGET) $(HYPERFED_TARGET)
	$(DOCKER) rmi $(REGISTRY)/$(TARGET):$(GIT_VERSION) || true
