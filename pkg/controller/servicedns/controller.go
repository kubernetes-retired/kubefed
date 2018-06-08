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
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	kubeclientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/client-go/util/workqueue"
	crclientset "k8s.io/cluster-registry/pkg/client/clientset_generated/clientset"

	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	dnsv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/multiclusterdns/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset_generated/clientset"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
)

const (
	allClustersKey = "ALL_CLUSTERS"
)

// Controller manages the MultiClusterServiceDNSRecord objects in federation.
type Controller struct {
	fedClient fedclientset.Interface

	// For triggering reconciliation of a single resource. This is
	// used when there is an add/update/delete operation on a resource
	// in federated API server.
	deliverer *util.DelayingDeliverer

	// For triggering reconciliation of all target resources. This is
	// used when a new cluster becomes available.
	clusterDeliverer *util.DelayingDeliverer

	// informer for service object from members of federation.
	serviceInformer util.FederatedInformer

	// informer for endpoint object from members of federation.
	endpointInformer util.FederatedInformer

	// Store for the MultiClusterServiceDNSRecord objects
	serviceDNSStore cache.Store
	// Informer for the MultiClusterServiceDNSRecord objects
	serviceDNSController cache.Controller

	// Work queue allowing parallel processing of resources
	workQueue workqueue.Interface

	// Backoff manager
	backoff *flowcontrol.Backoff

	reviewDelay             time.Duration
	clusterAvailableDelay   time.Duration
	clusterUnavailableDelay time.Duration
	smallDelay              time.Duration
}

// StartController starts the Controller for managing MultiClusterServiceDNSRecord objects.
func StartController(fedConfig, kubeConfig, crConfig *restclient.Config, stopChan <-chan struct{}, minimizeLatency bool) error {
	userAgent := "MultiClusterServiceDNS"
	restclient.AddUserAgent(fedConfig, userAgent)
	fedClient := fedclientset.NewForConfigOrDie(fedConfig)
	restclient.AddUserAgent(kubeConfig, userAgent)
	kubeClient := kubeclientset.NewForConfigOrDie(kubeConfig)
	restclient.AddUserAgent(crConfig, userAgent)
	crClient := crclientset.NewForConfigOrDie(crConfig)

	controller, err := newController(fedClient, kubeClient, crClient)
	if err != nil {
		return err
	}
	if minimizeLatency {
		controller.minimizeLatency()
	}
	glog.Infof("Starting MultiClusterServiceDNS controller")
	controller.Run(stopChan)
	return nil
}

