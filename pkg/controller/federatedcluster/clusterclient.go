/*
Copyright 2016 The Kubernetes Authors.

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

package federatedcluster

import (
	"strings"

	fedcommon "github.com/kubernetes-sigs/federation-v2/pkg/apis/federation/common"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/federation/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	crclientset "k8s.io/cluster-registry/pkg/client/clientset_generated/clientset"
)

const (
	UserAgentName = "Cluster-Controller"
)

type ClusterClient struct {
	kubeClient *kubeclientset.Clientset
}

func NewClusterClientSet(c *fedv1a1.FederatedCluster, kubeClient kubeclientset.Interface, crClient crclientset.Interface) (*ClusterClient, error) {
	clusterConfig, err := util.BuildClusterConfig(c, kubeClient, crClient)
	if err != nil {
		return nil, err
	}
	var clusterClientSet = ClusterClient{}
	if clusterConfig != nil {
		clusterClientSet.kubeClient = kubeclientset.NewForConfigOrDie((restclient.AddUserAgent(clusterConfig, UserAgentName)))
		if clusterClientSet.kubeClient == nil {
			return nil, nil
		}
	}
	return &clusterClientSet, nil
}

// GetClusterHealthStatus gets the kubernetes cluster health status by requesting "/healthz"
func (self *ClusterClient) GetClusterHealthStatus() *fedv1a1.FederatedClusterStatus {
	clusterStatus := fedv1a1.FederatedClusterStatus{}
	currentTime := metav1.Now()
	newClusterReadyCondition := fedv1a1.ClusterCondition{
		Type:               fedcommon.ClusterReady,
		Status:             corev1.ConditionTrue,
		Reason:             "ClusterReady",
		Message:            "/healthz responded with ok",
		LastProbeTime:      currentTime,
		LastTransitionTime: currentTime,
	}
	newClusterNotReadyCondition := fedv1a1.ClusterCondition{
		Type:               fedcommon.ClusterReady,
		Status:             corev1.ConditionFalse,
		Reason:             "ClusterNotReady",
		Message:            "/healthz responded without ok",
		LastProbeTime:      currentTime,
		LastTransitionTime: currentTime,
	}
	newNodeOfflineCondition := fedv1a1.ClusterCondition{
		Type:               fedcommon.ClusterOffline,
		Status:             corev1.ConditionTrue,
		Reason:             "ClusterNotReachable",
		Message:            "cluster is not reachable",
		LastProbeTime:      currentTime,
		LastTransitionTime: currentTime,
	}
	newNodeNotOfflineCondition := fedv1a1.ClusterCondition{
		Type:               fedcommon.ClusterOffline,
		Status:             corev1.ConditionFalse,
		Reason:             "ClusterReachable",
		Message:            "cluster is reachable",
		LastProbeTime:      currentTime,
		LastTransitionTime: currentTime,
	}
	body, err := self.kubeClient.DiscoveryClient.RESTClient().Get().AbsPath("/healthz").Do().Raw()
	if err != nil {
		clusterStatus.Conditions = append(clusterStatus.Conditions, newNodeOfflineCondition)
	} else {
		if !strings.EqualFold(string(body), "ok") {
			clusterStatus.Conditions = append(clusterStatus.Conditions, newClusterNotReadyCondition, newNodeNotOfflineCondition)
		} else {
			clusterStatus.Conditions = append(clusterStatus.Conditions, newClusterReadyCondition)
		}
	}

	return &clusterStatus
}
