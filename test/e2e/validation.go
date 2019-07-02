/*
Copyright 2019 The Kubernetes Authors.

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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"

	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	genericclient "sigs.k8s.io/kubefed/pkg/client/generic"
	"sigs.k8s.io/kubefed/pkg/controller/webhook/federatedtypeconfig"
	"sigs.k8s.io/kubefed/pkg/controller/webhook/kubefedcluster"
	"sigs.k8s.io/kubefed/pkg/kubefedctl/enable"
	"sigs.k8s.io/kubefed/test/common"
	"sigs.k8s.io/kubefed/test/e2e/framework"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("FederatedTypeConfig Core API Validation", func() {
	testBaseName := "core-api-validation"
	f := framework.NewKubeFedFramework(testBaseName)
	resourceName := federatedtypeconfig.ResourceName
	var hostConfig *restclient.Config
	var client genericclient.Client
	var validFtc *v1beta1.FederatedTypeConfig

	BeforeEach(func() {
		if framework.TestContext.InMemoryControllers {
			framework.Skipf("Running validation admission webhook outside of cluster not supported")
		}

		if hostConfig == nil {
			userAgent := fmt.Sprintf("test-%s-validation", resourceName)
			hostConfig = f.HostConfig(userAgent)
			client = f.Client(userAgent)
		}

		if validFtc == nil {
			// For the target API type, use an existing K8s API resource that
			// is not currently enabled by default. This simplifies logic and
			// avoids having to create a CRD that prevents validation tests
			// from running with LimitedScope.
			apiResource := metav1.APIResource{
				Group:      "apps",
				Version:    "v1",
				Kind:       "DaemonSet",
				Name:       "daemonsets",
				Namespaced: true,
			}
			enableTypeDirective := enable.NewEnableTypeDirective()
			validFtc = enable.GenerateTypeConfigForTarget(apiResource, enableTypeDirective).(*v1beta1.FederatedTypeConfig)
		}
		// Using TestNamespaceName() will ensure that for cluster-scoped
		// deployments, a different namespace from the kubefed system
		// namespace is used to make certain that the validation admission
		// webhook works across all namespaces.
		validFtc.Namespace = f.TestNamespaceName()
	})

	It(fmt.Sprintf("should fail when an invalid %s is created or updated", resourceName), func() {
		// This test also implicitly tests the successful creation of a valid
		// resource.
		By(fmt.Sprintf("Creating an invalid %s", resourceName))
		invalidFtc := validFtc.DeepCopyObject().(*v1beta1.FederatedTypeConfig)
		invalidFtc.Spec.FederatedType.Group = ""
		err := client.Create(context.TODO(), invalidFtc)
		if err == nil {
			f.Logger().Fatalf("Expected error creating invalid %s = %+v", resourceName, invalidFtc)
		}

		By(fmt.Sprintf("Creating a valid %s", resourceName))
		validFtcCopy := validFtc.DeepCopyObject().(*v1beta1.FederatedTypeConfig)
		err = client.Create(context.TODO(), validFtcCopy)
		if err != nil {
			f.Logger().Fatalf("Unexpected error creating valid %s = %+v, err: %v", resourceName, validFtcCopy, err)
		} else {
			framework.AddCleanupAction(func() {
				err := client.Delete(context.TODO(), validFtcCopy, validFtcCopy.Namespace, validFtcCopy.Name)
				if err != nil && !apierrors.IsNotFound(err) {
					f.Logger().Errorf("Error deleting %s %s: %v", resourceName, validFtcCopy.Name, err)
				}
			})
		}

		By(fmt.Sprintf("Updating with an invalid %s", resourceName))
		invalidFtc = validFtcCopy.DeepCopyObject().(*v1beta1.FederatedTypeConfig)
		invalidFtc.Spec.FederatedType.Kind = ""
		err = client.Update(context.TODO(), invalidFtc)
		if err == nil {
			f.Logger().Fatalf("Expected error updating invalid %s = %+v", resourceName, invalidFtc)
		}
	})

	When("running with namespace scoped deployment", func() {
		It(fmt.Sprintf("should succeed when an invalid %s is created outside the kubefed system namespace", resourceName), func() {
			if !framework.TestContext.LimitedScope {
				framework.Skipf("Cannot run validation admission webhook namespaced test in a cluster scoped deployment")
			}
			kubeClient := f.KubeClient(fmt.Sprintf("%s-create-namespace", testBaseName))
			invalidFtc := validFtc.DeepCopyObject().(*v1beta1.FederatedTypeConfig)
			namespace := framework.CreateTestNamespace(kubeClient, testBaseName)
			framework.AddCleanupAction(func() {
				framework.DeleteNamespace(kubeClient, namespace)
			})

			By(fmt.Sprintf("Creating an invalid %s in the separate test namespace %s", resourceName, namespace))
			invalidFtc.Namespace = namespace
			invalidFtc.Spec.FederatedType.Scope = "Unknown"
			err := client.Create(context.TODO(), invalidFtc)
			if err != nil {
				f.Logger().Fatalf("Unexpected error creating invalid %s = %+v in another test namespace %s, err: %v", resourceName, invalidFtc, namespace, err)
			}
		})
	})
})

// TODO(font): Generalize and refactor "Core API Validation" e2e tests for
// FederatedTypeConfig and KubeFedCluster.
var _ = Describe("KubeFedCluster Core API Validation", func() {
	testBaseName := "core-api-validation"
	f := framework.NewKubeFedFramework(testBaseName)
	resourceName := kubefedcluster.ResourceName
	var hostConfig *restclient.Config
	var client genericclient.Client
	var validKFC *v1beta1.KubeFedCluster

	BeforeEach(func() {
		if framework.TestContext.InMemoryControllers {
			framework.Skipf("Running validation admission webhook outside of cluster not supported")
		}

		if hostConfig == nil {
			userAgent := fmt.Sprintf("test-%s-validation", resourceName)
			hostConfig = f.HostConfig(userAgent)
			client = f.Client(userAgent)
		}

		if validKFC == nil {
			validKFC = common.ValidKubeFedCluster()
		}
		// Using TestNamespaceName() will ensure that for cluster-scoped
		// deployments, a different namespace from the kubefed system
		// namespace is used to make certain that the validation admission
		// webhook works across all namespaces.
		validKFC.Namespace = f.TestNamespaceName()
	})

	It(fmt.Sprintf("should fail when an invalid %s is created or updated", resourceName), func() {
		// This test also implicitly tests the successful creation of a valid
		// resource.
		By(fmt.Sprintf("Creating an invalid %s", resourceName))
		invalidKFC := validKFC.DeepCopyObject().(*v1beta1.KubeFedCluster)
		invalidKFC.Spec.APIEndpoint = ""
		err := client.Create(context.TODO(), invalidKFC)
		if err == nil {
			f.Logger().Fatalf("Expected error creating invalid %s = %+v", resourceName, invalidKFC)
		}

		By(fmt.Sprintf("Creating a valid %s", resourceName))
		validKFCCopy := validKFC.DeepCopyObject().(*v1beta1.KubeFedCluster)
		err = client.Create(context.TODO(), validKFCCopy)
		if err != nil {
			f.Logger().Fatalf("Unexpected error creating valid %s = %+v, err: %v", resourceName, validKFCCopy, err)
		}

		By(fmt.Sprintf("Updating with an invalid %s", resourceName))
		invalidKFC = validKFCCopy.DeepCopyObject().(*v1beta1.KubeFedCluster)
		invalidKFC.Spec.SecretRef.Name = ""
		err = client.Update(context.TODO(), invalidKFC)
		if err == nil {
			f.Logger().Fatalf("Expected error updating invalid %s = %+v", resourceName, invalidKFC)
		}

		err = client.Delete(context.TODO(), validKFCCopy, validKFCCopy.Namespace, validKFCCopy.Name)
		if err != nil && !apierrors.IsNotFound(err) {
			f.Logger().Errorf("Error deleting %s %s: %v", resourceName, validKFCCopy.Name, err)
		}
	})

	When("running with namespace scoped deployment", func() {
		It(fmt.Sprintf("should succeed when an invalid %s is created outside the kubefed system namespace", resourceName), func() {
			if !framework.TestContext.LimitedScope {
				framework.Skipf("Cannot run validation admission webhook namespaced test in a cluster scoped deployment")
			}
			kubeClient := f.KubeClient(fmt.Sprintf("%s-create-namespace", testBaseName))
			invalidKFC := validKFC.DeepCopyObject().(*v1beta1.KubeFedCluster)
			namespace := framework.CreateTestNamespace(kubeClient, testBaseName)
			framework.AddCleanupAction(func() {
				framework.DeleteNamespace(kubeClient, namespace)
			})

			By(fmt.Sprintf("Creating an invalid %s in the separate test namespace %s", resourceName, namespace))
			invalidKFC.Namespace = namespace
			invalidKFC.Spec.APIEndpoint = "https://"
			err := client.Create(context.TODO(), invalidKFC)
			if err != nil {
				f.Logger().Fatalf("Unexpected error creating invalid %s = %+v in another test namespace %s, err: %v", resourceName, invalidKFC, namespace, err)
			}
		})
	})
})
