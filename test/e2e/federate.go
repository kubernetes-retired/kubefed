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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	restclient "k8s.io/client-go/rest"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	genericclient "github.com/kubernetes-sigs/federation-v2/pkg/client/generic"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/kubefed2/federate"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/e2e/framework"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("Federate resource", func() {
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

	// Use one cluster scoped and one namespaced type to test complete flow
	toTest := []string{"clusterroles.rbac.authorization.k8s.io", "configmaps"}
	for _, testKey := range toTest {
		typeConfigName := testKey
		fixture := typeConfigFixtures[testKey]
		It(fmt.Sprintf("for %q should create an equivalant federated resource in federation", typeConfigName), func() {
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
			targetObject, err := common.NewTestTargetObject(typeConfig, f.TestNamespaceName(), fixture)
			if err != nil {
				tl.Fatalf("Error creating test object: %v", err)
			}

			createdTargetObject, err := common.CreateResource(kubeConfig, targetAPIResource, targetObject)
			if err != nil {
				tl.Fatalf("Error creating resource: %v", err)
			}

			typeName := util.QualifiedName{
				Name:      typeConfig.GetObjectMeta().Name,
				Namespace: typeConfig.GetObjectMeta().Namespace,
			}
			testResourceName := util.NewQualifiedName(createdTargetObject)

			defer deleteResources(f, tl, typeConfig, testResourceName)

			tl.Logf("Federating %s %q", kind, testResourceName)
			fedKind := typeConfig.GetFederatedType().Kind
			artifacts, err := federate.GetFederateArtifacts(kubeConfig, typeName, testResourceName, false, false)
			if err != nil {
				tl.Fatalf("Error getting %s from %s %q: %v", fedKind, kind, testResourceName, err)
			}

			err = federate.CreateFedResource(kubeConfig, artifacts, false)
			if err != nil {
				tl.Fatalf("Error creating %s %q: %v", fedKind, testResourceName, err)
			}

			validateTemplateEquality(tl, fedResourceFromAPI(tl, typeConfig, kubeConfig, testResourceName), createdTargetObject, fedKind)
		})
	}
})

func validateTemplateEquality(tl common.TestLogger, fedObj, targetObj *unstructured.Unstructured, fedKind string) {
	qualifiedName := util.NewQualifiedName(fedObj)
	templateMap, ok, err := unstructured.NestedFieldCopy(fedObj.Object, util.SpecField, util.TemplateField)
	if err != nil || !ok {
		tl.Fatalf("Error retrieving template from %s %q", fedKind, qualifiedName)
	}

	expectedObj := &unstructured.Unstructured{}
	expectedObj.Object = templateMap.(map[string]interface{})
	federate.RemoveUnwantedFields(expectedObj)
	federate.RemoveUnwantedFields(targetObj)

	if !reflect.DeepEqual(expectedObj, targetObj) {
		tl.Fatal("Federated object template and target object don't match for %s %q", fedKind, qualifiedName)
	}
}

func deleteResources(f framework.FederationFramework, tl common.TestLogger, typeConfig typeconfig.Interface, testResourceName util.QualifiedName) {
	client := getClient(tl, typeConfig, f.KubeConfig())
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
	client := getClient(tl, typeConfig, kubeConfig)
	fedResource, err := client.Resources(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
	if err != nil {
		tl.Fatalf("Federated resource %q not found: %v", err)
	}
	return fedResource
}

func getClient(tl common.TestLogger, typeConfig typeconfig.Interface, kubeConfig *restclient.Config) util.ResourceClient {
	fedAPIResource := typeConfig.GetFederatedType()
	fedKind := fedAPIResource.Kind
	client, err := util.NewResourceClient(kubeConfig, &fedAPIResource)
	if err != nil {
		tl.Fatalf("Error getting resource client for %s", fedKind)
	}
	return client
}
