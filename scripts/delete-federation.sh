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

function delete-helm-deployment() {
  if [[ ! "${NAMESPACED}" || "${DELETE_CLUSTER_RESOURCE}" ]]; then
    # Clean cluster registry
    ${KCD} crd clusters.clusterregistry.k8s.io
  fi

  if [[ ! "${NAMESPACED}" ]]; then
      ${KCD} namespace ${PUBLIC_NS}
  fi

  # Clean federation resources
  ${KCD} -n "${NS}" FederatedTypeConfig --all
  if [[ ! "${NAMESPACED}" || "${DELETE_CLUSTER_RESOURCE}" ]]; then
    ${KCD} crd $(kubectl get crd | grep -E 'federation.k8s.io' | awk '{print $1}')
  fi

  if [[ "${NAMESPACED}" ]]; then
    helm delete --purge federation-v2-${NS}
  else
    helm delete --purge federation-v2
  fi
}

KCD="kubectl --ignore-not-found=true delete"
NS="${FEDERATION_NAMESPACE:-federation-system}"
PUBLIC_NS=kube-multicluster-public
NAMESPACED="${NAMESPACED:-}"
DELETE_CLUSTER_RESOURCE="${DELETE_CLUSTER_RESOURCE:-}"

IMAGE_NAME=`kubectl get deploy -n ${NS} -oyaml | grep "image:" | awk '{print $2}'`
LATEST_IMAGE_NAME=quay.io/kubernetes-multicluster/federation-v2:latest
if [[ "${IMAGE_NAME}" == "$LATEST_IMAGE_NAME" ]]; then
  USE_LATEST=y
else
  USE_LATEST=
fi

KF_NS_ARG="--federation-namespace=${NS} "

# Unjoin clusters by removing objects added by kubefedctl.
HOST_CLUSTER="$(kubectl config current-context)"
JOINED_CLUSTERS="$(kubectl -n "${NS}" get federatedclusters -o=jsonpath='{range .items[*]}{.metadata.name}{" "}{end}')"
for c in ${JOINED_CLUSTERS}; do
  ./bin/kubefedctl unjoin "${c}" --host-cluster-context "${HOST_CLUSTER}" --remove-from-registry --v=2 ${KF_NS_ARG}
done

# Deploy federation resources
delete-helm-deployment

${KCD} ns "${NS}"

# Wait for the namespaces to be removed
function ns-deleted() {
  kubectl get ns "${1}" &> /dev/null
  [[ "$?" = "1" ]]
}
util::wait-for-condition "removal of namespace '${NS}'" "ns-deleted ${NS}" 120
if [[ ! "${NAMESPACED}" ]]; then
  util::wait-for-condition "removal of namespace '${PUBLIC_NS}'" "ns-deleted ${PUBLIC_NS}" 120
fi
