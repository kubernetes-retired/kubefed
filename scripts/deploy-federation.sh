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
# This script depends on crinit, kubectl, and apiserver-boot being
# installed in the path.  These binaries can be installed via the
# download-binaries.sh script, which downloads to ./bin:
#
#   $ ./scripts/download-binaries.sh
#   $ export PATH=$(pwd)/bin:${PATH}
#
# To redeploy federation from scratch, prefix the deploy invocation with the deletion script:
#
#   # WARNING: The deletion script will remove federation and cluster registry data
#   $ ./scripts/delete-federation.sh && ./scripts/deploy-federation.sh <image>
#

set -o errexit
set -o nounset
set -o pipefail

source "$(dirname "${BASH_SOURCE}")/util.sh"

NS=federation
IMAGE_NAME="${1:-}"

if [[ -z "${IMAGE_NAME}" ]]; then
  >&2 echo "Usage: $0 <image>

<image> should be in the form <containerregistry>/<username>/<imagename>:<tagname>

Example: docker.io/<username>/federation-v2:test

If intending to use the docker hub as the container registry to push
the federation image to, make sure to login to the local docker daemon
to ensure credentials are available for push:

  $ docker login --username <username>
"
  exit 2
fi

# Wait for the storage provisioner to be ready.  If crinit is called
# before dynamic provisioning is enabled, the pvc for etcd will never
# be bound.
function storage-provisioner-ready() {
  local result="$(kubectl -n kube-system get pod storage-provisioner -o jsonpath='{.status.conditions[?(@.type == "Ready")].status}' 2> /dev/null)"
  [[ "${result}" = "True" ]]
}
util::wait-for-condition "storage provisioner readiness" 'storage-provisioner-ready' 60

crinit aggregated init mycr

kubectl create ns "${NS}"

apiserver-boot run in-cluster --name "${NS}" --namespace "${NS}" --image "${IMAGE_NAME}" --controller-args="-logtostderr,-v=4"

# Create a permissive rolebinding to allow federation controllers to run.
# TODO(marun) Make this more restrictive.
kubectl create clusterrolebinding federation-admin --clusterrole=cluster-admin --serviceaccount="${NS}:default"

# Increase the memory limit of the apiserver to ensure it can start.
kubectl -n federation patch deploy federation -p '{"spec":{"template":{"spec":{"containers":[{"name":"apiserver","resources":{"limits":{"memory":"128Mi"},"requests":{"memory":"64Mi"}}}]}}}}'

# Wait for the apiserver to become available so that join can be performed.
function apiserver-available() {
  # Being able to retrieve without error indicates the apiserver is available
  kubectl get federatedcluster 2> /dev/null
}
util::wait-for-condition "apiserver availability" 'apiserver-available' 60

# Enable available types
for filename in ./config/federatedtypes/*.yaml; do
  kubectl create -f "${filename}"
done

# Join the host cluster
go build -o bin/kubefnord cmd/kubefnord/kubefnord.go
CONTEXT="$(kubectl config current-context)"
./bin/kubefnord join "${CONTEXT}" --host-cluster-context "${CONTEXT}" --add-to-registry --v=2
