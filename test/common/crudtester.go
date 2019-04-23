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
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/common"
	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	genericclient "github.com/kubernetes-sigs/federation-v2/pkg/client/generic"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/sync"
	versionmanager "github.com/kubernetes-sigs/federation-v2/pkg/controller/sync/version"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util/deletionhelper"
	"github.com/kubernetes-sigs/federation-v2/pkg/kubefed2/federate"
)

// FederatedTypeCrudTester exercises Create/Read/Update/Delete operations for
// federated types via the Federation API and validates that the
// results of those operations are propagated to clusters that are
// members of a federation.
type FederatedTypeCrudTester struct {
	tl                TestLogger
	typeConfig        typeconfig.Interface
	targetIsNamespace bool
	client            genericclient.Client
	kubeConfig        *rest.Config
	testClusters      map[string]TestCluster
	waitInterval      time.Duration
	// Federation operations will use wait.ForeverTestTimeout.  Any
	// operation that involves member clusters may take longer due to
	// propagation latency.
	clusterWaitTimeout time.Duration
}

type TestClusterConfig struct {
	Config    *rest.Config
	IsPrimary bool
}

type TestCluster struct {
	TestClusterConfig
	Client util.ResourceClient
}

func NewFederatedTypeCrudTester(testLogger TestLogger, typeConfig typeconfig.Interface, kubeConfig *rest.Config, testClusters map[string]TestCluster, waitInterval, clusterWaitTimeout time.Duration) (*FederatedTypeCrudTester, error) {
	return &FederatedTypeCrudTester{
		tl:                 testLogger,
		typeConfig:         typeConfig,
		targetIsNamespace:  typeConfig.GetTarget().Kind == util.NamespaceKind,
		client:             genericclient.NewForConfigOrDie(kubeConfig),
		kubeConfig:         kubeConfig,
		testClusters:       testClusters,
		waitInterval:       waitInterval,
		clusterWaitTimeout: clusterWaitTimeout,
	}, nil
}

func (c *FederatedTypeCrudTester) CheckLifecycle(targetObject *unstructured.Unstructured, overrides []interface{}) {
	fedObject := c.CheckCreate(targetObject, overrides)

	c.CheckStatusCreated(util.NewQualifiedName(fedObject))

	c.CheckUpdate(fedObject)
	c.CheckPlacementChange(fedObject)

	// Validate the golden path - removal of resources from member
	// clusters.  A test of orphaning is performed in the
	// namespace-scoped crd crud test.
	c.CheckDelete(fedObject, false)
}

func (c *FederatedTypeCrudTester) Create(targetObject *unstructured.Unstructured, overrides []interface{}) *unstructured.Unstructured {
	qualifiedName := util.NewQualifiedName(targetObject)
	kind := c.typeConfig.GetTarget().Kind
	fedKind := c.typeConfig.GetFederatedType().Kind
	fedObject, err := federate.FederatedResourceFromTargetResource(c.typeConfig, targetObject)
	if err != nil {
		c.tl.Fatalf("Error obtaining %s from %s %q: %v", fedKind, kind, qualifiedName, err)
	}

	fedObject, err = c.setAdditionalTestData(fedObject, overrides, targetObject.GetGenerateName())
	if err != nil {
		c.tl.Fatalf("Error setting overrides and placement on %s %q: %v", fedKind, qualifiedName, err)
	}

	return c.createResource(c.typeConfig.GetFederatedType(), fedObject)
}

func (c *FederatedTypeCrudTester) createResource(apiResource metav1.APIResource, desiredObj *unstructured.Unstructured) *unstructured.Unstructured {
	createdObj, err := CreateResource(c.kubeConfig, apiResource, desiredObj)
	if err != nil {
		c.tl.Fatalf("Error creating resource: %v", err)
	}

	qualifiedName := util.NewQualifiedName(createdObj)
	c.tl.Logf("Created new %s %q", apiResource.Kind, qualifiedName)

	return createdObj
}

func (c *FederatedTypeCrudTester) resourceClient(apiResource metav1.APIResource) util.ResourceClient {
	client, err := util.NewResourceClient(c.kubeConfig, &apiResource)
	if err != nil {
		c.tl.Fatalf("Error creating resource client: %v", err)
	}
	return client
}

