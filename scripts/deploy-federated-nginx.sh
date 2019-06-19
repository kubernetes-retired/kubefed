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

# This script has an end to end steps in deploying a federated deployment

set -o errexit
set -o nounset
set -o pipefail

source "$(dirname "${BASH_SOURCE}")/util.sh"

TEST_NS="test-namespace"

# Prerequisite:
# - a cluster scoped KubeFed control plane with `cluster1` and
#   `cluster2` joined as member clusters
# - kubectl config use-context cluster1
#
echo "Validating KubeFed walkthrough."


# Create kubernetes resources
echo
echo "Creating a test namespace ${TEST_NS}."
kubectl create ns ${TEST_NS}

echo
echo "Creating a ngnix application resources in namespace ${TEST_NS}."
kubectl apply -n ${TEST_NS} -f example/sample1/configmap.yaml
kubectl apply -n ${TEST_NS} -f example/sample1/deployment.yaml
kubectl apply -n ${TEST_NS} -f example/sample1/service.yaml


# Federate kubernetes resources to member clusters
echo
echo "Federating resources in namespace ${TEST_NS} to member clusters."
kubefedctl federate ns ${TEST_NS} --contents --skip-api-resources 'pods,secrets,serviceaccount,replicaset'

echo
echo "Checking status of federated resources."
util::wait-for-condition 'federated deployment status updated in cluster1' \
    "(kubectl get federateddeployment nginx -n ${TEST_NS} -o jsonpath='{.status.clusters[*].name}' | grep '\<cluster1\>') &> /dev/null" 120
util::wait-for-condition 'federated deployment status updated in cluster2' \
    "(kubectl get federateddeployment nginx -n ${TEST_NS} -o jsonpath='{.status.clusters[*].name}' | grep '\<cluster2\>') &> /dev/null" 120

echo
echo "Querying web server of deployment nginx application from member clusters."
IP1=$(kubectl get node -o jsonpath="{.items[0].status.addresses[0].address}" --context cluster1)
IP2=$(kubectl get node -o jsonpath="{.items[0].status.addresses[0].address}" --context cluster2)

util::wait-for-condition 'nodePort ok in cluster1' \
    "kubectl --namespace ${TEST_NS} --context cluster1 get service -o jsonpath='{.items[0].spec.ports[0].nodePort}' &> /dev/null" 30
util::wait-for-condition 'nodePort ok in cluster2' \
    "kubectl --namespace ${TEST_NS} --context cluster2 get service -o jsonpath='{.items[0].spec.ports[0].nodePort}' &> /dev/null" 30
PORT1=$(kubectl get service -n ${TEST_NS} --context cluster1 -o jsonpath="{.items[0].spec.ports[0].nodePort}")
PORT2=$(kubectl get service -n ${TEST_NS} --context cluster2 -o jsonpath="{.items[0].spec.ports[0].nodePort}")

util::wait-for-condition 'nginx web server ok in cluster1' "curl ${IP1}:${PORT1} &> /dev/null" 120
util::wait-for-condition 'nginx web server ok in cluster2' "curl ${IP2}:${PORT2} &> /dev/null" 120
echo "cluster1: $(curl -s ${IP1}:${PORT1})"
echo "cluster2: $(curl -s ${IP2}:${PORT2})"


# Modify federated resources to update kubernetes resources in member clusters
echo
echo "Changing index.html in federated configmap."
kubectl patch federatedconfigmap web-file -n ${TEST_NS} --type=merge -p '{"spec": {"template": {"data": {"index.html": "Hello from KubeFed!"}}}}'

echo
echo "Querying web server of deployed nginx application from member clusters."
util::wait-for-condition 'web content changed in cluster1' "(curl ${IP1}:${PORT1} | grep '^Hello from KubeFed') &> /dev/null" 120
util::wait-for-condition 'web content changed in cluster2' "(curl ${IP2}:${PORT2} | grep '^Hello from KubeFed') &> /dev/null" 120
echo "cluster1: $(curl -s ${IP1}:${PORT1})"
echo "cluster2: $(curl -s ${IP2}:${PORT2})"

echo
echo "Updating override of federated deployment nginx to increase 'replicas' to 2 in cluster2."
kubectl patch federateddeployment nginx -n ${TEST_NS} --type=merge --patch \
    '{"spec" : {"overrides": [{"clusterName" : "cluster2", "clusterOverrides": [{"path": "/spec/replicas", "value" : 2}]}]}}'
echo "Checking 'replicas' of deployment nginx in cluster2."
util::wait-for-condition 'replicas updated in cluster2' \
    "(kubectl get deployment nginx -n ${TEST_NS} --context cluster2 -o jsonpath='{.status.replicas}' | grep "^2$") &> /dev/null" 120

echo
echo "Updating placement to include only 'cluster2' so that the deployment will be removed from 'cluster1'"
kubectl patch federateddeployment nginx -n ${TEST_NS} --type=merge --patch '{"spec": {"placement": {"clusters": [{"name": "cluster2"}]}}}'
echo "Checking status of federated deployment nginx."
util::wait-for-condition 'federated deployment status updated' \
    "(kubectl get federateddeployment nginx -n ${TEST_NS} -o jsonpath='{.status.clusters[*].name}' | grep "^cluster2$") &> /dev/null" 120

echo
echo "Successfully completed KubeFed walkthrough."
