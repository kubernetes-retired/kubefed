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
	"fmt"
	"strings"

	apicommon "github.com/kubernetes-sigs/federation-v2/pkg/apis/federation/common"
	"github.com/kubernetes-sigs/federation-v2/pkg/apis/federation/typeconfig"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/federation/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/e2e/framework"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	. "github.com/onsi/ginkgo"
)

type testObjectAccessor func(namespace string, clusterNames []string) (template, placement, override *unstructured.Unstructured, err error)

var _ = Describe("Federated types", func() {
	f := framework.NewFederationFramework("federated-types")

	tl := framework.NewE2ELogger()

	typeConfigs, err := common.FederatedTypeConfigs()
	if err != nil {
		tl.Fatalf("Error loading type configs: %v", err)
	}

	for i, _ := range typeConfigs {
		// Bind the type config inside the loop to ensure the ginkgo
		// closure gets a different value for every loop iteration.
		//
		// Reference: https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		typeConfig := typeConfigs[i]
		templateKind := typeConfig.GetTemplate().Kind

		Describe(fmt.Sprintf("%q resources", templateKind), func() {
			It("should be created, read, updated and deleted successfully", func() {
				// TODO (font): e2e tests for federated Namespace using a
				// test managed federation does not work until k8s
				// namespace controller is added.
				if framework.TestContext.TestManagedFederation &&
					templateKind == util.NamespaceKind {
					framework.Skipf("%s not supported for test managed federation.", templateKind)
				}

				testObjectFunc := func(namespace string, clusterNames []string) (template, placement, override *unstructured.Unstructured, err error) {
					return common.NewTestObjects(typeConfig, namespace, clusterNames)
				}
				validateCrud(f, tl, typeConfig, testObjectFunc)
			})
		})
	}

	Describe("CRD resources", func() {
		It("should be created, read, updated and deleted successfully", func() {

			// TODO(marun) Is there a better way to create crd's from code?

			crdKind := "FedTestCrd"

			userAgent := fmt.Sprintf("test-%s-crud", strings.ToLower(crdKind))

			kubeConfig := f.KubeConfig()
			rest.AddUserAgent(kubeConfig, userAgent)

			pool := dynamic.NewDynamicClientPool(kubeConfig)
			crdApiResource := &metav1.APIResource{
				Group:      "apiextensions.k8s.io",
				Version:    "v1beta1",
				Name:       "customresourcedefinitions",
				Namespaced: false,
			}
			crdClient, err := util.NewResourceClient(pool, crdApiResource)

			// TODO(marun) Need to create the target crd in all member
			// clusters to support more than a host cluster
			// federation.
			testClusters := f.ClusterDynamicClients(crdApiResource, userAgent)
			if len(testClusters) > 1 {
				framework.Skipf("Testing of CRD not yet supported for multiple clusters")
			}

			crd := newTestCrd(tl, crdKind)
			crd, err = crdClient.Resources("").Create(crd)
			if err != nil {
				tl.Fatalf("Error creating crd %s: %v", crdKind, err)
			}
			// TODO(marun) CRD cleanup needs use AfterEach to maximize
			// the chances of removal.  The cluster-scoped nature of
			// CRDs mean cleanup is even more important.
			defer crdClient.Resources("").Delete(crd.GetName(), nil)

			// Create a template crd
			templateKind := fmt.Sprintf("Federated%s", crdKind)
			templateCrd := newTestCrd(tl, templateKind)
			templateCrd, err = crdClient.Resources("").Create(templateCrd)
			if err != nil {
				tl.Fatalf("Error creating template crd: %v", err)
			}
			defer crdClient.Resources("").Delete(templateCrd.GetName(), nil)

			// Create a placement crd
			placementKind := fmt.Sprintf("Federated%sPlacement", crdKind)
			placementCrd := newTestCrd(tl, placementKind)
			placementCrd, err = crdClient.Resources("").Create(placementCrd)
			if err != nil {
				tl.Fatalf("Error creating placement crd: %v", err)
			}
			defer crdClient.Resources("").Delete(placementCrd.GetName(), nil)

			// Create a type config for these types
			version := "v1alpha1"
			typeConfig := &fedv1a1.FederatedTypeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: crd.GetName(),
				},
				Spec: fedv1a1.FederatedTypeConfigSpec{
					Target: fedv1a1.APIResource{
						Version: version,
						Kind:    crdKind,
					},
					Namespaced:         true,
					ComparisonField:    apicommon.ResourceVersionField,
					PropagationEnabled: true,
					Template: fedv1a1.APIResource{
						Group:   "example.com",
						Version: version,
						Kind:    templateKind,
					},
					Placement: fedv1a1.APIResource{
						Kind: placementKind,
					},
				},
			}

			// Set defaults that would normally be set by the api
			fedv1a1.SetFederatedTypeConfigDefaults(typeConfig)

			// Wait for the CRDs to become available in the API
			waitForCrd(pool, tl, typeConfig.GetTarget())
			waitForCrd(pool, tl, typeConfig.GetTemplate())
			waitForCrd(pool, tl, typeConfig.GetPlacement())

			// If not using in-memory controllers, create the type
			// config in the api to ensure a propagation controller
			// will be started for the crd.
			if !framework.TestContext.InMemoryControllers {
				fedClient := f.FedClient(userAgent)
				_, err := fedClient.FederationV1alpha1().FederatedTypeConfigs().Create(typeConfig)
				if err != nil {
					tl.Fatalf("Error creating FederatedTypeConfig %q: %v", crd.GetName(), err)
				}
				defer fedClient.FederationV1alpha1().FederatedTypeConfigs().Delete(typeConfig.Name, nil)
				// TODO(marun) Wait until the controller has started
			}

			testObjectFunc := func(namespace string, clusterNames []string) (template, placement, override *unstructured.Unstructured, err error) {
				templateYaml := `
apiVersion: %s
kind: %s
metadata:
  generateName: "test-crd-"
  namespace: %s
spec:
  template:
    spec:
      bar: baz
`
				data := fmt.Sprintf(templateYaml, "example.com/v1alpha1", templateKind, namespace)
				template, err = common.ReaderToObj(strings.NewReader(data))
				if err != nil {
					return nil, nil, nil, fmt.Errorf("Error reading test template: %v", err)
				}

				placement, err = common.GetPlacementTestObject(typeConfig, namespace, clusterNames)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("Error reading test placement: %v", err)
				}

				return template, placement, nil, nil
			}

			validateCrud(f, tl, typeConfig, testObjectFunc)
		})
	})
})

