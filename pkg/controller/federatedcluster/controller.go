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
	"sync"
	"time"

	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	genericclient "github.com/kubernetes-sigs/federation-v2/pkg/client/generic"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/features"
)

// ClusterData stores cluster client and previous health check probe results of individual cluster.
type ClusterData struct {
	// clusterKubeClient is the kube client for the cluster.
	clusterKubeClient *ClusterClient

	// clusterStatus is the cluster status as of last sampling.
	clusterStatus *fedv1a1.FederatedClusterStatus

	// How many times in a row the probe has returned the same result.
	resultRun int
}

// ClusterController is responsible for maintaining the health status of each
// FederatedCluster in a particular namespace.
type ClusterController struct {
	client genericclient.Client

	// clusterHealthCheckConfig is the configurable parameters for cluster health check
	clusterHealthCheckConfig util.ClusterHealthCheckConfig

	mu sync.RWMutex

	// clusterDataMap is a mapping of clusterName and the cluster specific details.
	clusterDataMap map[string]*ClusterData

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
func StartClusterController(config *util.ControllerConfig, clusterHealthCheckConfig util.ClusterHealthCheckConfig, stopChan <-chan struct{}) error {
	controller, err := newClusterController(config, clusterHealthCheckConfig)
	if err != nil {
		return err
	}
	glog.Infof("Starting cluster controller")
	controller.Run(stopChan)
	return nil
}

// newClusterController returns a new cluster controller
func newClusterController(config *util.ControllerConfig, clusterHealthCheckConfig util.ClusterHealthCheckConfig) (*ClusterController, error) {
	kubeConfig := restclient.CopyConfig(config.KubeConfig)
	kubeConfig.Timeout = time.Duration(clusterHealthCheckConfig.TimeoutSeconds) * time.Second
	client := genericclient.NewForConfigOrDieWithUserAgent(kubeConfig, "cluster-controller")

	cc := &ClusterController{
		client:                   client,
		clusterHealthCheckConfig: clusterHealthCheckConfig,
		clusterDataMap:           make(map[string]*ClusterData),
		fedNamespace:             config.FederationNamespace,
		clusterNamespace:         config.ClusterNamespace,
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

// delFromClusterSet removes a cluster from the cluster data map
func (cc *ClusterController) delFromClusterSet(obj interface{}) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cluster := obj.(*fedv1a1.FederatedCluster)
	glog.V(1).Infof("ClusterController observed a cluster deletion: %v", cluster.Name)
	delete(cc.clusterDataMap, cluster.Name)
}

// addToClusterSet creates a new client for the cluster and stores it in cluster data map.
func (cc *ClusterController) addToClusterSet(obj interface{}) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cluster := obj.(*fedv1a1.FederatedCluster)
	clusterData := cc.clusterDataMap[cluster.Name]
	if clusterData != nil && clusterData.clusterKubeClient != nil {
		return
	}
	glog.V(1).Infof("ClusterController observed a new cluster: %v", cluster.Name)
	// create the restclient of cluster
	restClient, err := NewClusterClientSet(cluster, cc.client, cc.fedNamespace, cc.clusterNamespace)
	if err != nil || restClient == nil {
		glog.Errorf("Failed to create corresponding restclient of kubernetes cluster: %v", err)
		return
	}
	cc.clusterDataMap[cluster.Name] = &ClusterData{clusterKubeClient: restClient}
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
	}, time.Duration(cc.clusterHealthCheckConfig.PeriodSeconds)*time.Second, stopChan)
}

// updateClusterStatus checks cluster health and updates status of all FederatedClusters
func (cc *ClusterController) updateClusterStatus() error {
	clusters := &fedv1a1.FederatedClusterList{}
	err := cc.client.List(context.TODO(), clusters, cc.fedNamespace)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	for _, obj := range clusters.Items {
		cc.mu.RLock()
		cluster := obj.DeepCopy()
		clusterData := cc.clusterDataMap[cluster.Name]
		cc.mu.RUnlock()
		if clusterData == nil {
			glog.Warningf("Failed to retrieve stored data for cluster %s", cluster.Name)
			continue
		}

		wg.Add(1)
		go cc.updateIndividualClusterStatus(cluster, clusterData, &wg)
	}

	wg.Wait()
	return nil
}

