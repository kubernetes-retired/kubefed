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

package federatedtypeconfig

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/client-go/util/workqueue"

	corev1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	corev1alpha1client "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned/typed/core/v1alpha1"
	synccontroller "github.com/kubernetes-sigs/federation-v2/pkg/controller/sync"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
)

const finalizer string = "core.federation.k8s.io/federated-type-config"

// Controller manages the FederatedTypeConfig objects in federation.
type Controller struct {
	client corev1alpha1client.CoreV1alpha1Interface

	// Reference to config used to start new controllers
	config *rest.Config

	// The name of the federation system namespace
	fedNamespace string

	// The cluster registry namespace
	clusterNamespace string

	// The namespace to target
	targetNamespace string

	// Map of running sync controllers keyed by qualified target type
	stopChannels map[string]chan struct{}
	lock         sync.RWMutex

	// For triggering reconciliation of a single resource. This is
	// used when there is an add/update/delete operation on a resource
	// in federated API server.
	deliverer *util.DelayingDeliverer

	// Store for the FederatedTypeConfig objects
	store cache.Store
	// Informer for the FederatedTypeConfig objects
	controller cache.Controller

	// Work queue allowing parallel processing of resources
	workQueue workqueue.Interface

	// Backoff manager
	backoff *flowcontrol.Backoff
}

// StartController starts the Controller for managing FederatedTypeConfig objects.
func StartController(config *restclient.Config, fedNamespace, clusterNamespace, targetNamespace string, stopChan <-chan struct{}) error {
	userAgent := "FederatedTypeConfig"
	restclient.AddUserAgent(config, userAgent)
	client := fedclientset.NewForConfigOrDie(config).CoreV1alpha1()

	controller, err := newController(client, config, fedNamespace, clusterNamespace, targetNamespace)
	if err != nil {
		return err
	}
	glog.Infof("Starting FederatedTypeConfig controller")
	controller.Run(stopChan)
	return nil
}

// newController returns a new controller to manage FederatedTypeConfig objects.
func newController(client corev1alpha1client.CoreV1alpha1Interface, config *restclient.Config, fedNamespace, clusterNamespace, targetNamespace string) (*Controller, error) {
	c := &Controller{
		client:           client,
		config:           config,
		fedNamespace:     fedNamespace,
		clusterNamespace: clusterNamespace,
		targetNamespace:  targetNamespace,
		stopChannels:     make(map[string]chan struct{}),
		workQueue:        workqueue.New(),
		backoff:          flowcontrol.NewBackOff(5*time.Second, time.Minute),
	}

	c.deliverer = util.NewDelayingDeliverer()

	c.store, c.controller = cache.NewInformer(
		&cache.ListWatch{
			// Only watch the federation namespace to ensure
			// restrictive authz can be applied to a namespaced
			// control plane.
			ListFunc: func(options metav1.ListOptions) (pkgruntime.Object, error) {
				return client.FederatedTypeConfigs(fedNamespace).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return client.FederatedTypeConfigs(fedNamespace).Watch(options)
			},
		},
		&corev1a1.FederatedTypeConfig{},
		util.NoResyncPeriod,
		util.NewTriggerOnAllChanges(func(obj pkgruntime.Object) {
			c.deliverObj(obj, 0, false)
		}),
	)

	return c, nil
}

// Run runs the Controller.
func (c *Controller) Run(stopChan <-chan struct{}) {
	go c.controller.Run(stopChan)
	c.deliverer.StartWithHandler(func(item *util.DelayingDelivererItem) {
		c.workQueue.Add(item)
	})

	// wait for the caches to synchronize before starting the worker
	if !cache.WaitForCacheSync(stopChan, c.controller.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for cache to sync"))
		return
	}

	// TODO: Allow multiple workers.
	go wait.Until(c.worker, time.Second, stopChan)

	util.StartBackoffGC(c.backoff, stopChan)

	// Ensure all goroutines are cleaned up when the stop channel closes
	go func() {
		<-stopChan
		c.workQueue.ShutDown()
		c.deliverer.Stop()
		c.shutDown()
	}()
}

