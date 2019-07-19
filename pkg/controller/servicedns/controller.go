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

package servicedns

import (
	"context"
	"encoding/json"
	"reflect"
	"sort"
	"time"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
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

// Controller manages ServiceDNSRecord resources in the host cluster.
type Controller struct {
	client genericclient.Client

	// For triggering reconciliation of all target resources. This is
	// used when a new cluster becomes available.
	clusterDeliverer *util.DelayingDeliverer

	// informer for service resources in member clusters
	serviceInformer util.FederatedInformer

	// informer for endpoint resources in member clusters
	endpointInformer util.FederatedInformer

	// Store for the ServiceDNSRecord objects
	serviceDNSStore cache.Store
	// Informer for the ServiceDNSRecord objects
	serviceDNSController cache.Controller

	// Store for the Domain objects
	domainStore cache.Store
	// Informer for the Domain objects
	domainController cache.Controller

	worker util.ReconcileWorker

	clusterAvailableDelay   time.Duration
	clusterUnavailableDelay time.Duration
	smallDelay              time.Duration

	fedNamespace string
}

// StartController starts the Controller for managing ServiceDNSRecord objects.
func StartController(config *util.ControllerConfig, stopChan <-chan struct{}) error {
	controller, err := newController(config)
	if err != nil {
		return err
	}
	if config.MinimizeLatency {
		controller.minimizeLatency()
	}
	klog.Infof("Starting ServiceDNS controller")
	controller.Run(stopChan)
	return nil
}

// newController returns a new controller to manage ServiceDNSRecord objects.
func newController(config *util.ControllerConfig) (*Controller, error) {
	client := genericclient.NewForConfigOrDieWithUserAgent(config.KubeConfig, "ServiceDNS")
	s := &Controller{
		client:                  client,
		clusterAvailableDelay:   config.ClusterAvailableDelay,
		clusterUnavailableDelay: config.ClusterUnavailableDelay,
		smallDelay:              time.Second * 3,
		fedNamespace:            config.KubeFedNamespace,
	}

	s.worker = util.NewReconcileWorker(s.reconcile, util.WorkerTiming{
		ClusterSyncDelay: s.clusterAvailableDelay,
	})

	// Build deliverer for triggering cluster reconciliations.
	s.clusterDeliverer = util.NewDelayingDeliverer()

	// Informer for ServiceDNSRecord resources in the host cluster
	var err error
	s.serviceDNSStore, s.serviceDNSController, err = util.NewGenericInformer(
		config.KubeConfig,
		config.TargetNamespace,
		&dnsv1a1.ServiceDNSRecord{},
		util.NoResyncPeriod,
		s.worker.EnqueueObject,
	)
	if err != nil {
		return nil, err
	}

	// Informer for the Domain resource
	s.domainStore, s.domainController, err = util.NewGenericInformer(
		config.KubeConfig,
		config.KubeFedNamespace,
		&dnsv1a1.Domain{},
		util.NoResyncPeriod,
		func(pkgruntime.Object) {
			s.clusterDeliverer.DeliverAt(allClustersKey, nil, time.Now())
		},
	)
	if err != nil {
		return nil, err
	}

	// Federated informer for service resources in member clusters
	s.serviceInformer, err = util.NewFederatedInformer(
		config,
		client,
		&metav1.APIResource{
			Group:        "",
			Version:      "v1",
			Kind:         "Service",
			Name:         "services",
			SingularName: "service",
			Namespaced:   true},
		s.worker.EnqueueObject,
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

	// Federated informers on endpoints in federated clusters.
	// This will enable to check if service ingress endpoints in federated clusters are reachable
	s.endpointInformer, err = util.NewFederatedInformer(
		config,
		client,
		&metav1.APIResource{
			Group:        "",
			Version:      "v1",
			Kind:         "Endpoints",
			Name:         "endpoints",
			SingularName: "endpoint",
			Namespaced:   true},
		s.worker.EnqueueObject,
		&util.ClusterLifecycleHandlerFuncs{},
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
	go c.serviceDNSController.Run(stopChan)
	go c.domainController.Run(stopChan)
	c.serviceInformer.Start()
	c.endpointInformer.Start()
	c.clusterDeliverer.StartWithHandler(func(_ *util.DelayingDelivererItem) {
		c.reconcileOnClusterChange()
	})

	c.worker.Run(stopChan)

	// Ensure all goroutines are cleaned up when the stop channel closes
	go func() {
		<-stopChan
		c.serviceInformer.Stop()
		c.endpointInformer.Stop()
		c.clusterDeliverer.Stop()
	}()
}

// Check whether all data stores are in sync. False is returned if any of the serviceInformer/stores is not yet
// synced with the corresponding api server.
func (c *Controller) isSynced() bool {
	if !c.serviceInformer.ClustersSynced() {
		klog.V(2).Infof("Cluster list not synced")
		return false
	}
	clusters, err := c.serviceInformer.GetReadyClusters()
	if err != nil {
		runtime.HandleError(errors.Wrap(err, "Failed to get ready clusters"))
		return false
	}
	if !c.serviceInformer.GetTargetStore().ClustersSynced(clusters) {
		return false
	}

	if !c.endpointInformer.ClustersSynced() {
		klog.V(2).Infof("Cluster list not synced")
		return false
	}
	clusters, err = c.endpointInformer.GetReadyClusters()
	if err != nil {
		runtime.HandleError(errors.Wrap(err, "Failed to get ready clusters"))
		return false
	}
	if !c.endpointInformer.GetTargetStore().ClustersSynced(clusters) {
		return false
	}

	return true
}

// The function triggers reconciliation of all target federated resources.
func (c *Controller) reconcileOnClusterChange() {
	if !c.isSynced() {
		c.clusterDeliverer.DeliverAt(allClustersKey, nil, time.Now().Add(c.clusterAvailableDelay))
	}
	for _, obj := range c.serviceDNSStore.List() {
		qualifiedName := util.NewQualifiedName(obj.(pkgruntime.Object))
		c.worker.EnqueueWithDelay(qualifiedName, c.smallDelay)
	}
}

func (c *Controller) reconcile(qualifiedName util.QualifiedName) util.ReconciliationStatus {
	if !c.isSynced() {
		return util.StatusNotSynced
	}

	key := qualifiedName.String()

	klog.V(4).Infof("Starting to reconcile ServiceDNS resource: %v", key)
	startTime := time.Now()
	defer klog.V(4).Infof("Finished reconciling ServiceDNS resource %v (duration: %v)", key, time.Since(startTime))

	cachedObj, exist, err := c.serviceDNSStore.GetByKey(key)
	if err != nil {
		runtime.HandleError(errors.Wrapf(err, "Failed to query ServiceDNS store for %q", key))
		return util.StatusError
	}
	if !exist {
		return util.StatusAllOK
	}
	cachedDNS := cachedObj.(*dnsv1a1.ServiceDNSRecord)

	domainKey := util.QualifiedName{Namespace: c.fedNamespace, Name: cachedDNS.Spec.DomainRef}.String()
	cachedDomain, exist, err := c.domainStore.GetByKey(domainKey)
	if err != nil {
		runtime.HandleError(errors.Wrapf(err, "Failed to query Domain store for %q", domainKey))
		return util.StatusError
	}
	if !exist {
		return util.StatusAllOK
	}
	domainObj := cachedDomain.(*dnsv1a1.Domain)

	fedDNS := &dnsv1a1.ServiceDNSRecord{
		ObjectMeta: util.DeepCopyRelevantObjectMeta(cachedDNS.ObjectMeta),
		Spec:       *cachedDNS.Spec.DeepCopy(),
	}

	clusters, err := c.serviceInformer.GetReadyClusters()
	if err != nil {
		runtime.HandleError(errors.Wrap(err, "Failed to get ready cluster list"))
		return util.StatusError
	}

	var fedDNSStatus []dnsv1a1.ClusterDNS
	// Iterate through all ready clusters and aggregate the service status for the key
	for _, cluster := range clusters {
		if cluster.Status.Region == nil || *cluster.Status.Region == "" || len(cluster.Status.Zones) == 0 {
			runtime.HandleError(errors.Wrapf(err, "Cluster %q does not have Region or Zones Attributes", cluster.Name))
			return util.StatusError
		}
		clusterDNS := dnsv1a1.ClusterDNS{
			Cluster: cluster.Name,
			Region:  *cluster.Status.Region,
			Zones:   cluster.Status.Zones,
		}

		// If there are no endpoints for the service, the service is not backed by pods
		// and traffic is not routable to the service. We avoid such service shards while
		// writing DNS records, except when user specified to AllowServiceWithoutEndpoints
		endpointsExist, err := c.serviceBackedByEndpointsInCluster(cluster.Name, key)
		if err != nil {
			return util.StatusError
		}
		if cachedDNS.Spec.AllowServiceWithoutEndpoints || endpointsExist {
			lbStatus, err := c.getServiceStatusInCluster(cluster.Name, key)
			if err != nil {
				return util.StatusError
			}
			clusterDNS.LoadBalancer = *lbStatus
		}
		fedDNSStatus = append(fedDNSStatus, clusterDNS)
	}

	// We should preserve ClusterDNS entries (without loadbalancers) for offline clusters
	// so that any user within the offline cluster targeting the federated service can
	// be redirected (via CNAME DNS record) to any of the service shards of the federated
	// service in online clusters.
	offlineClusters, err := c.serviceInformer.GetUnreadyClusters()
	if err != nil {
		runtime.HandleError(errors.Wrap(err, "Failed to get unready cluster list"))
		return util.StatusError
	}
	for _, cluster := range offlineClusters {
		for _, clusterDNS := range fedDNS.Status.DNS {
			if clusterDNS.Cluster == cluster.Name {
				offlineClusterDNS := clusterDNS
				offlineClusterDNS.LoadBalancer = corev1.LoadBalancerStatus{}
				fedDNSStatus = append(fedDNSStatus, offlineClusterDNS)
				klog.V(5).Infof("Cluster %s is Offline, Preserving previously available status for Service %s", cluster.Name, key)
				break
			}
		}
	}

	sort.Slice(fedDNSStatus, func(i, j int) bool {
		return fedDNSStatus[i].Cluster < fedDNSStatus[j].Cluster
	})
	fedDNS.Status.DNS = fedDNSStatus
	fedDNS.Status.Domain = domainObj.Domain

	if !reflect.DeepEqual(cachedDNS.Status, fedDNS.Status) {
		err = c.client.UpdateStatus(context.TODO(), fedDNS)
		if err != nil {
			runtime.HandleError(errors.Wrapf(err, "Error updating the ServiceDNS object %s", key))
			return util.StatusError
		}
	}

	return util.StatusAllOK
}

// getServiceStatusInCluster returns service status in federated cluster
func (c *Controller) getServiceStatusInCluster(cluster, key string) (*corev1.LoadBalancerStatus, error) {
	lbStatus := &corev1.LoadBalancerStatus{}

	clusterServiceObj, serviceFound, err := c.serviceInformer.GetTargetStore().GetByKey(cluster, key)
	if err != nil {
		runtime.HandleError(errors.Wrapf(err, "Failed to get %s service from %s", key, cluster))
		return lbStatus, err
	}
	if serviceFound {
		//TODO(shashi): Find better alternative to convert Unstructured to a given type
		clusterService, ok := clusterServiceObj.(*unstructured.Unstructured)
		if !ok {
			runtime.HandleError(errors.Errorf("Failed to cast the object to unstructured object: %v", clusterServiceObj))
			return lbStatus, err
		}
		content, err := clusterService.MarshalJSON()
		if err != nil {
			runtime.HandleError(errors.Errorf("Failed to marshall the unstructured object: %v", clusterService))
			return lbStatus, err
		}
		service := corev1.Service{}
		err = json.Unmarshal(content, &service)
		if err == nil {
			// Sort the ingress slice, so that we return comparable service status.
			ingress := service.Status.LoadBalancer.Ingress
			sort.Slice(ingress, func(i, j int) bool {
				if ingress[i].IP == ingress[j].IP {
					return ingress[i].Hostname < ingress[j].Hostname
				}
				return ingress[i].IP < ingress[j].IP
			})
			lbStatus = &service.Status.LoadBalancer
		}
	}
	return lbStatus, nil
}

// serviceBackedByEndpointsInCluster returns ready endpoints corresponding to service in federated cluster
func (c *Controller) serviceBackedByEndpointsInCluster(cluster, key string) (bool, error) {
	addresses := []corev1.EndpointAddress{}

	clusterEndpointObj, endpointFound, err := c.endpointInformer.GetTargetStore().GetByKey(cluster, key)
	if err != nil {
		runtime.HandleError(errors.Wrapf(err, "Failed to get %s endpoint from %s", key, cluster))
		return false, err
	}
	if endpointFound {
		//TODO(shashi): Find better alternative to convert Unstructured to a given type
		clusterEndpoints, ok := clusterEndpointObj.(*unstructured.Unstructured)
		if !ok {
			runtime.HandleError(errors.Errorf("Failed to cast the object to unstructured object: %v", clusterEndpointObj))
			return false, err
		}
		content, err := clusterEndpoints.MarshalJSON()
		if err != nil {
			runtime.HandleError(errors.Errorf("Failed to marshall the unstructured object: %v", clusterEndpoints))
			return false, err
		}
		endpoints := corev1.Endpoints{}
		err = json.Unmarshal(content, &endpoints)
		if err == nil {
			for _, subset := range endpoints.Subsets {
				if len(subset.Addresses) > 0 {
					addresses = append(addresses, subset.Addresses...)
				}
			}
		}
	}
	return (len(addresses) > 0), nil
}
