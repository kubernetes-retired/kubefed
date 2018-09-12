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
# the path.  These and other test binaries can be installed via the
# download-binaries.sh script, which downloads to ./bin:
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

KF_NS_ARGS=
if [[ "${NAMESPACED}" ]]; then
  KF_NS_ARGS="--federation-namespace=${NS} --registry-namespace=${NS} --limited-scope=true"
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

# Use DOCKER_PUSH=false ./scripts/deploy-federation.sh <image> to skip docker
# push on container image when not using latest image.
DOCKER_PUSH=${DOCKER_PUSH:-true}
DOCKER_PUSH_CMD="docker push ${IMAGE_NAME}"
if [[ "${DOCKER_PUSH}" == false ]]; then
    DOCKER_PUSH_CMD=
fi

if [[ ! "${USE_LATEST}" ]]; then
  base_dir="$(cd "$(dirname "$0")/.." ; pwd)"
  dockerfile_dir="${base_dir}/images/federation-v2"
  go build -o "${dockerfile_dir}"/controller-manager "${base_dir}"/cmd/controller-manager/main.go
  docker build ${dockerfile_dir} -t "${IMAGE_NAME}"
  ${DOCKER_PUSH_CMD}
fi

if [[ "${NAMESPACED}" ]]; then
  if ! kubectl get ns "${NS}" > /dev/null 2>&1; then
    kubectl create ns "${NS}"
  fi
else
  if ! kubectl get ns "${PUBLIC_NS}" > /dev/null 2>&1; then
    kubectl create ns "${PUBLIC_NS}"
  fi
fi

# Create a permissive clusterrolebinding to allow federation controllers to run.
# TODO(marun) Make this more restrictive.
# TODO(marun) Use a rolebinding when namespaced once the
# federaatedtypeconfig controller can limit informer scope to the
# namespace.
kubectl create clusterrolebinding federation-admin --clusterrole=cluster-admin --serviceaccount="${NS}:default"

if [[ ! "${USE_LATEST}" ]]; then
  if [[ "${NAMESPACED}" ]]; then
    INSTALL_YAML="${INSTALL_YAML}" IMAGE_NAME="${IMAGE_NAME}" scripts/generate-namespaced-install-yaml.sh
  else
    INSTALL_YAML="${INSTALL_YAML}" IMAGE_NAME="${IMAGE_NAME}" scripts/generate-install-yaml.sh
  fi
fi

# TODO(marun) kubebuilder-generated installation yaml fails validation
# for seemingly harmless reasons on kube >= 1.11.  Ignore validation
# until the generated crd yaml can pass it.
kubectl -n "${NS}" apply --validate=false -f "${INSTALL_YAML}"
kubectl apply --validate=false -f vendor/k8s.io/cluster-registry/cluster-registry-crd.yaml

# TODO(marun) Ensure federatdtypeconfig is available before creating instances
# TODO(marun) Ensure crds are created for a given federated type before starting sync controller for that type

# Enable available types
for filename in ./config/federatedtypes/*.yaml; do
  # The namespace controller is not required when the control plane is namespaced
  if [[ "${NAMESPACED}" && "${filename}" == *namespace.yaml ]]; then
    continue
  fi
  kubectl -n "${NS}" apply -f "${filename}"
done

# Join the host cluster
go build -o bin/kubefed2 cmd/kubefed2/kubefed2.go
CONTEXT="$(kubectl config current-context)"
./bin/kubefed2 join "${CONTEXT}" --host-cluster-context "${CONTEXT}" --add-to-registry --v=2 ${KF_NS_ARGS}

for c in ${JOIN_CLUSTERS}; do
  ./bin/kubefed2 join "${c}" --host-cluster-context "${CONTEXT}" --add-to-registry --v=2 ${KF_NS_ARGS}

done
