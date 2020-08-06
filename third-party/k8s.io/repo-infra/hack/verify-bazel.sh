#!/usr/bin/env bash
# Copyright 2016 The Kubernetes Authors.
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


fail() {
  echo "ERROR: $1. Fix with:" >&2
  echo "  bazel run @io_k8s_repo_infra//hack:update-bazel" >&2
  exit 1
}

if [[ -n "${TEST_WORKSPACE:-}" ]]; then # Running inside bazel
  echo "Validating bazel rules..." >&2
elif ! command -v bazel &> /dev/null; then
  echo "Install bazel at https://bazel.build" >&2
  exit 1
elif ! bazel query @//:all-srcs &>/dev/null; then
  fail "bazel rules need bootstrapping"
else
  (
    set -o xtrace
    bazel test --test_output=streamed @io_k8s_repo_infra//hack:verify-bazel
  )
  exit 0
fi

buildifier=$1
gazelle=$2
kazel=$3

gazelle_diff=$("$gazelle" fix --mode=diff --external=external || echo "ERROR: gazelle diffs")
kazel_diff=$("$kazel" --dry-run --print-diff --cfg-path=./.kazelcfg.json || echo "ERROR: kazel diffs")
# TODO(fejta): --mode=diff --lint=warn
buildifier_diff=$(find . \
  -name BUILD -o -name BUILD.bazel -o -name '*.bzl' -type f \
  \( -not -path '*/vendor/*' -prune \) \
  -exec "$buildifier" --mode=diff '{}' + 2>&1 || echo "ERROR: found buildifier diffs")

if [[ -n "${gazelle_diff}${kazel_diff}${buildifier_diff}" ]]; then
  echo "Current rules (-) do not match expected (+):" >&2
  echo "gazelle diff:"
  echo "${gazelle_diff}"
  echo "kazel diff:"
  echo "${kazel_diff}"
  echo "buildifier diff:"
  echo "$buildifier_diff"
  echo
  fail "bazel rules out of date"
fi