func (cc *ClusterController) updateIndividualClusterStatus(cluster *fedv1a1.FederatedCluster,
	storedData *ClusterData, wg *sync.WaitGroup) {
	clusterClient := storedData.clusterKubeClient

	currentClusterStatus := clusterClient.GetClusterHealthStatus()
	currentClusterStatus = thresholdAdjustedClusterStatus(currentClusterStatus, storedData, cc.clusterHealthCheckConfig)

	if utilfeature.DefaultFeatureGate.Enabled(features.CrossClusterServiceDiscovery) {
		currentClusterStatus = updateClusterZonesAndRegion(currentClusterStatus, cluster, clusterClient)
	}

	storedData.clusterStatus = currentClusterStatus
	cluster.Status = *currentClusterStatus
	if err := cc.client.UpdateStatus(context.TODO(), cluster); err != nil {
		glog.Warningf("Failed to update the status of cluster %q: %v", cluster.Name, err)
	}
	wg.Done()
}

func thresholdAdjustedClusterStatus(clusterStatus *fedv1a1.FederatedClusterStatus, storedData *ClusterData,
	clusterHealthCheckConfig util.ClusterHealthCheckConfig) *fedv1a1.FederatedClusterStatus {

	if storedData.clusterStatus == nil {
		storedData.resultRun = 1
		return clusterStatus
	}

	threshold := clusterHealthCheckConfig.FailureThreshold
	if util.IsClusterReady(clusterStatus) {
		threshold = clusterHealthCheckConfig.SuccessThreshold
	}

	if storedData.resultRun < threshold {
		// Success/Failure is below threshold - leave the probe state unchanged.
		probeTime := clusterStatus.Conditions[0].LastProbeTime
		clusterStatus = storedData.clusterStatus
		setProbeTime(clusterStatus, probeTime)
	} else {
		if clusterStatusEqual(clusterStatus, storedData.clusterStatus) {
			// preserve the last transition time
			setTransitionTime(clusterStatus, storedData.clusterStatus.Conditions[0].LastTransitionTime)
		}
	}

	if clusterStatusEqual(clusterStatus, storedData.clusterStatus) {
		// Increment the result run has there is no change in cluster condition
		storedData.resultRun++
	} else {
		// Reset the result run
		storedData.resultRun = 1
	}

	return clusterStatus
}

func updateClusterZonesAndRegion(clusterStatus *fedv1a1.FederatedClusterStatus, cluster *fedv1a1.FederatedCluster,
	clusterClient *ClusterClient) *fedv1a1.FederatedClusterStatus {

	if !util.IsClusterReady(clusterStatus) {
		return clusterStatus
	}

	zones, region, err := clusterClient.GetClusterZones()
	if err != nil {
		glog.Warningf("Failed to get zones and region for cluster %q: %v", clusterClient.clusterName, err)
		return clusterStatus
	}

	// If new zone & region are empty, preserve the old ones so that user configured zone & region
	// labels are effective
	if len(zones) == 0 {
		zones = cluster.Status.Zones
	}
	if len(region) == 0 {
		region = cluster.Status.Region
	}
	clusterStatus.Zones = zones
	clusterStatus.Region = region
	return clusterStatus
}

func clusterStatusEqual(newClusterStatus, oldClusterStatus *fedv1a1.FederatedClusterStatus) bool {
	return util.IsClusterReady(newClusterStatus) == util.IsClusterReady(oldClusterStatus)
}

func setProbeTime(clusterStatus *fedv1a1.FederatedClusterStatus, probeTime metav1.Time) {
	for i := 0; i < len(clusterStatus.Conditions); i++ {
		clusterStatus.Conditions[i].LastProbeTime = probeTime
	}
}

func setTransitionTime(clusterStatus *fedv1a1.FederatedClusterStatus, transitionTime metav1.Time) {
	for i := 0; i < len(clusterStatus.Conditions); i++ {
		clusterStatus.Conditions[i].LastTransitionTime = transitionTime
	}
}
