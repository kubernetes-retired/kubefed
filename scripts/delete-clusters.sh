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

# This script handles the deletion of multiple clusters using kind as well as
# the deletion of the container registry.

set -o errexit
set -o nounset
set -o pipefail

NUM_CLUSTERS="${NUM_CLUSTERS:-2}"

function delete-clusters() {
  local num_clusters=${1}

  for i in $(seq "${num_clusters}"); do
    # The context name has been changed when creating clusters by 'create-cluster.sh'.
    # This will result in the context can't be removed by kind when deleting a cluster.
    # So, we need to change context name back and let kind take care about it.
    kubectl config rename-context "cluster${i}" "kind-cluster${i}"

    kind delete cluster --name "cluster${i}"
  done
}

echo "Deleting ${NUM_CLUSTERS} clusters"
delete-clusters "${NUM_CLUSTERS}"

echo "Complete"
