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
	"context"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/client-go/tools/cache"
	fedv1a1 "sigs.k8s.io/federation-v2/pkg/apis/core/v1alpha1"
	"sigs.k8s.io/federation-v2/pkg/controller/util"
	"sigs.k8s.io/federation-v2/pkg/features"

	genericclient "sigs.k8s.io/federation-v2/pkg/client/generic"
)

// ClusterController is responsible for maintaining the health status of each
// FederatedCluster in a particular namespace.
type ClusterController struct {
	client genericclient.Client

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
func StartClusterController(config *util.ControllerConfig, stopChan <-chan struct{}, clusterMonitorPeriod time.Duration) error {
	controller, err := newClusterController(config, clusterMonitorPeriod)
	if err != nil {
		return err
	}
	glog.Infof("Starting cluster controller")
	controller.Run(stopChan)
	return nil
}

// newClusterController returns a new cluster controller
func newClusterController(config *util.ControllerConfig, clusterMonitorPeriod time.Duration) (*ClusterController, error) {
	client := genericclient.NewForConfigOrDieWithUserAgent(config.KubeConfig, "cluster-controller")

	cc := &ClusterController{
		knownClusterSet:      make(sets.String),
		client:               client,
		clusterMonitorPeriod: clusterMonitorPeriod,
		clusterStatusMap:     make(map[string]fedv1a1.FederatedClusterStatus),
		clusterKubeClientMap: make(map[string]ClusterClient),
		fedNamespace:         config.FederationNamespace,
		clusterNamespace:     config.ClusterNamespace,
	}
	var err error
	_, cc.clusterController, err = util.NewGenericInformerWithEventHandler(
		config.KubeConfig,
		config.FederationNamespace,
		&fedv1a1.FederatedCluster{},
		util.NoResyncPeriod,
		&cache.ResourceEventHandlerFuncs{
			DeleteFunc: cc.delFromClusterSet,
			AddFunc:    cc.addToClusterSet,
		},
	)
	return cc, err
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
	restClient, err := NewClusterClientSet(cluster, cc.client, cc.fedNamespace, cc.clusterNamespace)
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
	clusters := &fedv1a1.FederatedClusterList{}
	err := cc.client.List(context.TODO(), clusters, cc.fedNamespace)
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

		if utilfeature.DefaultFeatureGate.Enabled(features.CrossClusterServiceDiscovery) {
			zone, region, err := clusterClient.GetClusterZones()
			if err != nil {
				glog.Warningf("Failed to get zones and region for cluster %s: %v", cluster.Name, err)
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
		}

		cc.mu.Lock()
		cc.clusterStatusMap[cluster.Name] = *clusterStatusNew
		cc.mu.Unlock()
		cluster.Status = *clusterStatusNew
		err = cc.client.UpdateStatus(context.TODO(), &cluster)
		if err != nil {
			glog.Warningf("Failed to update the status of cluster: %v, error is : %v", cluster.Name, err)
			// Don't return err here, as we want to continue processing remaining clusters.
			continue
		}
	}
	return nil
}
