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
	"context"
	"fmt"
	"strings"

	"github.com/pborman/uuid"
	"github.com/pkg/errors"

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextv1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"

	fedv1b1 "sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	"sigs.k8s.io/kubefed/pkg/controller/util"
	"sigs.k8s.io/kubefed/pkg/kubefedctl"
	kfenable "sigs.k8s.io/kubefed/pkg/kubefedctl/enable"
	kfenableopts "sigs.k8s.io/kubefed/pkg/kubefedctl/options"
	"sigs.k8s.io/kubefed/test/common"
	"sigs.k8s.io/kubefed/test/e2e/framework"

	. "github.com/onsi/ginkgo" //nolint:stylecheck
)

var _ = Describe("Federated CRD resources", func() {
	f := framework.NewKubeFedFramework("crd-resources")

	namespaceScoped := []bool{
		true,
		false,
	}
	for i := range namespaceScoped {
		namespaced := namespaceScoped[i]
		Describe(fmt.Sprintf("with namespaced=%v", namespaced), func() {
			It("should be created, read, updated and deleted successfully", func() {
				if framework.TestContext.LimitedScope {
					// The service account of clusters registered with
					// a namespaced control plane won't have
					// sufficient permissions to create crds.
					//
					// TODO(marun) Revisit this if federation of crds (nee
					// cr/instances of crds) ever becomes a thing.
					framework.Skipf("Validation of cr federation is not supported for a namespaced control plane.")
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

func validateCrdCrud(f framework.KubeFedFramework, targetCrdKind string, namespaced bool) {
	tl := framework.NewE2ELogger()

	targetAPIResource := metav1.APIResource{
		// Need to reuse a group and version for which the helm chart
		// is granted rbac permissions for.  The default group and
		// version used by `kubefedctl enable` meets this criteria.
		Group:   kfenableopts.DefaultFederatedGroup,
		Version: kfenableopts.DefaultFederatedVersion,

		Kind:       targetCrdKind,
		Name:       fedv1b1.PluralName(targetCrdKind),
		Namespaced: namespaced,
	}

	validationSchema := kfenable.ValidationSchema(apiextv1.JSONSchemaProps{
		Type: "object",
		Properties: map[string]apiextv1.JSONSchemaProps{
			"bar": {
				Type: "array",
				Items: &apiextv1.JSONSchemaPropsOrArray{
					Schema: &apiextv1.JSONSchemaProps{
						Type: "string",
					},
				},
			},
		},
	})

	targetCrd := kfenable.CrdForAPIResource(targetAPIResource, validationSchema, nil)

	userAgent := fmt.Sprintf("test-%s-crud", strings.ToLower(targetCrdKind))

	// Create the target crd in all clusters
	clusterConfigs := f.ClusterConfigs(userAgent)
	configs := make([]*rest.Config, 0, len(clusterConfigs))
	var hostConfig *rest.Config
	for clusterName, clusterConfig := range clusterConfigs {
		configs = append(configs, clusterConfig.Config)
		crdClient := apiextv1client.NewForConfigOrDie(clusterConfig.Config)
		if clusterConfig.IsPrimary {
			hostConfig = clusterConfig.Config
			createCrdForHost(tl, crdClient, targetCrd)
		} else {
			createCrd(tl, crdClient, targetCrd, clusterName)
		}
	}

	targetName := targetAPIResource.Name
	err := wait.PollImmediate(framework.PollInterval, framework.TestContext.SingleCallTimeout, func() (bool, error) {
		_, err := kfenable.LookupAPIResource(hostConfig, targetName, targetAPIResource.Version)
		if err != nil {
			tl.Logf("An error was reported while waiting for target type %q to be published as an available resource: %v", targetName, err)
		}
		return (err == nil), nil
	})
	if err != nil {
		tl.Fatalf("Timed out waiting for target type %q to be published as an available resource", targetName)
	}

	enableTypeDirective := &kfenable.EnableTypeDirective{
		ObjectMeta: metav1.ObjectMeta{
			Name: targetAPIResource.Name,
		},
		Spec: kfenable.EnableTypeDirectiveSpec{
			TargetVersion:    targetAPIResource.Version,
			FederatedGroup:   targetAPIResource.Group,
			FederatedVersion: targetAPIResource.Version,
		},
	}

	resources, err := kfenable.GetResources(hostConfig, enableTypeDirective)
	if err != nil {
		tl.Fatalf("Error retrieving resources to enable federation of target type %q: %v", targetAPIResource.Kind, err)
	}
	typeConfig := resources.TypeConfig

	err = kfenable.CreateResources(nil, hostConfig, resources, f.KubeFedSystemNamespace(), false)
	if err != nil {
		tl.Fatalf("Error creating resources to enable federation of target type %q: %v", targetAPIResource.Kind, err)
	}
	framework.AddCleanupAction(func() {
		delete := true
		dryRun := false
		// TODO(marun) Make this more resilient so that removal of all
		// CRDs is attempted even if the removal of any one CRD fails.
		objectMeta := typeConfig.GetObjectMeta()
		qualifiedName := util.QualifiedName{Namespace: f.KubeFedSystemNamespace(), Name: objectMeta.Name}
		err := kubefedctl.DisableFederation(nil, hostConfig, enableTypeDirective, qualifiedName, delete, dryRun, false)
		if err != nil {
			tl.Fatalf("Error disabling federation of target type %q: %v", targetAPIResource.Kind, err)
		}
	})

	// Wait for the CRDs to become available in the API
	for _, c := range configs {
		waitForCrd(c, tl, typeConfig.GetTargetType())
	}
	waitForCrd(hostConfig, tl, typeConfig.GetFederatedType())

	// TODO(marun) If not using in-memory controllers, wait until the
	// controller has started.

	concreteTypeConfig := typeConfig.(*fedv1b1.FederatedTypeConfig)
	// FederateResource needs the typeconfig to carry ns within
	concreteTypeConfig.Namespace = f.KubeFedSystemNamespace()
	testObjectsFunc := func(namespace string, clusterNames []string) (*unstructured.Unstructured, []interface{}, error) {
		fixtureYAML := `
kind: fixture
template:
  spec:
    bar:
    - baz
    - bal
overrides:
  - clusterOverrides:
    - path: /spec/bar
      value:
      - fiz
      - bang
`
		fixture := &unstructured.Unstructured{}
		err = kfenable.DecodeYAML(strings.NewReader(fixtureYAML), fixture)
		if err != nil {
			return nil, nil, errors.Wrap(err, "Error reading test fixture")
		}
		targetObj, err := common.NewTestTargetObject(concreteTypeConfig, namespace, fixture)
		if err != nil {
			return nil, nil, err
		}
		overrides, err := common.OverridesFromFixture(clusterNames, fixture)
		if err != nil {
			return nil, nil, err
		}
		return targetObj, overrides, nil
	}

	crudTester, targetObject, overrides := initCrudTest(f, tl, f.KubeFedSystemNamespace(), typeConfig, testObjectsFunc)
	// Make a copy for use in the orphan check.
	deletionTargetObject := targetObject.DeepCopy()
	crudTester.CheckLifecycle(targetObject, overrides, nil)

	if namespaced {
		// This check should not fail so long as the main test loop
		// skips for limited scope.  If it does fail, manual cleanup
		// after CheckDelete may need to be implemented.
		if framework.TestContext.LimitedScope {
			tl.Fatalf("Test of orphaned deletion assumes deletion of the containing namespace")
		}
		// Perform a check of orphan deletion.
		fedObject := crudTester.CheckCreate(deletionTargetObject, nil, nil)
		orphanDeletion := true
		crudTester.CheckDelete(fedObject, orphanDeletion)
	}
}

func waitForCrd(config *rest.Config, tl common.TestLogger, apiResource metav1.APIResource) {
	client, err := util.NewResourceClient(config, &apiResource)
	if err != nil {
		tl.Fatalf("Error creating client for crd %q: %v", apiResource.Kind, err)
	}
	err = wait.PollImmediate(framework.PollInterval, framework.TestContext.SingleCallTimeout, func() (bool, error) {
		_, err := client.Resources("invalid").Get(context.Background(), "invalid", metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return (err == nil), err
	})
	if err != nil {
		tl.Fatalf("Error waiting for crd %q to become established: %v", apiResource.Kind, err)
	}
}

func createCrdForHost(tl common.TestLogger, client *apiextv1client.ApiextensionsV1Client, crd *apiextv1.CustomResourceDefinition) {
	createCrd(tl, client, crd, "")
}

func createCrd(tl common.TestLogger, client *apiextv1client.ApiextensionsV1Client, crd *apiextv1.CustomResourceDefinition, clusterName string) {
	createdCrd, err := client.CustomResourceDefinitions().Create(context.Background(), crd, metav1.CreateOptions{})
	if err != nil {
		tl.Fatalf("Error creating crd %s in %s: %v", crd.Name, clusterMsg(clusterName), err)
	}
	ensureCRDRemoval(tl, client, createdCrd.Name, clusterName)
}

func ensureCRDRemoval(tl common.TestLogger, client *apiextv1client.ApiextensionsV1Client, crdName, clusterName string) {
	framework.AddCleanupAction(func() {
		err := client.CustomResourceDefinitions().Delete(context.Background(), crdName, metav1.DeleteOptions{})
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
