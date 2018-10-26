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

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

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

	// Store for the FederatedTypeConfig objects
	store cache.Store
	// Informer for the FederatedTypeConfig objects
	controller cache.Controller

	worker util.ReconcileWorker
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
	}

	c.worker = util.NewReconcileWorker(c.reconcile, util.WorkerTiming{})

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
		util.NewTriggerOnAllChanges(c.worker.EnqueueObject),
	)

	return c, nil
}

// Run runs the Controller.
func (c *Controller) Run(stopChan <-chan struct{}) {
	go c.controller.Run(stopChan)

	// wait for the caches to synchronize before starting the worker
	if !cache.WaitForCacheSync(stopChan, c.controller.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for cache to sync"))
		return
	}

	c.worker.Run(stopChan)

	// Ensure all goroutines are cleaned up when the stop channel closes
	go func() {
		<-stopChan
		c.shutDown()
	}()
}

func (c *Controller) reconcile(qualifiedName util.QualifiedName) util.ReconciliationStatus {
	key := qualifiedName.String()

	glog.Infof("Running reconcile FederatedTypeConfig for %q", key)

	cachedObj, exist, err := c.store.GetByKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to query FederatedTypeConfig store for %q: %v", key, err))
		return util.StatusError
	}
	if !exist {
		return util.StatusAllOK
	}
	typeConfig := cachedObj.(*corev1a1.FederatedTypeConfig)

	enabled := typeConfig.Spec.PropagationEnabled

	limitedScope := c.targetNamespace != metav1.NamespaceAll
	if limitedScope && enabled && !typeConfig.GetNamespaced() {
		glog.Infof("Skipping start of sync controller for cluster-scoped resource %q.  It is not required for a namespaced federation control plane.", typeConfig.GetTemplate().Kind)
		return util.StatusAllOK
	}

	stopChan, running := c.getStopChannel(typeConfig.Name)

	deleted := typeConfig.DeletionTimestamp != nil
	if deleted {
		if running {
			c.stopController(typeConfig.Name, stopChan)
		}
		err := c.removeFinalizer(typeConfig)
		if err != nil {
			runtime.HandleError(fmt.Errorf("Failed to remove finalizer from FederatedTypeConfig %q: %v", key, err))
			return util.StatusError
		}
		return util.StatusAllOK
	}

	err = c.ensureFinalizer(typeConfig)
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to ensure finalizer for FederatedTypeConfig %q: %v", key, err))
		return util.StatusError
	}

	startNewController := !running && enabled
	stopController := running && !enabled
	if startNewController {
		if err := c.startController(typeConfig); err != nil {
			runtime.HandleError(err)
			return util.StatusError
		}
	} else if stopController {
		c.stopController(typeConfig.Name, stopChan)
	}

	return util.StatusAllOK
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
