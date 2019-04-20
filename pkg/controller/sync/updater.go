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
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
)

type FederatedUpdater interface {
	NoChanges() bool
	Wait() (map[string]string, bool)

	Create(clusterName string)
	Update(clusterName string, clusterObj *unstructured.Unstructured)
	Delete(clusterName string)
	RemoveManagedLabel(clusterName string, clusterObj *unstructured.Unstructured)
}

type updaterResult struct {
	clusterName string
	version     string
	ok          bool
}

type federatedUpdaterImpl struct {
	fedView util.FederationView

	fedResource FederatedResource

	resultChan     chan updaterResult
	operationCount int32

	timeout time.Duration
}

func NewFederatedUpdater(fedView util.FederationView, fedResource FederatedResource) FederatedUpdater {
	return &federatedUpdaterImpl{
		fedView:     fedView,
		fedResource: fedResource,
		resultChan:  make(chan updaterResult),
		timeout:     30 * time.Second, // TODO(marun) Make this configurable
	}
}

func (u *federatedUpdaterImpl) NoChanges() bool {
	return atomic.LoadInt32(&u.operationCount) == 0
}

func (u *federatedUpdaterImpl) Wait() (map[string]string, bool) {
	versions := make(map[string]string)
	ok := true

	timedOut := false
	start := time.Now()
	for i := int32(0); i < atomic.LoadInt32(&u.operationCount); i++ {
		now := time.Now()
		if !now.Before(start.Add(u.timeout)) {
			timedOut = true
			break
		}
		select {
		case result := <-u.resultChan:
			if !result.ok {
				ok = false
				break
			}
			if len(result.version) > 0 {
				versions[result.clusterName] = result.version
			}
		case <-time.After(start.Add(u.timeout).Sub(now)):
			timedOut = true
			break
		}
	}
	if timedOut {
		ok = false
		u.fedResource.RecordError("OperationTimeoutError", errors.Errorf("Failed to finish all operations in %v", u.timeout))
	}

	return versions, ok
}

func (u *federatedUpdaterImpl) Create(clusterName string) {
	u.incrementOperationCount()
	const op = "create"
	go u.clusterOperation(clusterName, op, func(client util.ResourceClient) (string, error) {
		u.recordEvent(clusterName, op, "Creating")

		obj, err := u.fedResource.ObjectForCluster(clusterName)
		if err != nil {
			return "", err
		}
		createdObj, err := client.Resources(obj.GetNamespace()).Create(obj, metav1.CreateOptions{})
		if err == nil {
			return util.ObjectVersion(createdObj), nil
		}
		// TODO(marun) Figure out why attempting to create a namespace that
		// already exists indicates ServerTimeout instead of AlreadyExists.
		alreadyExists := apierrors.IsAlreadyExists(err) || u.fedResource.TargetKind() == util.NamespaceKind && apierrors.IsServerTimeout(err)
		if !alreadyExists {
			return "", err
		}

		// Attempt to update the existing resource to ensure that it
		// is labeled as a managed resource.
		clusterObj, err := client.Resources(obj.GetNamespace()).Get(obj.GetName(), metav1.GetOptions{})
		if err != nil {
			return "", errors.Wrapf(err, "Failed to retrieve object potentially requiring adoption for cluster %q", clusterName)
		}
		u.Update(clusterName, clusterObj)
		return "", errors.Errorf("An update will be attempted instead of a creation due to an existing resource in cluster %q", clusterName)
	})
}

