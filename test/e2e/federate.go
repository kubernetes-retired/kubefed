/*
Copyright 2017 The Kubernetes Authors.

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

package e2e

import (
	"context"
	"fmt"
	"reflect"

	"github.com/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	restclient "k8s.io/client-go/rest"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	genericclient "github.com/kubernetes-sigs/federation-v2/pkg/client/generic"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/kubefedctl/federate"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/e2e/framework"

	. "github.com/onsi/ginkgo"
)

type testResources struct {
	targetResource *unstructured.Unstructured
	typeConfig     typeconfig.Interface
}

var _ = Describe("Federate ", func() {
	f := framework.NewFederationFramework("federate-resource")
	tl := framework.NewE2ELogger()
	typeConfigFixtures := common.TypeConfigFixturesOrDie(tl)

	var kubeConfig *restclient.Config
	var client genericclient.Client

	BeforeEach(func() {
		if kubeConfig == nil {
			var err error
			kubeConfig = f.KubeConfig()
			client, err = genericclient.New(kubeConfig)
			if err != nil {
				tl.Fatalf("Error initializing dynamic client: %v", err)
			}
		}
	})

	// Use one cluster scoped and one namespaced type to test federate a single resource
	toTest := []string{"clusterroles.rbac.authorization.k8s.io", "configmaps"}
	for _, testKey := range toTest {
		typeConfigName := testKey
		fixture := typeConfigFixtures[testKey]
		It(fmt.Sprintf("resource %q, should create an equivalant federated resource in federation", typeConfigName), func() {
			typeConfig := &fedv1a1.FederatedTypeConfig{}
			err := client.Get(context.Background(), typeConfig, f.FederationSystemNamespace(), typeConfigName)
			if err != nil {
				tl.Fatalf("Error retrieving federatedtypeconfig %q: %v", typeConfigName, err)
			}

			if framework.TestContext.LimitedScope && !typeConfig.GetNamespaced() {
				framework.Skipf("Federation of cluster-scoped type %s is not supported by a namespaced control plane.", typeConfigName)
			}

			kind := typeConfig.GetTarget().Kind
			targetAPIResource := typeConfig.GetTarget()
			targetResource, err := common.NewTestTargetObject(typeConfig, f.TestNamespaceName(), fixture)
			if err != nil {
				tl.Fatalf("Error creating test resource: %v", err)
			}

			createdTargetResource, err := common.CreateResource(kubeConfig, targetAPIResource, targetResource)
			if err != nil {
				tl.Fatalf("Error creating resource: %v", err)
			}

			typeName := typeConfig.GetObjectMeta().Name
			typeNamespace := typeConfig.GetObjectMeta().Namespace
			testResourceName := util.NewQualifiedName(createdTargetResource)

			defer deleteResources(f, tl, typeConfig, testResourceName)

			By(fmt.Sprintf("Federating %s %q", kind, testResourceName))

			fedKind := typeConfig.GetFederatedType().Kind
			artifacts, err := federate.GetFederateArtifacts(kubeConfig, typeName, typeNamespace, testResourceName, false, false)
			if err != nil {
				tl.Fatalf("Error getting %s from %s %q: %v", fedKind, kind, testResourceName, err)
			}

			artifactsList := []*federate.FederateArtifacts{}
			artifactsList = append(artifactsList, artifacts)
			err = federate.CreateResources(nil, kubeConfig, artifactsList, typeNamespace, false, false)
			if err != nil {
				tl.Fatalf("Error creating %s %q: %v", fedKind, testResourceName, err)
			}

			By("Comparing the test resource and the templates of target resource for equality")
			validateTemplateEquality(tl, fedResourceFromAPI(tl, typeConfig, kubeConfig, testResourceName), createdTargetResource, kind, fedKind)
		})
	}

	It("namespace with contents, should create equivalant federated resources for all namespaced resources", func() {
		if framework.TestContext.LimitedScope {
			framework.Skipf("Federate namespace with content is not tested when control plane is namespace scoped")
		}

		var testResources []testResources
		var err error
		systemNamespace := f.FederationSystemNamespace()
		testNamespace := f.TestNamespaceName()
		// Set of arbitrary contained resources in a namespace
		containedTypeNames := []string{"configmaps", "secrets", "replicasets.apps"}
		// Namespace itself
		namespaceTypeName := "namespaces"

		testResources, err = containedTestResources(f, client, typeConfigFixtures, containedTypeNames, kubeConfig)
		if err != nil {
			tl.Fatalf("Error creating target resources: %v", err)
		}

		namespaceTestResource := targetNamespaceTestResources(tl, client, kubeConfig, systemNamespace, testNamespace, namespaceTypeName)
		testResources = append(testResources, namespaceTestResource)

		namespaceTypeConfig := namespaceTestResource.typeConfig
		namespaceKind := namespaceTypeConfig.GetTarget().Kind
		namespaceResourceName := util.NewQualifiedName(namespaceTestResource.targetResource)

		By(fmt.Sprintf("Federating %s %q with content", namespaceKind, namespaceResourceName))

		// Artifacts for the parent, that is, the namespace
		artifacts, err := federate.GetFederateArtifacts(kubeConfig, namespaceTypeConfig.GetObjectMeta().Name, namespaceTypeConfig.GetObjectMeta().Namespace, namespaceResourceName, false, false)
		if err != nil {
			tl.Fatalf("Error getting %s from %s %q: %v", namespaceTypeConfig.GetFederatedType().Kind, namespaceKind, namespaceResourceName, err)
		}
		artifactsList := []*federate.FederateArtifacts{}
		artifactsList = append(artifactsList, artifacts)

		skipAPIResourceNames := "pods,replicasets.extensions"
		// Artifacts for the contained resources
		containedArtifactsList, err := federate.GetContainedArtifactsList(kubeConfig, testNamespace, systemNamespace, skipAPIResourceNames, false, false)
		if err != nil {
			tl.Fatalf("Error getting contained artifacts: %v", err)
		}
		artifactsList = append(artifactsList, containedArtifactsList...)

		err = federate.CreateResources(nil, kubeConfig, artifactsList, systemNamespace, false, false)
		if err != nil {
			tl.Fatalf("Error creating resources: %v", err)
		}

		By("Comparing the test resources with the templates of corresponding federated resources for equality")
		validateResourcesEquality(tl, testResources, kubeConfig)
	})
})

func validateResourcesEquality(tl common.TestLogger, testResources []testResources, kubeConfig *restclient.Config) {
	for _, resources := range testResources {
		typeConfig := resources.typeConfig
		kind := typeConfig.GetTarget().Kind
		targetResource := resources.targetResource
		testResourceName := util.NewQualifiedName(targetResource)
		if kind == util.NamespaceKind {
			testResourceName.Namespace = testResourceName.Name
		}
		fedResource := fedResourceFromAPI(tl, typeConfig, kubeConfig, testResourceName)
		validateTemplateEquality(tl, fedResource, targetResource, kind, typeConfig.GetFederatedType().Kind)
	}
}

func validateTemplateEquality(tl common.TestLogger, fedResource, targetResource *unstructured.Unstructured, kind, fedKind string) {
	qualifiedName := util.NewQualifiedName(fedResource)
	templateMap, ok, err := unstructured.NestedFieldCopy(fedResource.Object, util.SpecField, util.TemplateField)
	if err != nil || !ok {
		tl.Fatalf("Error retrieving template from %s %q", fedKind, qualifiedName)
	}

	expectedResource := &unstructured.Unstructured{}
	expectedResource.Object = templateMap.(map[string]interface{})
	federate.RemoveUnwantedFields(expectedResource)
	federate.RemoveUnwantedFields(targetResource)
	if kind == util.NamespaceKind {
		unstructured.RemoveNestedField(targetResource.Object, "spec", "finalizers")
	}

	if !reflect.DeepEqual(expectedResource, targetResource) {
		tl.Fatalf("Federated object template and target object don't match for %s %q; expected: %v, target: %v", fedKind, qualifiedName, expectedResource, targetResource)
	}
}

func deleteResources(f framework.FederationFramework, tl common.TestLogger, typeConfig typeconfig.Interface, testResourceName util.QualifiedName) {
	client := getFedClient(tl, typeConfig, f.KubeConfig())
	deleteResource(tl, client, testResourceName, typeConfig.GetFederatedType().Kind)

	targetAPIResource := typeConfig.GetTarget()
	// Namespaced resources will be deleted in ns cleanup
	if !targetAPIResource.Namespaced {
		testClusters := f.ClusterDynamicClients(&targetAPIResource, "federate-resource")
		for _, cluster := range testClusters {
			deleteResource(tl, cluster.Client, testResourceName, targetAPIResource.Kind)
		}
	}
}

func deleteResource(tl common.TestLogger, client util.ResourceClient, qualifiedName util.QualifiedName, kind string) {
	tl.Logf("Deleting %s %q", kind, qualifiedName)
	err := client.Resources(qualifiedName.Namespace).Delete(qualifiedName.Name, &metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		tl.Fatalf("Error deleting %s %q: %v", kind, qualifiedName, err)
	}

	err = wait.PollImmediate(framework.PollInterval, framework.TestContext.SingleCallTimeout, func() (bool, error) {
		_, err := client.Resources(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
	if err != nil {
		tl.Fatalf("Error deleting %s %q: %v", kind, qualifiedName, err)
	}
}

func fedResourceFromAPI(tl common.TestLogger, typeConfig typeconfig.Interface, kubeConfig *restclient.Config, qualifiedName util.QualifiedName) *unstructured.Unstructured {
	client := getFedClient(tl, typeConfig, kubeConfig)
	fedResource, err := client.Resources(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
	if err != nil {
		tl.Fatalf("Federated resource %q not found: %v", qualifiedName, err)
	}
	return fedResource
}

func targetResourceFromAPI(tl common.TestLogger, typeConfig typeconfig.Interface, kubeConfig *restclient.Config, qualifiedName util.QualifiedName) *unstructured.Unstructured {
	client := getTargetClient(tl, typeConfig, kubeConfig)
	targetResource, err := client.Resources(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
	if err != nil {
		tl.Fatalf("Test resource %q not found: %v", qualifiedName, err)
	}
	return targetResource
}

func getFedClient(tl common.TestLogger, typeConfig typeconfig.Interface, kubeConfig *restclient.Config) util.ResourceClient {
	fedAPIResource := typeConfig.GetFederatedType()
	fedKind := fedAPIResource.Kind
	client, err := util.NewResourceClient(kubeConfig, &fedAPIResource)
	if err != nil {
		tl.Fatalf("Error getting resource client for %s", fedKind)
	}
	return client
}

func getTargetClient(tl common.TestLogger, typeConfig typeconfig.Interface, kubeConfig *restclient.Config) util.ResourceClient {
	apiResource := typeConfig.GetTarget()
	fedKind := apiResource.Kind
	client, err := util.NewResourceClient(kubeConfig, &apiResource)
	if err != nil {
		tl.Fatalf("Error getting resource client for %s", fedKind)
	}
	return client
}

func containedTestResources(f framework.FederationFramework, client genericclient.Client, fixtures map[string]*unstructured.Unstructured,
	typeConfigNames []string, kubeConfig *restclient.Config) ([]testResources, error) {
	resources := []testResources{}
	for _, typeConfigName := range typeConfigNames {
		fixture := fixtures[typeConfigName]

		typeConfig := &fedv1a1.FederatedTypeConfig{}
		err := client.Get(context.Background(), typeConfig, f.FederationSystemNamespace(), typeConfigName)
		if err != nil {
			return resources, errors.Wrapf(err, "Error retrieving federatedtypeconfig %q", typeConfigName)
		}

		targetResource, err := common.NewTestTargetObject(typeConfig, f.TestNamespaceName(), fixture)
		if err != nil {
			return resources, errors.Wrapf(err, "Error getting test resource for %s", typeConfigName)
		}
		createdTargetResource, err := common.CreateResource(kubeConfig, typeConfig.GetTarget(), targetResource)
		if err != nil {
			return resources, errors.Wrapf(err, "Error creating target resource %q", util.NewQualifiedName(targetResource))
		}

		resources = append(resources, testResources{targetResource: createdTargetResource, typeConfig: typeConfig})
	}

	return resources, nil
}

func targetNamespaceTestResources(tl common.TestLogger, client genericclient.Client, kubeConfig *restclient.Config, fedSystemNamespace, targetNamespace, typeConfigName string) testResources {
	typeConfig := &fedv1a1.FederatedTypeConfig{}
	err := client.Get(context.Background(), typeConfig, fedSystemNamespace, typeConfigName)
	if err != nil {
		tl.Fatalf("Error retrieving federatedtypeconfig %q: %v", typeConfigName, err)
	}

	resourceName := util.QualifiedName{Name: targetNamespace, Namespace: targetNamespace}
	resource := targetResourceFromAPI(tl, typeConfig, kubeConfig, resourceName)

	return testResources{targetResource: resource, typeConfig: typeConfig}
}
