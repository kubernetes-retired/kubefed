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
	"strings"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/pkg/errors"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/kubefed/pkg/apis/core/common"
	"sigs.k8s.io/kubefed/pkg/apis/core/typeconfig"
	fedv1a1 "sigs.k8s.io/kubefed/pkg/apis/core/v1alpha1"
	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	genericclient "sigs.k8s.io/kubefed/pkg/client/generic"
	"sigs.k8s.io/kubefed/pkg/controller/sync"
	"sigs.k8s.io/kubefed/pkg/controller/sync/status"
	versionmanager "sigs.k8s.io/kubefed/pkg/controller/sync/version"
	"sigs.k8s.io/kubefed/pkg/controller/util"
	"sigs.k8s.io/kubefed/pkg/kubefedctl/federate"
)

// FederatedTypeCrudTester exercises Create/Read/Update/Delete
// operations for federated types via the KubeFed API and validates
// that the results of those operations are propagated to clusters
// registered with the KubeFed control plane.
type FederatedTypeCrudTester struct {
	tl                TestLogger
	typeConfig        typeconfig.Interface
	targetIsNamespace bool
	client            genericclient.Client
	kubeConfig        *rest.Config
	testClusters      map[string]TestCluster
	waitInterval      time.Duration
	// KubeFed operations will use wait.ForeverTestTimeout.  Any
	// operation that involves member clusters may take longer due to
	// propagation latency.
	clusterWaitTimeout time.Duration
	clustersNamespace  string
}

type TestClusterConfig struct {
	Config    *rest.Config
	IsPrimary bool
}

type TestCluster struct {
	TestClusterConfig
	Client util.ResourceClient
}

func NewFederatedTypeCrudTester(testLogger TestLogger, typeConfig typeconfig.Interface, kubeConfig *rest.Config, testClusters map[string]TestCluster, clustersNamespace string, waitInterval, clusterWaitTimeout time.Duration) (*FederatedTypeCrudTester, error) {
	return &FederatedTypeCrudTester{
		tl:                 testLogger,
		typeConfig:         typeConfig,
		targetIsNamespace:  typeConfig.GetTargetType().Kind == util.NamespaceKind,
		client:             genericclient.NewForConfigOrDie(kubeConfig),
		kubeConfig:         kubeConfig,
		testClusters:       testClusters,
		waitInterval:       waitInterval,
		clusterWaitTimeout: clusterWaitTimeout,
		clustersNamespace:  clustersNamespace,
	}, nil
}

func (c *FederatedTypeCrudTester) CheckLifecycle(targetObject *unstructured.Unstructured, overrides []interface{}, selectors map[string]string) {
	fedObject := c.CheckCreate(targetObject, overrides, selectors)

	c.CheckStatusCreated(util.NewQualifiedName(fedObject))

	c.CheckUpdate(fedObject)
	c.CheckPlacementChange(fedObject)

	// Validate the golden path - removal of resources from member
	// clusters.  A test of orphaning is performed in the
	// namespace-scoped crd crud test.
	c.CheckDelete(fedObject, false)
}