func (u *federatedUpdaterImpl) Update(clusterName string, clusterObj *unstructured.Unstructured) {
	u.incrementOperationCount()
	const op = "update"
	go u.clusterOperation(clusterName, op, func(client util.ResourceClient) (string, error) {
		obj, err := u.fedResource.ObjectForCluster(clusterName)
		if err != nil {
			return "", err
		}
		err = RetainClusterFields(u.fedResource.TargetKind(), obj, clusterObj, u.fedResource.Object())
		if err != nil {
			wrappedErr := errors.Wrapf(err, "Failed to retain fields from %s %q for cluster %q",
				u.fedResource.FederatedKind(), u.fedResource.FederatedName(), clusterName)
			return "", wrappedErr
		}

		version, err := u.fedResource.VersionForCluster(clusterName)
		if err != nil {
			return "", err
		}
		if !util.ObjectNeedsUpdate(obj, clusterObj, version) {
			// Resource is current
			return "", nil
		}

		// Only record an event if the resource is not current
		u.recordEvent(clusterName, op, "Updating")

		updatedObj, err := client.Resources(obj.GetNamespace()).Update(obj, metav1.UpdateOptions{})
		if err != nil {
			return "", err
		}
		return util.ObjectVersion(updatedObj), nil
	})
}

func (u *federatedUpdaterImpl) Delete(clusterName string) {
	u.incrementOperationCount()
	const op = "delete"
	go u.clusterOperation(clusterName, op, func(client util.ResourceClient) (string, error) {
		u.recordEvent(clusterName, op, "Deleting")

		qualifiedName := u.fedResource.TargetName()
		err := client.Resources(qualifiedName.Namespace).Delete(qualifiedName.Name, &metav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			err = nil
		}
		return "", err
	})
}

func (u *federatedUpdaterImpl) RemoveManagedLabel(clusterName string, clusterObj *unstructured.Unstructured) {
	u.incrementOperationCount()
	const op = "remove managed label"
	go u.clusterOperation(clusterName, op, func(client util.ResourceClient) (string, error) {
		u.recordEvent(clusterName, op, "Removing managed label")

		// Avoid mutating the resource in the informer cache
		updateObj := clusterObj.DeepCopy()

		util.RemoveManagedLabel(updateObj)
		_, err := client.Resources(updateObj.GetNamespace()).Update(updateObj, metav1.UpdateOptions{})
		return "", err
	})
}

func (u *federatedUpdaterImpl) incrementOperationCount() {
	atomic.AddInt32(&u.operationCount, 1)
}

func (u *federatedUpdaterImpl) clusterOperation(clusterName, op string, opFunc func(util.ResourceClient) (string, error)) {
	result := updaterResult{
		clusterName: clusterName,
		ok:          true,
	}

	// TODO(marun) Update to generic client and support cancellation
	// on timeout.
	client, err := u.fedView.GetClientForCluster(clusterName)
	if err != nil {
		u.recordError(clusterName, op, errors.Wrapf(err, "Error retrieving client for cluster"))
		result.ok = false
		u.resultChan <- result
		return
	}

	// TODO(marun) Retry on recoverable errors (e.g. IsConflict, AlreadyExists)
	version, err := opFunc(client)
	if err != nil {
		u.recordError(clusterName, op, err)
		result.ok = false
	}
	result.version = version
	u.resultChan <- result
}

func (u *federatedUpdaterImpl) recordError(clusterName, operation string, err error) {
	args := []interface{}{operation, u.fedResource.TargetKind(), u.fedResource.TargetName(), clusterName}

	// TODO(marun) It will be necessary to log rather than record an event when
	// the federated resource has been deleted.

	eventType := fmt.Sprintf("%sInClusterFailed", strings.Replace(strings.Title(operation), " ", "", -1))
	u.fedResource.RecordError(eventType, errors.Wrapf(err, "Failed to %s %s %q in cluster %q", args...))
}

func (u *federatedUpdaterImpl) recordEvent(clusterName, operation, operationContinuous string) {
	args := []interface{}{operationContinuous, u.fedResource.TargetKind(), u.fedResource.TargetName(), clusterName}

	// TODO(marun) It will be necessary to log rather than record an event when
	// the federated resource has been deleted.

	eventType := fmt.Sprintf("%sInCluster", strings.Replace(strings.Title(operation), " ", "", -1))
	u.fedResource.RecordEvent(eventType, "%s %s %q in cluster %q", args...)
}
