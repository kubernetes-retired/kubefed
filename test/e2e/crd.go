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

	"github.com/pborman/uuid"
	"github.com/pkg/errors"

	apiextv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextv1b1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"

	apicommon "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/common"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/kubefed2/federate"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/e2e/framework"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("Federated CRD resources", func() {
	f := framework.NewFederationFramework("crd-resources")

	namespaceScoped := []bool{
		true,
		false,
	}
	for i := range namespaceScoped {
		namespaced := namespaceScoped[i]
		Describe(fmt.Sprintf("with namespaced=%v", namespaced), func() {
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

				// Ensure the name the target is unique to avoid
				// affecting subsequent test runs if cleanup fails.
				targetCrdKind := fmt.Sprintf("TestFedTarget-%s", uuid.New()[0:10])

				if !namespaced {
					targetCrdKind = fmt.Sprintf("Cluster%s", targetCrdKind)
				}
				validateCrdCrud(f, targetCrdKind, namespaced)
			})
		})
	}

})

func validateCrdCrud(f framework.FederationFramework, targetCrdKind string, namespaced bool) {
	tl := framework.NewE2ELogger()

	group := "example.com"
	version := "v1alpha1"

	targetAPIResource := metav1.APIResource{
		Group:      group,
		Version:    version,
		Kind:       targetCrdKind,
		Name:       fedv1a1.PluralName(targetCrdKind),
		Namespaced: namespaced,
	}

	validationSchema := federate.ValidationSchema(apiextv1b1.JSONSchemaProps{
		Type: "object",
		Properties: map[string]apiextv1b1.JSONSchemaProps{
			"bar": {
				Type: "array",
			},
		},
	})

	targetCrd := federate.CrdForAPIResource(targetAPIResource, validationSchema)

	userAgent := fmt.Sprintf("test-%s-crud", strings.ToLower(targetCrdKind))

	// Create the target crd in all clusters
	var configs []*rest.Config
	var hostConfig *rest.Config
	for clusterName, clusterConfig := range f.ClusterConfigs(userAgent) {
		configs = append(configs, clusterConfig.Config)
		crdClient := apiextv1b1client.NewForConfigOrDie(clusterConfig.Config)
		if clusterConfig.IsPrimary {
			hostConfig = clusterConfig.Config
			createCrdForHost(tl, crdClient, targetCrd)
		} else {
			createCrd(tl, crdClient, targetCrd, clusterName)
		}
	}

	targetName := targetAPIResource.Name
	err := wait.PollImmediate(framework.PollInterval, framework.TestContext.SingleCallTimeout, func() (bool, error) {
		_, err := federate.LookupAPIResource(hostConfig, targetName, targetAPIResource.Version)
		if err != nil {
			tl.Logf("An error was reported while waiting for target type %q to be published as an available resource: %v", targetName, err)
		}
		return (err == nil), nil
	})
	if err != nil {
		tl.Fatalf("Timed out waiting for target type %q to be published as an available resource", targetName)
	}

	enableTypeDirective := &federate.EnableTypeDirective{
		ObjectMeta: metav1.ObjectMeta{
			Name: targetAPIResource.Name,
		},
		Spec: federate.EnableTypeDirectiveSpec{
			TargetVersion:     targetAPIResource.Version,
			FederationGroup:   targetAPIResource.Group,
			FederationVersion: targetAPIResource.Version,
			ComparisonField:   apicommon.ResourceVersionField,
		},
	}

	resources, err := federate.GetResources(hostConfig, enableTypeDirective)
	if err != nil {
		tl.Fatalf("Error retrieving resources to enable federation of target type %q: %v", targetAPIResource.Kind, err)
	}
	typeConfig := resources.TypeConfig

	err = federate.CreateResources(nil, hostConfig, resources, f.FederationSystemNamespace())
	if err != nil {
		tl.Fatalf("Error creating resources to enable federation of target type %q: %v", targetAPIResource.Kind, err)
	}
	framework.AddCleanupAction(func() {
		delete := true
		dryRun := false
		// TODO(marun) Make this more resilient so that removal of all
		// CRDs is attempted even if the removal of any one CRD fails.
		objectMeta := typeConfig.GetObjectMeta()
		qualifiedName := util.QualifiedName{Namespace: f.FederationSystemNamespace(), Name: objectMeta.Name}
		err := federate.DisableFederation(nil, hostConfig, qualifiedName, delete, dryRun)
		if err != nil {
			tl.Fatalf("Error disabling federation of target type %q: %v", targetAPIResource.Kind, err)
		}
	})

	// Wait for the CRDs to become available in the API
	for _, c := range configs {
		waitForCrd(c, tl, typeConfig.GetTarget())
	}
	waitForCrd(hostConfig, tl, typeConfig.GetFederatedType())

	// TODO(marun) If not using in-memory controllers, wait until the
	// controller has started.

	testObjectFunc := func(namespace string, clusterNames []string) (*unstructured.Unstructured, error) {
		fixtureYAML := `
kind: fixture
template:
  spec:
    bar:
    - baz
    - bal
overrides:
  - clusterOverrides:
    - path: bar
      value:
      - fiz
      - bang
`
		fixture := &unstructured.Unstructured{}
		err = federate.DecodeYAML(strings.NewReader(fixtureYAML), fixture)
		if err != nil {
			return nil, errors.Wrap(err, "Error reading test fixture")
		}
		return common.NewTestObject(typeConfig, namespace, clusterNames, fixture)
	}

	orphanDependents := false
	validateCrud(f, tl, typeConfig, testObjectFunc, &orphanDependents)

}

func waitForCrd(config *rest.Config, tl common.TestLogger, apiResource metav1.APIResource) {
	client, err := util.NewResourceClient(config, &apiResource)
	if err != nil {
		tl.Fatalf("Error creating client for crd %q: %v", apiResource.Kind, err)
	}
	err = wait.PollImmediate(framework.PollInterval, framework.TestContext.SingleCallTimeout, func() (bool, error) {
		_, err := client.Resources("invalid").Get("invalid", metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
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
	createdCrd, err := client.CustomResourceDefinitions().Create(crd)
	if err != nil {
		tl.Fatalf("Error creating crd %s in %s: %v", crd.Name, clusterMsg(clusterName), err)
	}
	ensureCRDRemoval(tl, client, createdCrd.Name, clusterName)
	return createdCrd
}

func ensureCRDRemoval(tl common.TestLogger, client *apiextv1b1client.ApiextensionsV1beta1Client, crdName, clusterName string) {
	framework.AddCleanupAction(func() {
		err := client.CustomResourceDefinitions().Delete(crdName, nil)
		if err != nil {
			tl.Errorf("Error deleting crd %q in %s: %v", crdName, clusterMsg(clusterName), err)
		}
	})
}

func clusterMsg(clusterName string) string {
	if len(clusterName) > 0 {
		return fmt.Sprintf("cluster %q", clusterName)
	}
	return "host cluster"
}
