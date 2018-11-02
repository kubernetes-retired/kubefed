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

package servicedns

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/golang/glog"

	corev1 "k8s.io/api/core/v1"
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

// Controller manages the ServiceDNSRecord objects in federation.
type Controller struct {
	fedClient fedclientset.Interface

	// For triggering reconciliation of all target resources. This is
	// used when a new cluster becomes available.
	clusterDeliverer *util.DelayingDeliverer

	// informer for service object from members of federation.
	serviceInformer util.FederatedInformer

	// informer for endpoint object from members of federation.
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
	fedClient, kubeClient, crClient := config.AllClients("ServiceDNS")
	controller, err := newController(config, fedClient, kubeClient, crClient)
	if err != nil {
		return err
	}
	if config.MinimizeLatency {
		controller.minimizeLatency()
	}
	glog.Infof("Starting ServiceDNS controller")
	controller.Run(stopChan)
	return nil
}

// newController returns a new controller to manage ServiceDNSRecord objects.
func newController(config *util.ControllerConfig, fedClient fedclientset.Interface, kubeClient kubeclientset.Interface, crClient crclientset.Interface) (*Controller, error) {
	s := &Controller{
		fedClient:               fedClient,
		clusterAvailableDelay:   config.ClusterAvailableDelay,
		clusterUnavailableDelay: config.ClusterUnavailableDelay,
		smallDelay:              time.Second * 3,
		fedNamespace:            config.FederationNamespace,
	}

	s.worker = util.NewReconcileWorker(s.reconcile, util.WorkerTiming{
		ClusterSyncDelay: s.clusterAvailableDelay,
	})

	// Build deliverer for triggering cluster reconciliations.
	s.clusterDeliverer = util.NewDelayingDeliverer()

	// Informer for the ServiceDNSRecord resource in federation.
	s.serviceDNSStore, s.serviceDNSController = cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (pkgruntime.Object, error) {
				return fedClient.MulticlusterdnsV1alpha1().ServiceDNSRecords(config.TargetNamespace).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return fedClient.MulticlusterdnsV1alpha1().ServiceDNSRecords(config.TargetNamespace).Watch(options)
			},
		},
		&dnsv1a1.ServiceDNSRecord{},
		util.NoResyncPeriod,
		util.NewTriggerOnAllChanges(s.worker.EnqueueObject),
	)

	// Informer for the Domain resource
	s.domainStore, s.domainController = cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (pkgruntime.Object, error) {
				return fedClient.MulticlusterdnsV1alpha1().Domains(s.fedNamespace).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return fedClient.MulticlusterdnsV1alpha1().Domains(s.fedNamespace).Watch(options)
			},
		},
		&dnsv1a1.Domain{},
		util.NoResyncPeriod,
		util.NewTriggerOnAllChanges(func(pkgruntime.Object) {
			s.clusterDeliverer.DeliverAt(allClustersKey, nil, time.Now())
		}),
	)

	// Federated serviceInformer for the service resource in members of federation.
	s.serviceInformer = util.NewFederatedInformer(
		fedClient,
		kubeClient,
		crClient,
		config.FederationNamespaces,
		&metav1.APIResource{
			Group:        "",
			Version:      "v1",
			Kind:         "Service",
			Name:         "services",
			SingularName: "service",
			Namespaced:   true},
		s.worker.EnqueueObject,
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

	// Federated informers on endpoints in federated clusters.
	// This will enable to check if service ingress endpoints in federated clusters are reachable
	s.endpointInformer = util.NewFederatedInformer(
		fedClient,
		kubeClient,
		crClient,
		config.FederationNamespaces,
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
		glog.V(2).Infof("Cluster list not synced")
		return false
	}
	clusters, err := c.serviceInformer.GetReadyClusters()
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to get ready clusters: %v", err))
		return false
	}
	if !c.serviceInformer.GetTargetStore().ClustersSynced(clusters) {
		return false
	}

	if !c.endpointInformer.ClustersSynced() {
		glog.V(2).Infof("Cluster list not synced")
		return false
	}
	clusters, err = c.endpointInformer.GetReadyClusters()
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to get ready clusters: %v", err))
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

	glog.V(4).Infof("Starting to reconcile ServiceDNS resource: %v", key)
	startTime := time.Now()
	defer glog.V(4).Infof("Finished reconciling ServiceDNS resource %v (duration: %v)", key, time.Now().Sub(startTime))

	cachedObj, exist, err := c.serviceDNSStore.GetByKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to query ServiceDNS store for %q: %v", key, err))
		return util.StatusError
	}
	if !exist {
		return util.StatusAllOK
	}
	cachedDNS := cachedObj.(*dnsv1a1.ServiceDNSRecord)

	domainKey := util.QualifiedName{Namespace: c.fedNamespace, Name: cachedDNS.Spec.DomainRef}.String()
	cachedDomain, exist, err := c.domainStore.GetByKey(domainKey)
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to query Domain store for %q: %v", domainKey, err))
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
		runtime.HandleError(fmt.Errorf("Failed to get ready cluster list: %v", err))
		return util.StatusError
	}

	var fedDNSStatus []dnsv1a1.ClusterDNS
	// Iterate through all ready clusters and aggregate the service status for the key
	for _, cluster := range clusters {
		clusterDNS := dnsv1a1.ClusterDNS{
			Cluster: cluster.Name,
			Region:  cluster.Status.Region,
			Zone:    cluster.Status.Zone,
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
		runtime.HandleError(fmt.Errorf("Failed to get unready cluster list: %v", err))
		return util.StatusError
	}
	for _, cluster := range offlineClusters {
		for _, clusterDNS := range fedDNS.Status.DNS {
			if clusterDNS.Cluster == cluster.Name {
				offlineClusterDNS := clusterDNS
				offlineClusterDNS.LoadBalancer = corev1.LoadBalancerStatus{}
				fedDNSStatus = append(fedDNSStatus, offlineClusterDNS)
				glog.V(5).Infof("Cluster %s is Offline, Preserving previously available status for Service %s", cluster.Name, key)
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
		_, err = c.fedClient.MulticlusterdnsV1alpha1().ServiceDNSRecords(fedDNS.Namespace).UpdateStatus(fedDNS)
		if err != nil {
			runtime.HandleError(fmt.Errorf("Error updating the ServiceDNS object %s: %v", key, err))
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
		runtime.HandleError(fmt.Errorf("Failed to get %s service from %s: %v", key, cluster, err))
		return lbStatus, err
	}
	if serviceFound {
		//TODO(shashi): Find better alternative to convert Unstructured to a given type
		clusterService, ok := clusterServiceObj.(*unstructured.Unstructured)
		if !ok {
			runtime.HandleError(fmt.Errorf("Failed to cast the object to unstructured object: %v", clusterServiceObj))
			return lbStatus, err
		}
		content, err := clusterService.MarshalJSON()
		if err != nil {
			runtime.HandleError(fmt.Errorf("Failed to marshall the unstructured object: %v", clusterService))
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
		runtime.HandleError(fmt.Errorf("Failed to get %s endpoint from %s: %v", key, cluster, err))
		return false, err
	}
	if endpointFound {
		//TODO(shashi): Find better alternative to convert Unstructured to a given type
		clusterEndpoints, ok := clusterEndpointObj.(*unstructured.Unstructured)
		if !ok {
			runtime.HandleError(fmt.Errorf("Failed to cast the object to unstructured object: %v", clusterEndpointObj))
			return false, err
		}
		content, err := clusterEndpoints.MarshalJSON()
		if err != nil {
			runtime.HandleError(fmt.Errorf("Failed to marshall the unstructured object: %v", clusterEndpoints))
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
