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

set -o errexit
set -o nounset
set -o pipefail

INSTALL_YAML="${INSTALL_YAML:-hack/install-namespaced.yaml}"
IMAGE_NAME="${IMAGE_NAME:-quay.io/kubernetes-multicluster/federation-v2:latest}"

INSTALL_YAML="${INSTALL_YAML}" IMAGE_NAME="${IMAGE_NAME}" scripts/generate-install-yaml.sh

# Remove namespace resource from the top of the file
sed -i -e '/---/,$!d' "${INSTALL_YAML}"

# Remove namespace fields
sed -i -e '/^  namespace: federation-system$/d' "${INSTALL_YAML}"

# Convert rbac from cluster- to namespace-scoped
sed -i -e 's/ClusterRole/Role/' "${INSTALL_YAML}"

# Add args to container
sed -i -e '/--install-crds=false/a\
        - --limited-scope=true\
        - --federation-namespace=$(FEDERATION_NAMESPACE)\
        - --registry-namespace=$(CLUSTER_REGISTRY_NAMESPACE)' "${INSTALL_YAML}"

# Add namespace env args to container
 sed -i -e '/terminationGracePeriodSeconds/i\
        env:\
        - name: FEDERATION_NAMESPACE\
          valueFrom:\
            fieldRef:\
              fieldPath: metadata.namespace\
        - name: CLUSTER_REGISTRY_NAMESPACE\
          valueFrom:\
            fieldRef:\
              fieldPath: metadata.namespace' "${INSTALL_YAML}"
