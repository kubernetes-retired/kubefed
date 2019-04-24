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
	"context"
	"sync"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	corev1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	genericclient "github.com/kubernetes-sigs/federation-v2/pkg/client/generic"
	statuscontroller "github.com/kubernetes-sigs/federation-v2/pkg/controller/status"
	synccontroller "github.com/kubernetes-sigs/federation-v2/pkg/controller/sync"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
)

const finalizer string = "core.federation.k8s.io/federated-type-config"

// Controller manages the FederatedTypeConfig objects in federation.
type Controller struct {
	// Arguments to use when starting new controllers
	controllerConfig *util.ControllerConfig

	client genericclient.Client

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
func StartController(config *util.ControllerConfig, stopChan <-chan struct{}) error {
	controller, err := newController(config)
	if err != nil {
		return err
	}
	glog.Infof("Starting FederatedTypeConfig controller")
	controller.Run(stopChan)
	return nil
}

// newController returns a new controller to manage FederatedTypeConfig objects.
func newController(config *util.ControllerConfig) (*Controller, error) {
	userAgent := "FederatedTypeConfig"
	kubeConfig := restclient.CopyConfig(config.KubeConfig)
	restclient.AddUserAgent(kubeConfig, userAgent)
	genericclient, err := genericclient.New(kubeConfig)
	if err != nil {
		return nil, err
	}

	c := &Controller{
		controllerConfig: config,
		client:           genericclient,
		stopChannels:     make(map[string]chan struct{}),
	}

	c.worker = util.NewReconcileWorker(c.reconcile, util.WorkerTiming{})

	// Only watch the federation namespace to ensure
	// restrictive authz can be applied to a namespaced
	// control plane.
	c.store, c.controller, err = util.NewGenericInformer(
		kubeConfig,
		config.FederationNamespace,
		&corev1a1.FederatedTypeConfig{},
		util.NoResyncPeriod,
		c.worker.EnqueueObject,
	)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// Run runs the Controller.
func (c *Controller) Run(stopChan <-chan struct{}) {
	go c.controller.Run(stopChan)

	// wait for the caches to synchronize before starting the worker
	if !cache.WaitForCacheSync(stopChan, c.controller.HasSynced) {
		runtime.HandleError(errors.New("Timed out waiting for cache to sync"))
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

	glog.V(3).Infof("Running reconcile FederatedTypeConfig for %q", key)

	cachedObj, err := c.objCopyFromCache(key)
	if err != nil {
		return util.StatusError
	}

	if cachedObj == nil {
		return util.StatusAllOK
	}
	typeConfig := cachedObj.(*corev1a1.FederatedTypeConfig)

	// TODO(marun) Perform this defaulting in a webhook
	corev1a1.SetFederatedTypeConfigDefaults(typeConfig)

	// TODO(marun) Replace with validation webhook
	err = typeconfig.CheckTypeConfigName(typeConfig)
	if err != nil {
		runtime.HandleError(err)
		return util.StatusError
	}

	syncEnabled := typeConfig.Spec.PropagationEnabled
	statusEnabled := typeConfig.Spec.EnableStatus

	limitedScope := c.controllerConfig.TargetNamespace != metav1.NamespaceAll
	if limitedScope && syncEnabled && !typeConfig.GetNamespaced() {
		_, ok := c.getStopChannel(typeConfig.Name)
		if !ok {
			holderChan := make(chan struct{})
			c.lock.Lock()
			c.stopChannels[typeConfig.Name] = holderChan
			c.lock.Unlock()
			glog.Infof("Skipping start of sync & status controller for cluster-scoped resource %q. It is not required for a namespaced federation control plane.", typeConfig.GetFederatedType().Kind)
		}

		typeConfig.Status.ObservedGeneration = typeConfig.Generation
		typeConfig.Status.PropagationController = corev1a1.ControllerStatusNotRunning
		typeConfig.Status.StatusController = corev1a1.ControllerStatusNotRunning
		err = c.client.UpdateStatus(context.TODO(), typeConfig)
		if err != nil {
			runtime.HandleError(errors.Wrapf(err, "Could not update status fields of the CRD: %q", key))
			return util.StatusError
		}
		return util.StatusAllOK
	}

	statusKey := typeConfig.Name + "/status"
	syncStopChan, syncRunning := c.getStopChannel(typeConfig.Name)
	statusStopChan, statusRunning := c.getStopChannel(statusKey)

	deleted := typeConfig.DeletionTimestamp != nil
	if deleted {
		if syncRunning {
			c.stopController(typeConfig.Name, syncStopChan)
		}
		if statusRunning {
			c.stopController(statusKey, statusStopChan)
		}

		if typeConfig.IsNamespace() {
			glog.Infof("Reconciling all namespaced FederatedTypeConfig resources on deletion of %q", key)
			c.reconcileOnNamespaceFTCUpdate()
		}

		err := c.removeFinalizer(typeConfig)
		if err != nil {
			runtime.HandleError(errors.Wrapf(err, "Failed to remove finalizer from FederatedTypeConfig %q", key))
			return util.StatusError
		}
		return util.StatusAllOK
	}

	updated, err := c.ensureFinalizer(typeConfig)
	if err != nil {
		runtime.HandleError(errors.Wrapf(err, "Failed to ensure finalizer for FederatedTypeConfig %q", key))
		return util.StatusError
	} else if updated && typeConfig.IsNamespace() {
		// Detected creation of the namespace FTC. If there are existing FTCs
		// which did not start their sync controllers due to the lack of a
		// namespace FTC, then reconcile them now so they can start.
		glog.Infof("Reconciling all namespaced FederatedTypeConfig resources on finalizer update for %q", key)
		c.reconcileOnNamespaceFTCUpdate()
	}

	startNewSyncController := !syncRunning && syncEnabled
	stopSyncController := syncRunning && (!syncEnabled || (typeConfig.GetNamespaced() && !c.namespaceFTCExists()))
	if startNewSyncController {
		if err := c.startSyncController(typeConfig); err != nil {
			runtime.HandleError(err)
			return util.StatusError
		}
	} else if stopSyncController {
		c.stopController(typeConfig.Name, syncStopChan)
	}

	startNewStatusController := !statusRunning && statusEnabled
	stopStatusController := statusRunning && !statusEnabled
	if startNewStatusController {
		if err := c.startStatusController(statusKey, typeConfig); err != nil {
			runtime.HandleError(err)
			return util.StatusError
		}
	} else if stopStatusController {
		c.stopController(statusKey, statusStopChan)
	}

	typeConfig.Status.ObservedGeneration = typeConfig.Generation
	syncControllerRunning := startNewSyncController || (syncRunning && !stopSyncController)
	if syncControllerRunning {
		typeConfig.Status.PropagationController = corev1a1.ControllerStatusRunning
	} else {
		typeConfig.Status.PropagationController = corev1a1.ControllerStatusNotRunning
	}

	statusControllerRunning := startNewStatusController || (statusRunning && !stopStatusController)
	if statusControllerRunning {
		typeConfig.Status.StatusController = corev1a1.ControllerStatusRunning
	} else {
		typeConfig.Status.StatusController = corev1a1.ControllerStatusNotRunning
	}
	err = c.client.UpdateStatus(context.TODO(), typeConfig)
	if err != nil {
		runtime.HandleError(errors.Wrapf(err, "Could not update status fields of the CRD: %q", key))
		return util.StatusError
	}
	return util.StatusAllOK
}

func (c *Controller) objCopyFromCache(key string) (pkgruntime.Object, error) {
	cachedObj, exist, err := c.store.GetByKey(key)
	if err != nil {
		wrappedErr := errors.Wrapf(err, "Failed to query FederatedTypeConfig store for %q", key)
		runtime.HandleError(wrappedErr)
		return nil, err
	}
	if !exist {
		return nil, nil
	}
	return cachedObj.(pkgruntime.Object).DeepCopyObject(), nil
}

func (c *Controller) shutDown() {
	c.lock.Lock()
	defer c.lock.Unlock()

	// Stop all sync and status controllers
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

func (c *Controller) startSyncController(tc *corev1a1.FederatedTypeConfig) error {
	// TODO(marun) Consider using a shared informer for federated
	// namespace that can be shared between all controllers of a
	// cluster-scoped federation control plane.  A namespace-scoped
	// control plane would still have to use a non-shared informer due
	// to it not being possible to limit its scope.
	kind := tc.Spec.FederatedType.Kind
	fedNamespaceAPIResource, err := c.getFederatedNamespaceAPIResource()
	if err != nil {
		return errors.Wrapf(err, "Unable to start sync controller for %q due to missing FederatedTypeConfig for namespaces", kind)
	}
	stopChan := make(chan struct{})
	err = synccontroller.StartFederationSyncController(c.controllerConfig, stopChan, tc, fedNamespaceAPIResource)
	if err != nil {
		close(stopChan)
		return errors.Wrapf(err, "Error starting sync controller for %q", kind)
	}
	glog.Infof("Started sync controller for %q", kind)
	c.lock.Lock()
	defer c.lock.Unlock()
	c.stopChannels[tc.Name] = stopChan
	return nil
}

func (c *Controller) startStatusController(statusKey string, tc *corev1a1.FederatedTypeConfig) error {
	kind := tc.Spec.FederatedType.Kind
	stopChan := make(chan struct{})
	err := statuscontroller.StartFederationStatusController(c.controllerConfig, stopChan, tc)
	if err != nil {
		close(stopChan)
		return errors.Wrapf(err, "Error starting status controller for %q", kind)
	}
	glog.Infof("Started status controller for %q", kind)
	c.lock.Lock()
	defer c.lock.Unlock()
	c.stopChannels[statusKey] = stopChan
	return nil
}

func (c *Controller) stopController(key string, stopChan chan struct{}) {
	glog.Infof("Stopping controller for %q", key)
	close(stopChan)
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.stopChannels, key)
}

func (c *Controller) ensureFinalizer(tc *corev1a1.FederatedTypeConfig) (bool, error) {
	accessor, err := meta.Accessor(tc)
	if err != nil {
		return false, err
	}
	finalizers := sets.NewString(accessor.GetFinalizers()...)
	if finalizers.Has(finalizer) {
		return false, nil
	}
	finalizers.Insert(finalizer)
	accessor.SetFinalizers(finalizers.List())
	err = c.client.Update(context.TODO(), tc)
	return true, err
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
	err = c.client.Update(context.TODO(), tc)
	return err
}

func (c *Controller) namespaceFTCExists() bool {
	_, err := c.getFederatedNamespaceAPIResource()
	return err == nil
}

func (c *Controller) getFederatedNamespaceAPIResource() (*metav1.APIResource, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	qualifiedName := util.QualifiedName{
		Namespace: c.controllerConfig.FederationNamespace,
		Name:      util.NamespaceName,
	}
	key := qualifiedName.String()
	cachedObj, exists, err := c.store.GetByKey(key)
	if err != nil {
		return nil, errors.Wrapf(err, "Error retrieving %q from the informer cache", key)
	}
	if !exists {
		return nil, errors.Errorf("Unable to find %q in the informer cache", key)
	}
	namespaceTypeConfig := cachedObj.(*corev1a1.FederatedTypeConfig)
	apiResource := namespaceTypeConfig.GetFederatedType()
	return &apiResource, nil
}

func (c *Controller) reconcileOnNamespaceFTCUpdate() {
	for _, cachedObj := range c.store.List() {
		typeConfig := cachedObj.(*corev1a1.FederatedTypeConfig)
		if typeConfig.GetNamespaced() && !typeConfig.IsNamespace() {
			c.worker.EnqueueObject(typeConfig)
		}
	}
}
