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
	"k8s.io/apimachinery/pkg/util/sets"
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
	clusterClients map[string]clientset.Interface
	waitInterval   time.Duration
	// Federation operations will use wait.ForeverTestTimeout.  Any
	// operation that involves member clusters may take longer due to
	// propagation latency.
	clusterWaitTimeout time.Duration
}

func NewFederatedTypeCrudTester(testLogger TestLogger, adapter federatedtypes.FederatedTypeAdapter, clusterClients map[string]clientset.Interface, waitInterval, clusterWaitTimeout time.Duration) *FederatedTypeCrudTester {
	return &FederatedTypeCrudTester{
		tl:                 testLogger,
		adapter:            adapter,
		kind:               adapter.Template().Kind(),
		clusterClients:     clusterClients,
		waitInterval:       waitInterval,
		clusterWaitTimeout: clusterWaitTimeout,
	}
}

func (c *FederatedTypeCrudTester) CheckLifecycle(desiredTemplate, desiredPlacement, desiredOverride pkgruntime.Object) {
	template, placement, override := c.CheckCreate(desiredTemplate, desiredPlacement, desiredOverride)

	c.CheckUpdate(template, placement, override)
	c.CheckPlacementChange(template, placement, override)

	// Validate the golden path - removal of dependents
	orphanDependents := false
	// TODO(marun) need to delete placement
	c.CheckDelete(template, &orphanDependents)
}

func (c *FederatedTypeCrudTester) Create(desiredTemplate, desiredPlacement, desiredOverride pkgruntime.Object) (pkgruntime.Object, pkgruntime.Object, pkgruntime.Object) {
	templateAdapter := c.adapter.Template()
	template := c.createFedResource(templateAdapter, desiredTemplate)
	// Test objects may use GenerateName.  Use the name of the
	// template resource for other resources.
	name := templateAdapter.ObjectMeta(template).Name

	placementAdapter := c.adapter.Placement()
	c.setFedResourceName(placementAdapter, desiredPlacement, name)
	placement := c.createFedResource(placementAdapter, desiredPlacement)

	var override pkgruntime.Object
	overrideAdapter := c.adapter.Override()
	if overrideAdapter != nil {
		c.setFedResourceName(overrideAdapter, desiredOverride, name)
		override = c.createFedResource(overrideAdapter, desiredOverride)
	}

	return template, placement, override
}

func (c *FederatedTypeCrudTester) setFedResourceName(adapter federatedtypes.FedApiAdapter, obj pkgruntime.Object, name string) {
	meta := adapter.ObjectMeta(obj)
	meta.Name = name
	meta.GenerateName = ""
}

func (c *FederatedTypeCrudTester) createFedResource(adapter federatedtypes.FedApiAdapter, desiredObj pkgruntime.Object) pkgruntime.Object {
	namespace := adapter.ObjectMeta(desiredObj).Namespace
	kind := adapter.Kind()
	resourceMsg := kind
	if len(namespace) > 0 {
		resourceMsg = fmt.Sprintf("%s in namespace %q", resourceMsg, namespace)
	}

	c.tl.Logf("Creating new %s", resourceMsg)

	obj, err := adapter.Create(desiredObj)
	if err != nil {
		c.tl.Fatalf("Error creating %s: %v", resourceMsg, err)
	}

	qualifiedName := federatedtypes.NewQualifiedName(obj)
	c.tl.Logf("Created new %s %q", kind, qualifiedName)

	return obj
}

func (c *FederatedTypeCrudTester) CheckCreate(desiredTemplate, desiredPlacement, desiredOverride pkgruntime.Object) (pkgruntime.Object, pkgruntime.Object, pkgruntime.Object) {
	template, placement, override := c.Create(desiredTemplate, desiredPlacement, desiredOverride)

	c.CheckPropagation(template, placement, override)

	return template, placement, override
}

func (c *FederatedTypeCrudTester) CheckUpdate(template, placement, override pkgruntime.Object) {
	qualifiedName := federatedtypes.NewQualifiedName(template)

	adapter := c.adapter.Template()
	kind := adapter.Kind()

	var initialAnnotation string
	meta := adapter.ObjectMeta(template)
	if meta.Annotations != nil {
		initialAnnotation = meta.Annotations[AnnotationTestFederationCrudUpdate]
	}

	c.tl.Logf("Updating %s %q", kind, qualifiedName)
	updatedTemplate, err := c.updateFedObject(adapter, template, func(template pkgruntime.Object) {
		// Target the metadata for simplicity (it's type-agnostic)
		meta := adapter.ObjectMeta(template)
		if meta.Annotations == nil {
			meta.Annotations = make(map[string]string)
		}
		meta.Annotations[AnnotationTestFederationCrudUpdate] = "updated"
	})
	if err != nil {
		c.tl.Fatalf("Error updating %s %q: %v", kind, qualifiedName, err)
	}

	// updateFedObject is expected to have changed the value of the annotation
	meta = adapter.ObjectMeta(updatedTemplate)
	updatedAnnotation := meta.Annotations[AnnotationTestFederationCrudUpdate]
	if updatedAnnotation == initialAnnotation {
		c.tl.Fatalf("%s %q not mutated", kind, qualifiedName)
	}

	c.CheckPropagation(updatedTemplate, placement, override)
}

