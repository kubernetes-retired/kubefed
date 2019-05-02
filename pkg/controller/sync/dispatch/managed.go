/*
Copyright 2019 The Kubernetes Authors.

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

package dispatch

import (
	"fmt"
	"strings"
	"sync"

	"github.com/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
)

// FederatedResourceForDispatch is the subset of the FederatedResource
// interface required for dispatching operations to managed resources.
type FederatedResourceForDispatch interface {
	TargetName() util.QualifiedName
	TargetKind() string
	Object() *unstructured.Unstructured
	VersionForCluster(clusterName string) (string, error)
	ObjectForCluster(clusterName string) (*unstructured.Unstructured, error)
	RecordError(errorCode string, err error)
	RecordEvent(reason, messageFmt string, args ...interface{})
}

// ManagedDispatcher dispatches operations to member clusters for resources
// managed by a federated resource.
type ManagedDispatcher interface {
	UnmanagedDispatcher

	Create(clusterName string)
	Update(clusterName string, clusterObj *unstructured.Unstructured)
	VersionMap() map[string]string
}

type managedDispatcherImpl struct {
	sync.RWMutex

	dispatcher            *operationDispatcherImpl
	unmanagedDispatcher   *unmanagedDispatcherImpl
	fedResource           FederatedResourceForDispatch
	versionMap            map[string]string
	skipAdoptingResources bool
}

func NewManagedDispatcher(clientAccessor clientAccessorFunc, fedResource FederatedResourceForDispatch, skipAdoptingResources bool) ManagedDispatcher {
	d := &managedDispatcherImpl{
		fedResource:           fedResource,
		versionMap:            make(map[string]string),
		skipAdoptingResources: skipAdoptingResources,
	}
	d.dispatcher = newOperationDispatcher(clientAccessor, d)
	d.unmanagedDispatcher = newUnmanagedDispatcher(d.dispatcher, d, fedResource.TargetKind(), fedResource.TargetName())
	return d
}

func (d *managedDispatcherImpl) Wait() (bool, error) {
	return d.dispatcher.Wait()
}

func (d *managedDispatcherImpl) Create(clusterName string) {
	d.dispatcher.incrementOperationsInitiated()
	const op = "create"
	go d.dispatcher.clusterOperation(clusterName, op, func(client util.ResourceClient) util.ReconciliationStatus {
		d.RecordEvent(clusterName, op, "Creating")

		obj, err := d.fedResource.ObjectForCluster(clusterName)
		if err != nil {
			d.RecordError(clusterName, op, err)
			return util.StatusError
		}
		createdObj, err := client.Resources(obj.GetNamespace()).Create(obj, metav1.CreateOptions{})
		if err == nil {
			version := util.ObjectVersion(createdObj)
			d.recordVersion(clusterName, version)
			return util.StatusAllOK
		}

		// TODO(marun) Figure out why attempting to create a namespace that
		// already exists indicates ServerTimeout instead of AlreadyExists.
		alreadyExists := apierrors.IsAlreadyExists(err) || d.fedResource.TargetKind() == util.NamespaceKind && apierrors.IsServerTimeout(err)
		if !alreadyExists {
			d.RecordError(clusterName, op, err)
			return util.StatusError
		}

		if d.skipAdoptingResources {
			d.RecordError(clusterName, op, errors.Errorf("Resource pre-exist in cluster"))
			return util.StatusAllOK
		}

		// Attempt to update the existing resource to ensure that it
		// is labeled as a managed resource.
		clusterObj, err := client.Resources(obj.GetNamespace()).Get(obj.GetName(), metav1.GetOptions{})
		if err != nil {
			wrappedErr := errors.Wrapf(err, "failed to retrieve object potentially requiring adoption")
			d.RecordError(clusterName, op, wrappedErr)
			return util.StatusError
		}
		d.RecordError(clusterName, op, errors.Errorf("An update will be attempted instead of a creation due to an existing resource"))
		d.Update(clusterName, clusterObj)
		return util.StatusAllOK
	})
}

func (d *managedDispatcherImpl) Update(clusterName string, clusterObj *unstructured.Unstructured) {
	d.dispatcher.incrementOperationsInitiated()
	const op = "update"
	go d.dispatcher.clusterOperation(clusterName, op, func(client util.ResourceClient) util.ReconciliationStatus {
		obj, err := d.fedResource.ObjectForCluster(clusterName)
		if err != nil {
			d.RecordError(clusterName, op, err)
			return util.StatusError
		}

		err = RetainClusterFields(d.fedResource.TargetKind(), obj, clusterObj, d.fedResource.Object())
		if err != nil {
			wrappedErr := errors.Wrapf(err, "failed to retain fields")
			d.RecordError(clusterName, op, wrappedErr)
			return util.StatusError
		}

		version, err := d.fedResource.VersionForCluster(clusterName)
		if err != nil {
			d.RecordError(clusterName, op, err)
			return util.StatusError
		}
		if !util.ObjectNeedsUpdate(obj, clusterObj, version) {
			// Resource is current
			return util.StatusAllOK
		}

		// Only record an event if the resource is not current
		d.RecordEvent(clusterName, op, "Updating")

		updatedObj, err := client.Resources(obj.GetNamespace()).Update(obj, metav1.UpdateOptions{})
		if err != nil {
			d.RecordError(clusterName, op, err)
			return util.StatusError
		}
		version = util.ObjectVersion(updatedObj)
		d.recordVersion(clusterName, version)
		return util.StatusAllOK
	})
}

func (d *managedDispatcherImpl) Delete(clusterName string) {
	d.unmanagedDispatcher.Delete(clusterName)
}

func (d *managedDispatcherImpl) RemoveManagedLabel(clusterName string, clusterObj *unstructured.Unstructured) {
	d.unmanagedDispatcher.RemoveManagedLabel(clusterName, clusterObj)
}

func (d *managedDispatcherImpl) RecordError(clusterName, operation string, err error) {
	args := []interface{}{operation, d.fedResource.TargetKind(), d.fedResource.TargetName(), clusterName}
	eventType := fmt.Sprintf("%sInClusterFailed", strings.Replace(strings.Title(operation), " ", "", -1))
	d.fedResource.RecordError(eventType, errors.Wrapf(err, "Failed to "+eventTemplate, args...))
}

func (d *managedDispatcherImpl) RecordEvent(clusterName, operation, operationContinuous string) {
	args := []interface{}{operationContinuous, d.fedResource.TargetKind(), d.fedResource.TargetName(), clusterName}
	eventType := fmt.Sprintf("%sInCluster", strings.Replace(strings.Title(operation), " ", "", -1))
	d.fedResource.RecordEvent(eventType, eventTemplate, args...)
}

func (d *managedDispatcherImpl) VersionMap() map[string]string {
	d.RLock()
	defer d.RUnlock()
	versionMap := make(map[string]string)
	for key, value := range d.versionMap {
		versionMap[key] = value
	}
	return versionMap
}

func (d *managedDispatcherImpl) recordVersion(clusterName, version string) {
	d.Lock()
	defer d.Unlock()
	d.versionMap[clusterName] = version
}
