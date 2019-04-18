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

# This script automates deployment of a federation - as documented in
# the README - to the current kubectl context.  It also joins the
# hosting cluster as a member of the federation.
#
# WARNING: The service account for the federation namespace will be
# granted the cluster-admin role.  Until more restrictive permissions
# are used, access to the federation namespace should be restricted to
# trusted users.
#
# If using minikube, a cluster must be started prior to invoking this
# script:
#
#   $ minikube start
#
# This script depends on kubectl and kubebuilder being installed in
# the path.  If you want to install Federation via helm chart, you may
# also need to install helm in the path. These and other test binaries
# can be installed via the download-binaries.sh script, which downloads
# to ./bin:
#
#   $ ./scripts/download-binaries.sh
#   $ export PATH=$(pwd)/bin:${PATH}
#
# To redeploy federation from scratch, prefix the deploy invocation with the deletion script:
#
#   # WARNING: The deletion script will remove federation and cluster registry data
#   $ ./scripts/delete-federation.sh [join-cluster]... && ./scripts/deploy-federation.sh <image> [join-cluster]...
#

set -o errexit
set -o nounset
set -o pipefail

source "$(dirname "${BASH_SOURCE}")/util.sh"

function deploy-with-helm() {
  # Don't install tiller if we already have a working install.
  if ! helm version --server 2>/dev/null; then
    # RBAC should be enabled to avoid CI fail because CI K8s uses RBAC for Tiller
    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tiller
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: tiller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: tiller
    namespace: kube-system
EOF

    helm init --service-account tiller
    util::wait-for-condition "Tiller to become ready" "helm version --server &> /dev/null" 120
  fi

  local repository=${IMAGE_NAME%/*}
  local image_tag=${IMAGE_NAME##*/}
  local image=${image_tag%:*}
  local tag=${image_tag#*:}

  local cmd
  if [[ "${NAMESPACED}" ]]; then
    cmd="$(helm-deploy-cmd federation-v2-${NS} ${NS} ${repository} ${image} ${tag})"
    cmd="${cmd} --set global.scope=Namespaced"
  else
    cmd="$(helm-deploy-cmd federation-v2 ${NS} ${repository} ${image} ${tag})"
  fi

  if [[ "${IMAGE_PULL_POLICY:-}" ]]; then
    cmd="${cmd} --set controllermanager.imagePullPolicy=${IMAGE_PULL_POLICY}"
  fi

  ${cmd}
}

function helm-deploy-cmd {
  # Required arguments
  local name="${1}"
  local ns="${2}"
  local repo="${3}"
  local image="${4}"
  local tag="${5}"

  echo "helm install charts/federation-v2 --name ${name} --namespace ${ns} \
      --set controllermanager.repository=${repo} --set controllermanager.image=${image} \
      --set controllermanager.tag=${tag}"
}

NS="${FEDERATION_NAMESPACE:-federation-system}"
PUBLIC_NS=kube-multicluster-public
IMAGE_NAME="${1:-}"
NAMESPACED="${NAMESPACED:-}"

LATEST_IMAGE_NAME=quay.io/kubernetes-multicluster/federation-v2:latest
if [[ "${IMAGE_NAME}" == "$LATEST_IMAGE_NAME" ]]; then
  USE_LATEST=y
else
  USE_LATEST=
fi

KF_NS_ARGS="--federation-namespace=${NS} "

if [[ -z "${IMAGE_NAME}" ]]; then
  >&2 echo "Usage: $0 <image> [join-cluster]...

<image>        should be in the form <containerregistry>/<username>/<imagename>:<tagname>

Example: docker.io/<username>/federation-v2:test

If intending to use the docker hub as the container registry to push
the federation image to, make sure to login to the local docker daemon
to ensure credentials are available for push:

  $ docker login --username <username>

<join-cluster> should be the kubeconfig context name for the additional cluster to join.
               NOTE: The host cluster is already included in the join.

"
  exit 2
fi

shift
# Allow for no specific JOIN_CLUSTERS: they probably want to kubefed2 themselves.
JOIN_CLUSTERS="${*-}"

# Use DOCKER_PUSH= ./scripts/deploy-federation.sh <image> to skip docker
# push on container image when not using latest image.
DOCKER_PUSH="${DOCKER_PUSH:-y}"
DOCKER_PUSH_CMD="docker push ${IMAGE_NAME}"
if [[ ! "${DOCKER_PUSH}" ]]; then
    DOCKER_PUSH_CMD=
fi

# Build federation binaries and image
if [[ ! "${USE_LATEST}" ]]; then
  cd "$(dirname "$0")/.."
  make container IMAGE_NAME=${IMAGE_NAME}
  cd -
  ${DOCKER_PUSH_CMD}
fi
cd "$(dirname "$0")/.."
make kubefed2
cd -

if ! kubectl get ns "${NS}" > /dev/null 2>&1; then
  kubectl create ns "${NS}"
fi

if [[ ! "${NAMESPACED}" ]]; then
  if ! kubectl get ns "${PUBLIC_NS}" > /dev/null 2>&1; then
    kubectl create ns "${PUBLIC_NS}"
  fi
fi

# Deploy federation resources
deploy-with-helm

# Join the host cluster
CONTEXT="$(kubectl config current-context)"
./bin/kubefed2 join "${CONTEXT}" --host-cluster-context "${CONTEXT}" --add-to-registry --v=2 ${KF_NS_ARGS}

for c in ${JOIN_CLUSTERS}; do
  ./bin/kubefed2 join "${c}" --host-cluster-context "${CONTEXT}" --add-to-registry --v=2 ${KF_NS_ARGS}
done
