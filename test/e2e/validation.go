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
	"k8s.io/apiserver/pkg/storage/names"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	"sigs.k8s.io/kubefed/pkg/controller/util"
	ftc "sigs.k8s.io/kubefed/pkg/controller/webhook/federatedtypeconfig"
	kfcluster "sigs.k8s.io/kubefed/pkg/controller/webhook/kubefedcluster"
	kfconfig "sigs.k8s.io/kubefed/pkg/controller/webhook/kubefedconfig"
	"sigs.k8s.io/kubefed/pkg/kubefedctl/enable"
	"sigs.k8s.io/kubefed/test/common"
	"sigs.k8s.io/kubefed/test/e2e/framework"

	. "github.com/onsi/ginkgo" //nolint:stylecheck
)

var _ = Describe("Core API Validation", func() {
	testBaseName := "core-api-validation"
	f := framework.NewKubeFedFramework(testBaseName)

	BeforeEach(func() {
		if framework.TestContext.InMemoryControllers {
			framework.Skipf("Running validation admission webhook outside of cluster not supported")
		}
	})

	resourcesToValidate := []string{ftc.ResourceName, kfcluster.ResourceName, kfconfig.ResourceName}
	for i := range resourcesToValidate {
		resourceName := resourcesToValidate[i]
		vrt := newValidationResourceTest(resourceName)
		vrt.initialize()
		runValidationResourceTests(f, vrt, resourceName, testBaseName)
	}
})

func runValidationResourceTests(f framework.KubeFedFramework, vrt validationResourceTest, resourceName, testBaseName string) {
	It(fmt.Sprintf("for %s should fail when an invalid %s is created or updated", resourceName, resourceName), func() {
		userAgent := fmt.Sprintf("test-%s-validation", resourceName)
		client := f.Client(userAgent)
		namespace := f.TestNamespaceName()
		vrt.getObjectMeta().Namespace = namespace

		By(fmt.Sprintf("Creating an invalid %s", resourceName))
		invalidObj := vrt.invalidObject("")
		err := client.Create(context.TODO(), invalidObj)
		if err == nil {
			f.Logger().Fatalf("Expected error creating invalid %s = %+v", resourceName, invalidObj)
		}

		By(fmt.Sprintf("Creating a valid %s", resourceName))
		validObj := vrt.validObject()
		err = client.Create(context.TODO(), validObj)
		if err != nil {
			f.Logger().Fatalf("Unexpected error creating valid %s = %+v, err: %v", resourceName, validObj, err)
		}

		By(fmt.Sprintf("Updating with an invalid %s", resourceName))
		invalidObj = vrt.invalidObjectFromValid(validObj)
		err = client.Update(context.TODO(), invalidObj)
		if err == nil {
			f.Logger().Fatalf("Expected error updating invalid %s = %+v", resourceName, vrt)
		}

		By(fmt.Sprintf("Patching with an invalid %s", resourceName))
		patch := runtimeclient.MergeFrom(validObj)
		err = client.Patch(context.TODO(), invalidObj, patch)
		if err == nil {
			f.Logger().Fatalf("Expected error patching invalid %s = %+v", resourceName, vrt)
		}

		// Immediately delete the created test resource to avoid errors in
		// other e2e tests that rely on the original e2e testing setup. For
		// example for KubeFedCluster, delete the test cluster we just
		// created as it's not a properly joined member cluster that's part
		// of the original e2e test setup.
		validObjName := vrt.getObjectMeta().Name
		err = client.Delete(context.TODO(), validObj, namespace, validObjName)
		if err != nil && !apierrors.IsNotFound(err) {
			f.Logger().Errorf("Error deleting %s %s: %v", resourceName, validObjName, err)
		}
	})

	// TODO(font): Consider removing once webhook singleton is implemented.
	When("running with namespace scoped deployment", func() {
		It(fmt.Sprintf("for %s should succeed when an invalid %s is created outside the kubefed system namespace", resourceName, resourceName), func() {
			if !framework.TestContext.LimitedScope {
				framework.Skipf("Cannot run validation admission webhook namespaced test in a cluster scoped deployment")
			}
			userAgent := fmt.Sprintf("test-%s-validation", resourceName)
			client := f.Client(userAgent)
			kubeClient := f.KubeClient(fmt.Sprintf("%s-create-namespace", testBaseName))
			namespace := framework.CreateTestNamespace(kubeClient, testBaseName)
			framework.AddCleanupAction(func() {
				framework.DeleteNamespace(kubeClient, namespace)
			})

			By(fmt.Sprintf("Creating an invalid %s in the separate test namespace %s", resourceName, namespace))
			invalidObj := vrt.invalidObject(namespace)
			err := client.Create(context.TODO(), invalidObj)
			if err != nil {
				f.Logger().Fatalf("Unexpected error creating invalid %s = %+v in another test namespace %s, err: %v", resourceName, invalidObj, namespace, err)
			}
		})
	})
}

type validationResourceTest interface {
	initialize() runtimeclient.Object
	getObjectMeta() *metav1.ObjectMeta
	invalidObject(namespace string) runtimeclient.Object
	setInvalidField(obj runtimeclient.Object)
	validObject() runtimeclient.Object
	invalidObjectFromValid(obj runtimeclient.Object) runtimeclient.Object
}

