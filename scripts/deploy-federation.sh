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

function deploy-with-script() {
  if [[ ! "${USE_LATEST}" ]]; then
    if [[ "${NAMESPACED}" ]]; then
      INSTALL_YAML="${INSTALL_YAML}" IMAGE_NAME="${IMAGE_NAME}" scripts/generate-namespaced-install-yaml.sh
    else
      INSTALL_YAML="${INSTALL_YAML}" IMAGE_NAME="${IMAGE_NAME}" FEDERATION_NAMESPACE="${NS}" scripts/generate-install-yaml.sh
    fi
  fi

  # TODO(marun) kubebuilder-generated installation yaml fails validation
  # for seemingly harmless reasons on kube >= 1.11.  Ignore validation
  # until the generated crd yaml can pass it.
  kubectl -n "${NS}" apply --validate=false -f "${INSTALL_YAML}"
  kubectl apply --validate=false -f vendor/k8s.io/cluster-registry/cluster-registry-crd.yaml

  # TODO(marun) Ensure federatdtypeconfig is available before creating instances
  # TODO(marun) Ensure crds are created for a given federated type before starting sync controller for that type

  # Wait for the propagation of the clusterregistry CRD
  util::wait-for-condition "propagation of the clusterregistry CRD" \
    "kubectl api-resources | grep clusterregistry.k8s.io > /dev/null" \
    10

  # Enable available types
  for filename in ./config/enabletypedirectives/*.yaml; do
    ./bin/kubefed2 enable -f "${filename}" --federation-namespace="${NS}"
  done
}

function deploy-with-helm() {
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
  util::wait-for-condition "Tiller is ready" "helm version --server &> /dev/null" 120

  REPOSITORY=${IMAGE_NAME%/*}
  IMAGE_TAG=${IMAGE_NAME##*/}
  IMAGE=${IMAGE_TAG%:*}
  TAG=${IMAGE_TAG#*:}

  if [[ "${NAMESPACED}" ]]; then
      helm install charts/federation-v2 --name federation-v2-${NS} --namespace ${NS} \
          --set controllermanager.repository=${REPOSITORY} --set controllermanager.image=${IMAGE} --set controllermanager.tag=${TAG} \
          --set global.limitedScope=true
  else
      helm install charts/federation-v2 --name federation-v2 --namespace ${NS} \
          --set controllermanager.repository=${REPOSITORY} --set controllermanager.image=${IMAGE} --set controllermanager.tag=${TAG}
  fi
}

NS="${FEDERATION_NAMESPACE:-federation-system}"
PUBLIC_NS=kube-multicluster-public
IMAGE_NAME="${1:-}"
NAMESPACED="${NAMESPACED:-}"

LATEST_IMAGE_NAME=quay.io/kubernetes-multicluster/federation-v2:latest
if [[ "${IMAGE_NAME}" == "$LATEST_IMAGE_NAME" ]]; then
  USE_LATEST=y
  INSTALL_YAML=hack/install-latest.yaml
else
  USE_LATEST=
  INSTALL_YAML=hack/install.yaml
fi

KF_NS_ARGS="--federation-namespace=${NS} "
if [[ "${NAMESPACED}" ]]; then
  KF_NS_ARGS+="--registry-namespace=${NS} --limited-scope=true"
  INSTALL_YAML=hack/install-namespaced.yaml
fi

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
JOIN_CLUSTERS="${*}"

# Use DOCKER_PUSH= ./scripts/deploy-federation.sh <image> to skip docker
# push on container image when not using latest image.
DOCKER_PUSH="${DOCKER_PUSH:-y}"
DOCKER_PUSH_CMD="docker push ${IMAGE_NAME}"
if [[ ! "${DOCKER_PUSH}" ]]; then
    DOCKER_PUSH_CMD=
fi

# Build federation binaries and image
if [[ ! "${USE_LATEST}" ]]; then
  base_dir="$(cd "$(dirname "$0")/.." ; pwd)"
  dockerfile_dir="${base_dir}/images/federation-v2"
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${dockerfile_dir}"/hyperfed "${base_dir}"/cmd/hyperfed/main.go
  docker build ${dockerfile_dir} -t "${IMAGE_NAME}"
  ${DOCKER_PUSH_CMD}
fi
go build -o bin/kubefed2 cmd/kubefed2/kubefed2.go

if ! kubectl get ns "${NS}" > /dev/null 2>&1; then
  kubectl create ns "${NS}"
fi

if [[ ! "${NAMESPACED}" ]]; then
  if ! kubectl get ns "${PUBLIC_NS}" > /dev/null 2>&1; then
    kubectl create ns "${PUBLIC_NS}"
  fi
fi

# Create a permissive clusterrolebinding to allow federation controllers to run.
if [[ "${NAMESPACED}" ]]; then
  # TODO(marun) Investigate why cluster-admin is required to view cluster registry clusters in a namespace
  kubectl -n "${NS}" create rolebinding federation-admin --clusterrole=cluster-admin --serviceaccount="${NS}:default"
else
  # TODO(marun) Make this more restrictive.
  kubectl create clusterrolebinding federation-admin --clusterrole=cluster-admin --serviceaccount="${NS}:default"
fi

# Deploy federation resources
USE_CHART="${USE_CHART:-}"
if [[ "${USE_CHART}" ]]; then
  deploy-with-helm
else
  deploy-with-script
fi

# Join the host cluster
CONTEXT="$(kubectl config current-context)"
./bin/kubefed2 join "${CONTEXT}" --host-cluster-context "${CONTEXT}" --add-to-registry --v=2 ${KF_NS_ARGS}

for c in ${JOIN_CLUSTERS}; do
  ./bin/kubefed2 join "${c}" --host-cluster-context "${CONTEXT}" --add-to-registry --v=2 ${KF_NS_ARGS}
done
