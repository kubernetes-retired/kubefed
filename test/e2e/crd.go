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

	apicommon "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/common"
	apiextv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextv1b1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"

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
	targetCrd := federate.CrdForAPIResource(targetAPIResource)

	userAgent := fmt.Sprintf("test-%s-crud", strings.ToLower(targetCrdKind))

	// Create the target crd in all clusters
	var pools []dynamic.ClientPool
	var hostPool dynamic.ClientPool
	for clusterName, clusterConfig := range f.ClusterConfigs(userAgent) {
		pool := dynamic.NewDynamicClientPool(clusterConfig.Config)
		pools = append(pools, pool)
		crdClient := apiextv1b1client.NewForConfigOrDie(clusterConfig.Config)
		if clusterConfig.IsPrimary {
			hostPool = pool
			createCrdForHost(tl, crdClient, targetCrd)
		} else {
			createCrd(tl, crdClient, targetCrd, clusterName)
		}
	}

	hostConfig := f.KubeConfig()

	targetName := targetAPIResource.Name
	err := wait.PollImmediate(framework.PollInterval, framework.TestContext.SingleCallTimeout, func() (bool, error) {
		_, err := federate.LookupAPIResource(hostConfig, targetName)
		if err != nil {
			tl.Logf("An error was reported while waiting for target type %q to be published as an available resource: %v", targetName, err)
		}
		return (err == nil), nil
	})
	if err != nil {
		tl.Fatalf("Timed out waiting for target type %q to be published as an available resource", targetName)
	}

	overridePaths := []string{"spec.bar"}
	typeConfig, err := federate.EnableFederation(
		hostConfig, f.FederationSystemNamespace(), targetAPIResource.Name,
		targetAPIResource.Group, targetAPIResource.Version,
		apicommon.ResourceVersionField, overridePaths, false,
	)
	if err != nil {
		tl.Fatalf("Error enabling federation of target type %q: %v", targetAPIResource.Kind, err)
	}
	framework.AddCleanupAction(func() {
		delete := true
		dryRun := false
		// TODO(marun) Make this more resilient so that removal of all
		// CRDs is attempted even if the removal of any one CRD fails.
		objectMeta := typeConfig.GetObjectMeta()
		qualifiedName := util.QualifiedName{Namespace: objectMeta.Namespace, Name: objectMeta.Name}
		err := federate.DisableFederation(hostConfig, qualifiedName, delete, dryRun)
		if err != nil {
			tl.Fatalf("Error disabling federation of target type %q: %v", targetAPIResource.Kind, err)
		}
	})

	// Wait for the CRDs to become available in the API
	for _, pool := range pools {
		waitForCrd(pool, tl, typeConfig.GetTarget())
	}
	waitForCrd(hostPool, tl, typeConfig.GetTemplate())
	waitForCrd(hostPool, tl, typeConfig.GetPlacement())

	// TODO(marun) If not using in-memory controllers, wait until the
	// controller has started.

	testObjectFunc := func(namespace string, clusterNames []string) (template, placement, override *unstructured.Unstructured, err error) {
		templateYaml := `
apiVersion: %s/%s
kind: %s
metadata:
  generateName: "test-crd-"
spec:
  template:
    spec:
      bar: baz
`
		templateData := fmt.Sprintf(templateYaml, group, version, typeConfig.GetTemplate().Kind)
		template, err = common.ReaderToObj(strings.NewReader(templateData))
		if err != nil {
			return nil, nil, nil, fmt.Errorf("Error reading test template: %v", err)
		}
		if namespaced {
			template.SetNamespace(namespace)
		}

		placement, err = common.GetPlacementTestObject(typeConfig, namespace, clusterNames)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("Error reading test placement: %v", err)
		}

		overrideYaml := `
apiVersion: %s/%s
kind: %s
metadata:
  name: placeholder
spec:
  overrides:
  - clusterName: placeholder
    bar: foo
`
		overrideData := fmt.Sprintf(overrideYaml, group, version, typeConfig.GetOverride().Kind)
		override, err = common.ReaderToObj(strings.NewReader(overrideData))
		if err != nil {
			return nil, nil, nil, fmt.Errorf("Error reading test override: %v", err)
		}
		common.UpdateOverrideObject(typeConfig, namespace, clusterNames, override)

		return template, placement, override, nil
	}

	validateCrud(f, tl, typeConfig, testObjectFunc)

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
