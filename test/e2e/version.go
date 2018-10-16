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
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/pborman/uuid"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeclientset "k8s.io/client-go/kubernetes"

	apicommon "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/common"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	corev1alpha1 "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned/typed/core/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/sync/version"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	testcommon "github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/e2e/framework"

	. "github.com/onsi/ginkgo"
)

type testVersionAdapter interface {
	version.VersionAdapter

	// Test-specific version methods
	Delete(qualifiedName util.QualifiedName) error
	Update(obj pkgruntime.Object) (pkgruntime.Object, error)

	// Type-agnostic template methods
	CreateTemplate(obj pkgruntime.Object) (pkgruntime.Object, error)
	DeleteTemplate(qualifiedName util.QualifiedName) error
	GetTemplate(qualifiedName util.QualifiedName) (pkgruntime.Object, error)
	TemplateType() string
	TemplateYAML() string
	TemplateInstance() pkgruntime.Object
}

type testNamespacedVersionAdapter struct {
	version.VersionAdapter
	coreClient corev1alpha1.CoreV1alpha1Interface
	kubeClient kubeclientset.Interface
}

func (a *testNamespacedVersionAdapter) Delete(qualifiedName util.QualifiedName) error {
	return a.coreClient.PropagatedVersions(qualifiedName.Namespace).Delete(qualifiedName.Name, nil)
}

func (a *testNamespacedVersionAdapter) Update(obj pkgruntime.Object) (pkgruntime.Object, error) {
	version := obj.(*fedv1a1.PropagatedVersion)
	return a.coreClient.PropagatedVersions(version.Namespace).Update(version)
}

func (a *testNamespacedVersionAdapter) CreateTemplate(obj pkgruntime.Object) (pkgruntime.Object, error) {
	configMap := obj.(*corev1.ConfigMap)
	return a.kubeClient.CoreV1().ConfigMaps(configMap.Namespace).Create(configMap)
}

func (a *testNamespacedVersionAdapter) DeleteTemplate(qualifiedName util.QualifiedName) error {
	return a.kubeClient.CoreV1().ConfigMaps(qualifiedName.Namespace).Delete(qualifiedName.Name, nil)
}

