#!/usr/bin/env bash

# Copyright 2019 The Kubernetes Authors.
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

set -o errexit
set -o nounset
set -o pipefail

# This script updates APIEndpoints of KubeFedCluster resources for
# Docker on MacOS.
#
# By default, `https://<kind-pod-ip>:6443` is used to ensure
# compatibility with control plane components running in a kind
# cluster.
#
# If LOCAL_TESTING is set, the api endpoint defined in the local
# kubeconfig is used to ensure compatibility with control plane
# components run in-memory by local e2e tests.
LOCAL_TESTING="${LOCAL_TESTING:-}"

if [ "`uname`" != 'Darwin' ]; then
  >&2 echo "This script is only intended for use on MacOS"
  exit 1
fi

NS="${KUBEFED_NAMESPACE:-kube-federation-system}"

INSPECT_PATH='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}'

CLUSTERS="$(kubectl get kubefedclusters -n "${NS}" -o jsonpath='{range .items[*]}{.metadata.name}{" "}{end}')"
for cluster in ${CLUSTERS};
do
  if [[ "${LOCAL_TESTING}" ]]; then
    ENDPOINT="$(kubectl config view -o jsonpath='{.clusters[?(@.name == "'"${cluster}"'")].cluster.server}')"
  else
    IP_ADDR="$(docker inspect -f "${INSPECT_PATH}" "${cluster}-control-plane")"
    ENDPOINT="https://${IP_ADDR}:6443"
  fi
  kubectl patch kubefedclusters -n "${NS}" "${cluster}" --type merge \
          --patch "{\"spec\":{\"apiEndpoint\":\"${ENDPOINT}\"}}"
done
