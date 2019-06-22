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
	"time"

	apiextv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage/names"

	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1/defaults"
	genericclient "sigs.k8s.io/kubefed/pkg/client/generic"
	"sigs.k8s.io/kubefed/pkg/controller/util"
	"sigs.k8s.io/kubefed/pkg/controller/webhook/kubefedconfig"
	"sigs.k8s.io/kubefed/pkg/features"
	"sigs.k8s.io/kubefed/test/e2e/framework"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("Default", func() {
	f := framework.NewKubeFedFramework("defaulting")
	tl := f.Logger()
	resourceName := kubefedconfig.ResourceName
	var client genericclient.Client
	var defaultKubeFedConfig *v1beta1.KubeFedConfig
	var qualifiedName *util.QualifiedName

	kubeFedConfigGetter := func(namespace, name string) (pkgruntime.Object, error) {
		kubeFedConfig := &v1beta1.KubeFedConfig{}
		err := client.Get(context.TODO(), kubeFedConfig, namespace, name)
		return kubeFedConfig, err
	}

	BeforeEach(func() {
		if client == nil {
			userAgent := fmt.Sprintf("test-%s-defaulting", resourceName)
			client = f.Client(userAgent)
		}

		if qualifiedName == nil {
			kubeFedConfigName := util.KubeFedConfigName
			if framework.TestContext.LimitedScope {
				// Default KubeFedConfig name will already exist when running with
				// LimitedScope so generate a unique name.
				kubeFedConfigName = names.SimpleNameGenerator.GenerateName(util.KubeFedConfigName + "-")
			}
			qualifiedName = &util.QualifiedName{
				Name: kubeFedConfigName,
			}
		}

		if defaultKubeFedConfig == nil {
			defaultKubeFedConfig = &v1beta1.KubeFedConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: qualifiedName.Name,
				},
				Spec: v1beta1.KubeFedConfigSpec{
					// Set some values different from the defaults used. This
					// ensures they are retained after defaulting is applied.
					ControllerDuration: &v1beta1.DurationConfig{
						AvailableDelay: &metav1.Duration{
							Duration: defaults.DefaultClusterAvailableDelay + 11*time.Second,
						},
					},
					FeatureGates: []v1beta1.FeatureGatesConfig{
						{
							Name:          string(features.PushReconciler),
							Configuration: v1beta1.ConfigurationDisabled,
						},
					},
					ClusterHealthCheck: &v1beta1.ClusterHealthCheckConfig{

						Period: &metav1.Duration{
							Duration: defaults.DefaultClusterHealthCheckPeriod + 11*time.Second,
						},
					},
					Scope: apiextv1b1.ClusterScoped, // Required
				},
			}
		}

		// Ensure these get set for every test even if already initialized.
		qualifiedName.Namespace = f.TestNamespaceName()
		defaultKubeFedConfig.ObjectMeta.Namespace = qualifiedName.Namespace
	})

	It(fmt.Sprintf("%s is created by mutating admissing webhook and preserves values set by user", resourceName), func() {
		By(fmt.Sprintf("Creating a %s defaulted by the mutating admission webhook", resourceName))
		webhookDefaultedKubeFedConfig := defaultKubeFedConfig.DeepCopyObject().(*v1beta1.KubeFedConfig)
		err := client.Create(context.TODO(), webhookDefaultedKubeFedConfig)
		framework.ExpectNoError(err, fmt.Sprintf("Error creating default %q", *qualifiedName))

		By(fmt.Sprintf("Creating a %s defaulted explicitly in this test", resourceName))
		myDefaultedKubeFedConfig := defaultKubeFedConfig.DeepCopyObject().(*v1beta1.KubeFedConfig)
		defaults.SetDefaultKubeFedConfig(myDefaultedKubeFedConfig)

		By(fmt.Sprintf("Verifying the %s defaulted by the mutating admission webhook matches the one defaulted explicitly in this test", resourceName))
		framework.WaitForObject(tl, qualifiedName.Namespace, qualifiedName.Name, kubeFedConfigGetter, myDefaultedKubeFedConfig, util.ObjectMetaAndSpecEquivalent)

		if framework.TestContext.LimitedScope {
			// Delete the KubeFedConfig we created since the kubefed system
			// namespace will not be deleted when running namespace scoped.
			framework.AddCleanupAction(func() {
				fedConfig := webhookDefaultedKubeFedConfig
				err := client.Delete(context.TODO(), fedConfig, fedConfig.Namespace, fedConfig.Name)
				if err != nil && !apierrors.IsNotFound(err) {
					f.Logger().Errorf("Error deleting %s %q: %v", resourceName, *qualifiedName, err)
				}
			})
		}
	})

	// TODO(font): This test explicitly verifies the controller-manager detects
	// a valid KubeFedConfig that is created by the helm chart upon deployment
	// of kubefed until https://github.com/kubernetes-sigs/kubefed/issues/983
	// is resolved.
	It(fmt.Sprintf("%s does not cause controller-manager to fail", resourceName), func() {
		if framework.TestContext.LimitedScope {
			framework.Skipf(fmt.Sprintf("Testing of default %s requires an isolated test namespace which is only possible with a cluster-scoped control plane", resourceName))
		}

		By(fmt.Sprintf("Creating a %s defaulted by the mutating admission webhook", resourceName))
		webhookDefaultedKubeFedConfig := defaultKubeFedConfig.DeepCopyObject().(*v1beta1.KubeFedConfig)
		err := client.Create(context.TODO(), webhookDefaultedKubeFedConfig)
		framework.ExpectNoError(err, fmt.Sprintf("Error creating default %q", *qualifiedName))

		By("Spawning a new controller-manager process")
		args := []string{fmt.Sprintf("--kubeconfig=%s", framework.TestContext.KubeConfig),
			fmt.Sprintf("--kubefed-namespace=%s", qualifiedName.Namespace),
		}
		controllerManager, logStream, err := framework.StartControllerManager(args)
		framework.ExpectNoError(err)

		By("Verifying the controller-manager logs successful forward progress messages")
		verifyLogMessages := []string{
			fmt.Sprintf("Using valid %s %q", resourceName, *qualifiedName),
			"Starting cluster controller",
		}

		for _, logMsg := range verifyLogMessages {
			if framework.WaitUntilLogStreamContains(tl, logStream, logMsg) {
				tl.Log(fmt.Sprintf("Successfully verified log message: %q", logMsg))
			} else {
				_ = controllerManager.Process.Kill()
				tl.Fatal(fmt.Sprintf("Failed to verify log message: %q", logMsg))
			}
		}

		err = controllerManager.Process.Kill()
		framework.ExpectNoError(err)
	})
})
