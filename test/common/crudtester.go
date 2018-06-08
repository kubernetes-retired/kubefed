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

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/common"
	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/client/clientset_generated/clientset"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
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
	typeConfig       typeconfig.Interface
	comparisonHelper util.ComparisonHelper
	fedClient        clientset.Interface
	fedPool          dynamic.ClientPool
	kubePool         dynamic.ClientPool
	testClusters     map[string]TestCluster
	waitInterval     time.Duration
	// Federation operations will use wait.ForeverTestTimeout.  Any
	// operation that involves member clusters may take longer due to
	// propagation latency.
	clusterWaitTimeout time.Duration
}

type TestCluster struct {
	Client    util.ResourceClient
	IsPrimary bool
}

func NewFederatedTypeCrudTester(testLogger TestLogger, typeConfig typeconfig.Interface, fedConfig, kubeConfig *rest.Config, testClusters map[string]TestCluster, waitInterval, clusterWaitTimeout time.Duration) (*FederatedTypeCrudTester, error) {
	compare, err := util.NewComparisonHelper(typeConfig.GetComparisonField())
	if err != nil {
		return nil, err
	}

	return &FederatedTypeCrudTester{
		tl:                 testLogger,
		typeConfig:         typeConfig,
		comparisonHelper:   compare,
		fedClient:          clientset.NewForConfigOrDie(fedConfig),
		fedPool:            dynamic.NewDynamicClientPool(fedConfig),
		kubePool:           dynamic.NewDynamicClientPool(kubeConfig),
		testClusters:       testClusters,
		waitInterval:       waitInterval,
		clusterWaitTimeout: clusterWaitTimeout,
	}, nil
}

func (c *FederatedTypeCrudTester) CheckLifecycle(desiredTemplate, desiredPlacement, desiredOverride *unstructured.Unstructured) {
	template, placement, override := c.CheckCreate(desiredTemplate, desiredPlacement, desiredOverride)

	// TOOD(marun) Make sure these next steps work!!
	c.CheckUpdate(template, placement, override)
	c.CheckPlacementChange(template, placement, override)

	// Validate the golden path - removal of dependents
	orphanDependents := false
	// TODO(marun) need to delete placement and overrides
	c.CheckDelete(template, &orphanDependents)
}

func (c *FederatedTypeCrudTester) Create(desiredTemplate, desiredPlacement, desiredOverride *unstructured.Unstructured) (*unstructured.Unstructured, *unstructured.Unstructured, *unstructured.Unstructured) {
	template := c.createFedResource(c.typeConfig.GetTemplate(), desiredTemplate)

	// Test objects may use GenerateName.  Use the name of the
	// template resource for other resources.
	name := template.GetName()
	desiredPlacement.SetName(name)
	placement := c.createFedResource(c.typeConfig.GetPlacement(), desiredPlacement)

	var override *unstructured.Unstructured
	if overrideAPIResource := c.typeConfig.GetOverride(); overrideAPIResource != nil {
		desiredOverride.SetName(name)
		override = c.createFedResource(*overrideAPIResource, desiredOverride)
	}

	return template, placement, override
}

func (c *FederatedTypeCrudTester) createFedResource(apiResource metav1.APIResource, desiredObj *unstructured.Unstructured) *unstructured.Unstructured {
	namespace := desiredObj.GetNamespace()
	kind := apiResource.Kind
	resourceMsg := kind
	if len(namespace) > 0 {
		resourceMsg = fmt.Sprintf("%s in namespace %q", resourceMsg, namespace)
	}

	c.tl.Logf("Creating new %s", resourceMsg)

	client := c.fedResourceClient(apiResource)
	obj, err := client.Resources(namespace).Create(desiredObj)
	if err != nil {
		c.tl.Fatalf("Error creating %s: %v", resourceMsg, err)
	}

	qualifiedName := util.NewQualifiedName(obj)
	c.tl.Logf("Created new %s %q", kind, qualifiedName)

	return obj
}

func (c *FederatedTypeCrudTester) fedResourceClient(apiResource metav1.APIResource) util.ResourceClient {
	pool := c.fedPool
	// TODO(marun) Revisit this when federation of primary types to CRD
	if apiResource.Group != "federation.k8s.io" {
		pool = c.kubePool
	}
	client, err := util.NewResourceClient(pool, &apiResource)
	if err != nil {
		c.tl.Fatalf("Error creating resource client: %v", err)
	}
	return client
}

