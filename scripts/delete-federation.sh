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

function delete-script-deployment() {
  DISABLE_OPTIONS="--delete-federated-crd=false"
  if [[ "${DELETE_GLOBAL_RESOURCE}" ]]; then
    DISABLE_OPTIONS="--delete-federated-crd=true"

    # Remove cluster registry CRD
    ${KCD} -f vendor/k8s.io/cluster-registry/cluster-registry-crd.yaml
  fi

  # Remove public namespace
  if [[ ! "${NAMESPACED}" ]]; then
    ${KCD} namespace ${PUBLIC_NS}
  fi

  # Disable federation of all types
  for ftc in $(kubectl get federatedtypeconfig -n "${NS}" -o=jsonpath={.items..metadata.name}); do
    ./bin/kubefed2 disable "${ftc}" --delete-from-api --federation-namespace="${NS}" ${DISABLE_OPTIONS}
  done

  # Remove federation CRDs, namespace, RBAC and deployment resources.
  if [[ ! "${USE_LATEST}" ]]; then
    if [[ "${NAMESPACED}" ]]; then
      if [[ "${DELETE_GLOBAL_RESOURCE}" ]]; then
        ${KCD} -n "${NS}" -f hack/install-namespaced.yaml
      else
        ${KCD} -n "${NS}" role federation-role
        ${KCD} -n "${NS}" rolebinding federation-rolebinding
        ${KCD} -n "${NS}" service federation-controller-manager-service
        ${KCD} -n "${NS}" statefulset federation-controller-manager
      fi
    else
      ${KCD} -f hack/install.yaml
    fi
  else
    ${KCD} -f hack/install-latest.yaml
  fi
}

function delete-helm-deployment() {
  if [[ "${DELETE_GLOBAL_RESOURCE}" ]]; then
    # Clean cluster registry
    ${KCD} crd clusters.clusterregistry.k8s.io
  fi

  if [[ ! "${NAMESPACED}" ]]; then
      ${KCD} namespace ${PUBLIC_NS}
  fi

  # Clean federation resources
  ${KCD} -n "${NS}" FederatedTypeConfig --all

  if [[ "${DELETE_GLOBAL_RESOURCE}" ]]; then
    ${KCD} crd $(kubectl get crd | grep -E 'federation.k8s.io' | awk '{print $1}')
    ${KCD} clusterrolebindings
    ${KCD} clusterrole
  fi

  helm delete --purge federation-v2-${NS}
}

KCD="kubectl --ignore-not-found=true delete"
NS="${FEDERATION_NAMESPACE:-federation-system}"
PUBLIC_NS=kube-multicluster-public
NAMESPACED="${NAMESPACED:-}"
DELETE_GLOBAL_RESOURCE="${DELETE_GLOBAL_RESOURCE:-}"

IMAGE_NAME=`kubectl get sts -n ${NS} -oyaml | grep "image:" | awk '{print $2}'`
LATEST_IMAGE_NAME=quay.io/kubernetes-multicluster/federation-v2:latest
if [[ "${IMAGE_NAME}" == "$LATEST_IMAGE_NAME" ]]; then
  USE_LATEST=y
else
  USE_LATEST=
fi

KF_NS_ARG="--federation-namespace=${NS} "
if [[ "${NAMESPACED}" ]]; then
  KF_NS_ARG+="--registry-namespace=${NS} "
  if [[ ! "${DELETE_GLOBAL_RESOURCE}" ]]; then
      KF_NS_ARG+="--delete-global-resource=false "
  fi
fi

# Unjoin clusters by removing objects added by kubefed2.
HOST_CLUSTER="$(kubectl config current-context)"
JOINED_CLUSTERS="$(kubectl -n "${NS}" get federatedclusters -o=jsonpath='{range .items[*]}{.metadata.name}{" "}{end}')"
for c in ${JOINED_CLUSTERS}; do
  ./bin/kubefed2 unjoin "${c}" --host-cluster-context "${HOST_CLUSTER}" --remove-from-registry --v=2 ${KF_NS_ARG}
done

# Deploy federation resources
USE_CHART=${USE_CHART:-false}
if [[ ${USE_CHART} == true ]]; then
  delete-helm-deployment
else
  delete-script-deployment
fi

# Remove permissive rolebinding that allows federation controllers to run.
if [[ "${NAMESPACED}" ]]; then
  ${KCD} -n "${NS}" rolebinding federation-admin
else
  ${KCD} clusterrolebinding federation-admin
fi
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
