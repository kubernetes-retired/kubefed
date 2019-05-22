/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	fedv1b1 "sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	genericclient "sigs.k8s.io/kubefed/pkg/client/generic"
	"sigs.k8s.io/kubefed/pkg/controller/util"
	"sigs.k8s.io/kubefed/test/common"
)

func WaitForClusterReadiness(tl common.TestLogger, client genericclient.Client,
	namespace string, interval, timeout time.Duration) {
	clusterList := ListKubeFedClusters(tl, client, namespace)
	for _, cluster := range clusterList.Items {
		clusterIsReadyOrFail(tl, client, namespace, interval, timeout, &cluster)
	}
	tl.Logf("All federated clusters are ready")
}

func ListKubeFedClusters(tl common.TestLogger, client genericclient.Client, namespace string) *fedv1b1.KubeFedClusterList {
	clusterList := &fedv1b1.KubeFedClusterList{}
	err := client.List(context.TODO(), clusterList, namespace)
	if err != nil {
		tl.Fatalf("Error retrieving list of federated clusters: %+v", err)
	}
	if len(clusterList.Items) == 0 {
		tl.Fatal("No federated clusters found")
	}
	return clusterList
}

func clusterIsReadyOrFail(tl common.TestLogger, client genericclient.Client,
	namespace string, interval, timeout time.Duration, cluster *fedv1b1.KubeFedCluster) {
	clusterName := cluster.Name
	tl.Logf("Checking readiness for federated cluster %q", clusterName)
	if util.IsClusterReady(&cluster.Status) {
		return
	}
	err := wait.Poll(interval, timeout, func() (bool, error) {
		cluster := &fedv1b1.KubeFedCluster{}
		err := client.Get(context.TODO(), cluster, namespace, clusterName)
		if err != nil {
			return false, err
		}
		return util.IsClusterReady(&cluster.Status), nil
	})
	if err != nil {
		tl.Fatalf("Error determining readiness for cluster %q: %+v", clusterName, err)
	}
}
