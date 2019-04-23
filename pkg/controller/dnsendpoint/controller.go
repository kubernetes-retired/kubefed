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

package dnsendpoint

import (
	"context"
	"reflect"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	feddnsv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/multiclusterdns/v1alpha1"
	genericclient "github.com/kubernetes-sigs/federation-v2/pkg/client/generic"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
)

const (
	minRetryDelay = 5 * time.Second
	maxRetryDelay = 300 * time.Second
	maxRetries    = 5

	// TODO: consider making numWorkers configurable
	numWorkers = 2
)

type GetEndpointsFunc func(interface{}) ([]*feddnsv1a1.Endpoint, error)

type controller struct {
	client genericclient.Client
	// Informer Store for DNS objects
	dnsObjectStore cache.Store
	// Informer controller for DNS objects
	dnsObjectController cache.Controller

	dnsObjectKind string
	getEndpoints  GetEndpointsFunc

	queue         workqueue.RateLimitingInterface
	minRetryDelay time.Duration
	maxRetryDelay time.Duration
}

func newDNSEndpointController(config *util.ControllerConfig, objectType pkgruntime.Object, objectKind string,
	getEndpoints GetEndpointsFunc, minimizeLatency bool) (*controller, error) {
	client, err := genericclient.New(config.KubeConfig)
	if err != nil {
		return nil, err
	}

	d := &controller{
		client:        client,
		dnsObjectKind: objectKind,
		getEndpoints:  getEndpoints,
		minRetryDelay: minRetryDelay,
		maxRetryDelay: maxRetryDelay,
	}

	// Start informer for DNS objects
	// TODO: Change this to shared informer
	d.dnsObjectStore, d.dnsObjectController, err = util.NewGenericInformer(
		config.KubeConfig,
		config.TargetNamespace,
		objectType,
		util.NoResyncPeriod,
		d.enqueueObject,
	)
	if err != nil {
		return nil, err
	}

	if minimizeLatency {
		d.minimizeLatency()
	}
	d.queue = workqueue.NewNamedRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(
		d.minRetryDelay, d.maxRetryDelay), objectKind+"DNSEndpoint")

	return d, nil
}

// minimizeLatency reduces delays and timeouts to make the controller more responsive (useful for testing).
func (d *controller) minimizeLatency() {
	d.minRetryDelay = 50 * time.Millisecond
	d.maxRetryDelay = 2 * time.Second
}

func (d *controller) Run(stopCh <-chan struct{}) {
	defer runtime.HandleCrash()
	defer d.queue.ShutDown()

	glog.Infof("Starting %q DNSEndpoint controller", d.dnsObjectKind)
	defer glog.Infof("Shutting down %q DNSEndpoint controller", d.dnsObjectKind)

	go d.dnsObjectController.Run(stopCh)

	// wait for the caches to synchronize before starting the worker
	if !cache.WaitForCacheSync(stopCh, d.dnsObjectController.HasSynced) {
		runtime.HandleError(errors.New("Timed out waiting for caches to sync"))
		return
	}

	glog.Infof("%q DNSEndpoint controller synced and ready", d.dnsObjectKind)

	for i := 0; i < numWorkers; i++ {
		go wait.Until(d.worker, time.Second, stopCh)
	}

	<-stopCh
}

func (d *controller) enqueueObject(obj pkgruntime.Object) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		glog.Errorf("Couldn't get key for object %#v: %v", obj, err)
		return
	}
	d.queue.Add(key)
}

func (d *controller) worker() {
	// processNextWorkItem will automatically wait until there's work available
	for d.processNextItem() {
		// continue looping
	}
}

func (d *controller) processNextItem() bool {
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

func (d *controller) processItem(key string) error {
	startTime := time.Now()
	glog.V(4).Infof("Processing change to %q DNSEndpoint %s", d.dnsObjectKind, key)
	defer func() {
		glog.V(4).Infof("Finished processing %q DNSEndpoint %q (%v)", d.dnsObjectKind, key, time.Since(startTime))
	}()

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	// Prefix the name of DNSEndpoint object with DNS Object kind
	name = d.dnsObjectKind + "-" + name

	obj, exists, err := d.dnsObjectStore.GetByKey(key)
	if err != nil {
		return errors.Wrapf(err, "error fetching object with key %s from store", key)
	}

	if !exists {
		//delete corresponding DNSEndpoint object
		dnsEndpointObject := &feddnsv1a1.DNSEndpoint{}
		err = d.client.Delete(context.TODO(), dnsEndpointObject, namespace, name)
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	dnsEndpoints, err := d.getEndpoints(obj)
	if err != nil {
		return err
	}

	dnsEndpointObject := &feddnsv1a1.DNSEndpoint{}
	err = d.client.Get(context.TODO(), dnsEndpointObject, namespace, name)
	if apierrors.IsNotFound(err) {
		newDNSEndpointObject := &feddnsv1a1.DNSEndpoint{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: feddnsv1a1.DNSEndpointSpec{
				Endpoints: dnsEndpoints,
			},
		}
		return d.client.Create(context.TODO(), newDNSEndpointObject)
	}
	if err != nil {
		return err
	}

	// Update only if the new endpoints are not equal to the existing ones.
	if !reflect.DeepEqual(dnsEndpointObject.Spec.Endpoints, dnsEndpoints) {
		dnsEndpointObject.Spec.Endpoints = dnsEndpoints
		return d.client.Update(context.TODO(), dnsEndpointObject)
	}

	return nil
}
