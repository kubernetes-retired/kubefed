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
	"sync"
	"time"

	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	crclientset "k8s.io/cluster-registry/pkg/client/clientset/versioned"

	"github.com/golang/glog"
)

// ClusterController is responsible for maintaining the health status of each
// FederatedCluster in a particular namespace.
type ClusterController struct {
	// fedClient is used to access Federation resources in the host cluster.
	fedClient fedclientset.Interface

	// kubeClient is used to access Secrets in the host cluster.
	kubeClient kubeclientset.Interface

	// crClient is used to access the cluster registry in the host cluster.
	crClient crclientset.Interface

	// clusterMonitorPeriod is the period for updating status of cluster
	clusterMonitorPeriod time.Duration

	mu sync.RWMutex

	// knownClusterSet is the set of clusters known to this controller.
	knownClusterSet sets.String

	// clusterStatusMap is a mapping of clusterName and cluster status as
	// of last sampling.
	clusterStatusMap map[string]fedv1a1.FederatedClusterStatus

	// clusterKubeClientMap is a mapping of clusterName and the ClusterClient
	// for that cluster.
	clusterKubeClientMap map[string]ClusterClient

	// clusterController is the cache.Controller where callbacks are registered
	// for events on FederatedClusters.
	clusterController cache.Controller

	// fedNamespace is the name of the namespace containing
	// FederatedCluster resources and their associated secrets.
	fedNamespace string

	// clusterNamespace is the namespace containing Cluster resources.
	clusterNamespace string
}

// StartClusterController starts a new cluster controller.
func StartClusterController(config *util.ControllerConfig, stopChan <-chan struct{}, clusterMonitorPeriod time.Duration) {
	fedClient, kubeClient, crClient := config.AllClients("cluster-controller")
	controller := newClusterController(fedClient, kubeClient, crClient, config.FederationNamespaces, clusterMonitorPeriod)
	glog.Infof("Starting cluster controller")
	controller.Run(stopChan)
}

// newClusterController returns a new cluster controller
func newClusterController(fedClient fedclientset.Interface, kubeClient kubeclientset.Interface, crClient crclientset.Interface, namespaces util.FederationNamespaces, clusterMonitorPeriod time.Duration) *ClusterController {
	cc := &ClusterController{
		knownClusterSet:      make(sets.String),
		fedClient:            fedClient,
		kubeClient:           kubeClient,
		crClient:             crClient,
		clusterMonitorPeriod: clusterMonitorPeriod,
		clusterStatusMap:     make(map[string]fedv1a1.FederatedClusterStatus),
		clusterKubeClientMap: make(map[string]ClusterClient),
		fedNamespace:         namespaces.FederationNamespace,
		clusterNamespace:     namespaces.ClusterNamespace,
	}
	_, cc.clusterController = cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return cc.fedClient.CoreV1alpha1().FederatedClusters(cc.fedNamespace).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return cc.fedClient.CoreV1alpha1().FederatedClusters(cc.fedNamespace).Watch(options)
			},
		},
		&fedv1a1.FederatedCluster{},
		util.NoResyncPeriod,
		cache.ResourceEventHandlerFuncs{
			DeleteFunc: cc.delFromClusterSet,
			AddFunc:    cc.addToClusterSet,
		},
	)
	return cc
}

// delFromClusterSet delete a cluster from clusterSet and
// delete the corresponding restclient from the map clusterKubeClientMap
func (cc *ClusterController) delFromClusterSet(obj interface{}) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cluster := obj.(*fedv1a1.FederatedCluster)
	cc.delFromClusterSetByName(cluster.Name)
}

// delFromClusterSetByName delete a cluster from clusterSet by name and
// delete the corresponding restclient from the map clusterKubeClientMap.
// Caller must make sure that they hold the mutex
func (cc *ClusterController) delFromClusterSetByName(clusterName string) {
	glog.V(1).Infof("ClusterController observed a cluster deletion: %v", clusterName)
	cc.knownClusterSet.Delete(clusterName)
	delete(cc.clusterKubeClientMap, clusterName)
	delete(cc.clusterStatusMap, clusterName)
}

func (cc *ClusterController) addToClusterSet(obj interface{}) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cluster := obj.(*fedv1a1.FederatedCluster)
	cc.addToClusterSetWithoutLock(cluster)
}

