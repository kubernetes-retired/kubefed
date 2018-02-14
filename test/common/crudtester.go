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

package common

import (
	"fmt"
	"time"

	"github.com/marun/fnord/pkg/federatedtypes"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
)

const (
	AnnotationTestFederationCrudUpdate string = "federation.kubernetes.io/test-federation-crud-update"
)

// FederatedTypeCrudTester exercises Create/Read/Update/Delete operations for
// federated types via the Federation API and validates that the
// results of those operations are propagated to clusters that are
// members of a federation.
type FederatedTypeCrudTester struct {
	tl             TestLogger
	adapter        federatedtypes.FederatedTypeAdapter
	kind           string
	clusterClients []clientset.Interface
	waitInterval   time.Duration
	// Federation operations will use wait.ForeverTestTimeout.  Any
	// operation that involves member clusters may take longer due to
	// propagation latency.
	clusterWaitTimeout time.Duration
}

func NewFederatedTypeCrudTester(testLogger TestLogger, adapter federatedtypes.FederatedTypeAdapter, clusterClients []clientset.Interface, waitInterval, clusterWaitTimeout time.Duration) *FederatedTypeCrudTester {
	return &FederatedTypeCrudTester{
		tl:                 testLogger,
		adapter:            adapter,
		kind:               adapter.FedKind(),
		clusterClients:     clusterClients,
		waitInterval:       waitInterval,
		clusterWaitTimeout: clusterWaitTimeout,
	}
}

func (c *FederatedTypeCrudTester) CheckLifecycle(desiredObject pkgruntime.Object) {
	obj := c.CheckCreate(desiredObject)
	c.CheckUpdate(obj)

	// Validate the golden path - removal of dependents
	orphanDependents := false
	c.CheckDelete(obj, &orphanDependents)
}

func (c *FederatedTypeCrudTester) Create(desiredObject pkgruntime.Object) pkgruntime.Object {
	namespace := c.adapter.FedObjectMeta(desiredObject).Namespace
	resourceMsg := c.kind
	if len(namespace) > 0 {
		resourceMsg = fmt.Sprintf("%s in namespace %q", resourceMsg, namespace)
	}

	c.tl.Logf("Creating new %s", resourceMsg)

	obj, err := c.adapter.FedCreate(desiredObject)
	if err != nil {
		c.tl.Fatalf("Error creating %s: %v", resourceMsg, err)
	}

	qualifiedName := federatedtypes.NewQualifiedName(obj)
	c.tl.Logf("Created new %s %q", c.kind, qualifiedName)

	return obj
}

func (c *FederatedTypeCrudTester) CheckCreate(desiredObject pkgruntime.Object) pkgruntime.Object {
	obj := c.Create(desiredObject)

	c.CheckPropagation(obj)

	return obj
}

func (c *FederatedTypeCrudTester) CheckUpdate(obj pkgruntime.Object) {
	qualifiedName := federatedtypes.NewQualifiedName(obj)

	var initialAnnotation string
	meta := c.adapter.FedObjectMeta(obj)
	if meta.Annotations != nil {
		initialAnnotation = meta.Annotations[AnnotationTestFederationCrudUpdate]
	}

	c.tl.Logf("Updating %s %q", c.kind, qualifiedName)
	updatedObj, err := c.updateFedObject(obj)
	if err != nil {
		c.tl.Fatalf("Error updating %s %q: %v", c.kind, qualifiedName, err)
	}

	// updateFedObject is expected to have changed the value of the annotation
	meta = c.adapter.FedObjectMeta(updatedObj)
	updatedAnnotation := meta.Annotations[AnnotationTestFederationCrudUpdate]
	if updatedAnnotation == initialAnnotation {
		c.tl.Fatalf("%s %q not mutated", c.kind, qualifiedName)
	}

	c.CheckPropagation(updatedObj)
}