// CheckPlacementChange verifies that a change in the list of clusters
// in a placement resource has the desired impact on member cluster
// state.
func (c *FederatedTypeCrudTester) CheckPlacementChange(template, placement, override pkgruntime.Object) {
	qualifiedName := federatedtypes.NewQualifiedName(placement)

	adapter := c.adapter.Placement()
	kind := adapter.Kind()

	clusterNames := adapter.ClusterNames(placement)

	c.tl.Logf("Updating %s %q", kind, qualifiedName)
	updatedPlacement, err := c.updateFedObject(adapter, placement, func(placement pkgruntime.Object) {
		clusterNames := adapter.ClusterNames(placement)
		// Remove a cluster name
		clusterNames = append(clusterNames[:0], clusterNames[1:]...)
		adapter.SetClusterNames(placement, clusterNames)
	})
	if err != nil {
		c.tl.Fatalf("Error updating %s %q: %v", kind, qualifiedName, err)
	}

	// updateFedObject is expected to have reduced the size of the cluster list
	updatedClusterNames := adapter.ClusterNames(updatedPlacement)
	if len(updatedClusterNames) > len(clusterNames) {
		c.tl.Fatalf("%s %q not mutated", kind, qualifiedName)
	}

	c.CheckPropagation(template, updatedPlacement, override)
}

func (c *FederatedTypeCrudTester) CheckDelete(obj pkgruntime.Object, orphanDependents *bool) {
	qualifiedName := federatedtypes.NewQualifiedName(obj)

	templateAdapter := c.adapter.Template()

	// TODO(marun) delete related resources
	c.tl.Logf("Deleting %s %q", c.kind, qualifiedName)
	err := templateAdapter.Delete(qualifiedName, &metav1.DeleteOptions{OrphanDependents: orphanDependents})
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
		_, err := templateAdapter.Get(qualifiedName)
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
	targetAdapter := c.adapter.Target()
	for _, client := range c.clusterClients {
		_, err := targetAdapter.Get(client, qualifiedName)
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
func (c *FederatedTypeCrudTester) CheckPropagation(template, placement, override pkgruntime.Object) {
	qualifiedName := federatedtypes.NewQualifiedName(template)

	clusterNames := c.adapter.Placement().ClusterNames(placement)
	selectedClusters := sets.NewString(clusterNames...)

	// TODO(marun) run checks in parallel
	for clusterName, client := range c.clusterClients {
		objExpected := selectedClusters.Has(clusterName)

		operation := "to be deleted from"
		if objExpected {
			operation = "in"
		}
		c.tl.Logf("Waiting for %s %q %s cluster %q", c.kind, qualifiedName, operation, clusterName)

		if objExpected {
			expectedObj := c.adapter.ObjectForCluster(template, override, clusterName)
			err := c.waitForResource(client, expectedObj)
			switch {
			case err == wait.ErrWaitTimeout:
				c.tl.Fatalf("Timeout verifying %s %q in cluster %q: %v", c.kind, qualifiedName, clusterName, err)
			case err != nil:
				c.tl.Fatalf("Failed to verify %s %q in cluster %q: %v", c.kind, qualifiedName, clusterName, err)
			}
		} else {
			err := c.waitForResourceDeletion(client, qualifiedName)
			switch {
			case err == wait.ErrWaitTimeout:
				if objExpected {
					c.tl.Fatalf("Timeout verifying deletion of %s %q in cluster %q: %v", c.kind, qualifiedName, clusterName, err)
				}
			case err != nil:
				c.tl.Fatalf("Failed to verify deletion of %s %q in cluster %q: %v", c.kind, qualifiedName, clusterName, err)
			}

		}
	}
}

func (c *FederatedTypeCrudTester) waitForResource(client clientset.Interface, expectedObj pkgruntime.Object) error {
	qualifiedName := federatedtypes.NewQualifiedName(expectedObj)
	targetAdapter := c.adapter.Target()
	err := wait.PollImmediate(c.waitInterval, c.clusterWaitTimeout, func() (bool, error) {
		clusterObj, err := targetAdapter.Get(client, qualifiedName)
		if err == nil && targetAdapter.Equivalent(clusterObj, expectedObj) {
			return true, nil
		}
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	})
	return err
}

func (c *FederatedTypeCrudTester) waitForResourceDeletion(client clientset.Interface, qualifiedName federatedtypes.QualifiedName) error {
	err := wait.PollImmediate(c.waitInterval, c.clusterWaitTimeout, func() (bool, error) {
		_, err := c.adapter.Target().Get(client, qualifiedName)
		if errors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
	return err
}

func (c *FederatedTypeCrudTester) updateFedObject(adapter federatedtypes.FedApiAdapter, obj pkgruntime.Object, mutateResourceFunc func(pkgruntime.Object)) (pkgruntime.Object, error) {
	err := wait.PollImmediate(c.waitInterval, wait.ForeverTestTimeout, func() (bool, error) {
		mutateResourceFunc(obj)

		_, err := adapter.Update(obj)
		if errors.IsConflict(err) {
			// The resource was updated by the federation controller.
			// Get the latest version and retry.
			qualifiedName := federatedtypes.NewQualifiedName(obj)
			obj, err = adapter.Get(qualifiedName)
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
