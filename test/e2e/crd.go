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

package e2e

import (
	"fmt"
	"strings"

	apiextv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextv1b1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"

	apicommon "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/common"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/e2e/framework"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("Federated CRD resources", func() {
	f := framework.NewFederationFramework("crd-resources")

	scopes := []apiextv1b1.ResourceScope{
		apiextv1b1.ClusterScoped,
		apiextv1b1.NamespaceScoped,
	}
	for i, _ := range scopes {
		scope := scopes[i]
		Describe(fmt.Sprintf("with scope=%s", scope), func() {
			It("should be created, read, updated and deleted successfully", func() {
				if framework.TestContext.LimitedScope {
					// The service account of member clusters for
					// namespaced federation won't have sufficient
					// permissions to create crds.
					//
					// TODO(marun) Revisit this if federation of crds (nee
					// cr/instances of crds) ever becomes a thing.
					framework.Skipf("Validation of cr federation is not supported for namespaced federation.")
				}

				targetCrdKind := "FedTestCrd"
				if scope == apiextv1b1.ClusterScoped {
					targetCrdKind = fmt.Sprintf("%s%s", scope, targetCrdKind)
				}
				validateCrdCrud(f, targetCrdKind, scope)
			})
		})
	}
})

func validateCrdCrud(f framework.FederationFramework, targetCrdKind string, scope apiextv1b1.ResourceScope) {
	tl := framework.NewE2ELogger()

	targetCrd := newTestCrd(targetCrdKind, scope)
	targetCrdName := targetCrd.GetName()

	userAgent := fmt.Sprintf("test-%s-crud", strings.ToLower(targetCrdKind))

	// Create the target crd in all clusters
	var pools []dynamic.ClientPool
	var hostPool dynamic.ClientPool
	var hostCrdClient *apiextv1b1client.ApiextensionsV1beta1Client
	for clusterName, clusterConfig := range f.ClusterConfigs(userAgent) {
		pool := dynamic.NewDynamicClientPool(clusterConfig.Config)
		pools = append(pools, pool)
		crdClient := apiextv1b1client.NewForConfigOrDie(clusterConfig.Config)
		if clusterConfig.IsPrimary {
			hostPool = pool
			hostCrdClient = crdClient
			createCrdForHost(tl, crdClient, targetCrd)
		} else {
			createCrd(tl, crdClient, targetCrd, clusterName)
		}
	}

	// Create a template crd
	templateKind := fmt.Sprintf("Federated%s", targetCrdKind)
	templateCrd := newTestCrd(templateKind, scope)
	createCrdForHost(tl, hostCrdClient, templateCrd)

	// Create a placement crd
	placementKind := fmt.Sprintf("Federated%sPlacement", targetCrdKind)
	placementCrd := newTestCrd(placementKind, scope)
	createCrdForHost(tl, hostCrdClient, placementCrd)

	// Create a type config for these types
	version := "v1alpha1"
	fedNamespace := f.FederationSystemNamespace()
	typeConfig := &fedv1a1.FederatedTypeConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      targetCrdName,
			Namespace: fedNamespace,
		},
		Spec: fedv1a1.FederatedTypeConfigSpec{
			Target: fedv1a1.APIResource{
				Version: version,
				Kind:    targetCrdKind,
			},
			Namespaced:         scope == apiextv1b1.NamespaceScoped,
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
	for _, pool := range pools {
		waitForCrd(pool, tl, typeConfig.GetTarget())
	}
	waitForCrd(hostPool, tl, typeConfig.GetTemplate())
	waitForCrd(hostPool, tl, typeConfig.GetPlacement())

	// If not using in-memory controllers, create the type
	// config in the api to ensure a propagation controller
	// will be started for the crd.
	if !framework.TestContext.InMemoryControllers {
		fedClient := f.FedClient(userAgent)
		_, err := fedClient.CoreV1alpha1().FederatedTypeConfigs(fedNamespace).Create(typeConfig)
		if err != nil {
			tl.Fatalf("Error creating FederatedTypeConfig for type %q: %v", targetCrdName, err)
		}
		defer fedClient.CoreV1alpha1().FederatedTypeConfigs(fedNamespace).Delete(typeConfig.Name, nil)
		// TODO(marun) Wait until the controller has started
	}

	testObjectFunc := func(namespace string, clusterNames []string) (template, placement, override *unstructured.Unstructured, err error) {
		templateYaml := `
apiVersion: %s
kind: %s
metadata:
  generateName: "test-crd-"
spec:
  template:
    spec:
      bar: baz
`
		data := fmt.Sprintf(templateYaml, "example.com/v1alpha1", templateKind)
		template, err = common.ReaderToObj(strings.NewReader(data))
		if err != nil {
			return nil, nil, nil, fmt.Errorf("Error reading test template: %v", err)
		}
		if scope == apiextv1b1.NamespaceScoped {
			template.SetNamespace(namespace)
		}

		placement, err = common.GetPlacementTestObject(typeConfig, namespace, clusterNames)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("Error reading test placement: %v", err)
		}

		return template, placement, nil, nil
	}

	validateCrud(f, tl, typeConfig, testObjectFunc)

}

func newTestCrd(kind string, scope apiextv1b1.ResourceScope) *apiextv1b1.CustomResourceDefinition {
	plural := fedv1a1.PluralName(kind)
	group := "example.com"
	return &apiextv1b1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s.%s", plural, group),
		},
		Spec: apiextv1b1.CustomResourceDefinitionSpec{
			Group:   group,
			Version: "v1alpha1",
			Scope:   scope,
			Names: apiextv1b1.CustomResourceDefinitionNames{
				Plural:   plural,
				Singular: strings.ToLower(kind),
				Kind:     kind,
			},
		},
	}
}

func waitForCrd(pool dynamic.ClientPool, tl common.TestLogger, apiResource metav1.APIResource) {
	client, err := util.NewResourceClient(pool, &apiResource)
	if err != nil {
		tl.Fatalf("Error creating client for crd %q: %v", apiResource.Kind, err)
	}
	err = wait.PollImmediate(framework.PollInterval, framework.TestContext.SingleCallTimeout, func() (bool, error) {
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

func createCrdForHost(tl common.TestLogger, client *apiextv1b1client.ApiextensionsV1beta1Client, crd *apiextv1b1.CustomResourceDefinition) *apiextv1b1.CustomResourceDefinition {
	return createCrd(tl, client, crd, "")
}

func createCrd(tl common.TestLogger, client *apiextv1b1client.ApiextensionsV1beta1Client, crd *apiextv1b1.CustomResourceDefinition, clusterName string) *apiextv1b1.CustomResourceDefinition {
	clusterMsg := "host cluster"
	if len(clusterName) > 0 {
		clusterMsg = fmt.Sprintf("cluster %q", clusterName)
	}
	createdCrd, err := client.CustomResourceDefinitions().Create(crd)
	if err != nil {
		tl.Fatalf("Error creating crd %s in %s: %v", crd.Name, clusterMsg, err)
	}

	// Using a cleanup action is more reliable than defer()
	framework.AddCleanupAction(func() {
		crdName := createdCrd.Name
		err := client.CustomResourceDefinitions().Delete(crdName, nil)
		if err != nil {
			tl.Errorf("Error deleting crd %q in %s: %v", crdName, clusterMsg, err)
		}
	})

	return createdCrd
}
