/*
Copyright 2018 The Federation v2 Authors.

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

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/federation/common"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/federation/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/federatedtypes"
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
	tl               TestLogger
	adapter          federatedtypes.FederatedTypeAdapter
	kind             string
	comparisonHelper util.ComparisonHelper
	testClusters     map[string]TestCluster
	waitInterval     time.Duration
	// Federation operations will use wait.ForeverTestTimeout.  Any
	// operation that involves member clusters may take longer due to
	// propagation latency.
	clusterWaitTimeout time.Duration
}

type TestCluster struct {
	Client    clientset.Interface
	IsPrimary bool
}

func NewFederatedTypeCrudTester(testLogger TestLogger, adapter federatedtypes.FederatedTypeAdapter, testClusters map[string]TestCluster, waitInterval, clusterWaitTimeout time.Duration) (*FederatedTypeCrudTester, error) {
	compare, err := util.NewComparisonHelper(adapter.Target().VersionCompareType())
	if err != nil {
		return nil, err
	}

	return &FederatedTypeCrudTester{
		tl:                 testLogger,
		adapter:            adapter,
		kind:               adapter.Template().Kind(),
		comparisonHelper:   compare,
		testClusters:       testClusters,
		waitInterval:       waitInterval,
		clusterWaitTimeout: clusterWaitTimeout,
	}, nil
}

func (c *FederatedTypeCrudTester) CheckLifecycle(desiredTemplate, desiredPlacement, desiredOverride pkgruntime.Object) {
	template, placement, override := c.CheckCreate(desiredTemplate, desiredPlacement, desiredOverride)

	// TOOD(marun) Make sure these next steps work!!
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
	templateMeta := util.MetaAccessor(template)

	// Test objects may use GenerateName.  Use the name of the
	// template resource for other resources.
	name := templateMeta.GetName()
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
	meta := util.MetaAccessor(obj)
	meta.SetName(name)
	meta.SetGenerateName("")
}

func (c *FederatedTypeCrudTester) createFedResource(adapter federatedtypes.FedApiAdapter, desiredObj pkgruntime.Object) pkgruntime.Object {
	namespace := util.MetaAccessor(desiredObj).GetNamespace()
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

	overrideAdapter := c.adapter.Override()
	if overrideAdapter != nil {
		c.tl.Logf("Resource versions for %s: template %q, placement %q, override %q",
			federatedtypes.NewQualifiedName(template),
			util.MetaAccessor(template).GetResourceVersion(),
			util.MetaAccessor(placement).GetResourceVersion(),
			util.MetaAccessor(override).GetResourceVersion())
	} else {
		c.tl.Logf("Resource versions for %s: template %q, placement %q",
			federatedtypes.NewQualifiedName(template),
			util.MetaAccessor(template).GetResourceVersion(),
			util.MetaAccessor(placement).GetResourceVersion())
	}

	c.CheckPropagation(template, placement, override)

	return template, placement, override
}

func (c *FederatedTypeCrudTester) CheckUpdate(template, placement, override pkgruntime.Object) {
	qualifiedName := federatedtypes.NewQualifiedName(template)

	adapter := c.adapter.Template()

	var initialAnnotation string
	meta := util.MetaAccessor(template)
	annotations := meta.GetAnnotations()
	if annotations != nil {
		initialAnnotation = annotations[AnnotationTestFederationCrudUpdate]
	}

	c.tl.Logf("Updating %s %q", c.kind, qualifiedName)
	updatedTemplate, err := c.updateFedObject(adapter, template, func(template pkgruntime.Object) {
		// Target the metadata for simplicity (it's type-agnostic)
		meta := util.MetaAccessor(template)
		annotations := meta.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations[AnnotationTestFederationCrudUpdate] = "updated"
		meta.SetAnnotations(annotations)
	})
	if err != nil {
		c.tl.Fatalf("Error updating %s %q: %v", c.kind, qualifiedName, err)
	}

	// updateFedObject is expected to have changed the value of the annotation
	meta = util.MetaAccessor(updatedTemplate)
	updatedAnnotations := meta.GetAnnotations()
	updatedAnnotation := updatedAnnotations[AnnotationTestFederationCrudUpdate]
	if updatedAnnotation == initialAnnotation {
		c.tl.Fatalf("%s %q not mutated", c.kind, qualifiedName)
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

	// Skip if we're a namespace, we only have one cluster, and it's the
	// primary cluster. Skipping avoids deleting the namespace from the entire
	// federation by removing this single cluster from the placement, because
	// if deleted, this fails the next CheckDelete test.
	if kind == federatedtypes.FederatedNamespacePlacementKind && len(clusterNames) == 1 &&
		clusterNames[0] == c.getPrimaryClusterName() {
		c.tl.Logf("Skipping %s placement update for %q due to single primary cluster",
			kind, qualifiedName)
		return
	}

	c.tl.Logf("Updating %s %q", kind, qualifiedName)
	updatedPlacement, err := c.updateFedObject(adapter, placement, func(placement pkgruntime.Object) {
		clusterNames := adapter.ClusterNames(placement)
		clusterNames = c.deleteOneNonPrimaryCluster(clusterNames)
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

	// Version tracker should also be removed
	versionName := c.versionName(qualifiedName.Name)
	if federatedtypes.IsNamespaceKind(c.kind) {
		_, err = c.adapter.FedClient().FederationV1alpha1().PropagatedVersions(qualifiedName.Name).Get(versionName, metav1.GetOptions{})
	} else {
		_, err = c.adapter.FedClient().FederationV1alpha1().PropagatedVersions(qualifiedName.Namespace).Get(versionName, metav1.GetOptions{})
	}
	if !errors.IsNotFound(err) {
		c.tl.Fatalf("Expecting PropagatedVersion %s to be deleted", versionName)
	}

	var stateMsg string = "present"
	if deletingInCluster {
		stateMsg = "not present"
	}
	targetAdapter := c.adapter.Target()
	for _, testCluster := range c.testClusters {
		_, err := targetAdapter.Get(testCluster.Client, qualifiedName)
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

	// If we are a namespace, there is only one cluster, and the cluster is the
	// host cluster, then do not check for PropagatedVersion as it will never
	// be created.
	if federatedtypes.IsNamespaceKind(c.kind) && len(clusterNames) == 1 &&
		clusterNames[0] == c.getPrimaryClusterName() {
		return
	}

	version, err := c.waitForPropagatedVersion(template, placement, override)
	if err != nil {
		c.tl.Fatalf("Error waiting for propagated version for %s %q: %v", c.kind, qualifiedName, err)
	}

	// TODO(marun) run checks in parallel
	for clusterName, testCluster := range c.testClusters {
		objExpected := selectedClusters.Has(clusterName)

		operation := "to be deleted from"
		if objExpected {
			operation = "in"
		}
		c.tl.Logf("Waiting for %s %q %s cluster %q", c.kind, qualifiedName, operation, clusterName)

		expectedVersion := c.propagatedVersion(version, clusterName)
		if objExpected {
			if expectedVersion == "" {
				c.tl.Fatalf("Failed to determine expected resource version of %s %q in cluster %q.", c.kind, qualifiedName, clusterName)
			}
			err := c.waitForResource(testCluster.Client, qualifiedName, expectedVersion)
			switch {
			case err == wait.ErrWaitTimeout:
				c.tl.Fatalf("Timeout verifying %s %q in cluster %q: %v", c.kind, qualifiedName, clusterName, err)
			case err != nil:
				c.tl.Fatalf("Failed to verify %s %q in cluster %q: %v", c.kind, qualifiedName, clusterName, err)
			}
		} else {
			if expectedVersion != "" {
				c.tl.Fatalf("Expected resource version for %s %q in cluster %q to be removed", c.kind, qualifiedName, clusterName)
			}
			err := c.waitForResourceDeletion(testCluster.Client, qualifiedName)
			// Once resource deletion is complete, wait for the status to reflect the deletion

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

func (c *FederatedTypeCrudTester) waitForResource(client clientset.Interface, qualifiedName federatedtypes.QualifiedName, expectedVersion string) error {
	targetAdapter := c.adapter.Target()

	err := wait.PollImmediate(c.waitInterval, c.clusterWaitTimeout, func() (bool, error) {
		clusterObj, err := targetAdapter.Get(client, qualifiedName)
		targetMeta := util.MetaAccessor(clusterObj)
		if err == nil && c.comparisonHelper.GetVersion(targetMeta) == expectedVersion {
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

func (c *FederatedTypeCrudTester) waitForPropagatedVersion(template, placement, override pkgruntime.Object) (*fedv1a1.PropagatedVersion, error) {
	templateMeta := util.MetaAccessor(template)
	templateQualifiedName := federatedtypes.NewQualifiedName(template)

	overrideVersion := ""
	overrideAdapter := c.adapter.Override()
	if overrideAdapter != nil {
		overrideVersion = util.MetaAccessor(override).GetResourceVersion()
	}

	client := c.adapter.FedClient()
	namespace := templateMeta.GetNamespace()
	name := c.versionName(templateMeta.GetName())

	clusterNames := c.adapter.Placement().ClusterNames(placement)
	selectedClusters := sets.NewString(clusterNames...)
	if federatedtypes.IsNamespaceKind(c.kind) {
		// Delete the primary cluster as it will never be included in
		// PropagatedVersion's list of cluster versions.
		selectedClusters.Delete(c.getPrimaryClusterName())
	}

	var version *fedv1a1.PropagatedVersion
	err := wait.PollImmediate(c.waitInterval, c.clusterWaitTimeout, func() (bool, error) {
		var err error
		if federatedtypes.IsNamespaceKind(c.kind) {
			version, err = client.FederationV1alpha1().PropagatedVersions(templateMeta.GetName()).Get(name, metav1.GetOptions{})
		} else {
			version, err = client.FederationV1alpha1().PropagatedVersions(namespace).Get(name, metav1.GetOptions{})
		}
		if errors.IsNotFound(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		template, err := c.adapter.Template().Get(templateQualifiedName)
		if err != nil {
			return false, err
		}
		templateVersion := util.MetaAccessor(template).GetResourceVersion()
		if version.Status.TemplateVersion == templateVersion && version.Status.OverrideVersion == overrideVersion {
			// Check that the list of clusters propagated to matches the list of selected clusters
			propagatedClusters := sets.String{}
			for _, clusterVersion := range version.Status.ClusterVersions {
				propagatedClusters.Insert(clusterVersion.ClusterName)
			}
			if propagatedClusters.Equal(selectedClusters) {
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	return version, nil
}

func (c *FederatedTypeCrudTester) versionName(resourceName string) string {
	targetKind := c.adapter.Target().Kind()
	return common.PropagatedVersionName(targetKind, resourceName)
}

func (c *FederatedTypeCrudTester) getPrimaryClusterName() string {
	for name, testCluster := range c.testClusters {
		if testCluster.IsPrimary {
			return name
		}
	}
	return ""
}

func (c *FederatedTypeCrudTester) deleteOneNonPrimaryCluster(clusterNames []string) []string {
	primaryClusterName := c.getPrimaryClusterName()

	for i, name := range clusterNames {
		if name == primaryClusterName {
			continue
		} else {
			clusterNames = append(clusterNames[:i], clusterNames[i+1:]...)
			break
		}
	}

	return clusterNames
}

func (c *FederatedTypeCrudTester) propagatedVersion(version *fedv1a1.PropagatedVersion, clusterName string) string {
	// For namespaces, since we do not store the primary cluster's namespace
	// version in PropagatedVersion's ClusterVersions slice, grab it from the
	// TemplateVersion field instead.
	if federatedtypes.IsNamespaceKind(c.kind) && clusterName == c.getPrimaryClusterName() {
		return version.Status.TemplateVersion
	}

	for _, clusterVersion := range version.Status.ClusterVersions {
		if clusterVersion.ClusterName == clusterName {
			return clusterVersion.Version
		}
	}
	return ""
}
