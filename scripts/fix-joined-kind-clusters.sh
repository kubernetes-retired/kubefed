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

# This script updates APIEndpoints of KubeFedClusters for Docker
# running in a VM on MacOS.

if [ "`uname`" != 'Darwin' ]; then
  >&2 echo "This script is only intended for use on MacOS"
  exit 1
fi

NS="${KUBEFED_NAMESPACE:-kube-federation-system}"

INSPECT_PATH='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}'

CLUSTERS="$(kubectl get kubefedclusters -n "${NS}" -o jsonpath='{range .items[*]}{.metadata.name}{" "}{end}')"
for cluster in ${CLUSTERS};
do
  IP_ADDR="$(docker inspect -f "${INSPECT_PATH}" "${cluster}-control-plane")"
  kubectl patch kubefedclusters -n "${NS}" "${cluster}" --type merge \
          --patch "{\"spec\":{\"apiEndpoint\":\"https://${IP_ADDR}:6443\"}}"
done