// addToClusterSetWithoutLock inserts the new cluster to clusterSet and create
// a corresponding restclient to map clusterKubeClientMap if the cluster is not
// known. Caller must make sure that they hold the mutex.
func (cc *ClusterController) addToClusterSetWithoutLock(cluster *fedv1a1.FederatedCluster) {
	if cc.knownClusterSet.Has(cluster.Name) {
		return
	}
	glog.V(1).Infof("ClusterController observed a new cluster: %v", cluster.Name)
	cc.knownClusterSet.Insert(cluster.Name)
	// create the restclient of cluster
	restClient, err := NewClusterClientSet(cluster, cc.kubeClient, cc.crClient, cc.fedNamespace, cc.clusterNamespace)
	if err != nil || restClient == nil {
		glog.Errorf("Failed to create corresponding restclient of kubernetes cluster: %v", err)
		return
	}
	cc.clusterKubeClientMap[cluster.Name] = *restClient
}

// Run begins watching and syncing.
func (cc *ClusterController) Run(stopChan <-chan struct{}) {
	defer utilruntime.HandleCrash()
	go cc.clusterController.Run(stopChan)
	// monitor cluster status periodically, in phase 1 we just get the health state from "/healthz"
	go wait.Until(func() {
		if err := cc.updateClusterStatus(); err != nil {
			glog.Errorf("Error monitoring cluster status: %v", err)
		}
	}, cc.clusterMonitorPeriod, stopChan)
}

// updateClusterStatus checks cluster status and get the metrics from cluster's restapi
func (cc *ClusterController) updateClusterStatus() error {
	clusters, err := cc.fedClient.CoreV1alpha1().FederatedClusters(cc.fedNamespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, cluster := range clusters.Items {
		cc.mu.RLock()
		// skip updating status of the cluster which is not yet added to knownClusterSet.
		if !cc.knownClusterSet.Has(cluster.Name) {
			cc.mu.RUnlock()
			continue
		}
		clusterClient, clientFound := cc.clusterKubeClientMap[cluster.Name]
		clusterStatusOld, statusFound := cc.clusterStatusMap[cluster.Name]
		cc.mu.RUnlock()

		if !clientFound {
			glog.Warningf("Failed to get client for cluster %s", cluster.Name)
			continue
		}
		clusterStatusNew := clusterClient.GetClusterHealthStatus()
		if !statusFound {
			glog.Infof("There is no status stored for cluster: %v before", cluster.Name)
		} else {
			hasTransition := false
			if len(clusterStatusNew.Conditions) != len(clusterStatusOld.Conditions) {
				hasTransition = true
			} else {
				for i := 0; i < len(clusterStatusNew.Conditions); i++ {
					if !(strings.EqualFold(string(clusterStatusNew.Conditions[i].Type), string(clusterStatusOld.Conditions[i].Type)) &&
						strings.EqualFold(string(clusterStatusNew.Conditions[i].Status), string(clusterStatusOld.Conditions[i].Status))) {
						hasTransition = true
						break
					}
				}
			}

			if !hasTransition {
				for j := 0; j < len(clusterStatusNew.Conditions); j++ {
					clusterStatusNew.Conditions[j].LastTransitionTime = clusterStatusOld.Conditions[j].LastTransitionTime
				}
			}
		}

		zone, region, err := clusterClient.GetClusterZones()
		if err != nil {
			glog.Warningf("Failed to get zones and region for cluster with client %v: %v", clusterClient, err)
		} else {
			if len(zone) == 0 {
				zone = cluster.Status.Zone
			}
			if len(region) == 0 {
				region = cluster.Status.Region
			}
			clusterStatusNew.Zone = zone
			clusterStatusNew.Region = region
		}

		cc.mu.Lock()
		cc.clusterStatusMap[cluster.Name] = *clusterStatusNew
		cc.mu.Unlock()
		cluster.Status = *clusterStatusNew
		_, err = cc.fedClient.CoreV1alpha1().FederatedClusters(cc.fedNamespace).UpdateStatus(&cluster)
		if err != nil {
			glog.Warningf("Failed to update the status of cluster: %v, error is : %v", cluster.Name, err)
			// Don't return err here, as we want to continue processing remaining clusters.
			continue
		}
	}
	return nil
}