func (a *testNamespacedVersionAdapter) GetTemplate(qualifiedName util.QualifiedName) (pkgruntime.Object, error) {
	return a.kubeClient.CoreV1().ConfigMaps(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
}

func (a *testNamespacedVersionAdapter) TemplateType() string {
	return "ConfigMap"
}

func (a *testNamespacedVersionAdapter) TemplateYAML() string {
	return `
apiVersion: v1
kind: ConfigMap
metadata:
  generateName: test-version-manager
data:
  foo: bar
`
}

func (a *testNamespacedVersionAdapter) TemplateInstance() pkgruntime.Object {
	return &corev1.ConfigMap{}
}

type testClusterVersionAdapter struct {
	version.VersionAdapter
	coreClient corev1alpha1.CoreV1alpha1Interface
	kubeClient kubeclientset.Interface
}

func (a *testClusterVersionAdapter) Delete(qualifiedName util.QualifiedName) error {
	return a.coreClient.ClusterPropagatedVersions().Delete(qualifiedName.Name, nil)
}

func (a *testClusterVersionAdapter) Update(obj pkgruntime.Object) (pkgruntime.Object, error) {
	version := obj.(*fedv1a1.ClusterPropagatedVersion)
	return a.coreClient.ClusterPropagatedVersions().Update(version)
}

func (a *testClusterVersionAdapter) CreateTemplate(obj pkgruntime.Object) (pkgruntime.Object, error) {
	role := obj.(*rbacv1.ClusterRole)
	return a.kubeClient.RbacV1().ClusterRoles().Create(role)
}

func (a *testClusterVersionAdapter) DeleteTemplate(qualifiedName util.QualifiedName) error {
	return a.kubeClient.RbacV1().ClusterRoles().Delete(qualifiedName.String(), nil)
}

func (a *testClusterVersionAdapter) GetTemplate(qualifiedName util.QualifiedName) (pkgruntime.Object, error) {
	return a.kubeClient.RbacV1().ClusterRoles().Get(qualifiedName.String(), metav1.GetOptions{})
}

func (a *testClusterVersionAdapter) TemplateType() string {
	return "ClusterRole"
}

func (a *testClusterVersionAdapter) TemplateYAML() string {
	return `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  generateName: test-version-manager
`
}

func (a *testClusterVersionAdapter) TemplateInstance() pkgruntime.Object {
	return &rbacv1.ClusterRole{}
}

func newTestVersionAdapter(fedClient fedclientset.Interface, kubeClient kubeclientset.Interface, namespaced bool) testVersionAdapter {
	adapter := version.NewVersionAdapter(fedClient, namespaced)
	coreClient := fedClient.CoreV1alpha1()
	if namespaced {
		return &testNamespacedVersionAdapter{adapter, coreClient, kubeClient}
	}
	return &testClusterVersionAdapter{adapter, coreClient, kubeClient}
}

var _ = Describe("VersionManager", func() {
	userAgent := "test-version-manager"

	f := framework.NewFederationFramework(userAgent)

	tl := framework.NewE2ELogger()

	templateKind := "FederatedFoo"

	var targetKind string
	var namespace string
	var template, override *unstructured.Unstructured
	var versionManager *version.VersionManager
	var expectedStatus fedv1a1.PropagatedVersionStatus
	var clusterNames []string
	var versionMap map[string]string
	var fedClient fedclientset.Interface
	var versionName, templateName util.QualifiedName
	var adapter testVersionAdapter
	var versionType string

	var stopChan chan struct{}

	versionTests := []struct {
		name       string
		namespaced bool
	}{
		{"NamespacedVersionManager", true},
		{"ClusterVersionManager", false},
	}

	for i, _ := range versionTests {
		versionTest := versionTests[i]
		namespaced := versionTest.namespaced

		Describe(versionTest.name, func() {
			BeforeEach(func() {
				// Clear the resource names to avoid AfterEach using
				// stale names if an error occurs in BeforeEach.
				versionName = util.QualifiedName{}
				templateName = util.QualifiedName{}

				// Use a random string for the target kind to ensure test
				// isolation for any test that depends on the state of the
				// API (like the sync test).  The target kind is used by
				// the version manager to prefix version names written to
				// the API.  Prefixing the uuid with an alphanumeric
				// ensures the prefix will be a valid kube name.
				targetKind = fmt.Sprintf("v%s", uuid.New())

				namespace = f.TestNamespaceName()

				clusterNames = []string{"cluster1", "cluster2", "cluster3"}

				fedClient = f.FedClient(userAgent)

				kubeClient := f.KubeClient(userAgent)
				adapter = newTestVersionAdapter(fedClient, kubeClient, namespaced)
				versionType = adapter.TypeName()

				// Use a non-template type as the template to avoid having
				// sync controllers add a deletion finalizer to the
				// created template that would complicate validating
				// garbage collection.
				var err error
				template, err = testcommon.ReaderToObj(strings.NewReader(adapter.TemplateYAML()))
				if err != nil {
					tl.Fatalf("Failed to parse template yaml: %v", err)
				}
				if namespaced {
					template.SetNamespace(namespace)
				}
				content, err := template.MarshalJSON()
				if err != nil {
					tl.Fatalf("Failed to marshall template yaml: %v", err)
				}
				templateInstance := adapter.TemplateInstance()
				err = json.Unmarshal(content, templateInstance)
				if err != nil {
					tl.Fatalf("Failed to unmarshall template: %v", err)
				}

				// Create the template to ensure that created versions
				// will have a valid owner and not be subject to immediate
				// garbage collection.
				createdObj, err := adapter.CreateTemplate(templateInstance)
				if err != nil {
					tl.Fatalf("Error creating template: %v", err)
				}
				metaAccessor, err := meta.Accessor(createdObj)
				if err != nil {
					tl.Fatalf("Failed to retrieve meta accessor for template: %s", err)
				}
				template.SetName(metaAccessor.GetName())
				template.SetUID(metaAccessor.GetUID())
				template.SetResourceVersion(metaAccessor.GetResourceVersion())
				templateName = util.NewQualifiedName(template)

				propVerName := apicommon.PropagatedVersionName(targetKind, template.GetName())
				versionNamespace := namespace
				if !namespaced {
					versionNamespace = ""
				}
				versionName = util.QualifiedName{Namespace: versionNamespace, Name: propVerName}

				versionMap = make(map[string]string)
				for _, name := range clusterNames {
					versionMap[name] = name
				}

				expectedStatus = fedv1a1.PropagatedVersionStatus{
					TemplateVersion: template.GetResourceVersion(),
					OverrideVersion: "",
					ClusterVersions: version.VersionMapToClusterVersions(versionMap),
				}

				versionManager = version.NewVersionManager(fedClient, namespaced, templateKind, targetKind, versionNamespace)
				stopChan = make(chan struct{})
				// There shouldn't be any api objects to load, but Sync
				// also starts the worker that will write to the API.
				versionManager.Sync(stopChan)
			})

			AfterEach(func() {
				close(stopChan)
				// Ensure removal of the template, which will prompt the
				// removal of owned versions by the garbage collector.
				if len(templateName.Name) > 0 {
					err := adapter.DeleteTemplate(templateName)
					if err != nil && !errors.IsNotFound(err) {
						tl.Errorf("Error deleting %s %q: %v", adapter.TemplateType(), templateName, err)
					}
				}
				// Managed fixture doesn't run the garbage collector, so
				// manual cleanup of propagated versions is required.
				if len(versionName.Name) > 0 && framework.TestContext.TestManagedFederation {
					err := adapter.Delete(versionName)
					if err != nil && !errors.IsNotFound(err) {
						tl.Errorf("Error deleting %s %q: %v", versionType, versionName, err)
					}
				}
			})

			inSupportedScopeIt("should create a new version in the API", namespaced, func() {
				versionManager.Update(template, override, clusterNames, versionMap)
				waitForPropVer(tl, adapter, versionName, expectedStatus)
			})

			inSupportedScopeIt("should load versions from the API on sync", namespaced, func() {
				versionManager.Update(template, override, clusterNames, versionMap)
				waitForPropVer(tl, adapter, versionName, expectedStatus)

				// Create a second manager and sync it
				otherManager := version.NewVersionManager(fedClient, namespaced, templateKind, targetKind, namespace)
				otherManager.Sync(stopChan)

				// Ensure that the second manager loaded the version
				// written by the first manager.
				retrievedVersionMap := otherManager.Get(template, override)
				if !reflect.DeepEqual(versionMap, retrievedVersionMap) {
					tl.Fatalf("Sync failed to load version from api")
				}
			})

			inSupportedScopeIt("should refresh and update after out-of-band creation", namespaced, func() {
				// Create a second manager and use it to write a version to the api
				otherManager := version.NewVersionManager(fedClient, namespaced, templateKind, targetKind, namespace)
				otherManager.Sync(stopChan)
				otherManager.Update(template, override, clusterNames, versionMap)
				waitForPropVer(tl, adapter, versionName, expectedStatus)

				// Ensure that an updated version is written successfully
				clusterNames, versionMap = removeOneCluster(clusterNames, versionMap)
				versionManager.Update(template, override, clusterNames, versionMap)
				expectedStatus.ClusterVersions = version.VersionMapToClusterVersions(versionMap)
				waitForPropVer(tl, adapter, versionName, expectedStatus)
			})

			inSupportedScopeIt("should refresh and update after out-of-band update", namespaced, func() {
				versionManager.Update(template, override, clusterNames, versionMap)
				waitForPropVer(tl, adapter, versionName, expectedStatus)

				// Update an annotation out-of-band to ensure a conflict
				propVer, err := adapter.Get(versionName)
				if err != nil {
					tl.Errorf("Error retrieving %s %q: %v", versionType, versionName, err)
				}
				metaAccessor, err := meta.Accessor(propVer)
				if err != nil {
					tl.Fatalf("Failed to retrieve meta accessor for %s %q: %s", versionType, versionName, err)
				}
				metaAccessor.SetAnnotations(map[string]string{"foo": "bar"})
				_, err = adapter.Update(propVer)
				if err != nil {
					tl.Errorf("Error updating %s %q: %v", versionType, versionName, err)
				}

				// Ensure that an updated version is written successfully
				clusterNames, versionMap = removeOneCluster(clusterNames, versionMap)
				versionManager.Update(template, override, clusterNames, versionMap)
				expectedStatus.ClusterVersions = version.VersionMapToClusterVersions(versionMap)
				waitForPropVer(tl, adapter, versionName, expectedStatus)
			})

			inSupportedScopeIt("should recreate after out-of-band deletion", namespaced, func() {
				versionManager.Update(template, override, clusterNames, versionMap)
				waitForPropVer(tl, adapter, versionName, expectedStatus)

				// Delete the written version out-of-band
				err := adapter.Delete(versionName)
				if err != nil {
					tl.Fatalf("Error deleting %s %q: %v", versionType, versionName, err)
				}

				// Ensure a modified version is written successfully
				clusterNames, versionMap = removeOneCluster(clusterNames, versionMap)
				versionManager.Update(template, override, clusterNames, versionMap)
				expectedStatus.ClusterVersions = version.VersionMapToClusterVersions(versionMap)
				waitForPropVer(tl, adapter, versionName, expectedStatus)
			})

			inSupportedScopeIt("should add owner reference that ensures garbage collection when template is deleted", namespaced, func() {
				if framework.TestContext.TestManagedFederation {
					// Test-managed fixture does not run the garbage collector.
					framework.Skipf("Validation of garbage collection is not supported for test-managed federation.")
				}

				versionManager.Update(template, override, clusterNames, versionMap)
				waitForPropVer(tl, adapter, versionName, expectedStatus)

				// Removal of the template should prompt the garbage
				// collection of the associated version resource due to
				// the owner reference added by the version manager.
				err := adapter.DeleteTemplate(templateName)
				if err != nil {
					tl.Fatalf("Error deleting %s %q: %v", adapter.TemplateType(), templateName, err)
				}

				// The removal of the ConfigMap instance is the
				// precondition for the garbage collection of the version
				// resource.
				checkForDeletion(tl, adapter.TemplateType(), templateName, func() error {
					_, err := adapter.GetTemplate(templateName)
					return err
				})
				checkForDeletion(tl, adapter.TypeName(), versionName, func() error {
					_, err = adapter.Get(versionName)
					return err

				})
			})
		})
	}
})

func checkForDeletion(tl common.TestLogger, typeName string, qualifiedName util.QualifiedName, checkFunc func() error) {
	err := wait.PollImmediate(framework.PollInterval, framework.TestContext.SingleCallTimeout, func() (bool, error) {
		err := checkFunc()
		if errors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			tl.Errorf("Error checking %s %q for deletion: %v", typeName, qualifiedName, err)
		}
		return false, nil
	})
	if err != nil {
		tl.Fatalf("Timed out waiting for %s %q to be deleted.", typeName, qualifiedName)
	}
}

