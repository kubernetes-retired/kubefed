#!/usr/bin/env bash

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

# This script validates that binaries can be built and that all tests pass.

set -o errexit
set -o nounset
set -o pipefail

# Make sure, we run in the root of the repo and
# therefore run the tests on all packages
base_dir="$( cd "$(dirname "$0")/.." && pwd )"
cd "$base_dir" || {
  echo "Cannot cd to '$base_dir'. Aborting." >&2
  exit 1

}

rc=0

echo "Downloading binary dependencies"
./scripts/download-binaries.sh
rc=$((rc || $?))

echo "Building federation binaries"
PATH="${base_dir}/bin:${PATH}" apiserver-boot build executables
go build -o bin/kubefnord cmd/kubefnord/kubefnord.go

echo "Running go test"
# Ensure the test binaries are in the path
PATH="${base_dir}/bin:${PATH}" go test -v ./...
rc=$((rc || $?))

exit $rc