func waitForCrd(pool dynamic.ClientPool, tl common.TestLogger, apiResource metav1.APIResource) {
	client, err := util.NewResourceClient(pool, &apiResource)
	if err != nil {
		tl.Fatalf("Error creating client for crd %q: %v", apiResource.Kind, err)
	}
	err = wait.PollImmediate(framework.PollInterval, framework.SingleCallTimeout, func() (bool, error) {
		_, err := client.Resources("invalid").Get("invalid", metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return true, nil
		}
		return (err == nil), err
	})
	if err != nil {
		tl.Fatalf("Error waiting for crd %q to become established: %v", apiResource.Kind, err)
	}
}

func validateCrud(f framework.FederationFramework, tl common.TestLogger, typeConfig typeconfig.Interface, testObjectFunc testObjectAccessor) {
	// Initialize an in-memory controller if configuration requires
	f.SetUpControllerFixture(typeConfig)

	templateKind := typeConfig.GetTemplate().Kind

	userAgent := fmt.Sprintf("test-%s-crud", strings.ToLower(templateKind))

	fedConfig := f.FedConfig()
	kubeConfig := f.KubeConfig()
	targetAPIResource := typeConfig.GetTarget()
	testClusters := f.ClusterDynamicClients(&targetAPIResource, userAgent)
	crudTester, err := common.NewFederatedTypeCrudTester(tl, typeConfig, fedConfig, kubeConfig, testClusters, framework.PollInterval, framework.SingleCallTimeout)
	if err != nil {
		tl.Fatalf("Error creating crudtester for %q: %v", templateKind, err)
	}

	clusterNames := []string{}
	for name, _ := range testClusters {
		clusterNames = append(clusterNames, name)
	}
	template, placement, override, err := testObjectFunc(f.TestNamespaceName(), clusterNames)
	if err != nil {
		tl.Fatalf("Error creating test objects: %v", err)
	}

	crudTester.CheckLifecycle(template, placement, override)
}

func newTestCrd(tl common.TestLogger, kind string) *unstructured.Unstructured {
	template := `
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: %s
spec:
  group: %s
  version: v1alpha1
  scope: Namespaced
  names:
    plural: %s
    singular: %s
    kind: %s
`
	group := "example.com"
	singular := strings.ToLower(kind)
	plural := singular + "s"
	name := fmt.Sprintf("%s.%s", plural, group)
	data := fmt.Sprintf(template, name, group, plural, singular, kind)
	obj, err := common.ReaderToObj(strings.NewReader(data))
	if err != nil {
		tl.Fatalf("Error loading test object: %v", err)
	}
	return obj
}