// newController returns a new controller to manage MultiClusterServiceDNSRecord objects.
func newController(fedClient fedclientset.Interface, kubeClient kubeclientset.Interface, crClient crclientset.Interface) (*Controller, error) {
	s := &Controller{
		fedClient:               fedClient,
		reviewDelay:             time.Second * 10,
		clusterAvailableDelay:   time.Second * 20,
		clusterUnavailableDelay: time.Second * 60,
		smallDelay:              time.Second * 3,
		workQueue:               workqueue.New(),
		backoff:                 flowcontrol.NewBackOff(5*time.Second, time.Minute),
	}

	// Build deliverers for triggering reconciliations.
	s.deliverer = util.NewDelayingDeliverer()
	s.clusterDeliverer = util.NewDelayingDeliverer()

	// Informer for the MultiClusterServiceDNSRecord resource in federation.
	s.serviceDNSStore, s.serviceDNSController = cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (pkgruntime.Object, error) {
				return fedClient.MulticlusterdnsV1alpha1().MultiClusterServiceDNSRecords(metav1.NamespaceAll).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return fedClient.MulticlusterdnsV1alpha1().MultiClusterServiceDNSRecords(metav1.NamespaceAll).Watch(options)
			},
		},
		&dnsv1a1.MultiClusterServiceDNSRecord{},
		util.NoResyncPeriod,
		util.NewTriggerOnAllChanges(func(obj pkgruntime.Object) {
			s.deliverObj(obj, 0, false)
		}),
	)

	// Federated serviceInformer for the service resource in members of federation.
	s.serviceInformer = util.NewFederatedInformer(
		fedClient,
		kubeClient,
		crClient,
		&metav1.APIResource{
			Group:        "",
			Version:      "v1",
			Kind:         "Service",
			Name:         "services",
			SingularName: "service",
			Namespaced:   true},
		func(obj pkgruntime.Object) {
			s.deliverObj(obj, 0, false)
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

	// Federated informers on endpoints in federated clusters.
	// This will enable to check if service ingress endpoints in federated clusters are reachable
	s.endpointInformer = util.NewFederatedInformer(
		fedClient,
		kubeClient,
		crClient,
		&metav1.APIResource{
			Group:        "",
			Version:      "v1",
			Kind:         "Endpoints",
			Name:         "endpoints",
			SingularName: "endpoint",
			Namespaced:   true},
		func(obj pkgruntime.Object) {
			s.deliverObj(obj, 0, false)
		},
		&util.ClusterLifecycleHandlerFuncs{},
	)

	return s, nil
}

// minimizeLatency reduces delays and timeouts to make the controller more responsive (useful for testing).
func (c *Controller) minimizeLatency() {
	c.clusterAvailableDelay = time.Second
	c.clusterUnavailableDelay = time.Second
	c.reviewDelay = 50 * time.Millisecond
	c.smallDelay = 20 * time.Millisecond
}

// Run runs the Controller.
func (c *Controller) Run(stopChan <-chan struct{}) {
	go c.serviceDNSController.Run(stopChan)
	c.serviceInformer.Start()
	c.endpointInformer.Start()
	c.deliverer.StartWithHandler(func(item *util.DelayingDelivererItem) {
		c.workQueue.Add(item)
	})
	c.clusterDeliverer.StartWithHandler(func(_ *util.DelayingDelivererItem) {
		c.reconcileOnClusterChange()
	})

	// TODO: Allow multiple workers.
	go wait.Until(c.worker, time.Second, stopChan)

	util.StartBackoffGC(c.backoff, stopChan)

	// Ensure all goroutines are cleaned up when the stop channel closes
	go func() {
		<-stopChan
		c.serviceInformer.Stop()
		c.endpointInformer.Stop()
		c.workQueue.ShutDown()
		c.deliverer.Stop()
		c.clusterDeliverer.Stop()
	}()
}

type reconciliationStatus int

const (
	statusAllOK reconciliationStatus = iota
	statusNeedsRecheck
	statusError
	statusNotSynced
)

func (c *Controller) worker() {
	for {
		obj, quit := c.workQueue.Get()
		if quit {
			return
		}

		item := obj.(*util.DelayingDelivererItem)
		qualifiedName := item.Value.(*util.QualifiedName)
		status := c.reconcile(*qualifiedName)
		c.workQueue.Done(item)

		switch status {
		case statusAllOK:
			break
		case statusError:
			c.deliver(*qualifiedName, 0, true)
		case statusNeedsRecheck:
			c.deliver(*qualifiedName, c.reviewDelay, false)
		case statusNotSynced:
			c.deliver(*qualifiedName, c.clusterAvailableDelay, false)
		}
	}
}

func (c *Controller) deliverObj(obj pkgruntime.Object, delay time.Duration, failed bool) {
	qualifiedName := util.NewQualifiedName(obj)
	c.deliver(qualifiedName, delay, failed)
}

// Adds backoff to delay if this delivery is related to some failure. Resets backoff if there was no failure.
func (c *Controller) deliver(qualifiedName util.QualifiedName, delay time.Duration, failed bool) {
	key := qualifiedName.String()
	if failed {
		c.backoff.Next(key, time.Now())
		delay = delay + c.backoff.Get(key)
	} else {
		c.backoff.Reset(key)
	}
	c.deliverer.DeliverAfter(key, &qualifiedName, delay)
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
		c.deliver(qualifiedName, c.smallDelay, false)
	}
}

func (c *Controller) reconcile(qualifiedName util.QualifiedName) reconciliationStatus {
	if !c.isSynced() {
		return statusNotSynced
	}

	key := qualifiedName.String()

	glog.V(4).Infof("Starting to reconcile MultiClusterServiceDNS resource: %v", key)
	startTime := time.Now()
	defer glog.V(4).Infof("Finished reconciling MultiClusterServiceDNS resource %v (duration: %v)", key, time.Now().Sub(startTime))

	cachedObj, exist, err := c.serviceDNSStore.GetByKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to query MultiClusterServiceDNS store for %q: %v", key, err))
		return statusError
	}
	if !exist {
		return statusAllOK
	}
	cachedDNS := cachedObj.(*dnsv1a1.MultiClusterServiceDNSRecord)

	fedDNS := &dnsv1a1.MultiClusterServiceDNSRecord{
		ObjectMeta: util.DeepCopyRelevantObjectMeta(cachedDNS.ObjectMeta),
		Spec:       *cachedDNS.Spec.DeepCopy(),
	}

	clusters, err := c.serviceInformer.GetReadyClusters()
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to get ready cluster list: %v", err))
		return statusError
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
		// and traffic is not routable to the service.
		// We avoid such service shards while writing DNS records.
		endpointsExist, err := c.serviceBackedByEndpointsInCluster(cluster.Name, key)
		if err != nil {
			return statusError
		}
		if endpointsExist {
			lbStatus, err := c.getServiceStatusInCluster(cluster.Name, key)
			if err != nil {
				return statusError
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
		return statusError
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

	if !reflect.DeepEqual(cachedDNS.Status, fedDNS.Status) {
		_, err = c.fedClient.MulticlusterdnsV1alpha1().MultiClusterServiceDNSRecords(fedDNS.Namespace).UpdateStatus(fedDNS)
		if err != nil {
			runtime.HandleError(fmt.Errorf("Error updating the MultiClusterServiceDNS object %s: %v", key, err))
			return statusError
		}
	}

	return statusAllOK
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
