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

if [ "$1" == "" ];then
    echo "kubernetes cluster context list need to be provided. eg. cluster1,cluster2,cluster3"; exit 1
fi
CLUSTER_CONTEXT=${1//,/ }

if [ "`uname`" == 'Darwin' ];then

    # We need to fix cluster ip addr in cluster-registry for mac os.
    # Assume all context was contained in current kubeconfig.
    for c in ${CLUSTER_CONTEXT};
    do
        ip_addr=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' ${c}-control-plane)

        kubectl patch clusters -n kube-multicluster-public ${c} --type merge \
            --patch "{\"spec\":{\"kubernetesApiEndpoints\":{\"serverEndpoints\":[{\"clientCIDR\":\"0.0.0.0/0\", \"serverAddress\":\"https://${ip_addr}:6443\"}]}}}"
    done
fi

echo "cluster $1 address patched successfully."