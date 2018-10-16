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

	apicommon "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/common"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/e2e/framework"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("CRD resources", func() {
	f := framework.NewFederationFramework("federated-types")

	tl := framework.NewE2ELogger()

	It("should be created, read, updated and deleted successfully", func() {
		if framework.TestContext.LimitedScope {
			// TODO(marun) Revisit this if federation of crds (nee
			// cr/instances of crds) ever becomes a thing.
			framework.Skipf("Validation of cr federation is not supported for namespaced federation.")
		}

		// TODO(marun) Is there a better way to create crd's from code?

		targetCrdKind := "FedTestCrd"
		targetCrd := newTestCrd(tl, targetCrdKind)
		targetCrdName := targetCrd.GetName()

		userAgent := fmt.Sprintf("test-%s-crud", strings.ToLower(targetCrdKind))

		// Create the target crd in all clusters
		var pools []dynamic.ClientPool
		var hostPool dynamic.ClientPool
		var hostCrdClient util.ResourceClient
		crdApiResource := &metav1.APIResource{
			Group:      "apiextensions.k8s.io",
			Version:    "v1beta1",
			Name:       "customresourcedefinitions",
			Namespaced: false,
		}
		testClusters := f.ClusterDynamicClients(crdApiResource, userAgent)
		for clusterName, cluster := range testClusters {
			pool := dynamic.NewDynamicClientPool(cluster.Config)
			crdClient, err := util.NewResourceClient(pool, crdApiResource)
			if err != nil {
				tl.Fatalf("Error creating crd resource client for cluster %s: %v", clusterName, err)
			}

			pools = append(pools, pool)
			if cluster.IsPrimary {
				hostPool = pool
				hostCrdClient = crdClient
			}

			_, err = crdClient.Resources("").Create(targetCrd)
			if err != nil {
				tl.Fatalf("Error creating crd %s in cluster %s: %v", targetCrdKind, clusterName, err)
			}
			// TODO(marun) CRD cleanup needs use AfterEach to maximize
			// the chances of removal.  The cluster-scoped nature of
			// CRDs mean cleanup is even more important.
			defer crdClient.Resources("").Delete(targetCrdName, nil)
		}

		// Create a template crd
		templateKind := fmt.Sprintf("Federated%s", targetCrdKind)
		templateCrd := newTestCrd(tl, templateKind)
		templateCrd, err := hostCrdClient.Resources("").Create(templateCrd)
		if err != nil {
			tl.Fatalf("Error creating template crd: %v", err)
		}
		defer hostCrdClient.Resources("").Delete(templateCrd.GetName(), nil)

		// Create a placement crd
		placementKind := fmt.Sprintf("Federated%sPlacement", targetCrdKind)
		placementCrd := newTestCrd(tl, placementKind)
		placementCrd, err = hostCrdClient.Resources("").Create(placementCrd)
		if err != nil {
			tl.Fatalf("Error creating placement crd: %v", err)
		}
		defer hostCrdClient.Resources("").Delete(placementCrd.GetName(), nil)

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
