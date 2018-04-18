#!/bin/bash

set -ex

# Make sure, we run in the root of the repo and
# therefore run the tests on all packages
base_dir="$( cd "$(dirname "$0")/.." && pwd )"
cd "$base_dir" || {
  echo "Cannot cd to '$base_dir'. Aborting." >&2
  exit 1

}

rc=0

echo "Installing test and build binaries"
./scripts/download-test-binaries.sh
rc=$((rc || $?))

echo "Building binaries"
PATH="${base_dir}/bin:${PATH}" apiserver-boot build executables
go build -o bin/kubefnord cmd/kubefnord/kubefnord.go

echo "Running go test"
# Ensure the test binaries are in the path
PATH="${base_dir}/bin:${PATH}" go test -v ./...
rc=$((rc || $?))

exit $rc
