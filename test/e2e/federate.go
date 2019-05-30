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
	"io/ioutil"
	"os"
	"reflect"

	"github.com/pborman/uuid"
	"github.com/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	restclient "k8s.io/client-go/rest"

	"sigs.k8s.io/kubefed/pkg/apis/core/typeconfig"
	fedv1b1 "sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	genericclient "sigs.k8s.io/kubefed/pkg/client/generic"
	"sigs.k8s.io/kubefed/pkg/controller/util"
	"sigs.k8s.io/kubefed/pkg/kubefedctl/federate"
	"sigs.k8s.io/kubefed/test/common"
	"sigs.k8s.io/kubefed/test/e2e/framework"

	. "github.com/onsi/ginkgo"
)

type testResources struct {
	targetResource *unstructured.Unstructured
	typeConfig     typeconfig.Interface
}

var _ = Describe("Federate ", func() {
	f := framework.NewKubeFedFramework("federate-resource")
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
		It(fmt.Sprintf("resource %q, should create an equivalant federated resource in the host cluster", typeConfigName), func() {
			typeConfig := &fedv1b1.FederatedTypeConfig{}
			err := client.Get(context.Background(), typeConfig, f.KubeFedSystemNamespace(), typeConfigName)
			if err != nil {
				tl.Fatalf("Error retrieving federatedtypeconfig %q: %v", typeConfigName, err)
			}

			if framework.TestContext.LimitedScope && !typeConfig.GetNamespaced() {
				framework.Skipf("Federation of cluster-scoped type %s is not supported by a namespaced control plane.", typeConfigName)
			}

			kind := typeConfig.GetTargetType().Kind
			targetAPIResource := typeConfig.GetTargetType()
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

		systemNamespace := f.KubeFedSystemNamespace()
		testNamespace := f.TestNamespaceName()
		// Set of arbitrary contained resources in a namespace
		containedTypeNames := []string{"configmaps", "secrets", "replicasets.apps"}
		// Namespace itself
		namespaceTypeName := "namespaces"

		targetTestResources, err := getTargetTestResources(client, typeConfigFixtures, systemNamespace, testNamespace, containedTypeNames)
		if err != nil {
			tl.Fatalf("Error getting target test resources: %v", err)
		}
		createdTargetResources, err := createTargetResources(targetTestResources, kubeConfig)
		if err != nil {
			tl.Fatalf("Error creating target test resources: %v", err)
		}

		namespaceTestResource := targetNamespaceTestResources(tl, client, kubeConfig, systemNamespace, testNamespace, namespaceTypeName)
		createdTargetResources = append(createdTargetResources, namespaceTestResource)

		namespaceTypeConfig := namespaceTestResource.typeConfig
		namespaceKind := namespaceTypeConfig.GetTargetType().Kind
		namespaceResourceName := util.NewQualifiedName(namespaceTestResource.targetResource)

		By(fmt.Sprintf("Federating %s %q with content", namespaceKind, namespaceResourceName))

		// Artifacts for the parent, that is, the namespace
		artifacts, err := federate.GetFederateArtifacts(kubeConfig, namespaceTypeConfig.GetObjectMeta().Name, namespaceTypeConfig.GetObjectMeta().Namespace, namespaceResourceName, false, false)
		if err != nil {
			tl.Fatalf("Error getting %s from %s %q: %v", namespaceTypeConfig.GetFederatedType().Kind, namespaceKind, namespaceResourceName, err)
		}
		artifactsList := []*federate.FederateArtifacts{}
		artifactsList = append(artifactsList, artifacts)

		skipAPIResourceNames := []string{"pods", "replicasets.extensions"}
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
		validateResourcesEqualityFromAPI(tl, createdTargetResources, kubeConfig)
	})

	It("input yaml from a file, should emit equivalant federated resources", func() {
		tmpFile, err := ioutil.TempFile("", "tmp-")
		if err != nil {
			tl.Fatalf("Error creating temperory file: %v", err)
		}
		defer func() {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
		}()

		systemNamespace := f.KubeFedSystemNamespace()
		testNamespace := f.TestNamespaceName()
		// Set of arbitrary  resources representing both namespaced and non namespaced types
		testTypeNames := []string{"clusterroles.rbac.authorization.k8s.io", "configmaps", "replicasets.apps"}

		targetTestResources, err := getTargetTestResources(client, typeConfigFixtures, systemNamespace, testNamespace, testTypeNames)
		if err != nil {
			tl.Fatalf("Error getting target test resources: %v", err)
		}

		By("Creating a yaml file with a set of test resources")
		err = federate.WriteUnstructuredObjsToYaml(namedTestTargetResources(targetTestResources), tmpFile)
		if err != nil {
			tl.Fatalf("Error writing test resources to yaml")
		}

		By("Decoding the yaml resources back")
		testResourcesFromFile, err := federate.DecodeUnstructuredFromFile(tmpFile.Name())
		if err != nil {
			tl.Fatalf("Failed to decode yaml from file: %v", err)
		}

		By("Federating the decoded resources")
		federatedResources, err := federate.FederateResources(testResourcesFromFile)
		if err != nil {
			tl.Fatalf("Error federating resources: %v", err)
		}

		By("Comparing the original test target resources to the templates in federated resources for equality")
		validateResourcesEquality(tl, targetTestResources, federatedResources)

	})
})