func (c *FederatedTypeCrudTester) CheckCreate(targetObject *unstructured.Unstructured, overrides []interface{}) *unstructured.Unstructured {
	fedObject := c.Create(targetObject, overrides)

	c.CheckPropagation(fedObject)
	return fedObject
}

// AdditionalTestData additionally sets fixture overrides and placement clusternames into federated object
func (c *FederatedTypeCrudTester) setAdditionalTestData(fedObject *unstructured.Unstructured, overrides []interface{}, generateName string) (*unstructured.Unstructured, error) {
	fedKind := c.typeConfig.GetFederatedType().Kind
	qualifiedName := util.NewQualifiedName(fedObject)

	if overrides != nil {
		err := unstructured.SetNestedField(fedObject.Object, overrides, util.SpecField, util.OverridesField)
		if err != nil {
			c.tl.Fatalf("Error updating overrides in %s %q: %v", fedKind, qualifiedName, err)
		}
	}
	clusterNames := []string{}
	for name := range c.testClusters {
		clusterNames = append(clusterNames, name)
	}
	err := util.SetClusterNames(fedObject, clusterNames)
	if err != nil {
		c.tl.Fatalf("Error setting cluster names in %s %q: %v", fedKind, qualifiedName, err)
	}
	fedObject.SetGenerateName(generateName)

	return fedObject, err
}

func (c *FederatedTypeCrudTester) CheckUpdate(fedObject *unstructured.Unstructured) {
	apiResource := c.typeConfig.GetFederatedType()
	kind := apiResource.Kind
	qualifiedName := util.NewQualifiedName(fedObject)

	key := "metadata.labels"
	value := map[string]interface{}{
		"crudtester-operation":           "update",
		util.ManagedByFederationLabelKey: util.ManagedByFederationLabelValue,
	}

	c.tl.Logf("Updating %s %q", kind, qualifiedName)
	updatedFedObject, err := c.updateObject(apiResource, fedObject, func(obj *unstructured.Unstructured) {
		overrides, err := util.GetOverrides(obj)
		if err != nil {
			c.tl.Fatalf("Error retrieving overrides for %s %q: %v", kind, qualifiedName, err)
		}
		for clusterName := range c.testClusters {
			clusterOverrides, ok := overrides[clusterName]
			if !ok {
				clusterOverrides = make(util.ClusterOverridesMap)
				overrides[clusterName] = clusterOverrides
			}
			_, ok = clusterOverrides[key]
			if ok {
				c.tl.Fatalf("An override for %q already exists for cluster %q", key, clusterName)
			}
			clusterOverrides[key] = value
		}

		if err := util.SetOverrides(obj, overrides); err != nil {
			c.tl.Fatalf("Unexpected error: %v", err)
		}
	})
	if err != nil {
		c.tl.Fatalf("Error updating %s %q: %v", kind, qualifiedName, err)
	}

	c.CheckPropagation(updatedFedObject)
}

// CheckPlacementChange verifies that a change in the list of clusters
// in a placement resource has the desired impact on member cluster
// state.
func (c *FederatedTypeCrudTester) CheckPlacementChange(fedObject *unstructured.Unstructured) {
	apiResource := c.typeConfig.GetFederatedType()
	kind := apiResource.Kind
	qualifiedName := util.NewQualifiedName(fedObject)

	// Any cluster can be removed for non-namespace targets.
	clusterNameToRemove := ""
	if c.targetIsNamespace {
		// The primary cluster should be removed for namespace targets.  This
		// will ensure that unlabeling is validated.
		clusterNameToRemove = c.getPrimaryClusterName()
	}

	c.tl.Logf("Updating %s %q", kind, qualifiedName)
	updatedFedObject, err := c.updateObject(apiResource, fedObject, func(obj *unstructured.Unstructured) {
		clusterNames, err := util.GetClusterNames(obj)
		if err != nil {
			c.tl.Fatalf("Error retrieving cluster names: %v", err)
		}
		updatedClusterNames := c.removeOneClusterName(clusterNames, clusterNameToRemove)
		if len(updatedClusterNames) != len(clusterNames)-1 {
			// This test depends on a cluster name being removed from
			// the placement resource to validate that the sync
			// controller will then remove the resource from the
			// cluster whose name was removed.
			c.tl.Fatalf("Expected %d cluster names, got %d", len(clusterNames)-1, len(updatedClusterNames))
		}
		err = util.SetClusterNames(obj, updatedClusterNames)
		if err != nil {
			c.tl.Fatalf("Error setting cluster names for %s %q: %v", kind, qualifiedName, err)
		}
	})
	if err != nil {
		c.tl.Fatalf("Error updating %s %q: %v", kind, qualifiedName, err)
	}

	c.CheckPropagation(updatedFedObject)
}