func waitForPropVer(tl common.TestLogger, adapter testVersionAdapter, qualifiedName util.QualifiedName, expectedStatus fedv1a1.PropagatedVersionStatus) {
	err := wait.PollImmediate(framework.PollInterval, framework.TestContext.SingleCallTimeout, func() (bool, error) {
		propVer, err := adapter.Get(qualifiedName)
		if errors.IsNotFound(err) {
			return false, nil
		}
		if err != nil {
			tl.Fatalf("Error retrieving status of %s %q: %v", adapter.TypeName(), qualifiedName, err)
		}
		status := adapter.GetStatus(propVer)
		return util.PropagatedVersionStatusEquivalent(&expectedStatus, status), nil
	})
	if err != nil {
		tl.Fatalf("Timed out waiting for %s %q: %v", adapter.TypeName(), qualifiedName, err)
	}
}

func removeOneCluster(clusterNames []string, versionMap map[string]string) ([]string, map[string]string) {
	targetName := clusterNames[0]
	clusterNames[0] = clusterNames[len(clusterNames)-1]
	clusterNames = clusterNames[:len(clusterNames)-1]
	delete(versionMap, targetName)
	return clusterNames, versionMap
}

func inSupportedScopeIt(description string, namespaced bool, f interface{}) {
	// TODO(marun) Refactor test to support skipping before setup so
	// that skip details can be logged.  The implicit skipping this
	// function performs doesn't provide a good indication of which
	// tests are skipped and why.
	if namespaced && framework.TestContext.LimitedScope {
		// Validation of cluster-scoped versioning is not supported for namespaced federation
		PIt(description, f)
	} else {
		It(description, f)
	}
}