func (c *FederatedTypeCrudTester) Create(targetObject *unstructured.Unstructured, overrides []interface{}, selectors map[string]string) *unstructured.Unstructured {
	qualifiedName := util.NewQualifiedName(targetObject)
	kind := c.typeConfig.GetTargetType().Kind
	fedKind := c.typeConfig.GetFederatedType().Kind
	fedObject, err := federate.FederatedResourceFromTargetResource(c.typeConfig, targetObject)
	if err != nil {
		c.tl.Fatalf("Error obtaining %s from %s %q: %v", fedKind, kind, qualifiedName, err)
	}

	fedObject = c.setAdditionalTestData(fedObject, overrides, selectors, targetObject.GetGenerateName())

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

func (c *FederatedTypeCrudTester) CheckCreate(targetObject *unstructured.Unstructured, overrides []interface{}, selectors map[string]string) *unstructured.Unstructured {
	fedObject := c.Create(targetObject, overrides, selectors)

	c.CheckPropagation(fedObject)
	return fedObject
}

// AdditionalTestData additionally sets fixture overrides and placement clusternames into federated object
func (c *FederatedTypeCrudTester) setAdditionalTestData(fedObject *unstructured.Unstructured, overrides []interface{}, selectors map[string]string, generateName string) *unstructured.Unstructured {
	fedKind := c.typeConfig.GetFederatedType().Kind
	qualifiedName := util.NewQualifiedName(fedObject)

	if overrides != nil {
		err := unstructured.SetNestedField(fedObject.Object, overrides, util.SpecField, util.OverridesField)
		if err != nil {
			c.tl.Fatalf("Error updating overrides in %s %q: %v", fedKind, qualifiedName, err)
		}
	}
	if selectors != nil {
		if err := util.SetClusterSelector(fedObject, selectors); err != nil {
			c.tl.Fatalf("Error setting cluster selectors for %s/%s: %v", fedObject.GetKind(), fedObject.GetName(), err)
		}
	} else {
		clusterNames := []string{}
		for name := range c.testClusters {
			clusterNames = append(clusterNames, name)
		}
		err := util.SetClusterNames(fedObject, clusterNames)
		if err != nil {
			c.tl.Fatalf("Error setting cluster names in %s %q: %v", fedKind, qualifiedName, err)
		}
	}
	fedObject.SetGenerateName(generateName)

	return fedObject
}

func (c *FederatedTypeCrudTester) CheckUpdate(fedObject *unstructured.Unstructured) {
	apiResource := c.typeConfig.GetFederatedType()
	kind := apiResource.Kind
	qualifiedName := util.NewQualifiedName(fedObject)

	key := "/metadata/labels"
	value := map[string]interface{}{
		"crudtester-operation":        "update",
		util.ManagedByKubeFedLabelKey: util.ManagedByKubeFedLabelValue,
	}

	c.tl.Logf("Updating %s %q", kind, qualifiedName)
	updatedFedObject, err := c.updateObject(apiResource, fedObject, func(obj *unstructured.Unstructured) {
		overrides, err := util.GetOverrides(obj)
		if err != nil {
			c.tl.Fatalf("Error retrieving overrides for %s %q: %v", kind, qualifiedName, err)
		}
		for clusterName := range c.testClusters {
			if _, ok := overrides[clusterName]; !ok {
				overrides[clusterName] = util.ClusterOverrides{}
			}
			paths := sets.NewString()
			for _, overrideItem := range overrides[clusterName] {
				paths.Insert(overrideItem.Path)
			}
			if paths.Has(key) {
				c.tl.Fatalf("An override for %q already exists for cluster %q", key, clusterName)
			}
			paths.Insert(key)
			overrides[clusterName] = append(overrides[clusterName], util.ClusterOverride{Path: key, Value: value})
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
		orphanKey := util.OrphanManagedResourcesAnnotation
		err := wait.PollImmediate(c.waitInterval, wait.ForeverTestTimeout, func() (bool, error) {
			var err error
			if fedObject == nil {
				fedObject, err = client.Resources(namespace).Get(context.Background(), name, metav1.GetOptions{})
				if err != nil {
					c.tl.Logf("Error retrieving %s %q to add the %q annotation: %v", federatedKind, qualifiedName, orphanKey, err)
					return false, nil
				}
			}
			if util.IsOrphaningEnabled(fedObject) {
				return true, nil
			}
			util.EnableOrphaning(fedObject)
			fedObject, err = client.Resources(namespace).Update(context.Background(), fedObject, metav1.UpdateOptions{})
			if err == nil {
				return true, nil
			}
			c.tl.Logf("Will retry updating %s %q to include the %q annotation after error: %v", federatedKind, qualifiedName, orphanKey, err)
			// Clear fedObject to ensure its attempted retrieval in the next iteration
			fedObject = nil
			return false, nil
		})
		if err != nil {
			c.tl.Fatalf("Timed out trying to add %q annotation to %s %q", orphanKey, federatedKind, qualifiedName)
		}
	}

	c.tl.Logf("Deleting %s %q", federatedKind, qualifiedName)
	err := client.Resources(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
	if err != nil {
		c.tl.Fatalf("Error deleting %s %q: %v", federatedKind, qualifiedName, err)
	}

	deletingInCluster := !orphanDependents

	waitTimeout := wait.ForeverTestTimeout
	if deletingInCluster {
		// May need extra time to delete both federated and cluster resources
		waitTimeout = c.clusterWaitTimeout
	}

	// Wait for deletion.  The federated resource will only be removed once managed resources have
	// been deleted or orphaned.
	err = wait.PollImmediate(c.waitInterval, waitTimeout, func() (bool, error) {
		_, err := client.Resources(namespace).Get(context.Background(), name, metav1.GetOptions{})
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

	targetKind := c.typeConfig.GetTargetType().Kind

	// TODO(marun) Consider using informer to detect expected deletion state.
	var stateMsg string = "unlabeled"
	if deletingInCluster {
		stateMsg = "not present"
	}
	clusters, err := util.ComputePlacement(fedObject, c.getClusters(), false)
	if err != nil {
		c.tl.Fatalf("Couldn't retrieve clusters for %s/%s: %v", federatedKind, name, err)
	}
	for clusterName, testCluster := range c.testClusters {
		if !clusters.Has(clusterName) {
			continue
		}
		namespace = util.QualifiedNameForCluster(clusterName, qualifiedName).Namespace
		err = wait.PollImmediate(c.waitInterval, waitTimeout, func() (bool, error) {
			obj, err := testCluster.Client.Resources(namespace).Get(context.Background(), name, metav1.GetOptions{})
			switch {
			case !deletingInCluster && apierrors.IsNotFound(err):
				return false, errors.Errorf("%s %q was unexpectedly deleted from cluster %q", targetKind, qualifiedName, clusterName)
			case deletingInCluster && err == nil:
				if c.targetIsNamespace && clusterName == c.getPrimaryClusterName() {
					// A namespace in the host cluster should have the
					// managed label removed instead of being deleted.
					return !util.HasManagedLabel(obj), nil
				}
				// Continue checking for deletion or label removal
				return false, nil
			case !deletingInCluster && err == nil:
				return !util.HasManagedLabel(obj), nil
			case err != nil && !apierrors.IsNotFound(err):
				c.tl.Errorf("Error while checking whether %s %q is %s in cluster %q: %v", targetKind, qualifiedName, stateMsg, clusterName, err)
				// This error may be recoverable
				return false, nil
			default:
				return true, nil
			}
		})
		if err != nil {
			c.tl.Fatalf("Failed to confirm whether %s %q is %s in cluster %q: %v", targetKind, qualifiedName, stateMsg, clusterName, err)
		}
	}
}

func (c *FederatedTypeCrudTester) SetDeleteOption(fedObject *unstructured.Unstructured, opts ...client.DeleteOption) {
	apiResource := c.typeConfig.GetFederatedType()
	qualifiedName := util.NewQualifiedName(fedObject)
	kind := apiResource.Kind
	_, err := c.updateObject(apiResource, fedObject, func(obj *unstructured.Unstructured) {
		err := util.ApplyDeleteOptions(obj, opts...)
		if err != nil {
			c.tl.Fatalf("Error apply delete options for %s %q: %v", kind, qualifiedName, err)
		}
	})
	if err != nil {
		c.tl.Fatalf("Error updating %s %q: %v", kind, qualifiedName, err)
	}
}

func (c *FederatedTypeCrudTester) CheckReplicaSet(fedObject *unstructured.Unstructured) {
	lb, ok, _ := unstructured.NestedStringMap(fedObject.Object, "spec", "selector", "matchLabels")
	if !ok {
		c.tl.Fatal("Failed to get matchLabels on the target deployment")
	}

	matchingLabels := (client.MatchingLabels)(lb)

	for clusterName := range c.testClusters {
		clusterConfig := c.testClusters[clusterName].Config

		kubeClient := kubeclientset.NewForConfigOrDie(clusterConfig)
		WaitForNamespaceOrDie(c.tl, kubeClient, clusterName, fedObject.GetNamespace(),
			c.waitInterval, 30*time.Second)

		clusterClient := genericclient.NewForConfigOrDie(clusterConfig)

		c.tl.Log("Checking that the ReplicaSet still exists in every cluster")

		err := wait.PollImmediate(c.waitInterval, wait.ForeverTestTimeout, func() (bool, error) {
			objList := &appsv1.ReplicaSetList{}
			err := clusterClient.List(context.TODO(), objList, fedObject.GetNamespace(), matchingLabels)
			if err != nil {
				return false, errors.Errorf("Error retrieving ReplicatSet: %v", err)
			}

			if len(objList.Items) == 0 {
				return false, errors.Errorf("ReplicatSet was unexpectedly deleted from cluster %q", clusterName)
			}

			c.tl.Log("Checking that OwnerReferences has been removed from the ReplicaSet")
			hasOwner := false
			for _, rs := range objList.Items {
				if len(rs.OwnerReferences) > 0 {
					hasOwner = true
				}
			}
			return !hasOwner, nil
		})
		if err != nil {
			c.tl.Fatalf("Failed to confirm whether ReplicatSet is in cluster %q: %v", clusterName, err)
		}
	}
}

func (c *FederatedTypeCrudTester) getClusters() []*v1beta1.KubeFedCluster {
	client, err := genericclient.New(c.kubeConfig)
	if err != nil {
		c.tl.Fatalf("Failed to get kubefed clientset: %v", err)
	}

	fedClusters := []*v1beta1.KubeFedCluster{}
	for cluster := range c.testClusters {
		clusterResource := &v1beta1.KubeFedCluster{}
		err = client.Get(context.Background(), clusterResource, c.clustersNamespace, cluster)
		if err != nil {
			c.tl.Fatalf("Cannot get cluster %s: %v", cluster, err)
		}
		fedClusters = append(fedClusters, clusterResource)
	}
	return fedClusters
}

// CheckPropagation checks propagation for the crud tester's clients
func (c *FederatedTypeCrudTester) CheckPropagation(fedObject *unstructured.Unstructured) {
	federatedKind := c.typeConfig.GetFederatedType().Kind
	qualifiedName := util.NewQualifiedName(fedObject)

	selectedClusters, err := util.ComputePlacement(fedObject, c.getClusters(), false)
	if err != nil {
		c.tl.Fatalf("Error retrieving cluster names for %s %q: %v", federatedKind, qualifiedName, err)
	}

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

	targetKind := c.typeConfig.GetTargetType().Kind

	// TODO(marun) run checks in parallel
	primaryClusterName := c.getPrimaryClusterName()
	for clusterName, testCluster := range c.testClusters {
		targetName := util.QualifiedNameForCluster(clusterName, qualifiedName)

		objExpected := selectedClusters.Has(clusterName)

		operation := "to be deleted from"
		if objExpected {
			operation = "in"
		}
		c.tl.Logf("Waiting for %s %q %s cluster %q", targetKind, targetName, operation, clusterName)

		switch {
		case objExpected:
			err := c.waitForResource(testCluster.Client, targetName, overridesMap[clusterName], func() string {
				version, _ := c.expectedVersion(qualifiedName, templateVersion, overrideVersion, clusterName)
				return version
			})
			switch {
			case err == wait.ErrWaitTimeout:
				c.tl.Fatalf("Timeout verifying %s %q in cluster %q: %v", targetKind, targetName, clusterName, err)
			case err != nil:
				c.tl.Fatalf("Failed to verify %s %q in cluster %q: %v", targetKind, targetName, clusterName, err)
			}
		case c.targetIsNamespace && clusterName == primaryClusterName:
			c.checkHostNamespaceUnlabeled(testCluster.Client, targetName, targetKind, clusterName)
		default:
			err := c.waitForResourceDeletion(testCluster.Client, targetName, func() bool {
				version, ok := c.expectedVersion(qualifiedName, templateVersion, overrideVersion, clusterName)
				return version == "" && ok
			})
			// Once resource deletion is complete, wait for the status to reflect the deletion

			switch {
			case err == wait.ErrWaitTimeout:
				if objExpected {
					c.tl.Fatalf("Timeout verifying deletion of %s %q in cluster %q: %v", targetKind, targetName, clusterName, err)
				}
			case err != nil:
				c.tl.Fatalf("Failed to verify deletion of %s %q in cluster %q: %v", targetKind, targetName, clusterName, err)
			}
		}

		// Use a longer wait interval to avoid spamming the test log.
		waitInterval := 1 * time.Second
		var waitingForError error
		err := wait.PollImmediate(waitInterval, c.clusterWaitTimeout, func() (bool, error) {
			ok, err := c.checkFederatedStatus(fedObject, clusterName, objExpected)
			if err != nil {
				// Logging lots of waiting messages would clutter the
				// logs.  Instead, track the most recent message
				// indicating a wait and log it if the waiting fails.
				if strings.HasPrefix(err.Error(), "Waiting") {
					waitingForError = err
					return false, nil
				}
				return false, err
			}
			return ok, nil
		})
		if err != nil {
			if waitingForError != nil {
				c.tl.Fatalf("Failed to check status for %s %q: %v", federatedKind, qualifiedName, waitingForError)
			}
			c.tl.Fatalf("Failed to check status for %s %q: %v", federatedKind, qualifiedName, err)
		}
	}
}

// checkFederatedStatus ensures that the federated resource status
// reflects the expected propagation state.
func (c *FederatedTypeCrudTester) checkFederatedStatus(fedObject *unstructured.Unstructured, clusterName string, objExpected bool) (bool, error) {
	federatedKind := fedObject.GetKind()
	qualifiedName := util.NewQualifiedName(fedObject)

	// Retrieve the resource from the API to ensure the latest status
	// is considered.
	resource, err := GetGenericResource(c.client, fedObject.GroupVersionKind(), qualifiedName)
	if err != nil {
		return false, err
	}
	if resource.Status == nil {
		c.tl.Logf("Status is not yet available for %s %q", federatedKind, qualifiedName)
		return false, nil
	}
	fedStatus := resource.Status

	if fedStatus.ObservedGeneration != fedObject.GetGeneration() {
		return false, errors.Errorf("Waiting for status.observedGeneration to match metadata.generation for %s %q", federatedKind, qualifiedName)
	}

	// Check that aggregate status is ok
	conditionTrue := false
	for _, condition := range fedStatus.Conditions {
		if condition.Type == status.PropagationConditionType {
			if condition.Status == apiv1.ConditionTrue {
				conditionTrue = true
			}
			break
		}
	}
	if !conditionTrue {
		return false, errors.Errorf("Waiting for the propagated condition of %s %q to have status True", federatedKind, qualifiedName)
	}

	// Check that the cluster status is correct
	if objExpected {
		clusterStatusOK := false
		for _, cluster := range fedStatus.Clusters {
			if cluster.Name == clusterName && cluster.Status == status.ClusterPropagationOK {
				clusterStatusOK = true
				break
			}
		}
		if !clusterStatusOK {
			return false, errors.Errorf("Waiting for %s %q to have ok status for cluster %q", federatedKind, qualifiedName, clusterName)
		}
	} else {
		clusterRemoved := true
		for _, cluster := range fedStatus.Clusters {
			if cluster.Name == clusterName && cluster.Status != status.WaitingForRemoval {
				clusterRemoved = false
				break
			}
		}
		if !clusterRemoved {
			return false, errors.Errorf("Waiting for cluster %q to be removed from the status of %s %q", clusterName, federatedKind, qualifiedName)
		}
	}
	return true, nil
}

func (c *FederatedTypeCrudTester) checkHostNamespaceUnlabeled(client util.ResourceClient, qualifiedName util.QualifiedName, targetKind, clusterName string) {
	// A namespace in the host cluster should end up unlabeled instead of
	// deleted when it is not targeted by placement.

	err := wait.PollImmediate(c.waitInterval, c.clusterWaitTimeout, func() (bool, error) {
		hostNamespace, err := client.Resources("").Get(context.Background(), qualifiedName.Name, metav1.GetOptions{})
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

func (c *FederatedTypeCrudTester) waitForResource(client util.ResourceClient, qualifiedName util.QualifiedName, expectedOverrides util.ClusterOverrides, expectedVersionFunc func() string) error {
	err := wait.PollImmediate(c.waitInterval, c.clusterWaitTimeout, func() (bool, error) {
		expectedVersion := expectedVersionFunc()
		if len(expectedVersion) == 0 {
			return false, nil
		}

		clusterObj, err := client.Resources(qualifiedName.Namespace).Get(context.Background(), qualifiedName.Name, metav1.GetOptions{})
		if err == nil && util.ObjectVersion(clusterObj) == expectedVersion {
			// Validate that the resource has been labeled properly,
			// indicating creation or adoption by the sync controller.  This
			// labeling also ensures that the federated informer will be able
			// to cache the resource.
			if !util.HasManagedLabel(clusterObj) {
				c.tl.Errorf("Expected resource to be labeled with %q", fmt.Sprintf("%s: %s", util.ManagedByKubeFedLabelKey, util.ManagedByKubeFedLabelValue))
				return false, nil
			}

			// Validate that the expected override was applied
			if len(expectedOverrides) > 0 {
				expectedClusterObject := clusterObj.DeepCopy()
				// Applying overrides on copy of received cluster object should not change the cluster object if the overrides are properly applied.
				if err := util.ApplyJSONPatch(expectedClusterObject, expectedOverrides); err != nil {
					c.tl.Fatalf("Failed to apply json patch: %v", err)
				}

				// Kubernetes 1.21 introduced a label kubernetes.io/metadata.name to all namespaces so regardless of what we
				// override we should always add this label here to this check.
				if expectedClusterObject.GetObjectKind().GroupVersionKind() == apiv1.SchemeGroupVersion.WithKind("Namespace") {
					labels := expectedClusterObject.GetLabels()
					labels[apiv1.LabelMetadataName] = expectedClusterObject.GetName()
					expectedClusterObject.SetLabels(labels)
				}

				expectedClusterObjectJSON, err := expectedClusterObject.MarshalJSON()
				if err != nil {
					c.tl.Fatalf("Failed to marshal expected cluster object to json: %v", err)
				}

				clusterObjectJSON, err := clusterObj.MarshalJSON()
				if err != nil {
					c.tl.Fatalf("Failed to marshal cluster object to json: %v", err)
				}

				if !jsonpatch.Equal(expectedClusterObjectJSON, clusterObjectJSON) {
					c.tl.Errorf("Cluster object is not as expected. expected: %s, actual: %s", expectedClusterObjectJSON, clusterObjectJSON)
					return false, nil
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
		_, err := client.Resources(qualifiedName.Namespace).Get(context.Background(), qualifiedName.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			if !versionRemoved() {
				c.tl.Logf("Removal of %q %s successful, but propagated version still exists", c.typeConfig.GetTargetType().Kind, qualifiedName)
				return false, nil
			}
			return true, nil
		}
		if err != nil {
			c.tl.Errorf("Error checking that %q %s was deleted: %v", c.typeConfig.GetTargetType().Kind, qualifiedName, err)
		}
		return false, nil
	})
	return err
}

func (c *FederatedTypeCrudTester) updateObject(apiResource metav1.APIResource, obj *unstructured.Unstructured, mutateResourceFunc func(*unstructured.Unstructured)) (*unstructured.Unstructured, error) {
	client := c.resourceClient(apiResource)
	var updatedObj *unstructured.Unstructured
	err := wait.PollImmediate(c.waitInterval, wait.ForeverTestTimeout, func() (bool, error) {
		mutateResourceFunc(obj)

		var err error
		updatedObj, err = client.Resources(obj.GetNamespace()).Update(context.Background(), obj, metav1.UpdateOptions{})
		if apierrors.IsConflict(err) {
			// The resource was updated by the KubeFed controller.
			// Get the latest version and retry.
			obj, err = client.Resources(obj.GetNamespace()).Get(context.Background(), obj.GetName(), metav1.GetOptions{})
			return false, err
		}
		// Be tolerant of a slow server
		if apierrors.IsServerTimeout(err) {
			return false, nil
		}
		return (err == nil), err
	})
	return updatedObj, err
}

// expectedVersion retrieves the version of the resource expected in the named cluster
func (c *FederatedTypeCrudTester) expectedVersion(qualifiedName util.QualifiedName, templateVersion, overrideVersion, clusterName string) (string, bool) {
	targetKind := c.typeConfig.GetTargetType().Kind
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

func (c *FederatedTypeCrudTester) CheckRemoteStatus(fedObject *unstructured.Unstructured, targetObject *unstructured.Unstructured) {
	for clusterName := range c.testClusters {
		clusterConfig := c.testClusters[clusterName].Config

		kubeClient := kubeclientset.NewForConfigOrDie(clusterConfig)
		WaitForNamespaceOrDie(c.tl, kubeClient, clusterName, targetObject.GetNamespace(),
			c.waitInterval, 30*time.Second)

		util.AddManagedLabel(targetObject)
		labeledObj, err := CreateResource(clusterConfig, c.typeConfig.GetTargetType(), targetObject)
		if err != nil {
			c.tl.Fatalf("Failed to create labeled resource in cluster %q: %v", clusterName, err)
		}

		clusterClient := genericclient.NewForConfigOrDie(clusterConfig)

		c.tl.Log("Checking that the resource has status")
		var objStatus interface{}
		err = wait.PollImmediate(c.waitInterval, wait.ForeverTestTimeout, func() (bool, error) {
			obj := &unstructured.Unstructured{}
			obj.SetGroupVersionKind(labeledObj.GroupVersionKind())
			err := clusterClient.Get(context.TODO(), obj, labeledObj.GetNamespace(), labeledObj.GetName())
			if err != nil {
				c.tl.Errorf("Error retrieving kubefed cluster object resource: %v", err)
				return false, nil
			}

			objStatus = obj.Object[util.StatusField]
			c.tl.Logf("Show kubefed cluster object status: %v", objStatus)
			return (obj.Object[util.StatusField] != nil), nil
		})
		if err != nil {
			c.tl.Fatal("Timed out waiting for the resource to have a status field")
		}
		c.tl.Logf("Kubefed cluster object status: %v", objStatus)

		c.tl.Log("Checking that the federated resource has a remote status field")
		objRemoteStatus, err := c.getRemoteStatus(fedObject, clusterName)
		if err != nil {
			c.tl.Fatal("Timed out waiting for the federated resource to have a remote status field")
		}
		if objRemoteStatus == nil {
			c.tl.Fatal("Federated object remote status is empty")
		}
		c.tl.Logf("Show federated object remote status %v", objRemoteStatus)

		err = clusterClient.Delete(context.TODO(), labeledObj, labeledObj.GetNamespace(), labeledObj.GetName())
		if err != nil {
			c.tl.Fatalf("Unexpected error deleting the labeled resource: %v", err)
		}
	}
}

func (c *FederatedTypeCrudTester) getRemoteStatus(fedObject *unstructured.Unstructured, clusterName string) (interface{}, error) {
	apiResource := c.typeConfig.GetFederatedType()
	qualifiedName := util.NewQualifiedName(fedObject)

	client := c.resourceClient(apiResource)
	var remoteStatusObj interface{}
	// The default is normally 30 seconds
	waitTimeout := 6 * wait.ForeverTestTimeout
	err := wait.PollImmediate(c.waitInterval, waitTimeout, func() (bool, error) {
		fedObj, err := client.Resources(qualifiedName.Namespace).Get(context.Background(), qualifiedName.Name, metav1.GetOptions{})
		if err != nil {
			c.tl.Errorf("An unexpected error occurred while polling for remote status: %v", err)
			return false, nil
		}

		resource := &status.GenericFederatedResource{}
		err = util.UnstructuredToInterface(fedObj, resource)
		if err != nil {
			return false, err
		}
		if resource.Status != nil {
			for _, cluster := range resource.Status.Clusters {
				c.tl.Logf("Current status of resource for cluster '%s' with value: %v", cluster.Name, resource.Status)
				if cluster.Name == clusterName && cluster.Status == status.ClusterPropagationOK {
					c.tl.Logf("resource remote status for cluster '%s': %v", cluster.Name, cluster.RemoteStatus)
					if cluster.RemoteStatus != nil {
						remoteStatusObj = cluster.RemoteStatus
						return true, nil
					}
				}
			}
		}
		return false, nil
	})

	if err != nil {
		c.tl.Fatalf("Timed out waiting for the remote status: %q", qualifiedName)
		return nil, err
	}

	return remoteStatusObj, nil
}

func (c *FederatedTypeCrudTester) CheckStatusCreated(qualifiedName util.QualifiedName) {
	if !c.typeConfig.GetStatusEnabled() {
		return
	}

	statusAPIResource := c.typeConfig.GetStatusType()
	// TODO (hectorj2f): To be deprecated.
	// I added a check to support current FederatedServiceStatus and new remote resource
	// status. The new mechanism reuses the statusCollection flag but it does not define
	// a StatusType type.
	if statusAPIResource != nil {
		statusKind := statusAPIResource.Kind

		c.tl.Logf("Checking creation of %s %q", statusKind, qualifiedName)

		client := c.resourceClient(*statusAPIResource)
		err := wait.PollImmediate(c.waitInterval, wait.ForeverTestTimeout, func() (bool, error) {
			_, err := client.Resources(qualifiedName.Namespace).Get(context.Background(), qualifiedName.Name, metav1.GetOptions{})
			if err != nil && !apierrors.IsNotFound(err) {
				c.tl.Errorf("An unexpected error occurred while polling for desired status: %v", err)
			}
			return (err == nil), nil
		})

		if err != nil {
			c.tl.Fatalf("Timed out waiting for %s %q", statusKind, qualifiedName)
		}
	}
}

// GetGenericResource retrieves a federated resource and converts it to
// the generic resource struct.
func GetGenericResource(client genericclient.Client, gvk schema.GroupVersionKind,
	qualifiedName util.QualifiedName) (*status.GenericFederatedResource, error) {
	fedObject := &unstructured.Unstructured{}
	fedObject.SetGroupVersionKind(gvk)
	err := client.Get(context.TODO(), fedObject, qualifiedName.Namespace, qualifiedName.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to retrieve federated resource from the API")
	}

	resource := &status.GenericFederatedResource{}
	err = util.UnstructuredToInterface(fedObject, resource)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to unmarshall federated resource to generic resource struct")
	}

	return resource, nil
}
