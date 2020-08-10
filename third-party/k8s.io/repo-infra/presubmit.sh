#!/usr/bin/env bash
# Copyright 2019 The Kubernetes Authors.
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

set -o nounset
set -o errexit
set -o pipefail

cd "$(git rev-parse --show-toplevel)"

if [[ -n "${GOOGLE_APPLICATION_CREDENTIALS:-}" ]]; then
  echo "Service account detected. Adding --config=ci to bazel commands" >&2
  mkdir -p "$HOME"
  touch "$HOME/.bazelrc"
  echo "build --config=ci" >> "$HOME/.bazelrc"
fi
(
  set -o xtrace
  bazel test //... # This also builds everything
  ./verify/verify-boilerplate.sh --rootdir="$(pwd)" -v # TODO(fejta) migrate to bazel
)
