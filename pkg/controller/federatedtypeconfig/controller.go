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
	"log"
	"sync"

	"github.com/kubernetes-sigs/kubebuilder/pkg/controller"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"

	corev1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	corev1alpha1client "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned/typed/core/v1alpha1"
	corev1alpha1informer "github.com/kubernetes-sigs/federation-v2/pkg/client/informers/externalversions/core/v1alpha1"
	corev1alpha1lister "github.com/kubernetes-sigs/federation-v2/pkg/client/listers/core/v1alpha1"
	synccontroller "github.com/kubernetes-sigs/federation-v2/pkg/controller/sync"

	"github.com/kubernetes-sigs/federation-v2/pkg/inject/args"
)

const (
	finalizer string = "core.federation.k8s.io/federated-type-config"
)

type FederatedTypeConfigController struct {
	lister corev1alpha1lister.FederatedTypeConfigLister
	client corev1alpha1client.CoreV1alpha1Interface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	// Reference to config used to start new controllers
	config *rest.Config

	// The name of the federation system namespace
	fedNamespace string

	// Map of running sync controllers keyed by qualified target type
	stopChannels map[string]chan struct{}
	lock         sync.RWMutex
}

func ProvideController(arguments args.InjectArgs, fedNamespace string, stopChan <-chan struct{}) (*controller.GenericController, error) {
	name := "FederatedTypeConfigController"
	c := &FederatedTypeConfigController{
		lister:       arguments.ControllerManager.GetInformerProvider(&corev1a1.FederatedTypeConfig{}).(corev1alpha1informer.FederatedTypeConfigInformer).Lister(),
		client:       arguments.Clientset.CoreV1alpha1(),
		recorder:     arguments.CreateRecorder(name),
		config:       arguments.Config,
		fedNamespace: fedNamespace,
		stopChannels: make(map[string]chan struct{}),
	}
	// Ensure launched controllers are shutdown cleanly
	go func() {
		<-stopChan
		c.shutDown()
	}()

	gc := &controller.GenericController{
		Name:             name,
		Reconcile:        c.Reconcile,
		InformerRegistry: arguments.ControllerManager,
	}

	if err := gc.Watch(&corev1a1.FederatedTypeConfig{}); err != nil {
		return gc, err
	}
	return gc, nil
}

func (c *FederatedTypeConfigController) Reconcile(k types.ReconcileKey) error {
	log.Printf("Running reconcile FederatedTypeConfig for %s\n", k.Name)

	typeConfig, err := c.lister.Get(k.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("federatedtypeconfig '%s' in work queue no longer exists", k))
			return nil
		}
		return err
	}

	stopChan, running := c.getStopChannel(k.Name)

	deleted := typeConfig.DeletionTimestamp != nil
	if deleted {
		if running {
			c.stopController(k.Name, stopChan)
		}
		return c.removeFinalizer(typeConfig)
	}

	err = c.ensureFinalizer(typeConfig)
	if err != nil {
		return err
	}

	enabled := typeConfig.Spec.PropagationEnabled
	startNewController := !running && enabled
	stopController := running && !enabled
	if startNewController {
		return c.startController(typeConfig)
	} else if stopController {
		c.stopController(k.Name, stopChan)
	}

	return nil
}

func (c *FederatedTypeConfigController) shutDown() {
	c.lock.Lock()
	defer c.lock.Unlock()

	// Stop all sync controllers
	for key, stopChannel := range c.stopChannels {
		close(stopChannel)
		delete(c.stopChannels, key)
	}
}

func (c *FederatedTypeConfigController) getStopChannel(name string) (chan struct{}, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	stopChan, ok := c.stopChannels[name]
	return stopChan, ok
}

func (c *FederatedTypeConfigController) startController(tc *corev1a1.FederatedTypeConfig) error {
	kind := tc.Spec.Template.Kind
	log.Printf("Starting sync controller for %s \n", kind)

	// TODO(marun) Perform this defaulting in a webhook
	corev1a1.SetFederatedTypeConfigDefaults(tc)

	stopChan := make(chan struct{})
	err := synccontroller.StartFederationSyncController(tc, c.config, c.fedNamespace, stopChan, false)
	if err != nil {
		close(stopChan)
		return fmt.Errorf("Error starting sync controller for %q: %v", kind, err)
	}
	log.Printf("Started sync controller for %s \n", kind)
	c.lock.Lock()
	defer c.lock.Unlock()
	c.stopChannels[tc.Name] = stopChan
	return nil
}

func (c *FederatedTypeConfigController) stopController(key string, stopChan chan struct{}) {
	log.Printf("Stopping sync controller for %s \n", key)
	close(stopChan)
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.stopChannels, key)
}

func (c *FederatedTypeConfigController) ensureFinalizer(tc *corev1a1.FederatedTypeConfig) error {
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
	_, err = c.client.FederatedTypeConfigs().Update(tc)
	return err
}

func (c *FederatedTypeConfigController) removeFinalizer(tc *corev1a1.FederatedTypeConfig) error {
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
	_, err = c.client.FederatedTypeConfigs().Update(tc)
	return err
}
