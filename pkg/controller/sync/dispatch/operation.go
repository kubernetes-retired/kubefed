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
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/util/runtime"

	"sigs.k8s.io/kubefed/pkg/client/generic"
	"sigs.k8s.io/kubefed/pkg/controller/sync/status"
	"sigs.k8s.io/kubefed/pkg/controller/util"
)

type clientAccessorFunc func(clusterName string) (generic.Client, error)

type dispatchRecorder interface {
	recordEvent(clusterName, operation, operationContinuous string)
	recordOperationError(status status.PropagationStatus, clusterName, operation string, err error) util.ReconciliationStatus
}

// OperationDispatcher provides an interface to wait for operations
// dispatched to member clusters.
type OperationDispatcher interface {
	// Wait returns true for ok if all operations completed
	// successfully and false if only some operations completed
	// successfully.  An error is returned on timeout.
	Wait() (ok bool, timeoutErr error)
}

type operationDispatcherImpl struct {
	clientAccessor clientAccessorFunc

	resultChan          chan util.ReconciliationStatus
	operationsInitiated int32

	timeout time.Duration

	recorder dispatchRecorder
}

func newOperationDispatcher(clientAccessor clientAccessorFunc, recorder dispatchRecorder) *operationDispatcherImpl {
	return &operationDispatcherImpl{
		clientAccessor: clientAccessor,
		resultChan:     make(chan util.ReconciliationStatus),
		timeout:        30 * time.Second, // TODO(marun) Make this configurable
		recorder:       recorder,
	}
}

func (d *operationDispatcherImpl) Wait() (bool, error) {
	ok := true
	timedOut := false
	start := time.Now()
	for i := int32(0); i < atomic.LoadInt32(&d.operationsInitiated); i++ {
		now := time.Now()
		if !now.Before(start.Add(d.timeout)) {
			timedOut = true
			break
		}
		select {
		case result := <-d.resultChan:
			if result == util.StatusError {
				ok = false
			}
			break
		case <-time.After(start.Add(d.timeout).Sub(now)):
			timedOut = true
			break
		}
	}
	if timedOut {
		return false, errors.Errorf("Failed to finish %d operations in %v", atomic.LoadInt32(&d.operationsInitiated), d.timeout)
	}
	return ok, nil
}

func (d *operationDispatcherImpl) clusterOperation(clusterName, op string, opFunc func(generic.Client) util.ReconciliationStatus) {
	// TODO(marun) Support cancellation of client calls on timeout.
	client, err := d.clientAccessor(clusterName)
	if err != nil {
		wrappedErr := errors.Wrapf(err, "Error retrieving client for cluster")
		if d.recorder == nil {
			runtime.HandleError(wrappedErr)
		} else {
			d.recorder.recordOperationError(status.ClientRetrievalFailed, clusterName, op, wrappedErr)
		}
		d.resultChan <- util.StatusError
		return
	}

	// TODO(marun) Retry on recoverable errors (e.g. IsConflict, AlreadyExists)
	ok := opFunc(client)
	d.resultChan <- ok
}

func (d *operationDispatcherImpl) incrementOperationsInitiated() {
	atomic.AddInt32(&d.operationsInitiated, 1)
}