func (c *FederatedTypeCrudTester) CheckCreate(desiredTemplate, desiredPlacement, desiredOverride *unstructured.Unstructured) (*unstructured.Unstructured, *unstructured.Unstructured, *unstructured.Unstructured) {
	template, placement, override := c.Create(desiredTemplate, desiredPlacement, desiredOverride)

	msg := fmt.Sprintf("Resource versions for %s: template %q, placement %q",
		util.NewQualifiedName(template),
		template.GetResourceVersion(),
		placement.GetResourceVersion(),
	)
	if override != nil {
		msg = fmt.Sprintf("%s, override %q", msg, override.GetResourceVersion())
	}
	c.tl.Log(msg)

	c.CheckPropagation(template, placement, override)

	return template, placement, override
}

func (c *FederatedTypeCrudTester) CheckUpdate(template, placement, override *unstructured.Unstructured) {
	templateAPIResource := c.typeConfig.GetTemplate()
	templateKind := templateAPIResource.Kind
	qualifiedName := util.NewQualifiedName(template)

	var initialAnnotation string
	annotations := template.GetAnnotations()
	if annotations != nil {
		initialAnnotation = annotations[AnnotationTestFederationCrudUpdate]
	}

	c.tl.Logf("Updating %s %q", templateKind, qualifiedName)
	updatedTemplate, err := c.updateFedObject(templateAPIResource, template, func(template *unstructured.Unstructured) {
		// Target the metadata for simplicity (it's type-agnostic)
		annotations := template.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations[AnnotationTestFederationCrudUpdate] = "updated"
		template.SetAnnotations(annotations)
	})
	if err != nil {
		c.tl.Fatalf("Error updating %s %q: %v", templateKind, qualifiedName, err)
	}

	// updateFedObject is expected to have changed the value of the annotation
	updatedAnnotations := updatedTemplate.GetAnnotations()
	updatedAnnotation := updatedAnnotations[AnnotationTestFederationCrudUpdate]
	if updatedAnnotation == initialAnnotation {
		c.tl.Fatalf("%s %q not mutated", templateKind, qualifiedName)
	}

	c.CheckPropagation(updatedTemplate, placement, override)
}

// CheckPlacementChange verifies that a change in the list of clusters
// in a placement resource has the desired impact on member cluster
// state.
func (c *FederatedTypeCrudTester) CheckPlacementChange(template, placement, override *unstructured.Unstructured) {
	placementAPIResource := c.typeConfig.GetPlacement()
	placementKind := placementAPIResource.Kind
	qualifiedName := util.NewQualifiedName(placement)

	clusterNames := util.GetClusterNames(placement)

	// Skip if we're a namespace, we only have one cluster, and it's the
	// primary cluster. Skipping avoids deleting the namespace from the entire
	// federation by removing this single cluster from the placement, because
	// if deleted, this fails the next CheckDelete test.
	if c.typeConfig.GetTarget().Kind == util.NamespaceKind && len(clusterNames) == 1 &&
		clusterNames[0] == c.getPrimaryClusterName() {
		c.tl.Logf("Skipping %s update for %q due to single primary cluster",
			placementKind, qualifiedName)
		return
	}

	c.tl.Logf("Updating %s %q", placementKind, qualifiedName)
	updatedPlacement, err := c.updateFedObject(placementAPIResource, placement, func(placement *unstructured.Unstructured) {
		clusterNames := util.GetClusterNames(placement)
		clusterNames = c.deleteOneNonPrimaryCluster(clusterNames)
		err := util.SetClusterNames(placement, clusterNames)
		if err != nil {
			c.tl.Fatalf("Error setting cluster names for %s %q: %v", placementKind, qualifiedName, err)
		}
	})
	if err != nil {
		c.tl.Fatalf("Error updating %s %q: %v", placementKind, qualifiedName, err)
	}

	// updateFedObject is expected to have reduced the size of the cluster list
	updatedClusterNames := util.GetClusterNames(updatedPlacement)
	if len(updatedClusterNames) > len(clusterNames) {
		c.tl.Fatalf("%s %q not mutated", placementKind, qualifiedName)
	}

	c.CheckPropagation(template, updatedPlacement, override)
}

