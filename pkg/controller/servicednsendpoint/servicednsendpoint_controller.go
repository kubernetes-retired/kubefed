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

package servicednsendpoint

import (
	"fmt"
	"net"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/golang/glog"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	feddnsv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/multiclusterdns/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
)

const (
	// minDNSTTL is the minimum safe DNS TTL value to use (in seconds).  We use this as the TTL for all DNS records.
	minDNSTTL = 180

	minRetryDelay = 5 * time.Second
	maxRetryDelay = 300 * time.Second
	maxRetries    = 5
	numWorkers    = 2

	userAgent = "ServiceDNSEndpoint"

	// RecordTypeA is a RecordType enum value
	RecordTypeA = "A"
	// RecordTypeCNAME is a RecordType enum value
	RecordTypeCNAME = "CNAME"
)

// Abstracting away the internet for testing purposes
type NetWrapper interface {
	LookupHost(host string) (addrs []string, err error)
}

type NetWrapperDefaultImplementation struct{}

func (r *NetWrapperDefaultImplementation) LookupHost(host string) (addrs []string, err error) {
	return net.LookupHost(host)
}

var netWrapper NetWrapper

func init() {
	netWrapper = &NetWrapperDefaultImplementation{}
}

type ServiceDNSEndpointController struct {
	// Client to federation api server
	client fedclientset.Interface
	// Informer Store for ServiceDNS objects
	serviceDNSObjectStore cache.Store
	// Informer controller for ServiceDNS objects
	serviceDNSObjectController cache.Controller

	queue         workqueue.RateLimitingInterface
	minRetryDelay time.Duration
	maxRetryDelay time.Duration
}

func StartController(config *restclient.Config, stopChan <-chan struct{}, minimizeLatency bool) error {
	restclient.AddUserAgent(config, userAgent)
	fedClient := fedclientset.NewForConfigOrDie(config)

	controller, err := newServiceDNSEndpointController(fedClient, minimizeLatency)
	if err != nil {
		return err
	}
	// TODO: consider making numWorkers configurable
	go controller.Run(stopChan, numWorkers)
	return nil
}

func newServiceDNSEndpointController(client fedclientset.Interface, minimizeLatency bool) (*ServiceDNSEndpointController, error) {
	d := &ServiceDNSEndpointController{
		client: client,
	}

	// Start informer in federated API servers on DNS objects
	// TODO: Change this to shared informer
	d.serviceDNSObjectStore, d.serviceDNSObjectController = cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (pkgruntime.Object, error) {
				return client.MulticlusterdnsV1alpha1().MultiClusterServiceDNSRecords(metav1.NamespaceAll).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return client.MulticlusterdnsV1alpha1().MultiClusterServiceDNSRecords(metav1.NamespaceAll).Watch(options)
			},
		},
		&feddnsv1a1.MultiClusterServiceDNSRecord{},
		util.NoResyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: d.enqueueObject,
			UpdateFunc: func(old, cur interface{}) {
				oldObj, ok1 := old.(*feddnsv1a1.MultiClusterServiceDNSRecord)
				curObj, ok2 := cur.(*feddnsv1a1.MultiClusterServiceDNSRecord)
				if !ok1 || !ok2 {
					glog.Errorf("Received unknown objects: %v, %v", old, cur)
					return
				}
				if d.needsUpdate(oldObj, curObj) {
					d.enqueueObject(cur)
				}
			},
			DeleteFunc: d.enqueueObject,
		},
	)

	d.minRetryDelay = minRetryDelay
	d.maxRetryDelay = maxRetryDelay
	if minimizeLatency {
		d.minimizeLatency()
	}
	d.queue = workqueue.NewNamedRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(d.minRetryDelay, d.maxRetryDelay), userAgent)

	return d, nil
}

// minimizeLatency reduces delays and timeouts to make the controller more responsive (useful for testing).
func (d *ServiceDNSEndpointController) minimizeLatency() {
	d.minRetryDelay = 50 * time.Millisecond
	d.maxRetryDelay = 2 * time.Second
}

func (d *ServiceDNSEndpointController) Run(stopCh <-chan struct{}, workers int) {
	defer runtime.HandleCrash()
	defer d.queue.ShutDown()

	glog.Infof("Starting ServiceDNSEndpoint controller")
	defer glog.Infof("Shutting down ServiceDNSEndpoint controller")

	go d.serviceDNSObjectController.Run(stopCh)

	// wait for the caches to synchronize before starting the worker
	if !cache.WaitForCacheSync(stopCh, d.serviceDNSObjectController.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	glog.Infof("ServiceDNSEndpoint controller synced and ready")

	for i := 0; i < workers; i++ {
		go wait.Until(d.worker, time.Second, stopCh)
	}

	<-stopCh
}

func (d *ServiceDNSEndpointController) needsUpdate(oldObject, newObject *feddnsv1a1.MultiClusterServiceDNSRecord) bool {
	if !reflect.DeepEqual(oldObject.Spec, newObject.Spec) {
		return true
	}
	if !reflect.DeepEqual(oldObject.Status, newObject.Status) {
		return true
	}

	return false
}

// obj could be an *feddnsv1a1.MultiClusterServiceDNSRecord, or a DeletionFinalStateUnknown marker item.
func (d *ServiceDNSEndpointController) enqueueObject(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		glog.Errorf("Couldn't get key for object %#v: %v", obj, err)
		return
	}
	d.queue.Add(key)
}

