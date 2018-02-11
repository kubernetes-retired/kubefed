#!/usr/bin/env bash
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

set -euo pipefail

SCRIPT_ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ret=0

# The go-to-protobuf tool requires goimports to be in the PATH.
command -v goimports >/dev/null 2>&1 || go get golang.org/x/tools/cmd/goimports

echo -e "\n########## Verifying Code Generation ##########\n"

if ! ${SCRIPT_ROOT}/update-codegen.sh --verify-only; then
    echo -e "\nGenerated code is out of date. Please run hack/update-codegen.sh\n"
    ret=1
fi

echo -e "\n########## Verifying OpenAPI Spec ##########\n"

if ! ${SCRIPT_ROOT}/verify-openapi-spec.sh; then
    # The verify script prints out an error message for us.
    echo
    ret=1
fi

exit ${ret}