func (c *FederatedTypeCrudTester) CheckDelete(obj pkgruntime.Object, orphanDependents *bool) {
	qualifiedName := federatedtypes.NewQualifiedName(obj)

	c.tl.Logf("Deleting %s %q", c.kind, qualifiedName)
	err := c.adapter.FedDelete(qualifiedName, &metav1.DeleteOptions{OrphanDependents: orphanDependents})
	if err != nil {
		c.tl.Fatalf("Error deleting %s %q: %v", c.kind, qualifiedName, err)
	}

	deletingInCluster := (orphanDependents != nil && *orphanDependents == false)

	waitTimeout := wait.ForeverTestTimeout
	if deletingInCluster {
		// May need extra time to delete both federation and cluster resources
		waitTimeout = c.clusterWaitTimeout
	}

	// Wait for deletion.  The federation resource will only be removed once orphan deletion has been
	// completed or deemed unnecessary.
	err = wait.PollImmediate(c.waitInterval, waitTimeout, func() (bool, error) {
		_, err := c.adapter.FedGet(qualifiedName)
		if errors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
	if err != nil {
		c.tl.Fatalf("Error deleting %s %q: %v", c.kind, qualifiedName, err)
	}

	var stateMsg string = "present"
	if deletingInCluster {
		stateMsg = "not present"
	}
	for _, client := range c.clusterClients {
		_, err := c.adapter.Get(client, qualifiedName)
		switch {
		case !deletingInCluster && errors.IsNotFound(err):
			c.tl.Fatalf("%s %q was unexpectedly deleted from a member cluster", c.kind, qualifiedName)
		case deletingInCluster && err == nil:
			c.tl.Fatalf("%s %q was unexpectedly orphaned in a member cluster", c.kind, qualifiedName)
		case err != nil && !errors.IsNotFound(err):
			c.tl.Fatalf("Error while checking whether %s %q is %s in member clusters: %v", c.kind, qualifiedName, stateMsg, err)
		}
	}
}

// CheckPropagation checks propagation for the crud tester's clients
func (c *FederatedTypeCrudTester) CheckPropagation(obj pkgruntime.Object) {
	c.CheckPropagationForClients(obj, c.clusterClients, true)
}

// CheckPropagationForClients checks propagation for the provided clients
func (c *FederatedTypeCrudTester) CheckPropagationForClients(obj pkgruntime.Object, clusterClients []clientset.Interface, objExpected bool) {
	// non-federated
	qualifiedName := federatedtypes.NewQualifiedName(obj)

	// TODO(marun) need the names of clusters to be able to support a different object per cluster
	expectedObj := c.adapter.ObjectForCluster(obj, "fake-cluster")

	c.tl.Logf("Waiting for %s %q in %d clusters", c.kind, qualifiedName, len(clusterClients))
	for _, client := range clusterClients {
		err := c.waitForResource(client, expectedObj)
		switch {
		case err == wait.ErrWaitTimeout:
			if objExpected {
				c.tl.Fatalf("Timeout verifying %s %q in a member cluster: %v", c.kind, qualifiedName, err)
			}
		case err != nil:
			c.tl.Fatalf("Failed to verify %s %q in a member cluster: %v", c.kind, qualifiedName, err)
		case err == nil && !objExpected:
			c.tl.Fatalf("Found unexpected object %s %q in a member cluster: %v", c.kind, qualifiedName, err)
		}
	}
}

func (c *FederatedTypeCrudTester) waitForResource(client clientset.Interface, obj pkgruntime.Object) error {
	qualifiedName := federatedtypes.NewQualifiedName(obj)
	err := wait.PollImmediate(c.waitInterval, c.clusterWaitTimeout, func() (bool, error) {
		clusterObj, err := c.adapter.Get(client, qualifiedName)
		if err == nil && c.adapter.Equivalent(clusterObj, obj) {
			return true, nil
		}
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	})
	return err
}

func (c *FederatedTypeCrudTester) updateFedObject(obj pkgruntime.Object) (pkgruntime.Object, error) {
	err := wait.PollImmediate(c.waitInterval, wait.ForeverTestTimeout, func() (bool, error) {
		// Target the metadata for simplicity (it's type-agnostic)
		meta := c.adapter.FedObjectMeta(obj)
		if meta.Annotations == nil {
			meta.Annotations = make(map[string]string)
		}
		meta.Annotations[AnnotationTestFederationCrudUpdate] = "updated"

		_, err := c.adapter.FedUpdate(obj)
		if errors.IsConflict(err) {
			// The resource was updated by the federation controller.
			// Get the latest version and retry.
			qualifiedName := federatedtypes.NewQualifiedName(obj)
			obj, err = c.adapter.FedGet(qualifiedName)
			return false, err
		}
		// Be tolerant of a slow server
		if errors.IsServerTimeout(err) {
			return false, nil
		}
		return (err == nil), err
	})
	return obj, err
}