func (d *ServiceDNSEndpointController) worker() {
	// processNextWorkItem will automatically wait until there's work available
	for d.processNextItem() {
		// continue looping
	}
}

func (d *ServiceDNSEndpointController) processNextItem() bool {
	key, quit := d.queue.Get()
	if quit {
		return false
	}
	defer d.queue.Done(key)

	err := d.processItem(key.(string))

	if err == nil {
		// No error, tell the queue to stop tracking history
		d.queue.Forget(key)
	} else if d.queue.NumRequeues(key) < maxRetries {
		glog.Errorf("Error processing %s (will retry): %v", key, err)
		// requeue the item to work on later
		d.queue.AddRateLimited(key)
	} else {
		// err != nil and too many retries
		glog.Errorf("Error processing %s (giving up): %v", key, err)
		d.queue.Forget(key)
		runtime.HandleError(err)
	}

	return true
}

func (d *ServiceDNSEndpointController) processItem(key string) error {
	startTime := time.Now()
	glog.V(4).Infof("Processing change to ServiceDNSEndpoint %s", key)
	defer func() {
		glog.V(4).Infof("Finished processing ServiceDNSEndpoint %q (%v)", key, time.Since(startTime))
	}()

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	obj, exists, err := d.serviceDNSObjectStore.GetByKey(key)
	if err != nil {
		return fmt.Errorf("error fetching object with key %s from store: %v", key, err)
	}

	if !exists {
		//delete corresponding DNSEndpoint object
		return d.client.MulticlusterdnsV1alpha1().DNSEndpoints(namespace).Delete(name, &metav1.DeleteOptions{})
	}

	dnsObject, ok := obj.(*feddnsv1a1.MultiClusterServiceDNSRecord)
	if !ok {
		return fmt.Errorf("recieved event for unknown object %v", obj)
	}

	dnsEndpoints, err := GetEndpointsForServiceDNSObject(dnsObject)
	if err != nil {
		return err
	}

	dnsEndpointObject, err := d.client.MulticlusterdnsV1alpha1().DNSEndpoints(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			newDNSEndpointObject := &feddnsv1a1.DNSEndpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: feddnsv1a1.DNSEndpointSpec{
					Endpoints: dnsEndpoints,
				},
			}

			_, err = d.client.MulticlusterdnsV1alpha1().DNSEndpoints(namespace).Create(newDNSEndpointObject)
		}
		return err
	}

	// Update only if the new endpoints are not equal to the existing ones.
	if !reflect.DeepEqual(dnsEndpointObject.Spec.Endpoints, dnsEndpoints) {
		dnsEndpointObject.Spec.Endpoints = dnsEndpoints
		_, err = d.client.MulticlusterdnsV1alpha1().DNSEndpoints(namespace).Update(dnsEndpointObject)
	}

	return err
}

// GetEndpointsForServiceDNSObject returns endpoint objects for each MultiClusterServiceDNSRecord object that should be processed.
func GetEndpointsForServiceDNSObject(dnsObject *feddnsv1a1.MultiClusterServiceDNSRecord) ([]*feddnsv1a1.Endpoint, error) {
	endpoints, err := getNamesForDNSObject(dnsObject)
	if err != nil {
		return nil, err
	}

	return DedupeAndMergeEndpoints(endpoints), nil
}

// Merge and remove duplicate endpoints
func DedupeAndMergeEndpoints(endpoints []*feddnsv1a1.Endpoint) (result []*feddnsv1a1.Endpoint) {
	// Sort endpoints by DNSName
	sort.Slice(endpoints, func(i, j int) bool {
		return endpoints[i].DNSName < endpoints[j].DNSName
	})

	// Remove the endpoint with no targets/ empty targets
	for i := 0; i < len(endpoints); {
		for j := 0; j < len(endpoints[i].Targets); {
			if endpoints[i].Targets[j] == "" {
				endpoints[i].Targets = append(endpoints[i].Targets[:j], endpoints[i].Targets[j+1:]...)
				continue
			}
			j++
		}
		if len(endpoints[i].Targets) == 0 {
			endpoints = append(endpoints[:i], endpoints[i+1:]...)
			continue
		}
		i++
	}

	// Merge endpoints with same DNSName
	for i := 1; i < len(endpoints); {
		if endpoints[i].DNSName == endpoints[i-1].DNSName {
			// Merge targets
			endpoints[i-1].Targets = append(endpoints[i-1].Targets, endpoints[i].Targets...)
			endpoints[i-1].Targets = sortAndRemoveDuplicateTargets(endpoints[i-1].Targets)

			// Remove the duplicate endpoint
			endpoints = append(endpoints[:i], endpoints[i+1:]...)
			continue
		}
		i++
	}

	return endpoints
}

