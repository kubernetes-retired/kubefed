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

# This script removes the cluster registry, unjoins any clusters passed as
# arguments, and removes the federation from the current kubectl context.

set -o errexit
set -o nounset
set -o pipefail

source "$(dirname "${BASH_SOURCE}")/util.sh"

KCD="kubectl --ignore-not-found=true delete"
NS=federation-system

PUBLIC_NS=kube-multicluster-public

# Remove public namespace
${KCD} namespace ${PUBLIC_NS}

# Remove cluster registry CRD
${KCD} -f vendor/k8s.io/cluster-registry/cluster-registry-crd.yaml

# Unjoin clusters by removing objects added by kubefed2.
HOST_CLUSTER="$(kubectl config current-context)"
JOIN_CLUSTERS="${HOST_CLUSTER} ${@}"
for c in ${JOIN_CLUSTERS}; do
  SA_NAME="${c}-${HOST_CLUSTER}"
  CONTROLLER_ROLE="federation-controller-manager:${SA_NAME}"

  if [[ "${c}" != "${HOST_CLUSTER}" ]]; then
    ${KCD} ns ${NS} --context=${c}
  else
    ${KCD} sa ${SA_NAME} --context=${c} -n ${NS}
  fi

  ${KCD} clusterrolebinding ${CONTROLLER_ROLE} --context=${c}
  ${KCD} clusterrole ${CONTROLLER_ROLE} --context=${c}
  ${KCD} clusters ${c} -n ${PUBLIC_NS}
  ${KCD} federatedclusters ${c} -n ${NS}
  CLUSTER_SECRET=$(kubectl -n ${NS} get secrets \
    -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}' | grep "${c}-*")
  ${KCD} secret ${CLUSTER_SECRET} -n ${NS}
done

# Disable available types
for filename in ./config/federatedtypes/*.yaml; do
  ${KCD} -f "${filename}"
done

# Remove federation CRDs, namespace, RBAC and deployment resources.
${KCD} -f hack/install-latest.yaml

# Remove permissive rolebinding that allows federation controllers to run.
${KCD} clusterrolebinding federation-admin

# Wait for the namespaces to be removed
function ns-deleted() {
  kubectl get ns "${1}" &> /dev/null
  [[ "$?" = "1" ]]
}
util::wait-for-condition "removal of namespace '${NS}'" "ns-deleted ${NS}" 120
util::wait-for-condition "removal of namespace '${PUBLIC_NS}'" "ns-deleted ${PUBLIC_NS}" 120