type ftcValidationTest struct {
	object runtimeclient.Object
}

type kfClusterValidationTest struct {
	object runtimeclient.Object
}

type kfConfigValidationTest struct {
	object runtimeclient.Object
}

func newValidationResourceTest(resourceName string) validationResourceTest {
	var vrt validationResourceTest
	switch resourceName {
	case ftc.ResourceName:
		vrt = &ftcValidationTest{}
	case kfcluster.ResourceName:
		vrt = &kfClusterValidationTest{}
	case kfconfig.ResourceName:
		vrt = &kfConfigValidationTest{}
	}
	return vrt
}

func (ftc *ftcValidationTest) initialize() runtimeclient.Object {
	if ftc.object != nil {
		return ftc.object
	}

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
	ftc.object = enable.GenerateTypeConfigForTarget(apiResource, enableTypeDirective).(*v1beta1.FederatedTypeConfig)
	return ftc.object
}

func (ftc *ftcValidationTest) getObjectMeta() *metav1.ObjectMeta {
	return &ftc.object.(*v1beta1.FederatedTypeConfig).ObjectMeta
}

func (ftc *ftcValidationTest) invalidObject(namespace string) runtimeclient.Object {
	invalidFtc := ftc.object.DeepCopyObject().(*v1beta1.FederatedTypeConfig)
	if namespace != "" {
		invalidFtc.Namespace = namespace
	}

	ftc.setInvalidField(invalidFtc)
	return invalidFtc
}

func (ftc *ftcValidationTest) setInvalidField(obj runtimeclient.Object) {
	obj.(*v1beta1.FederatedTypeConfig).Spec.FederatedType.Group = ""
}

func (ftc *ftcValidationTest) validObject() runtimeclient.Object {
	return ftc.object.DeepCopyObject().(runtimeclient.Object)
}

func (ftc *ftcValidationTest) invalidObjectFromValid(obj runtimeclient.Object) runtimeclient.Object {
	invalidFtc := obj.DeepCopyObject().(*v1beta1.FederatedTypeConfig)
	ftc.setInvalidField(invalidFtc)
	return invalidFtc
}

func (kfc *kfClusterValidationTest) initialize() runtimeclient.Object {
	if kfc.object != nil {
		return kfc.object
	}

	kfc.object = common.ValidKubeFedCluster()
	return kfc.object
}

func (kfc *kfClusterValidationTest) getObjectMeta() *metav1.ObjectMeta {
	return &kfc.object.(*v1beta1.KubeFedCluster).ObjectMeta
}

func (kfc *kfClusterValidationTest) invalidObject(namespace string) runtimeclient.Object {
	invalidKfc := kfc.object.DeepCopyObject().(*v1beta1.KubeFedCluster)
	if namespace != "" {
		invalidKfc.Namespace = namespace
	}

	kfc.setInvalidField(invalidKfc)
	return invalidKfc
}

func (kfc *kfClusterValidationTest) setInvalidField(obj runtimeclient.Object) {
	obj.(*v1beta1.KubeFedCluster).Spec.APIEndpoint = ""
}

func (kfc *kfClusterValidationTest) validObject() runtimeclient.Object {
	return kfc.object.DeepCopyObject().(runtimeclient.Object)
}

func (kfc *kfClusterValidationTest) invalidObjectFromValid(obj runtimeclient.Object) runtimeclient.Object {
	invalidKfc := obj.DeepCopyObject().(*v1beta1.KubeFedCluster)
	kfc.setInvalidField(invalidKfc)
	return invalidKfc
}

func (kfc *kfConfigValidationTest) initialize() runtimeclient.Object {
	if kfc.object != nil {
		return kfc.object
	}

	kfconfig := common.ValidKubeFedConfig()
	kfconfig.Name = names.SimpleNameGenerator.GenerateName(util.KubeFedConfigName + "-")
	kfc.object = kfconfig
	return kfc.object
}

func (kfc *kfConfigValidationTest) getObjectMeta() *metav1.ObjectMeta {
	return &kfc.object.(*v1beta1.KubeFedConfig).ObjectMeta
}

func (kfc *kfConfigValidationTest) invalidObject(namespace string) runtimeclient.Object {
	invalidKfc := kfc.object.DeepCopyObject().(*v1beta1.KubeFedConfig)
	if namespace != "" {
		invalidKfc.Namespace = namespace
	}

	kfc.setInvalidField(invalidKfc)
	return invalidKfc
}

func (kfc *kfConfigValidationTest) setInvalidField(obj runtimeclient.Object) {
	*obj.(*v1beta1.KubeFedConfig).Spec.SyncController.AdoptResources = "Unknown"
}

func (kfc *kfConfigValidationTest) validObject() runtimeclient.Object {
	return kfc.object.DeepCopyObject().(runtimeclient.Object)
}

func (kfc *kfConfigValidationTest) invalidObjectFromValid(obj runtimeclient.Object) runtimeclient.Object {
	invalidKfc := obj.DeepCopyObject().(*v1beta1.KubeFedConfig)
	kfc.setInvalidField(invalidKfc)
	return invalidKfc
}