func validateResourcesEquality(tl common.TestLogger, targetResources []testResources, federatedResources []*unstructured.Unstructured) {
	numResources := len(targetResources)
	if numResources != len(federatedResources) {
		tl.Fatalf("The number of federated resources does not match that of target test resources")
	}

	count := 0
	for _, t := range targetResources {
		targetResource := t.targetResource
		for _, federatedResource := range federatedResources {
			if targetResource.GetName() == federatedResource.GetName() {
				validateTemplateEquality(tl, federatedResource, targetResource, t.typeConfig.GetTargetType().Kind, t.typeConfig.GetFederatedType().Kind)
				count++
			}
		}
	}
	if count != numResources {
		tl.Fatalf("Some or all federated resources did not match their original target test resource")
	}
}

func validateResourcesEqualityFromAPI(tl common.TestLogger, testResources []testResources, kubeConfig *restclient.Config) {
	for _, resources := range testResources {
		typeConfig := resources.typeConfig
		kind := typeConfig.GetTargetType().Kind
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

func deleteResources(f framework.KubeFedFramework, tl common.TestLogger, typeConfig typeconfig.Interface, testResourceName util.QualifiedName) {
	client := getFedClient(tl, typeConfig, f.KubeConfig())
	deleteResource(tl, client, testResourceName, typeConfig.GetFederatedType().Kind)

	targetAPIResource := typeConfig.GetTargetType()
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
	apiResource := typeConfig.GetTargetType()
	fedKind := apiResource.Kind
	client, err := util.NewResourceClient(kubeConfig, &apiResource)
	if err != nil {
		tl.Fatalf("Error getting resource client for %s", fedKind)
	}
	return client
}

func namedTestTargetResources(testResources []testResources) []*unstructured.Unstructured {
	var resources []*unstructured.Unstructured
	for _, t := range testResources {
		r := t.targetResource
		// In some tests name is never populated as the resource is
		// not created in API. Setting a name enables matching resources using names.
		// Arg testResources stores the object pointer, updating the name
		// here also reflects in the passed testResources.
		r.SetName(fmt.Sprintf("%s-%s", r.GetGenerateName(), uuid.New()))
		resources = append(resources, r)
	}
	return resources
}

func getTargetTestResources(client genericclient.Client, fixtures map[string]*unstructured.Unstructured,
	systemNamespace, testNamespace string, typeConfigNames []string) ([]testResources, error) {
	resources := []testResources{}
	for _, typeConfigName := range typeConfigNames {
		fixture := fixtures[typeConfigName]

		typeConfig := &fedv1b1.FederatedTypeConfig{}
		err := client.Get(context.Background(), typeConfig, systemNamespace, typeConfigName)
		if err != nil {
			return resources, errors.Wrapf(err, "Error retrieving federatedtypeconfig %q", typeConfigName)
		}

		targetResource, err := common.NewTestTargetObject(typeConfig, testNamespace, fixture)
		if err != nil {
			return resources, errors.Wrapf(err, "Error getting test resource for %s", typeConfigName)
		}

		resources = append(resources, testResources{targetResource: targetResource, typeConfig: typeConfig})
	}

	return resources, nil
}

func createTargetResources(resources []testResources, kubeConfig *restclient.Config) ([]testResources, error) {
	createResources := []testResources{}
	for _, resource := range resources {
		typeConfig := resource.typeConfig
		targetResource := resource.targetResource
		createdTargetResource, err := common.CreateResource(kubeConfig, typeConfig.GetTargetType(), targetResource)
		if err != nil {
			return resources, errors.Wrapf(err, "Error creating target resource %q", util.NewQualifiedName(targetResource))
		}

		createResources = append(createResources, testResources{targetResource: createdTargetResource, typeConfig: typeConfig})
	}

	return createResources, nil
}

func targetNamespaceTestResources(tl common.TestLogger, client genericclient.Client, kubeConfig *restclient.Config, fedSystemNamespace, targetNamespace, typeConfigName string) testResources {
	typeConfig := &fedv1b1.FederatedTypeConfig{}
	err := client.Get(context.Background(), typeConfig, fedSystemNamespace, typeConfigName)
	if err != nil {
		tl.Fatalf("Error retrieving federatedtypeconfig %q: %v", typeConfigName, err)
	}

	resourceName := util.QualifiedName{Name: targetNamespace, Namespace: targetNamespace}
	resource := targetResourceFromAPI(tl, typeConfig, kubeConfig, resourceName)

	return testResources{targetResource: resource, typeConfig: typeConfig}
}
