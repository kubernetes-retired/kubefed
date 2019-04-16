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
# This script fixes up the configuration for k3s. (https://k3s.io/)
# Namely it updates ca.crt in secrets to match with the ones in
# KUBECONFIG. It's intended to be run after deploy-federation.sh
# script.
#
# Background:
# In k3s, different endpoints and certificates are configured for
# users (KUBECONFIG) and pods (service accounts).
# Because "kubefed2 join" uses the endpoint from KUBECONFIG and
# the certificate from a service account in the member cluster,
# the federation controller manager fails to communicate with the
# member clusters, producing the messages like the following.
#
# 	x509: certificate signed by unknown authority

set -o errexit
set -o nounset
set -o pipefail

CLUSTER_NAMES="${*-}"
NS="${FEDERATION_NAMESPACE:-federation-system}"

for CLUSTER_NAME in $CLUSTER_NAMES; do
	CA_CRT=$(kubectl config view --raw -o jsonpath='{.clusters[?(@.name=="'"${CLUSTER_NAME}"'")].cluster.certificate-authority-data}')
	SECRET_NAME=$(kubectl get -n ${NS} federatedclusters ${CLUSTER_NAME} -o jsonpath='{.spec.secretRef.name}')
	kubectl patch -n ${NS} secret ${SECRET_NAME} \
		--patch '{"data":{"ca.crt": "'${CA_CRT}'"}}'
done

# Restart the controller manager to make it reload the secrets
kubectl delete -n ${NS} po -l control-plane=controller-manager
