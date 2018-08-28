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
	"fmt"
	"strings"

	"github.com/golang/glog"

	fedcommon "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/common"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	crclientset "k8s.io/cluster-registry/pkg/client/clientset/versioned"
)

const (
	UserAgentName = "Cluster-Controller"

	// Following labels come from k8s.io/kubernetes/pkg/kubelet/apis
	LabelZoneFailureDomain = "failure-domain.beta.kubernetes.io/zone"
	LabelZoneRegion        = "failure-domain.beta.kubernetes.io/region"
)

// ClusterClient provides methods for determining the status and zones of a
// particular FederatedCluster.
type ClusterClient struct {
	kubeClient *kubeclientset.Clientset
}

// NewClusterClientSet returns a ClusterClient for the given FederatedCluster.
// The kubeClient and crClient are used to configure the ClusterClient's
// internal client with information from a kubeconfig stored in a kubernetes
// secret and an API endpoint from the cluster-registry.
func NewClusterClientSet(c *fedv1a1.FederatedCluster, kubeClient kubeclientset.Interface, crClient crclientset.Interface, fedNamespace, clusterNamespace string) (*ClusterClient, error) {
	clusterConfig, err := util.BuildClusterConfig(c, kubeClient, crClient, fedNamespace, clusterNamespace)
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

// GetClusterZones gets the kubernetes cluster zones and region by inspecting labels on nodes in the cluster.
func (self *ClusterClient) GetClusterZones() (zone, region string, err error) {
	nodes, err := self.kubeClient.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		glog.Errorf("Failed to list nodes while getting zone names: %v", err)
		return "", "", err
	}
	for i, node := range nodes.Items {
		zone, err = getZoneNameForNode(node)
		if err != nil {
			return "", "", err
		}
		if i == 0 {
			region, err = getRegionNameForNode(node)
			if err != nil {
				return "", "", err
			}
		}
		// TODO: Optimize this flow. All nodes will have the same zone label.
		// So just considering first node for now.
		break
	}
	return zone, region, nil
}

// Find the name of the zone in which a Node is running.
func getZoneNameForNode(node corev1.Node) (string, error) {
	for key, value := range node.Labels {
		if key == LabelZoneFailureDomain {
			return value, nil
		}
	}
	return "", fmt.Errorf("Zone name for node %s not found. No label with key %s",
		node.Name, LabelZoneFailureDomain)
}

// Find the name of the region in which a Node is running.
func getRegionNameForNode(node corev1.Node) (string, error) {
	for key, value := range node.Labels {
		if key == LabelZoneRegion {
			return value, nil
		}
	}
	return "", fmt.Errorf("Region name for node %s not found. No label with key %s",
		node.Name, LabelZoneRegion)
}