type reconciliationStatus int

const (
	statusAllOK reconciliationStatus = iota
	statusError
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

func (c *Controller) reconcile(qualifiedName util.QualifiedName) reconciliationStatus {
	key := qualifiedName.String()

	glog.Infof("Running reconcile FederatedTypeConfig for %q", key)

	cachedObj, exist, err := c.store.GetByKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to query FederatedTypeConfig store for %q: %v", key, err))
		return statusError
	}
	if !exist {
		return statusAllOK
	}
	typeConfig := cachedObj.(*corev1a1.FederatedTypeConfig)

	stopChan, running := c.getStopChannel(typeConfig.Name)

	deleted := typeConfig.DeletionTimestamp != nil
	if deleted {
		if running {
			c.stopController(typeConfig.Name, stopChan)
		}
		err := c.removeFinalizer(typeConfig)
		if err != nil {
			runtime.HandleError(fmt.Errorf("Failed to remove finalizer from FederatedTypeConfig %q: %v", key, err))
			return statusError
		}
		return statusAllOK
	}

	err = c.ensureFinalizer(typeConfig)
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to ensure finalizer for FederatedTypeConfig %q: %v", key, err))
		return statusError
	}

	enabled := typeConfig.Spec.PropagationEnabled
	startNewController := !running && enabled
	stopController := running && !enabled
	if startNewController {
		if err := c.startController(typeConfig); err != nil {
			runtime.HandleError(err)
			return statusError
		}
	} else if stopController {
		c.stopController(typeConfig.Name, stopChan)
	}

	return statusAllOK
}

func (c *Controller) shutDown() {
	c.lock.Lock()
	defer c.lock.Unlock()

	// Stop all sync controllers
	for key, stopChannel := range c.stopChannels {
		close(stopChannel)
		delete(c.stopChannels, key)
	}
}

func (c *Controller) getStopChannel(name string) (chan struct{}, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	stopChan, ok := c.stopChannels[name]
	return stopChan, ok
}

func (c *Controller) startController(tc *corev1a1.FederatedTypeConfig) error {
	kind := tc.Spec.Template.Kind

	// TODO(marun) Perform this defaulting in a webhook
	corev1a1.SetFederatedTypeConfigDefaults(tc)

	stopChan := make(chan struct{})
	err := synccontroller.StartFederationSyncController(tc, c.config, c.fedNamespace, c.clusterNamespace, c.targetNamespace, stopChan, false)
	if err != nil {
		close(stopChan)
		return fmt.Errorf("Error starting sync controller for %q: %v", kind, err)
	}
	glog.Infof("Started sync controller for %q", kind)
	c.lock.Lock()
	defer c.lock.Unlock()
	c.stopChannels[tc.Name] = stopChan
	return nil
}

func (c *Controller) stopController(key string, stopChan chan struct{}) {
	glog.Infof("Stopping sync controller for %q", key)
	close(stopChan)
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.stopChannels, key)
}

func (c *Controller) ensureFinalizer(tc *corev1a1.FederatedTypeConfig) error {
	accessor, err := meta.Accessor(tc)
	if err != nil {
		return err
	}
	finalizers := sets.NewString(accessor.GetFinalizers()...)
	if finalizers.Has(finalizer) {
		return nil
	}
	finalizers.Insert(finalizer)
	accessor.SetFinalizers(finalizers.List())
	_, err = c.client.FederatedTypeConfigs(tc.Namespace).Update(tc)
	return err
}

func (c *Controller) removeFinalizer(tc *corev1a1.FederatedTypeConfig) error {
	accessor, err := meta.Accessor(tc)
	if err != nil {
		return err
	}
	finalizers := sets.NewString(accessor.GetFinalizers()...)
	if !finalizers.Has(finalizer) {
		return nil
	}
	finalizers.Delete(finalizer)
	accessor.SetFinalizers(finalizers.List())
	_, err = c.client.FederatedTypeConfigs(tc.Namespace).Update(tc)
	return err
}