func (c *FederatedTypeCrudTester) CheckDelete(template *unstructured.Unstructured, orphanDependents *bool) {
	templateAPIResource := c.typeConfig.GetTemplate()
	templateKind := templateAPIResource.Kind
	qualifiedName := util.NewQualifiedName(template)
	name := qualifiedName.Name
	namespace := qualifiedName.Namespace

	client := c.fedResourceClient(templateAPIResource)

	// TODO(marun) delete related resources?
	c.tl.Logf("Deleting %s %q", templateKind, qualifiedName)
	err := client.Resources(namespace).Delete(name, &metav1.DeleteOptions{OrphanDependents: orphanDependents})
	if err != nil {
		c.tl.Fatalf("Error deleting %s %q: %v", templateKind, qualifiedName, err)
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
		_, err := client.Resources(namespace).Get(name, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
	if err != nil {
		c.tl.Fatalf("Error deleting %s %q: %v", templateKind, qualifiedName, err)
	}

	// Version tracker should also be removed. Wait for it to be removed.
	targetKind := c.typeConfig.GetTarget().Kind
	versionName := common.PropagatedVersionName(targetKind, name)
	err = wait.PollImmediate(c.waitInterval, waitTimeout, func() (bool, error) {
		if targetKind == util.NamespaceKind {
			_, err = c.fedClient.FederationV1alpha1().PropagatedVersions(name).Get(versionName, metav1.GetOptions{})
		} else {
			_, err = c.fedClient.FederationV1alpha1().PropagatedVersions(namespace).Get(versionName, metav1.GetOptions{})
		}
		if errors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
	if err != nil {
		c.tl.Fatalf("Expecting PropagatedVersion %s to be deleted", versionName)
	}

	var stateMsg string = "present"
	if deletingInCluster {
		stateMsg = "not present"
	}
	for _, testCluster := range c.testClusters {
		_, err := testCluster.Client.Resources(namespace).Get(name, metav1.GetOptions{})
		switch {
		case !deletingInCluster && errors.IsNotFound(err):
			c.tl.Fatalf("%s %q was unexpectedly deleted from a member cluster", targetKind, qualifiedName)
		case deletingInCluster && err == nil:
			c.tl.Fatalf("%s %q was unexpectedly orphaned in a member cluster", targetKind, qualifiedName)
		case err != nil && !errors.IsNotFound(err):
			c.tl.Fatalf("Error while checking whether %s %q is %s in member clusters: %v", targetKind, qualifiedName, stateMsg, err)
		}
	}
}

// CheckPropagation checks propagation for the crud tester's clients
func (c *FederatedTypeCrudTester) CheckPropagation(template, placement, override *unstructured.Unstructured) {
	targetKind := c.typeConfig.GetTarget().Kind
	qualifiedName := util.NewQualifiedName(template)

	clusterNames := util.GetClusterNames(placement)
	selectedClusters := sets.NewString(clusterNames...)

	// If we are a namespace, there is only one cluster, and the cluster is the
	// host cluster, then do not check for PropagatedVersion as it will never
	// be created.
	if targetKind == util.NamespaceKind && len(clusterNames) == 1 &&
		clusterNames[0] == c.getPrimaryClusterName() {
		return
	}

	version, err := c.waitForPropagatedVersion(template, placement, override)
	if err != nil {
		c.tl.Fatalf("Error waiting for propagated version for %s %q: %v", targetKind, qualifiedName, err)
	}

	// TODO(marun) run checks in parallel
	for clusterName, testCluster := range c.testClusters {
		objExpected := selectedClusters.Has(clusterName)

		operation := "to be deleted from"
		if objExpected {
			operation = "in"
		}
		c.tl.Logf("Waiting for %s %q %s cluster %q", targetKind, qualifiedName, operation, clusterName)

		expectedVersion := c.propagatedVersion(version, clusterName)
		if objExpected {
			if expectedVersion == "" {
				c.tl.Fatalf("Failed to determine expected version of %s %q in cluster %q.", targetKind, qualifiedName, clusterName)
			}
			err := c.waitForResource(testCluster.Client, qualifiedName, expectedVersion)
			switch {
			case err == wait.ErrWaitTimeout:
				c.tl.Fatalf("Timeout verifying %s %q in cluster %q: %v", targetKind, qualifiedName, clusterName, err)
			case err != nil:
				c.tl.Fatalf("Failed to verify %s %q in cluster %q: %v", targetKind, qualifiedName, clusterName, err)
			}
		} else {
			if expectedVersion != "" {
				c.tl.Fatalf("Expected resource version for %s %q in cluster %q to be removed", targetKind, qualifiedName, clusterName)
			}
			err := c.waitForResourceDeletion(testCluster.Client, qualifiedName)
			// Once resource deletion is complete, wait for the status to reflect the deletion

			switch {
			case err == wait.ErrWaitTimeout:
				if objExpected {
					c.tl.Fatalf("Timeout verifying deletion of %s %q in cluster %q: %v", targetKind, qualifiedName, clusterName, err)
				}
			case err != nil:
				c.tl.Fatalf("Failed to verify deletion of %s %q in cluster %q: %v", targetKind, qualifiedName, clusterName, err)
			}
		}
	}
}

func (c *FederatedTypeCrudTester) waitForResource(client util.ResourceClient, qualifiedName util.QualifiedName, expectedVersion string) error {
	err := wait.PollImmediate(c.waitInterval, c.clusterWaitTimeout, func() (bool, error) {
		clusterObj, err := client.Resources(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
		if err == nil && c.comparisonHelper.GetVersion(clusterObj) == expectedVersion {
			return true, nil
		}
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	})
	return err
}

func (c *FederatedTypeCrudTester) waitForResourceDeletion(client util.ResourceClient, qualifiedName util.QualifiedName) error {
	err := wait.PollImmediate(c.waitInterval, c.clusterWaitTimeout, func() (bool, error) {
		_, err := client.Resources(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
	return err
}

func (c *FederatedTypeCrudTester) updateFedObject(apiResource metav1.APIResource, obj *unstructured.Unstructured, mutateResourceFunc func(*unstructured.Unstructured)) (*unstructured.Unstructured, error) {
	client := c.fedResourceClient(apiResource)
	err := wait.PollImmediate(c.waitInterval, wait.ForeverTestTimeout, func() (bool, error) {
		mutateResourceFunc(obj)

		_, err := client.Resources(obj.GetNamespace()).Update(obj)
		if errors.IsConflict(err) {
			// The resource was updated by the federation controller.
			// Get the latest version and retry.
			obj, err = client.Resources(obj.GetNamespace()).Get(obj.GetName(), metav1.GetOptions{})
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

func (c *FederatedTypeCrudTester) waitForPropagatedVersion(template, placement, override *unstructured.Unstructured) (*fedv1a1.PropagatedVersion, error) {
	targetKind := c.typeConfig.GetTarget().Kind

	overrideVersion := ""
	if override != nil {
		overrideVersion = override.GetResourceVersion()
	}

	name := template.GetName()
	namespace := template.GetNamespace()

	versionName := common.PropagatedVersionName(targetKind, name)
	versionNamespace := namespace

	clusterNames := util.GetClusterNames(placement)
	selectedClusters := sets.NewString(clusterNames...)

	if targetKind == util.NamespaceKind {
		versionNamespace = name
		// Delete the primary cluster as it will never be included in
		// PropagatedVersion's list of cluster versions.
		selectedClusters.Delete(c.getPrimaryClusterName())
	}

	client := c.fedResourceClient(c.typeConfig.GetTemplate())

	var version *fedv1a1.PropagatedVersion
	err := wait.PollImmediate(c.waitInterval, c.clusterWaitTimeout, func() (bool, error) {
		var err error
		version, err = c.fedClient.FederationV1alpha1().PropagatedVersions(versionNamespace).Get(versionName, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		template, err := client.Resources(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if version.Status.TemplateVersion == template.GetResourceVersion() && version.Status.OverrideVersion == overrideVersion {
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
	if c.typeConfig.GetTarget().Kind == util.NamespaceKind && clusterName == c.getPrimaryClusterName() {
		return version.Status.TemplateVersion
	}

	for _, clusterVersion := range version.Status.ClusterVersions {
		if clusterVersion.ClusterName == clusterName {
			return clusterVersion.Version
		}
	}
	return ""
}
