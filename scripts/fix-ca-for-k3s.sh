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

# Usage:
# % ./scripts/fix-ca-for-k3s.sh [member-cluster] ...
#
# Description:
# This script fixes up the configuration for member clusters
# running older versions of k3s (< v0.7.0). (https://k3s.io/)
# Namely it updates caBundle for the member clusters to match with
# the ones in KUBECONFIG. It's intended to be run after joining
# member clusters successfully.
# Note that this is not necessary for k3s v0.7.0.
#
# Background:
# In k3s < v0.7.0, different endpoints and certificates are configured for
# users (KUBECONFIG) and pods (service accounts).
# Because "kubefedctl join" uses the endpoint from KUBECONFIG and
# the certificate from a service account in the member cluster,
# the kubefed controller manager fails to communicate with the
# member clusters, producing the messages like the following.
#
# 	x509: certificate signed by unknown authority
#
# k3s v0.7.0 has been changed to use the same CA cert to sign them. [1]
# Thus this workaround is no longer necessary.
# [1] https://github.com/rancher/k3s/commit/2c9444399b427ffb706818f5bf3892a8880673bf

set -o errexit
set -o nounset
set -o pipefail

CLUSTER_NAMES="${*-}"
NS="${KUBEFED_NAMESPACE:-kube-federation-system}"

for CLUSTER_NAME in $CLUSTER_NAMES; do
	CA_CRT=$(kubectl config view --raw -o jsonpath='{.clusters[?(@.name=="'"${CLUSTER_NAME}"'")].cluster.certificate-authority-data}')
	kubectl patch -n ${NS} kubefedclusters "${CLUSTER_NAME}" \
		--type='merge' \
		--patch '{"spec": {"caBundle": "'${CA_CRT}'"}}'
done

# Restart the controller manager to make it reload the caBundle
kubectl delete -n ${NS} po -l control-plane=controller-manager
