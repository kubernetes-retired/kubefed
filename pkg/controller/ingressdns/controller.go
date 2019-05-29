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

package ingressdns

import (
	"context"
	"encoding/json"
	"reflect"
	"sort"
	"time"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	extv1b1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"

	fedv1b1 "sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	dnsv1a1 "sigs.k8s.io/kubefed/pkg/apis/multiclusterdns/v1alpha1"
	genericclient "sigs.k8s.io/kubefed/pkg/client/generic"
	"sigs.k8s.io/kubefed/pkg/controller/util"
)

const (
	allClustersKey = "ALL_CLUSTERS"
)

// Controller manages the IngressDNSRecord objects in the host cluster.
type Controller struct {
	client genericclient.Client

	// For triggering reconciliation of all target resources. This is
	// used when a new cluster becomes available.
	clusterDeliverer *util.DelayingDeliverer

	// informer for ingress resources in member clusters
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
	controller, err := newController(config)
	if err != nil {
		return err
	}
	if config.MinimizeLatency {
		controller.minimizeLatency()
	}
	klog.Infof("Starting IngressDNS controller")
	controller.Run(stopChan)
	return nil
}

// newController returns a new controller to manage IngressDNSRecord objects.
func newController(config *util.ControllerConfig) (*Controller, error) {
	client := genericclient.NewForConfigOrDieWithUserAgent(config.KubeConfig, "IngressDNS")
	s := &Controller{
		client:                  client,
		clusterAvailableDelay:   config.ClusterAvailableDelay,
		clusterUnavailableDelay: config.ClusterUnavailableDelay,
		smallDelay:              time.Second * 3,
	}

	s.worker = util.NewReconcileWorker(s.reconcile, util.WorkerTiming{
		ClusterSyncDelay: s.clusterAvailableDelay,
	})

	// Build deliverer for triggering cluster reconciliations.
	s.clusterDeliverer = util.NewDelayingDeliverer()

	// Informer for the IngressDNSRecord resources in the host cluster
	var err error
	s.ingressDNSStore, s.ingressDNSController, err = util.NewGenericInformer(
		config.KubeConfig,
		config.TargetNamespace,
		&dnsv1a1.IngressDNSRecord{},
		util.NoResyncPeriod,
		s.worker.EnqueueObject,
	)
	if err != nil {
		return nil, err
	}

	// Federated informer for ingress resources in members clusters
	s.ingressFederatedInformer, err = util.NewFederatedInformer(
		config,
		client,
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
			ClusterAvailable: func(cluster *fedv1b1.KubeFedCluster) {
				// When new cluster becomes available process all the target resources again.
				s.clusterDeliverer.DeliverAt(allClustersKey, nil, time.Now().Add(s.clusterAvailableDelay))
			},
			// When a cluster becomes unavailable process all the target resources again.
			ClusterUnavailable: func(cluster *fedv1b1.KubeFedCluster, _ []interface{}) {
				s.clusterDeliverer.DeliverAt(allClustersKey, nil, time.Now().Add(s.clusterUnavailableDelay))
			},
		},
	)
	if err != nil {
		return nil, err
	}

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
		klog.V(2).Infof("Cluster list not synced")
		return false
	}
	clusters, err := c.ingressFederatedInformer.GetReadyClusters()
	if err != nil {
		runtime.HandleError(errors.Wrap(err, "Failed to get ready clusters"))
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

	klog.V(2).Infof("Starting to reconcile IngressDNS resource: %v", key)
	startTime := time.Now()
	defer klog.V(2).Infof("Finished reconciling IngressDNS resource %v (duration: %v)", key, time.Since(startTime))

	cachedIngressDNSObj, exist, err := c.ingressDNSStore.GetByKey(key)
	if err != nil {
		runtime.HandleError(errors.Wrapf(err, "Failed to query IngressDNS store for %q", key))
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
		runtime.HandleError(errors.Wrap(err, "Failed to get ready cluster list"))
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
		err = c.client.UpdateStatus(context.TODO(), newIngressDNS)
		if err != nil {
			runtime.HandleError(errors.Wrapf(err, "Error updating the IngressDNS object %s", key))
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
		runtime.HandleError(errors.Wrapf(err, "Failed to get %s ingress from %s", key, cluster))
		return lbStatus, err
	}
	if ingressFound {
		//TODO(shashi): Find better alternative to convert Unstructured to a given type
		clusterIngress, ok := clusterIngressObj.(*unstructured.Unstructured)
		if !ok {
			runtime.HandleError(errors.Errorf("Failed to cast the object to unstructured object: %v", clusterIngressObj))
			return lbStatus, err
		}
		content, err := clusterIngress.MarshalJSON()
		if err != nil {
			runtime.HandleError(errors.Wrapf(err, "Failed to marshall the unstructured object: %v", clusterIngress))
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
