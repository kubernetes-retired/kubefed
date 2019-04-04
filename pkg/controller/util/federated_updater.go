/*
Copyright 2016 The Kubernetes Authors.

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

// TODO(marun) Remove this once its use is entirely replaced with the
// new sync controller updater.

package util

import (
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

	apiv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

// Type of the operation that can be executed in Federated.
type FederatedOperationType string

const (
	OperationTypeAdd    = "add"
	OperationTypeUpdate = "update"
	OperationTypeDelete = "delete"
)

type operationResult struct {
	clusterName string
	version     string
	err         error
}

// FederatedOperation definition contains type (add/update/delete) and the object itself.
type FederatedOperation struct {
	Type        FederatedOperationType
	ClusterName string
	Obj         pkgruntime.Object
	Key         string
}

// A helper that executes the given set of updates on federation, in parallel.
type FederatedUpdater interface {
	// Executes the given set of operations.
	Update([]FederatedOperation) (map[string]string, []error)
}

// A function that executes some operation using the passed client and object.
type FederatedOperationHandler func(ResourceClient, pkgruntime.Object) (string, error)

type federatedUpdaterImpl struct {
	federation FederationView

	kind string

	timeout time.Duration

	eventRecorder record.EventRecorder

	addFunction    FederatedOperationHandler
	updateFunction FederatedOperationHandler
	deleteFunction FederatedOperationHandler
}

func NewFederatedUpdater(federation FederationView, kind string, timeout time.Duration, recorder record.EventRecorder, add, update, del FederatedOperationHandler) FederatedUpdater {
	return &federatedUpdaterImpl{
		federation:     federation,
		kind:           kind,
		timeout:        timeout,
		eventRecorder:  recorder,
		addFunction:    add,
		updateFunction: update,
		deleteFunction: del,
	}
}

func (fu *federatedUpdaterImpl) recordEvent(obj runtime.Object, eventType, reason, eventVerb string, args ...interface{}) {
	// TODO(marun) Ensure the federated updater is logging events to
	// objects in the federation api.  'Obj' is intended to appear in
	// the member cluser, not the federation api.
	//messageFmt := eventVerb + " %s %q in cluster %s"
	//fu.eventRecorder.Eventf(obj, eventType, reason, eventType, messageFmt, args...)
}

// Update executes the given set of operations within the timeout specified for
// the instance. Timeout is best-effort. There is no guarantee that the
// underlying operations are stopped when it is reached. However the function
// will return after the timeout with a non-nil error.
func (fu *federatedUpdaterImpl) Update(ops []FederatedOperation) (map[string]string, []error) {
	done := make(chan operationResult, len(ops))
	for _, op := range ops {
		go func(op FederatedOperation) {
			clusterName := op.ClusterName

			// TODO: Ensure that the client has reasonable timeout.
			client, err := fu.federation.GetClientForCluster(clusterName)
			if err != nil {
				done <- operationResult{err: err}
				return
			}

			eventArgs := []interface{}{fu.kind, op.Key, clusterName}
			baseEventType := fmt.Sprintf("%s", op.Type)
			eventType := fmt.Sprintf("%sInCluster", strings.Title(baseEventType))

			version := ""

			switch op.Type {
			case OperationTypeAdd:
				// TODO s+OperationTypeAdd+OperationTypeCreate+
				baseEventType = "create"
				eventType := "CreateInCluster"

				fu.recordEvent(op.Obj, apiv1.EventTypeNormal, eventType, "Creating", eventArgs...)
				version, err = fu.addFunction(client, op.Obj)
			case OperationTypeUpdate:
				fu.recordEvent(op.Obj, apiv1.EventTypeNormal, eventType, "Updating", eventArgs...)
				version, err = fu.updateFunction(client, op.Obj)
			case OperationTypeDelete:
				fu.recordEvent(op.Obj, apiv1.EventTypeNormal, eventType, "Deleting", eventArgs...)
				_, err = fu.deleteFunction(client, op.Obj)
				// IsNotFound error is fine since that means the object is deleted already.
				if apierrors.IsNotFound(err) {
					err = nil
				}
			}

			if err != nil {
				eventType := eventType + "Failed"
				messageFmt := "Failed to " + baseEventType + " %s %q in cluster %s: %v"
				eventArgs = append(eventArgs, err)
				err = errors.Errorf(messageFmt, eventArgs...)
				fu.recordEvent(op.Obj, apiv1.EventTypeWarning, eventType, messageFmt, eventArgs...)
			}

			done <- operationResult{
				clusterName: clusterName,
				version:     version,
				err:         err,
			}
		}(op)
	}

	versions := make(map[string]string)
	updateErrs := []error{}
	timedOut := false

	start := time.Now()
	for i := 0; i < len(ops); i++ {
		now := time.Now()
		if !now.Before(start.Add(fu.timeout)) {
			timedOut = true
			break
		}
		select {
		case result := <-done:
			if result.err != nil {
				updateErrs = append(updateErrs, result.err)
				break
			}
			versions[result.clusterName] = result.version
		case <-time.After(start.Add(fu.timeout).Sub(now)):
			timedOut = true
			break
		}
	}

	if timedOut {
		updateErrs = append(updateErrs, errors.Errorf("Failed to finish all operations in %v", fu.timeout))
	}

	return versions, updateErrs
}
