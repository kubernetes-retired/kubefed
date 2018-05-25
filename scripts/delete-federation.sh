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

# This script removes the cluster registry and federation from the
# current kubectl context.

set -o errexit
set -o nounset
set -o pipefail

source "$(dirname "${BASH_SOURCE}")/util.sh"

KCD="kubectl --ignore-not-found=true delete"

# Remove the federation service account for the current context.
CONTEXT="$(kubectl config current-context)"
SA_NAME="${CONTEXT}-${CONTEXT}"
${KCD} -n federation sa "${SA_NAME}"
${KCD} -n federation clusterrole "federation-controller-manager:${SA_NAME}"
${KCD} -n federation clusterrolebinding "federation-controller-manager:${SA_NAME}"

# Remove federation
${KCD} apiservice v1alpha1.federation.k8s.io
${KCD} namespace federation

# Remove cluster registry
crinit aggregated delete mycr

# Wait for the namespaces to be removed
function ns-deleted() {
  kubectl get ns "${1}" &> /dev/null
  [[ "$?" = "1" ]]
}
util::wait-for-condition "removal of namespace 'federation'" "ns-deleted federation" 120
util::wait-for-condition "removal of namespace 'clusterregistry'" "ns-deleted clusterregistry" 120
