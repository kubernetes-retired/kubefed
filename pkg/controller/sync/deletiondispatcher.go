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

package sync

import (
	"sync/atomic"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/runtime"

	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
)

const eventTemplate = "%s %s %q in cluster %q"

type clientAccessorFunc func(clusterName string) (util.ResourceClient, error)

type DispatchRecorder interface {
	RecordError(clusterName, operation string, err error)
	RecordEvent(clusterName, operation, operationContinuous string)
}

// DeletionDispatcher dispatches operations to member clusters for a
// federated resource that is marked as deleted.
type DeletionDispatcher interface {
	Wait() (ok bool)

	Delete(clusterName string, clusterObj *unstructured.Unstructured)
	RemoveManagedLabel(clusterName string, clusterObj *unstructured.Unstructured)
}

type deletionDispatcherImpl struct {
	clientAccessor clientAccessorFunc

	targetName util.QualifiedName
	targetKind string

	resultChan          chan util.ReconciliationStatus
	operationsInitiated int32

	timeout time.Duration

	recorder DispatchRecorder
}

func NewDeletionDispatcher(clientAccessor clientAccessorFunc, targetKind string, targetName util.QualifiedName) DeletionDispatcher {
	return newRecordedDeletionDispatcher(clientAccessor, targetKind, targetName, nil)
}

func newRecordedDeletionDispatcher(clientAccessor clientAccessorFunc, targetKind string, targetName util.QualifiedName, recorder DispatchRecorder) DeletionDispatcher {
	return &deletionDispatcherImpl{
		clientAccessor: clientAccessor,
		targetName:     targetName,
		targetKind:     targetKind,
		resultChan:     make(chan util.ReconciliationStatus),
		timeout:        30 * time.Second, // TODO(marun) Make this configurable
		recorder:       recorder,
	}
}

func (dd *deletionDispatcherImpl) Wait() bool {
	ok := true
	timedOut := false
	start := time.Now()
	for i := int32(0); i < atomic.LoadInt32(&dd.operationsInitiated); i++ {
		now := time.Now()
		if !now.Before(start.Add(dd.timeout)) {
			timedOut = true
			break
		}
		select {
		case result := <-dd.resultChan:
			if result == util.StatusError {
				ok = false
			}
			break
		case <-time.After(start.Add(dd.timeout).Sub(now)):
			timedOut = true
			break
		}
	}
	if timedOut {
		runtime.HandleError(errors.Errorf("Failed to finish %d operations in %v", atomic.LoadInt32(&dd.operationsInitiated), dd.timeout))
	}

	return ok
}

func (dd *deletionDispatcherImpl) clusterOperation(clusterName, op string, opFunc func(util.ResourceClient) util.ReconciliationStatus) {
	// TODO(marun) Update to generic client and support cancellation
	// on timeout.
	client, err := dd.clientAccessor(clusterName)
	if err != nil {
		wrappedErr := errors.Wrapf(err, "Error retrieving client for cluster")
		if dd.recorder == nil {
			runtime.HandleError(wrappedErr)
		} else {
			dd.recorder.RecordError(clusterName, op, wrappedErr)
		}
		dd.resultChan <- util.StatusError
		return
	}

	// TODO(marun) Retry on recoverable errors (e.g. IsConflict, AlreadyExists)
	ok := opFunc(client)
	dd.resultChan <- ok
}

func (dd *deletionDispatcherImpl) incrementOperationsInitiated() {
	atomic.AddInt32(&dd.operationsInitiated, 1)
}

func (dd *deletionDispatcherImpl) Delete(clusterName string, clusterObj *unstructured.Unstructured) {
	dd.incrementOperationsInitiated()
	const op = "delete"
	const opContinuous = "Deleting"
	go dd.clusterOperation(clusterName, op, func(client util.ResourceClient) util.ReconciliationStatus {
		if clusterObj.GetDeletionTimestamp() != nil {
			return util.StatusAllOK
		}

		if dd.recorder == nil {
			glog.V(2).Infof(eventTemplate, opContinuous, dd.targetKind, dd.targetName, clusterName)
		} else {
			dd.recorder.RecordEvent(clusterName, op, opContinuous)
		}

		err := client.Resources(dd.targetName.Namespace).Delete(dd.targetName.Name, &metav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			err = nil
		}
		if err != nil {
			wrappedErr := dd.wrapOperationError(err, clusterName, op)
			if dd.recorder == nil {
				runtime.HandleError(wrappedErr)
			} else {
				dd.recorder.RecordError(clusterName, op, wrappedErr)
			}
			return util.StatusError
		}
		return util.StatusAllOK
	})
}

func (dd *deletionDispatcherImpl) RemoveManagedLabel(clusterName string, clusterObj *unstructured.Unstructured) {
	dd.incrementOperationsInitiated()
	const op = "remove managed label"
	const opContinuous = "Removing managed label"
	go dd.clusterOperation(clusterName, op, func(client util.ResourceClient) util.ReconciliationStatus {
		if dd.recorder == nil {
			glog.V(2).Infof(eventTemplate, opContinuous, dd.targetKind, dd.targetName, clusterName)
		} else {
			dd.recorder.RecordEvent(clusterName, op, opContinuous)
		}

		// Avoid mutating the resource in the informer cache
		updateObj := clusterObj.DeepCopy()

		util.RemoveManagedLabel(updateObj)

		_, err := client.Resources(updateObj.GetNamespace()).Update(updateObj, metav1.UpdateOptions{})
		if err != nil {
			wrappedErr := dd.wrapOperationError(err, clusterName, op)
			if dd.recorder == nil {
				runtime.HandleError(wrappedErr)
			} else {
				dd.recorder.RecordError(clusterName, op, wrappedErr)
			}
			return util.StatusError
		}
		return util.StatusAllOK
	})
}

func (dd *deletionDispatcherImpl) wrapOperationError(err error, clusterName, operation string) error {
	return errors.Wrapf(err, "Failed to %s %s %q in cluster %q", operation, dd.targetKind, dd.targetName, clusterName)
}
