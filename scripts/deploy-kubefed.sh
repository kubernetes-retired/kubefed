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

# This script automates deployment of a KubeFed control plane to the
# current kubectl context.  It also registers the hosting cluster with
# the control plane.
#
# This script depends on kubectl, kubebuilder and helm being installed
# in the path.  These and other test binaries can be installed via the
# download-binaries.sh script, which downloads to ./bin:
#
#   $ ./scripts/download-binaries.sh
#   $ export PATH=$(pwd)/bin:${PATH}
#
# To redeploy KubeFed from scratch, prefix the deploy invocation with the deletion script:
#
#   # WARNING: The deletion script will remove KubeFed data
#   $ ./scripts/delete-kubefed.sh [join-cluster]... && ./scripts/deploy-kubefed.sh <image> [join-cluster]...
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
    cmd="$(helm-deploy-cmd kubefed-${NS} ${NS} ${repository} ${image} ${tag})"
    cmd="${cmd} --set global.scope=Namespaced"
  else
    cmd="$(helm-deploy-cmd kubefed ${NS} ${repository} ${image} ${tag})"
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

  echo "helm install charts/kubefed --name ${name} --namespace ${ns} \
      --set controllermanager.repository=${repo} --set controllermanager.image=${image} \
      --set controllermanager.tag=${tag}"
}

function kubefed-admission-webhook-ready() {
  local readyReplicas=$(kubectl -n ${1} get deployments.apps kubefed-admission-webhook -o jsonpath='{.status.readyReplicas}')
  [[ "${readyReplicas}" -ge "1" ]]
}

NS="${KUBEFED_NAMESPACE:-kube-federation-system}"
IMAGE_NAME="${1:-}"
NAMESPACED="${NAMESPACED:-}"

LATEST_IMAGE_NAME=quay.io/kubernetes-multicluster/kubefed:latest
if [[ "${IMAGE_NAME}" == "$LATEST_IMAGE_NAME" ]]; then
  USE_LATEST=y
else
  USE_LATEST=
fi

KF_NS_ARGS="--kubefed-namespace=${NS} "

if [[ -z "${IMAGE_NAME}" ]]; then
  >&2 echo "Usage: $0 <image> [join-cluster]...

<image>        should be in the form <containerregistry>/<username>/<imagename>:<tagname>

Example: docker.io/<username>/kubefed:test

If intending to use the docker hub as the container registry to push
the KubeFed image to, make sure to login to the local docker daemon
to ensure credentials are available for push:

  $ docker login --username <username>

<join-cluster> should be the kubeconfig context name for the additional cluster to join.
               NOTE: The host cluster is already included in the join.

"
  exit 2
fi

shift
# Allow for no specific JOIN_CLUSTERS: they probably want to kubefedctl themselves.
JOIN_CLUSTERS="${*-}"

# Use DOCKER_PUSH= ./scripts/deploy-kubefed.sh <image> to skip docker
# push on container image when not using latest image.
DOCKER_PUSH="${DOCKER_PUSH-y}"
DOCKER_PUSH_CMD="docker push ${IMAGE_NAME}"
if [[ ! "${DOCKER_PUSH}" ]]; then
    DOCKER_PUSH_CMD=
fi

# Build KubeFed binaries and image
if [[ ! "${USE_LATEST}" ]]; then
  cd "$(dirname "$0")/.."
  make container IMAGE_NAME=${IMAGE_NAME}
  cd -
  ${DOCKER_PUSH_CMD}
fi
cd "$(dirname "$0")/.."
make kubefedctl
cd -

# Deploy KubeFed resources
deploy-with-helm

# Wait for admission webhook server to be ready
util::wait-for-condition "kubefed admission webhook to be ready" "kubefed-admission-webhook-ready ${NS}" 120

# Join the host cluster
CONTEXT="$(kubectl config current-context)"
./bin/kubefedctl join "${CONTEXT}" --host-cluster-context "${CONTEXT}" --v=2 ${KF_NS_ARGS}

for c in ${JOIN_CLUSTERS}; do
  ./bin/kubefedctl join "${c}" --host-cluster-context "${CONTEXT}" --v=2 ${KF_NS_ARGS}
done