func sortAndRemoveDuplicateTargets(targets []string) []string {
	sort.Slice(targets, func(i, j int) bool {
		return targets[i] < targets[j]
	})
	for i := 1; i < len(targets); {
		if targets[i] == targets[i-1] {
			// Remove duplicate target
			targets = append(targets[:i], targets[i+1:]...)
			continue
		}
		i++
	}
	return targets
}

func getNamesForDNSObject(dnsObject *feddnsv1a1.MultiClusterServiceDNSRecord) ([]*feddnsv1a1.Endpoint, error) {
	endpoints := []*feddnsv1a1.Endpoint{}

	commonPrefix := strings.Join([]string{dnsObject.Name, dnsObject.Namespace, dnsObject.Spec.FederationName, "svc"}, ".")
	for _, clusterDNS := range dnsObject.Status.DNS {
		zone := clusterDNS.Zone
		region := clusterDNS.Region

		dnsNames := []string{
			strings.Join([]string{commonPrefix, zone, region, dnsObject.Spec.DNSSuffix}, "."), // zone level
			strings.Join([]string{commonPrefix, region, dnsObject.Spec.DNSSuffix}, "."),       // region level, one up from zone level
			strings.Join([]string{commonPrefix, dnsObject.Spec.DNSSuffix}, "."),               // global level, one up from region level
			"", // nowhere to go up from global level
		}

		zoneTargets, regionTargets, globalTargets := getHealthyTargets(zone, region, dnsObject)
		targets := [][]string{zoneTargets, regionTargets, globalTargets}

		for i, target := range targets {
			endpoint, err := generateEndpoint(dnsNames[i], target, dnsNames[i+1])
			if err != nil {
				return nil, err
			}
			endpoints = append(endpoints, endpoint)
		}
	}

	return endpoints, nil
}

// getHealthyTargets returns the hostnames and/or IP addresses of healthy endpoints for the service, at a zone, region and global level (or an error)
func getHealthyTargets(zone, region string, dnsObject *feddnsv1a1.MultiClusterServiceDNSRecord) (zoneTargets, regionTargets, globalTargets feddnsv1a1.Targets) {
	// If federated dnsObject is deleted, return empty endpoints, so that DNS records are removed
	if dnsObject.DeletionTimestamp != nil {
		return zoneTargets, regionTargets, globalTargets
	}

	for _, clusterDNS := range dnsObject.Status.DNS {
		if clusterDNS.Zone == zone {
			zoneTargets = append(zoneTargets, ExtractLoadBalancerTargets(clusterDNS.LoadBalancer)...)
		}
	}

	for _, clusterDNS := range dnsObject.Status.DNS {
		if clusterDNS.Region == region {
			regionTargets = append(regionTargets, ExtractLoadBalancerTargets(clusterDNS.LoadBalancer)...)
		}
	}

	for _, clusterDNS := range dnsObject.Status.DNS {
		globalTargets = append(globalTargets, ExtractLoadBalancerTargets(clusterDNS.LoadBalancer)...)
	}

	return zoneTargets, regionTargets, globalTargets
}

func generateEndpoint(name string, targets feddnsv1a1.Targets, uplevelCname string) (ep *feddnsv1a1.Endpoint, err error) {
	ep = &feddnsv1a1.Endpoint{
		DNSName:   name,
		RecordTTL: minDNSTTL,
	}

	if len(targets) > 0 {
		targets, err = getResolvedTargets(targets, netWrapper)
		if err != nil {
			return nil, err
		}
		ep.Targets = targets
		ep.RecordType = RecordTypeA
	} else {
		ep.Targets = []string{uplevelCname}
		ep.RecordType = RecordTypeCNAME
	}

	return ep, nil
}

// getResolvedTargets performs DNS resolution on the provided slice of endpoints (which might be DNS names
// or IPv4 addresses) and returns a list of IPv4 addresses.  If any of the endpoints are neither valid IPv4
// addresses nor resolvable DNS names, non-nil error is also returned (possibly along with a partially
// complete list of resolved endpoints.
func getResolvedTargets(targets feddnsv1a1.Targets, netWrapper NetWrapper) (feddnsv1a1.Targets, error) {
	resolvedTargets := sets.String{}
	for _, target := range targets {
		if net.ParseIP(target) == nil {
			// It's not a valid IP address, so assume it's a DNS name, and try to resolve it,
			// replacing its DNS name with its IP addresses in expandedEndpoints
			// through an interface abstracting the internet
			ipAddrs, err := netWrapper.LookupHost(target)
			if err != nil {
				glog.Errorf("Failed to resolve %s, err: %v", target, err)
				return resolvedTargets.List(), err
			}
			for _, ip := range ipAddrs {
				resolvedTargets.Insert(ip)
			}
		} else {
			resolvedTargets.Insert(target)
		}
	}
	return resolvedTargets.List(), nil
}

func ExtractLoadBalancerTargets(lbStatus corev1.LoadBalancerStatus) feddnsv1a1.Targets {
	var targets feddnsv1a1.Targets

	for _, lb := range lbStatus.Ingress {
		if lb.IP != "" {
			targets = append(targets, lb.IP)
		}
		if lb.Hostname != "" {
			targets = append(targets, lb.Hostname)
		}
	}

	return targets
}
