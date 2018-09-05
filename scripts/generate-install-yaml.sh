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

INSTALL_YAML="${INSTALL_YAML:-hack/install-latest.yaml}"
IMAGE_NAME="${IMAGE_NAME:-quay.io/kubernetes-multicluster/federation-v2:latest}"

kubebuilder create config --controller-image "${IMAGE_NAME}" \
            --name federation --output "${INSTALL_YAML}"

# Increase memory request and limit to avoid OOM issues.
sed -i 's/memory: 20Mi/memory: 64Mi/' "${INSTALL_YAML}"
sed -i 's/memory: 30Mi/memory: 128Mi/' "${INSTALL_YAML}"

# Delete the 'type' field from the openapi schema to avoid
# triggering validation errors in kube < 1.12 when a type specifies
# a subresource (e.g. status).  The 'type' field only triggers
# validation errors for resource types that define subresources, but
# it is simpler to fix for all types.
#
# Reference: https://github.com/kubernetes/kubernetes/issues/65293
sed -i -e '/^      type: object$/d' "${INSTALL_YAML}"
