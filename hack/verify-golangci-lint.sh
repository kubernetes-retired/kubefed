#!/bin/bash
# Copyright 2017 The Kubernetes Authors.
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

set -o errexit
set -o nounset
set -o pipefail

if [[ -n "${TEST_WORKSPACE:-}" ]]; then # Running inside bazel
  echo "Verifying golangci-lint..." >&2
elif ! command -v bazel &> /dev/null; then
  echo "Install bazel at https://bazel.build" >&2
  exit 1
else
  (
    set -o xtrace
    bazel test --test_output=streamed @io_k8s_repo_infra//hack:verify-golangci-lint
  )
  exit 0
fi

trap 'echo ERROR: golangci-lint failed >&2' ERR

if [[ ! -f .golangci.yml ]]; then
  echo 'ERROR: missing .golangci.yml in repo root' >&2
  exit 1
fi

golangci_lint=$2
export GO111MODULE=on
export GOPROXY=https://proxy.golang.org
export GOSUMDB=sum.golang.org
export HOME=$TEST_TMPDIR/home
export GOPATH=$HOME/go
PATH=$(dirname "$1"):$PATH
export PATH
shift 2
"$golangci_lint" run "$@"