func (c *FederatedTypeCrudTester) CheckDelete(fedObject *unstructured.Unstructured, orphanDependents bool) {
	apiResource := c.typeConfig.GetFederatedType()
	federatedKind := apiResource.Kind
	qualifiedName := util.NewQualifiedName(fedObject)
	name := qualifiedName.Name
	namespace := qualifiedName.Namespace

	client := c.resourceClient(apiResource)

	if orphanDependents {
		orphanKey := deletionhelper.OrphanManagedResources
		err := wait.PollImmediate(c.waitInterval, wait.ForeverTestTimeout, func() (bool, error) {
			var err error
			if fedObject == nil {
				fedObject, err = client.Resources(namespace).Get(name, metav1.GetOptions{})
				if err != nil {
					c.tl.Logf("Error retrieving %s %q to add the %q annotation: %v", federatedKind, qualifiedName, orphanKey, err)
					return false, nil
				}
			}
			// Set the orphan annotation if necessary
			annotations := fedObject.GetAnnotations()
			if annotations == nil {
				annotations = make(map[string]string)
			}
			if annotations[orphanKey] == "true" {
				return true, nil
			}
			annotations[orphanKey] = "true"
			fedObject.SetAnnotations(annotations)
			fedObject, err = client.Resources(namespace).Update(fedObject, metav1.UpdateOptions{})
			if err == nil {
				return true, nil
			}
			c.tl.Logf("Error updating %s %q to include the %q annotation: %v", federatedKind, qualifiedName, orphanKey, err)
			// Clear fedObject to ensure its attempted retrieval in the next iteration
			fedObject = nil
			return false, nil
		})
		if err != nil {
			c.tl.Fatalf("Timed out trying to add %q annotation to %s %q", orphanKey, federatedKind, qualifiedName)
		}
	}

	c.tl.Logf("Deleting %s %q", federatedKind, qualifiedName)
	err := client.Resources(namespace).Delete(name, nil)
	if err != nil {
		c.tl.Fatalf("Error deleting %s %q: %v", federatedKind, qualifiedName, err)
	}

	deletingInCluster := !orphanDependents

	waitTimeout := wait.ForeverTestTimeout
	if deletingInCluster {
		// May need extra time to delete both federation and cluster resources
		waitTimeout = c.clusterWaitTimeout
	}

	// Wait for deletion.  The federation resource will only be removed once orphan deletion has been
	// completed or deemed unnecessary.
	err = wait.PollImmediate(c.waitInterval, waitTimeout, func() (bool, error) {
		_, err := client.Resources(namespace).Get(name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
	if err != nil {
		c.tl.Fatalf("Error deleting %s %q: %v", federatedKind, qualifiedName, err)
	}

	if c.targetIsNamespace {
		namespace = ""
		qualifiedName = util.QualifiedName{Name: name}
	}

	targetKind := c.typeConfig.GetTarget().Kind

	// TODO(marun) Consider using informer to detect expected deletion state.
	var stateMsg string = "present"
	if deletingInCluster {
		stateMsg = "not present"
	}
	for clusterName, testCluster := range c.testClusters {
		err = wait.PollImmediate(c.waitInterval, waitTimeout, func() (bool, error) {
			_, err := testCluster.Client.Resources(namespace).Get(name, metav1.GetOptions{})
			switch {
			case !deletingInCluster && apierrors.IsNotFound(err):
				return false, errors.Errorf("%s %q was unexpectedly deleted from cluster %q", targetKind, qualifiedName, clusterName)
			case deletingInCluster && err == nil:
				// The namespace in the host cluster should not be removed.
				if c.targetIsNamespace && clusterName == c.getPrimaryClusterName() {
					// TODO(marun) Validate removal of the managed label once
					// the deletion helper is updated to support the capability.
					return true, nil
				}
				// Continue checking for deletion
				return false, nil
			case err != nil && !apierrors.IsNotFound(err):
				c.tl.Errorf("Error while checking whether %s %q is %s in cluster %q: %v", targetKind, qualifiedName, stateMsg, clusterName, err)
				// This error may be recoverable
				return false, nil
			default:
				return true, nil
			}
		})
		if err != nil {
			c.tl.Fatalf("Failed to confirm whether %s %q is %s in cluster: %v", targetKind, qualifiedName, stateMsg, clusterName, err)
		}
	}
}

// CheckPropagation checks propagation for the crud tester's clients
func (c *FederatedTypeCrudTester) CheckPropagation(fedObject *unstructured.Unstructured) {
	federatedKind := c.typeConfig.GetFederatedType().Kind
	qualifiedName := util.NewQualifiedName(fedObject)

	clusterNames, err := util.GetClusterNames(fedObject)
	if err != nil {
		c.tl.Fatalf("Error retrieving cluster names for %s %q: %v", federatedKind, qualifiedName, err)
	}
	selectedClusters := sets.NewString(clusterNames...)

	templateVersion, err := sync.GetTemplateHash(fedObject.Object)
	if err != nil {
		c.tl.Fatalf("Error computing template hash for %s %q: %v", federatedKind, qualifiedName, err)
	}

	overrideVersion, err := sync.GetOverrideHash(fedObject)
	if err != nil {
		c.tl.Fatalf("Error computing override hash for %s %q: %v", federatedKind, qualifiedName, err)
	}

	overridesMap, err := util.GetOverrides(fedObject)
	if err != nil {
		c.tl.Fatalf("Error reading cluster overrides for %s %q: %v", federatedKind, qualifiedName, err)
	}

	targetKind := c.typeConfig.GetTarget().Kind

	// TODO(marun) run checks in parallel
	primaryClusterName := c.getPrimaryClusterName()
	for clusterName, testCluster := range c.testClusters {
		objExpected := selectedClusters.Has(clusterName)

		operation := "to be deleted from"
		if objExpected {
			operation = "in"
		}
		c.tl.Logf("Waiting for %s %q %s cluster %q", targetKind, qualifiedName, operation, clusterName)

		if objExpected {
			err := c.waitForResource(testCluster.Client, qualifiedName, overridesMap[clusterName], func() string {
				version, _ := c.expectedVersion(qualifiedName, templateVersion, overrideVersion, clusterName)
				return version
			})
			switch {
			case err == wait.ErrWaitTimeout:
				c.tl.Fatalf("Timeout verifying %s %q in cluster %q: %v", targetKind, qualifiedName, clusterName, err)
			case err != nil:
				c.tl.Fatalf("Failed to verify %s %q in cluster %q: %v", targetKind, qualifiedName, clusterName, err)
			}
		} else if c.targetIsNamespace && clusterName == primaryClusterName {
			c.checkHostNamespaceUnlabeled(testCluster.Client, qualifiedName, targetKind, clusterName)
		} else {
			err := c.waitForResourceDeletion(testCluster.Client, qualifiedName, func() bool {
				version, ok := c.expectedVersion(qualifiedName, templateVersion, overrideVersion, clusterName)
				return version == "" && ok
			})
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

func (c *FederatedTypeCrudTester) checkHostNamespaceUnlabeled(client util.ResourceClient, qualifiedName util.QualifiedName, targetKind, clusterName string) {
	// A namespace in the host cluster should end up unlabeled instead of
	// deleted when it is not targeted by placement.

	err := wait.PollImmediate(c.waitInterval, c.clusterWaitTimeout, func() (bool, error) {
		hostNamespace, err := client.Resources("").Get(qualifiedName.Name, metav1.GetOptions{})
		if err != nil {
			c.tl.Errorf("Error retrieving %s %q in host cluster %q: %v", targetKind, qualifiedName, clusterName, err)
			return false, nil
		}
		// Validate that the namespace is without the managed label
		return !util.HasManagedLabel(hostNamespace), nil
	})
	if err != nil {
		c.tl.Fatalf("Timeout verifying removal of managed label from %s %q in host cluster %q: %v", targetKind, qualifiedName, clusterName, err)
	}
}

func (c *FederatedTypeCrudTester) waitForResource(client util.ResourceClient, qualifiedName util.QualifiedName, expectedOverrides util.ClusterOverridesMap, expectedVersionFunc func() string) error {
	err := wait.PollImmediate(c.waitInterval, c.clusterWaitTimeout, func() (bool, error) {
		expectedVersion := expectedVersionFunc()
		if len(expectedVersion) == 0 {
			return false, nil
		}

		clusterObj, err := client.Resources(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
		if err == nil && util.ObjectVersion(clusterObj) == expectedVersion {
			// Validate that the resource has been labeled properly,
			// indicating creation or adoption by the sync controller.  This
			// labeling also ensures that the federated informer will be able
			// to cache the resource.
			if !util.HasManagedLabel(clusterObj) {
				c.tl.Errorf("Expected resource to be labeled with %q", fmt.Sprintf("%s: %s", util.ManagedByFederationLabelKey, util.ManagedByFederationLabelValue))
				return false, nil
			}

			// Validate that the expected override was applied
			if len(expectedOverrides) > 0 {
				for path, expectedValue := range expectedOverrides {
					pathEntries := strings.Split(path, ".")
					value, ok, err := unstructured.NestedFieldCopy(clusterObj.Object, pathEntries...)
					if err != nil {
						c.tl.Fatalf("Error retrieving overridden path: %v", err)
					}
					if !ok {
						c.tl.Fatalf("Missing overridden path %s", path)
					}
					// Because the result of deserializing an override field differs from the value
					// retrieved by NestedFieldCopy, reflection is not able to accurately compare
					// numeric types that should otherwise be equal. For example, an override value
					// of 2 is deserialized as %!q(float64=2), but the same value retrieved by
					// NestedFieldCopy would be '\x02'.  String conversion is a hacky way of working
					// around this problem, with a fallback to reflection for non-numeric types.
					if fmt.Sprintf("%v", expectedValue) != fmt.Sprintf("%v", value) && !reflect.DeepEqual(expectedValue, value) {
						c.tl.Errorf("Expected field %s to be %q, got %q", path, expectedValue, value)
						return false, nil
					}
				}
			}

			return true, nil
		}
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	})
	return err
}

func (c *FederatedTypeCrudTester) TestClusters() map[string]TestCluster {
	return c.testClusters
}

func (c *FederatedTypeCrudTester) waitForResourceDeletion(client util.ResourceClient, qualifiedName util.QualifiedName, versionRemoved func() bool) error {
	err := wait.PollImmediate(c.waitInterval, c.clusterWaitTimeout, func() (bool, error) {
		_, err := client.Resources(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			if !versionRemoved() {
				c.tl.Logf("Removal of %q %s successful, but propagated version still exists", c.typeConfig.GetTarget().Kind, qualifiedName)
				return false, nil
			}
			return true, nil
		}
		if err != nil {
			c.tl.Errorf("Error checking that %q %s was deleted: %v", c.typeConfig.GetTarget().Kind, qualifiedName, err)
		}
		return false, nil
	})
	return err
}

func (c *FederatedTypeCrudTester) updateObject(apiResource metav1.APIResource, obj *unstructured.Unstructured, mutateResourceFunc func(*unstructured.Unstructured)) (*unstructured.Unstructured, error) {
	client := c.resourceClient(apiResource)
	err := wait.PollImmediate(c.waitInterval, wait.ForeverTestTimeout, func() (bool, error) {
		mutateResourceFunc(obj)

		_, err := client.Resources(obj.GetNamespace()).Update(obj, metav1.UpdateOptions{})
		if apierrors.IsConflict(err) {
			// The resource was updated by the federation controller.
			// Get the latest version and retry.
			obj, err = client.Resources(obj.GetNamespace()).Get(obj.GetName(), metav1.GetOptions{})
			return false, err
		}
		// Be tolerant of a slow server
		if apierrors.IsServerTimeout(err) {
			return false, nil
		}
		return (err == nil), err
	})
	return obj, err
}

// expectedVersion retrieves the version of the resource expected in the named cluster
func (c *FederatedTypeCrudTester) expectedVersion(qualifiedName util.QualifiedName, templateVersion, overrideVersion, clusterName string) (string, bool) {
	targetKind := c.typeConfig.GetTarget().Kind
	versionName := util.QualifiedName{
		Namespace: qualifiedName.Namespace,
		Name:      common.PropagatedVersionName(targetKind, qualifiedName.Name),
	}
	if c.targetIsNamespace {
		versionName.Namespace = qualifiedName.Name
	}

	loggedWaiting := false
	adapter := versionmanager.NewVersionAdapter(c.typeConfig.GetFederatedNamespaced())
	var version *fedv1a1.PropagatedVersionStatus
	err := wait.PollImmediate(c.waitInterval, wait.ForeverTestTimeout, func() (bool, error) {
		versionObj := adapter.NewObject()
		err := c.client.Get(context.TODO(), versionObj, versionName.Namespace, versionName.Name)
		if apierrors.IsNotFound(err) {
			if !loggedWaiting {
				loggedWaiting = true
				c.tl.Logf("Waiting for %s %q", adapter.TypeName(), versionName)
			}
			return false, nil
		}
		if err != nil {
			c.tl.Errorf("Error retrieving %s %q: %v", adapter.TypeName(), versionName, err)
			return false, nil
		}
		version = adapter.GetStatus(versionObj)
		return true, nil
	})
	if err != nil {
		c.tl.Errorf("Timed out waiting for %s %q", adapter.TypeName(), versionName)
		return "", false
	}

	matchedVersions := (version.TemplateVersion == templateVersion &&
		version.OverrideVersion == overrideVersion)
	if !matchedVersions {
		c.tl.Logf("Expected template and override versions (%q, %q), got (%q, %q)",
			templateVersion, overrideVersion,
			version.TemplateVersion, version.OverrideVersion,
		)
		return "", false
	}

	return c.versionForCluster(version, clusterName), true
}

func (c *FederatedTypeCrudTester) getPrimaryClusterName() string {
	for name, testCluster := range c.testClusters {
		if testCluster.IsPrimary {
			return name
		}
	}
	return ""
}

func (c *FederatedTypeCrudTester) removeOneClusterName(clusterNames []string, clusterNameToRemove string) []string {
	if len(clusterNameToRemove) == 0 {
		return clusterNames[:len(clusterNames)-1]
	}
	newClusterNames := []string{}
	for _, name := range clusterNames {
		if name == clusterNameToRemove {
			continue
		}
		newClusterNames = append(newClusterNames, name)
	}
	return newClusterNames
}

func (c *FederatedTypeCrudTester) versionForCluster(version *fedv1a1.PropagatedVersionStatus, clusterName string) string {
	for _, clusterVersion := range version.ClusterVersions {
		if clusterVersion.ClusterName == clusterName {
			return clusterVersion.Version
		}
	}
	return ""
}

func (c *FederatedTypeCrudTester) CheckStatusCreated(qualifiedName util.QualifiedName) {
	if !c.typeConfig.GetEnableStatus() {
		return
	}

	statusAPIResource := c.typeConfig.GetStatus()
	statusKind := statusAPIResource.Kind

	c.tl.Logf("Checking creation of %s %q", statusKind, qualifiedName)

	client := c.resourceClient(*statusAPIResource)
	err := wait.PollImmediate(c.waitInterval, wait.ForeverTestTimeout, func() (bool, error) {
		_, err := client.Resources(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			c.tl.Errorf("An unexpected error occurred while polling for desired status: %v", err)
		}
		return (err == nil), nil
	})

	if err != nil {
		c.tl.Fatalf("Timed out waiting for %s %q", statusKind, qualifiedName)
	}
}
