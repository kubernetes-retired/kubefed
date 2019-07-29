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

	"github.com/pborman/uuid"

	apiextv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextv1b1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/kubefed/pkg/apis/core/typeconfig"
	fedv1b1 "sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	genericclient "sigs.k8s.io/kubefed/pkg/client/generic"
	"sigs.k8s.io/kubefed/pkg/controller/util"
	"sigs.k8s.io/kubefed/pkg/kubefedctl"
	kfenable "sigs.k8s.io/kubefed/pkg/kubefedctl/enable"
	kfenableopts "sigs.k8s.io/kubefed/pkg/kubefedctl/options"
	"sigs.k8s.io/kubefed/test/common"
	"sigs.k8s.io/kubefed/test/e2e/framework"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("FTC controller", func() {
	f := framework.NewKubeFedFramework("ftc-controller")
	tl := f.Logger()

	var client genericclient.Client

	BeforeEach(func() {
		if framework.TestContext.RunControllers() {
			controllerFixture := framework.NewFederatedTypeConfigControllerFixture(tl, f.ControllerConfig())
			f.RegisterFixture(controllerFixture)
			client = f.Client("ftc-controller")
		}
	})

	It("should be refreshed if the target CRD version changed for a federated CRD", func() {
		if !framework.TestContext.RunControllers() {
			framework.Skipf("FTC controller can only be tested when controllers are running in-process.")
		}

		By("Creating a CRD with version 1")
		targetCrdKind := fmt.Sprintf("FedTarget-%s", uuid.New()[0:10])
		targetAPIResource := metav1.APIResource{
			Group:   kfenableopts.DefaultFederatedGroup,
			Version: kfenableopts.DefaultFederatedVersion,

			Kind:       targetCrdKind,
			Name:       fedv1b1.PluralName(targetCrdKind),
			Namespaced: true,
		}
		targetCrd := kfenable.CrdForAPIResource(targetAPIResource, nil, nil)
		crdClient := apiextv1b1client.NewForConfigOrDie(f.KubeConfig())

		targetCrd.Spec.Version = "v1"
		createCrdForHost(f.Logger(), crdClient, targetCrd)
		waitForTargetCrd(f.Logger(), f.KubeConfig(), targetAPIResource.Name, "v1")

		By("Enabling federation of the CRD v1")
		needCleanup := true
		typeConfig := enableResource(f, &targetAPIResource, "v1", needCleanup)

		By("Waiting for sync controller ready")
		objectMeta := typeConfig.GetObjectMeta()
		waitForGenerationSynced(tl, client, objectMeta.Namespace, objectMeta.Name)

		By("Upgrading the CRD version from v1 to v2")
		existingCrd, err := crdClient.CustomResourceDefinitions().Get(objectMeta.Name, metav1.GetOptions{})
		if err != nil {
			tl.Fatalf("Error retrieving target CRD %q: %v", objectMeta.Name, err)
		}
		existingCrd.Spec.Version = "v2"
		existingCrd.Spec.Versions = []apiextv1b1.CustomResourceDefinitionVersion{
			{
				Name:    "v2",
				Served:  true,
				Storage: true,
			},
			{
				Name:    "v1",
				Served:  false,
				Storage: false,
			},
		}
		_, err = crdClient.CustomResourceDefinitions().Update(existingCrd)
		if err != nil {
			tl.Fatalf("Error updating target CRD version %q: %v", existingCrd.Spec.Version, err)
		}

		By("Enabling federation of the CRD v2 to make target version of the FederatedTypeConfig updated")
		needCleanup = false
		typeConfig = enableResource(f, &targetAPIResource, "v2", needCleanup)

		By("Waiting for refreshed sync controller ready")
		objectMeta = typeConfig.GetObjectMeta()
		waitForGenerationSynced(tl, client, objectMeta.Namespace, objectMeta.Name)
	})
})

func enableResource(f framework.KubeFedFramework, targetAPIResource *metav1.APIResource, version string, needCleanup bool) typeconfig.Interface {
	tl := f.Logger()

	enableTypeDirective := &kfenable.EnableTypeDirective{
		ObjectMeta: metav1.ObjectMeta{
			Name: targetAPIResource.Name,
		},
		Spec: kfenable.EnableTypeDirectiveSpec{
			TargetVersion:    version,
			FederatedGroup:   targetAPIResource.Group,
			FederatedVersion: targetAPIResource.Version,
		},
	}

	resources, err := kfenable.GetResources(f.KubeConfig(), enableTypeDirective)
	if err != nil {
		tl.Fatalf("Error retrieving resources to enable federation of target type %q: %v", targetAPIResource.Kind, err)
	}
	typeConfig := resources.TypeConfig

	err = kfenable.CreateResources(nil, f.KubeConfig(), resources, f.KubeFedSystemNamespace())
	if err != nil {
		tl.Fatalf("Error creating resources to enable federation of target type %q: %v", targetAPIResource.Kind, err)
	}

	if !needCleanup {
		return typeConfig
	}

	framework.AddCleanupAction(func() {
		delete := true
		dryRun := false

		objectMeta := typeConfig.GetObjectMeta()
		qualifiedName := util.QualifiedName{Namespace: f.KubeFedSystemNamespace(), Name: objectMeta.Name}
		err := kubefedctl.DisableFederation(nil, f.KubeConfig(), enableTypeDirective, qualifiedName, delete, dryRun, false)
		if err != nil {
			tl.Fatalf("Error disabling federation of target type %q: %v", targetAPIResource.Kind, err)
		}
	})
	return typeConfig
}

func waitForTargetCrd(tl common.TestLogger, config *rest.Config, targetName, version string) {
	err := wait.PollImmediate(framework.PollInterval, framework.TestContext.SingleCallTimeout, func() (bool, error) {
		_, err := kfenable.LookupAPIResource(config, targetName, version)
		if err != nil {
			tl.Logf("An error was reported while waiting for target type %q to be published as an available resource: %v", targetName, err)
		}
		return (err == nil), nil
	})
	if err != nil {
		tl.Fatalf("Timed out waiting for target type %q to be published as an available resource", targetName)
	}
}

// waitForGenerationSynced indicates that sync controller is updated
func waitForGenerationSynced(tl common.TestLogger, client genericclient.Client, namespace, name string) {
	err := wait.PollImmediate(framework.PollInterval, framework.TestContext.SingleCallTimeout, func() (bool, error) {
		ftc := fedv1b1.FederatedTypeConfig{}
		err := client.Get(context.TODO(), &ftc, namespace, name)
		if err != nil {
			tl.Fatalf("Error retrieving status of FederatedTypeConfig %q: %v", util.QualifiedName{Namespace: namespace, Name: name}, err)
		}

		if ftc.Generation != ftc.Status.ObservedGeneration {
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		tl.Fatalf("Timed out waiting for sync controller to be refreshed")
	}
}
