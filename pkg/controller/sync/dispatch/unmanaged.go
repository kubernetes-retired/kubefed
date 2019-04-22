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
	"github.com/golang/glog"
	"github.com/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/runtime"

	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
)

const eventTemplate = "%s %s %q in cluster %q"

// UnmanagedDispatcher dispatches operations to member clusters for
// resources that are no longer managed by a federated resource.
type UnmanagedDispatcher interface {
	OperationDispatcher

	Delete(clusterName string)
	RemoveManagedLabel(clusterName string, clusterObj *unstructured.Unstructured)
}

type unmanagedDispatcherImpl struct {
	dispatcher *operationDispatcherImpl

	targetName util.QualifiedName
	targetKind string

	recorder DispatchRecorder
}

func NewUnmanagedDispatcher(clientAccessor clientAccessorFunc, targetKind string, targetName util.QualifiedName) UnmanagedDispatcher {
	dispatcher := newOperationDispatcher(clientAccessor, nil)
	return newUnmanagedDispatcher(dispatcher, nil, targetKind, targetName)
}

func newUnmanagedDispatcher(dispatcher *operationDispatcherImpl, recorder DispatchRecorder, targetKind string, targetName util.QualifiedName) *unmanagedDispatcherImpl {
	return &unmanagedDispatcherImpl{
		dispatcher: dispatcher,
		targetName: targetName,
		targetKind: targetKind,
		recorder:   recorder,
	}
}

func (d *unmanagedDispatcherImpl) Wait() (bool, error) {
	return d.dispatcher.Wait()
}

func (d *unmanagedDispatcherImpl) Delete(clusterName string) {
	d.dispatcher.incrementOperationsInitiated()
	const op = "delete"
	const opContinuous = "Deleting"
	go d.dispatcher.clusterOperation(clusterName, op, func(client util.ResourceClient) util.ReconciliationStatus {
		if d.recorder == nil {
			glog.V(2).Infof(eventTemplate, opContinuous, d.targetKind, d.targetName, clusterName)
		} else {
			d.recorder.RecordEvent(clusterName, op, opContinuous)
		}

		err := client.Resources(d.targetName.Namespace).Delete(d.targetName.Name, &metav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			err = nil
		}
		if err != nil {
			if d.recorder == nil {
				wrappedErr := d.wrapOperationError(err, clusterName, op)
				runtime.HandleError(wrappedErr)
			} else {
				d.recorder.RecordError(clusterName, op, err)
			}
			return util.StatusError
		}
		return util.StatusAllOK
	})
}

func (d *unmanagedDispatcherImpl) RemoveManagedLabel(clusterName string, clusterObj *unstructured.Unstructured) {
	d.dispatcher.incrementOperationsInitiated()
	const op = "remove managed label from"
	const opContinuous = "Removing managed label from"
	go d.dispatcher.clusterOperation(clusterName, op, func(client util.ResourceClient) util.ReconciliationStatus {
		if d.recorder == nil {
			glog.V(2).Infof(eventTemplate, opContinuous, d.targetKind, d.targetName, clusterName)
		} else {
			d.recorder.RecordEvent(clusterName, op, opContinuous)
		}

		// Avoid mutating the resource in the informer cache
		updateObj := clusterObj.DeepCopy()

		util.RemoveManagedLabel(updateObj)

		_, err := client.Resources(updateObj.GetNamespace()).Update(updateObj, metav1.UpdateOptions{})
		if err != nil {
			if d.recorder == nil {
				wrappedErr := d.wrapOperationError(err, clusterName, op)
				runtime.HandleError(wrappedErr)
			} else {
				d.recorder.RecordError(clusterName, op, err)
			}
			return util.StatusError
		}
		return util.StatusAllOK
	})
}

func (d *unmanagedDispatcherImpl) wrapOperationError(err error, clusterName, operation string) error {
	return errors.Wrapf(err, "Failed to "+eventTemplate, operation, d.targetKind, d.targetName, clusterName)
}
