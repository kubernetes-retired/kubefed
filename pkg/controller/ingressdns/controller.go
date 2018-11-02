/*
Copyright 2018 The Federation v2 Authors.

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

package ingressdns

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/golang/glog"

	corev1 "k8s.io/api/core/v1"
	extv1b1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	crclientset "k8s.io/cluster-registry/pkg/client/clientset/versioned"

	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	dnsv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/multiclusterdns/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
)

const (
	allClustersKey = "ALL_CLUSTERS"
)

// Controller manages the IngressDNSRecord objects in federation.
type Controller struct {
	fedClient fedclientset.Interface

	// For triggering reconciliation of all target resources. This is
	// used when a new cluster becomes available.
	clusterDeliverer *util.DelayingDeliverer

	// informer for ingress object from members of federation.
	ingressFederatedInformer util.FederatedInformer

	// Store for the IngressDNSRecord objects
	ingressDNSStore cache.Store
	// Informer for the IngressDNSRecord objects
	ingressDNSController cache.Controller

	worker util.ReconcileWorker

	clusterAvailableDelay   time.Duration
	clusterUnavailableDelay time.Duration
	smallDelay              time.Duration
}

// StartController starts the Controller for managing IngressDNSRecord objects.
func StartController(config *util.ControllerConfig, stopChan <-chan struct{}) error {
	fedClient, kubeClient, crClient := config.AllClients("IngressDNS")
	controller, err := newController(config, fedClient, kubeClient, crClient)
	if err != nil {
		return err
	}
	if config.MinimizeLatency {
		controller.minimizeLatency()
	}
	glog.Infof("Starting IngressDNS controller")
	controller.Run(stopChan)
	return nil
}

// newController returns a new controller to manage IngressDNSRecord objects.
func newController(config *util.ControllerConfig, fedClient fedclientset.Interface, kubeClient kubeclientset.Interface, crClient crclientset.Interface) (*Controller, error) {
	s := &Controller{
		fedClient:               fedClient,
		clusterAvailableDelay:   config.ClusterAvailableDelay,
		clusterUnavailableDelay: config.ClusterUnavailableDelay,
		smallDelay:              time.Second * 3,
	}

	s.worker = util.NewReconcileWorker(s.reconcile, util.WorkerTiming{
		ClusterSyncDelay: s.clusterAvailableDelay,
	})

	// Build deliverer for triggering cluster reconciliations.
	s.clusterDeliverer = util.NewDelayingDeliverer()

	// Informer for the IngressDNSRecord resource in federation.
	s.ingressDNSStore, s.ingressDNSController = cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (pkgruntime.Object, error) {
				return fedClient.MulticlusterdnsV1alpha1().IngressDNSRecords(config.TargetNamespace).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return fedClient.MulticlusterdnsV1alpha1().IngressDNSRecords(config.TargetNamespace).Watch(options)
			},
		},
		&dnsv1a1.IngressDNSRecord{},
		util.NoResyncPeriod,
		util.NewTriggerOnAllChanges(func(obj pkgruntime.Object) {
			s.worker.EnqueueObject(obj)
		}),
	)

	// Federated informer for the ingress resource in members of federation.
	s.ingressFederatedInformer = util.NewFederatedInformer(
		fedClient,
		kubeClient,
		crClient,
		config.FederationNamespaces,
		&metav1.APIResource{
			Group:        "extensions",
			Version:      "v1beta1",
			Kind:         "Ingress",
			Name:         "ingresses",
			SingularName: "ingress",
			Namespaced:   true},
		func(obj pkgruntime.Object) {
			s.worker.EnqueueObject(obj)
		},

		&util.ClusterLifecycleHandlerFuncs{
			ClusterAvailable: func(cluster *fedv1a1.FederatedCluster) {
				// When new cluster becomes available process all the target resources again.
				s.clusterDeliverer.DeliverAt(allClustersKey, nil, time.Now().Add(s.clusterAvailableDelay))
			},
			// When a cluster becomes unavailable process all the target resources again.
			ClusterUnavailable: func(cluster *fedv1a1.FederatedCluster, _ []interface{}) {
				s.clusterDeliverer.DeliverAt(allClustersKey, nil, time.Now().Add(s.clusterUnavailableDelay))
			},
		},
	)

	return s, nil
}

// minimizeLatency reduces delays and timeouts to make the controller more responsive (useful for testing).
func (c *Controller) minimizeLatency() {
	c.clusterAvailableDelay = time.Second
	c.clusterUnavailableDelay = time.Second
	c.smallDelay = 20 * time.Millisecond
	c.worker.SetDelay(50*time.Millisecond, c.clusterAvailableDelay)
}

// Run runs the Controller.
func (c *Controller) Run(stopChan <-chan struct{}) {
	go c.ingressDNSController.Run(stopChan)
	c.ingressFederatedInformer.Start()
	c.clusterDeliverer.StartWithHandler(func(_ *util.DelayingDelivererItem) {
		c.reconcileOnClusterChange()
	})

	c.worker.Run(stopChan)

	// Ensure all goroutines are cleaned up when the stop channel closes
	go func() {
		<-stopChan
		c.ingressFederatedInformer.Stop()
		c.clusterDeliverer.Stop()
	}()
}

// Check whether all data stores are in sync. False is returned if any of the ingressFederatedInformer/stores is not yet
// synced with the corresponding api server.
func (c *Controller) isSynced() bool {
	if !c.ingressFederatedInformer.ClustersSynced() {
		glog.V(2).Infof("Cluster list not synced")
		return false
	}
	clusters, err := c.ingressFederatedInformer.GetReadyClusters()
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to get ready clusters: %v", err))
		return false
	}
	if !c.ingressFederatedInformer.GetTargetStore().ClustersSynced(clusters) {
		return false
	}

	return true
}

// The function triggers reconciliation of all target federated resources.
func (c *Controller) reconcileOnClusterChange() {
	if !c.isSynced() {
		c.clusterDeliverer.DeliverAt(allClustersKey, nil, time.Now().Add(c.clusterAvailableDelay))
	}
	for _, obj := range c.ingressDNSStore.List() {
		qualifiedName := util.NewQualifiedName(obj.(pkgruntime.Object))
		c.worker.EnqueueWithDelay(qualifiedName, c.smallDelay)
	}
}

func (c *Controller) reconcile(qualifiedName util.QualifiedName) util.ReconciliationStatus {
	if !c.isSynced() {
		return util.StatusNotSynced
	}

	key := qualifiedName.String()

	glog.V(2).Infof("Starting to reconcile IngressDNS resource: %v", key)
	startTime := time.Now()
	defer glog.V(2).Infof("Finished reconciling IngressDNS resource %v (duration: %v)", key, time.Now().Sub(startTime))

	cachedIngressDNSObj, exist, err := c.ingressDNSStore.GetByKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to query IngressDNS store for %q: %v", key, err))
		return util.StatusError
	}
	if !exist {
		return util.StatusAllOK
	}
	cachedIngressDNS := cachedIngressDNSObj.(*dnsv1a1.IngressDNSRecord)

	newIngressDNS := &dnsv1a1.IngressDNSRecord{
		ObjectMeta: util.DeepCopyRelevantObjectMeta(cachedIngressDNS.ObjectMeta),
		Spec:       *cachedIngressDNS.Spec.DeepCopy(),
		Status:     dnsv1a1.IngressDNSRecordStatus{},
	}

	clusters, err := c.ingressFederatedInformer.GetReadyClusters()
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to get ready cluster list: %v", err))
		return util.StatusError
	}

	// Iterate through all ready clusters and aggregate the ingress status for the key
	for _, cluster := range clusters {
		clusterDNS := dnsv1a1.ClusterIngressDNS{
			Cluster: cluster.Name,
		}

		lbStatus, err := c.getIngressStatusInCluster(cluster.Name, key)
		if err != nil {
			return util.StatusError
		}
		clusterDNS.LoadBalancer = *lbStatus
		newIngressDNS.Status.DNS = append(newIngressDNS.Status.DNS, clusterDNS)
	}

	sort.Slice(newIngressDNS.Status.DNS, func(i, j int) bool {
		return newIngressDNS.Status.DNS[i].Cluster < newIngressDNS.Status.DNS[j].Cluster
	})

	if !reflect.DeepEqual(cachedIngressDNS.Status, newIngressDNS.Status) {
		_, err = c.fedClient.MulticlusterdnsV1alpha1().IngressDNSRecords(newIngressDNS.Namespace).UpdateStatus(newIngressDNS)
		if err != nil {
			runtime.HandleError(fmt.Errorf("Error updating the IngressDNS object %s: %v", key, err))
			return util.StatusError
		}
	}

	return util.StatusAllOK
}

// getIngressStatusInCluster returns ingress status in federated cluster
func (c *Controller) getIngressStatusInCluster(cluster, key string) (*corev1.LoadBalancerStatus, error) {
	lbStatus := &corev1.LoadBalancerStatus{}

	clusterIngressObj, ingressFound, err := c.ingressFederatedInformer.GetTargetStore().GetByKey(cluster, key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to get %s ingress from %s: %v", key, cluster, err))
		return lbStatus, err
	}
	if ingressFound {
		//TODO(shashi): Find better alternative to convert Unstructured to a given type
		clusterIngress, ok := clusterIngressObj.(*unstructured.Unstructured)
		if !ok {
			runtime.HandleError(fmt.Errorf("Failed to cast the object to unstructured object: %v", clusterIngressObj))
			return lbStatus, err
		}
		content, err := clusterIngress.MarshalJSON()
		if err != nil {
			runtime.HandleError(fmt.Errorf("Failed to marshall the unstructured object: %v", clusterIngress))
			return lbStatus, err
		}
		ingress := extv1b1.Ingress{}
		err = json.Unmarshal(content, &ingress)
		if err == nil {
			// Sort the lbIngress slice, so that we return comparable lbIngress status.
			lbIngress := ingress.Status.LoadBalancer.Ingress
			sort.Slice(lbIngress, func(i, j int) bool {
				if lbIngress[i].IP == lbIngress[j].IP {
					return lbIngress[i].Hostname < lbIngress[j].Hostname
				}
				return lbIngress[i].IP < lbIngress[j].IP
			})

			lbStatus.Ingress = lbIngress
		}
	}
	return lbStatus, nil
}
