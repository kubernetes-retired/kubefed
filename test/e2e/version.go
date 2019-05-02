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
	genericclient "github.com/kubernetes-sigs/federation-v2/pkg/client/generic"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/sync"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/sync/version"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	kfenable "github.com/kubernetes-sigs/federation-v2/pkg/kubefedctl/enable"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/e2e/framework"

	. "github.com/onsi/ginkgo"
)

type testVersionAdapter interface {
	version.VersionAdapter

	// Type-agnostic template methods
	CreateFederatedObject(obj pkgruntime.Object) (pkgruntime.Object, error)
	DeleteFederatedObject(qualifiedName util.QualifiedName) error
	GetFederatedObject(qualifiedName util.QualifiedName) (pkgruntime.Object, error)
	FederatedType() string
	FederatedObjectYAML() string
	FederatedTypeInstance() pkgruntime.Object
}

type testNamespacedVersionAdapter struct {
	version.VersionAdapter
	kubeClient kubeclientset.Interface
}

func (a *testNamespacedVersionAdapter) CreateFederatedObject(obj pkgruntime.Object) (pkgruntime.Object, error) {
	configMap := obj.(*corev1.ConfigMap)
	return a.kubeClient.CoreV1().ConfigMaps(configMap.Namespace).Create(configMap)
}

func (a *testNamespacedVersionAdapter) DeleteFederatedObject(qualifiedName util.QualifiedName) error {
	return a.kubeClient.CoreV1().ConfigMaps(qualifiedName.Namespace).Delete(qualifiedName.Name, nil)
}

func (a *testNamespacedVersionAdapter) GetFederatedObject(qualifiedName util.QualifiedName) (pkgruntime.Object, error) {
	return a.kubeClient.CoreV1().ConfigMaps(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
}

func (a *testNamespacedVersionAdapter) FederatedType() string {
	return "ConfigMap"
}

func (a *testNamespacedVersionAdapter) FederatedObjectYAML() string {
	return `
apiVersion: v1
kind: ConfigMap
metadata:
  generateName: test-version-manager
data:
  foo: bar
`
}

func (a *testNamespacedVersionAdapter) FederatedTypeInstance() pkgruntime.Object {
	return &corev1.ConfigMap{}
}

type testClusterVersionAdapter struct {
	version.VersionAdapter
	kubeClient kubeclientset.Interface
}

func (a *testClusterVersionAdapter) CreateFederatedObject(obj pkgruntime.Object) (pkgruntime.Object, error) {
	role := obj.(*rbacv1.ClusterRole)
	return a.kubeClient.RbacV1().ClusterRoles().Create(role)
}

func (a *testClusterVersionAdapter) DeleteFederatedObject(qualifiedName util.QualifiedName) error {
	return a.kubeClient.RbacV1().ClusterRoles().Delete(qualifiedName.String(), nil)
}

func (a *testClusterVersionAdapter) GetFederatedObject(qualifiedName util.QualifiedName) (pkgruntime.Object, error) {
	return a.kubeClient.RbacV1().ClusterRoles().Get(qualifiedName.String(), metav1.GetOptions{})
}

func (a *testClusterVersionAdapter) FederatedType() string {
	return "ClusterRole"
}

func (a *testClusterVersionAdapter) FederatedObjectYAML() string {
	return `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  generateName: test-version-manager
`
}

func (a *testClusterVersionAdapter) FederatedTypeInstance() pkgruntime.Object {
	return &rbacv1.ClusterRole{}
}

type testVersionedResource struct {
	federatedName   util.QualifiedName
	object          *unstructured.Unstructured
	templateVersion string
	overrideVersion string
}

func (r *testVersionedResource) FederatedName() util.QualifiedName {
	return r.federatedName
}

func (r *testVersionedResource) Object() *unstructured.Unstructured {
	return r.object
}

func (r *testVersionedResource) TemplateVersion() (string, error) {
	return r.templateVersion, nil
}
func (r *testVersionedResource) OverrideVersion() (string, error) {
	return r.overrideVersion, nil
}

func newTestVersionAdapter(client genericclient.Client, kubeClient kubeclientset.Interface, namespaced bool) testVersionAdapter {
	adapter := version.NewVersionAdapter(namespaced)
	if namespaced {
		return &testNamespacedVersionAdapter{adapter, kubeClient}
	}
	return &testClusterVersionAdapter{adapter, kubeClient}
}

var _ = Describe("VersionManager", func() {
	userAgent := "test-version-manager"

	f := framework.NewFederationFramework(userAgent)

	tl := framework.NewE2ELogger()

	federatedKind := "FederatedFoo"

	var targetKind string
	var namespace string
	var fedObject *unstructured.Unstructured
	var versionedResource *testVersionedResource
	var versionManager *version.VersionManager
	var expectedStatus fedv1a1.PropagatedVersionStatus
	var clusterNames []string
	var versionMap map[string]string
	var client genericclient.Client
	var versionName, fedObjectName util.QualifiedName
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

	for i := range versionTests {
		versionTest := versionTests[i]
		namespaced := versionTest.namespaced

		Describe(versionTest.name, func() {
			BeforeEach(func() {
				// Clear the resource names to avoid AfterEach using
				// stale names if an error occurs in BeforeEach.
				versionName = util.QualifiedName{}
				fedObjectName = util.QualifiedName{}

				// Use a random string for the target kind to ensure test
				// isolation for any test that depends on the state of the
				// API (like the sync test).  The target kind is used by
				// the version manager to prefix version names written to
				// the API.  Prefixing the uuid with an alphanumeric
				// ensures the prefix will be a valid kube name.
				targetKind = fmt.Sprintf("v%s", uuid.New())

				namespace = f.TestNamespaceName()

				clusterNames = []string{"cluster1", "cluster2", "cluster3"}

				client = f.Client(userAgent)

				kubeClient := f.KubeClient(userAgent)
				adapter = newTestVersionAdapter(client, kubeClient, namespaced)
				versionType = adapter.TypeName()

				// Use a non-federated type as the federated object to
				// avoid having sync controllers add a deletion
				// finalizer to the object that would complicate
				// validating garbage collection.
				fedObject = &unstructured.Unstructured{}
				err := kfenable.DecodeYAML(strings.NewReader(adapter.FederatedObjectYAML()), fedObject)
				if err != nil {
					tl.Fatalf("Failed to parse yaml: %v", err)
				}
				if namespaced {
					fedObject.SetNamespace(namespace)
				}
				content, err := fedObject.MarshalJSON()
				if err != nil {
					tl.Fatalf("Failed to marshall yaml: %v", err)
				}
				fedObjectInstance := adapter.FederatedTypeInstance()
				err = json.Unmarshal(content, fedObjectInstance)
				if err != nil {
					tl.Fatalf("Failed to unmarshall federated object: %v", err)
				}

				// Create the template to ensure that created versions
				// will have a valid owner and not be subject to immediate
				// garbage collection.
				createdObj, err := adapter.CreateFederatedObject(fedObjectInstance)
				if err != nil {
					tl.Fatalf("Error creating template: %v", err)
				}
				metaAccessor, err := meta.Accessor(createdObj)
				if err != nil {
					tl.Fatalf("Failed to retrieve meta accessor for template: %s", err)
				}
				fedObject.SetName(metaAccessor.GetName())
				fedObject.SetUID(metaAccessor.GetUID())
				fedObject.SetResourceVersion(metaAccessor.GetResourceVersion())
				fedObjectName = util.NewQualifiedName(fedObject)

				templateVersion, err := sync.GetTemplateHash(fedObject.Object)
				if err != nil {
					tl.Fatalf("Failed to determine template version: %v", err)
				}

				versionedResource = &testVersionedResource{
					federatedName:   fedObjectName,
					object:          fedObject,
					templateVersion: templateVersion,
				}

				propVerName := apicommon.PropagatedVersionName(targetKind, fedObject.GetName())
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
					TemplateVersion: templateVersion,
					OverrideVersion: "",
					ClusterVersions: version.VersionMapToClusterVersions(versionMap),
				}

				versionManager = version.NewVersionManager(client, namespaced, federatedKind, targetKind, versionNamespace)
				stopChan = make(chan struct{})
				// There shouldn't be any api objects to load, but Sync
				// also starts the worker that will write to the API.
				versionManager.Sync(stopChan)
			})

			AfterEach(func() {
				close(stopChan)
				// Ensure removal of the federated object, which will
				// prompt the removal of owned versions by the garbage
				// collector.
				if len(fedObjectName.Name) > 0 {
					err := adapter.DeleteFederatedObject(fedObjectName)
					if err != nil && !errors.IsNotFound(err) {
						tl.Errorf("Error deleting %s %q: %v", adapter.FederatedType(), fedObjectName, err)
					}
				}
			})

			inSupportedScopeIt("should create a new version in the API", namespaced, func() {
				err := versionManager.Update(versionedResource, clusterNames, versionMap)
				if err != nil {
					tl.Fatalf("Error updating version status: %v", err)
				}
				waitForPropVer(tl, adapter, client, versionName, expectedStatus)
			})

			inSupportedScopeIt("should load versions from the API on sync", namespaced, func() {
				err := versionManager.Update(versionedResource, clusterNames, versionMap)
				if err != nil {
					tl.Fatalf("Error updating version status: %v", err)
				}
				waitForPropVer(tl, adapter, client, versionName, expectedStatus)

				// Create a second manager and sync it
				otherManager := version.NewVersionManager(client, namespaced, federatedKind, targetKind, namespace)
				otherManager.Sync(stopChan)

				// Ensure that the second manager loaded the version
				// written by the first manager.
				retrievedVersionMap, err := otherManager.Get(versionedResource)
				if err != nil {
					tl.Fatalf("Error retrieving version map: %v", err)
				}
				if !reflect.DeepEqual(versionMap, retrievedVersionMap) {
					tl.Fatalf("Sync failed to load version from api")
				}
			})

			inSupportedScopeIt("should refresh and update after out-of-band creation", namespaced, func() {
				// Create a second manager and use it to write a version to the api
				otherManager := version.NewVersionManager(client, namespaced, federatedKind, targetKind, namespace)
				otherManager.Sync(stopChan)
				err := otherManager.Update(versionedResource, clusterNames, versionMap)
				if err != nil {
					tl.Fatalf("Error updating version status: %v", err)
				}
				waitForPropVer(tl, adapter, client, versionName, expectedStatus)

				// Ensure that an updated version is written successfully
				clusterNames, versionMap = removeOneCluster(clusterNames, versionMap)
				err = versionManager.Update(versionedResource, clusterNames, versionMap)
				if err != nil {
					tl.Fatalf("Error updating version status: %v", err)
				}
				expectedStatus.ClusterVersions = version.VersionMapToClusterVersions(versionMap)
				waitForPropVer(tl, adapter, client, versionName, expectedStatus)
			})

			inSupportedScopeIt("should refresh and update after out-of-band update", namespaced, func() {
				err := versionManager.Update(versionedResource, clusterNames, versionMap)
				if err != nil {
					tl.Fatalf("Error updating version status: %v", err)
				}
				waitForPropVer(tl, adapter, client, versionName, expectedStatus)

				// Update an annotation out-of-band to ensure a conflict
				propVer := adapter.NewObject()
				err = client.Get(context.TODO(), propVer, versionName.Namespace, versionName.Name)
				if err != nil {
					tl.Errorf("Error retrieving %s %q: %v", versionType, versionName, err)
				}
				metaAccessor, err := meta.Accessor(propVer)
				if err != nil {
					tl.Fatalf("Failed to retrieve meta accessor for %s %q: %s", versionType, versionName, err)
				}
				metaAccessor.SetAnnotations(map[string]string{"foo": "bar"})

				err = client.Update(context.TODO(), propVer)
				if err != nil {
					tl.Errorf("Error updating %s %q: %v", versionType, versionName, err)
				}

				// Ensure that an updated version is written successfully
				clusterNames, versionMap = removeOneCluster(clusterNames, versionMap)
				err = versionManager.Update(versionedResource, clusterNames, versionMap)
				if err != nil {
					tl.Fatalf("Error updating version status: %v", err)
				}
				expectedStatus.ClusterVersions = version.VersionMapToClusterVersions(versionMap)
				waitForPropVer(tl, adapter, client, versionName, expectedStatus)
			})

			inSupportedScopeIt("should recreate after out-of-band deletion", namespaced, func() {
				err := versionManager.Update(versionedResource, clusterNames, versionMap)
				if err != nil {
					tl.Fatalf("Error updating version status: %v", err)
				}
				waitForPropVer(tl, adapter, client, versionName, expectedStatus)

				// Delete the written version out-of-band
				err = client.Delete(context.TODO(), adapter.NewObject(), versionName.Namespace, versionName.Name)
				if err != nil {
					tl.Fatalf("Error deleting %s %q: %v", versionType, versionName, err)
				}

				// Ensure a modified version is written successfully
				clusterNames, versionMap = removeOneCluster(clusterNames, versionMap)
				err = versionManager.Update(versionedResource, clusterNames, versionMap)
				if err != nil {
					tl.Fatalf("Error updating version status: %v", err)
				}
				expectedStatus.ClusterVersions = version.VersionMapToClusterVersions(versionMap)
				waitForPropVer(tl, adapter, client, versionName, expectedStatus)
			})

			inSupportedScopeIt("should add owner reference that ensures garbage collection when template is deleted", namespaced, func() {
				if framework.TestContext.LimitedScope {
					// Garbage collection of a namespaced resource
					// takes an arbitrary amount of time and
					// attempting to verify it in an e2e test is
					// unlikely to be reliable.
					framework.Skipf("Full coverage of owner reference addition is already achieved by testing with cluster-scoped resources")
				}

				err := versionManager.Update(versionedResource, clusterNames, versionMap)
				if err != nil {
					tl.Fatalf("Error updating version status: %v", err)
				}
				waitForPropVer(tl, adapter, client, versionName, expectedStatus)

				// Removal of the template should prompt the garbage
				// collection of the associated version resource due to
				// the owner reference added by the version manager.
				err = adapter.DeleteFederatedObject(fedObjectName)
				if err != nil {
					tl.Fatalf("Error deleting %s %q: %v", adapter.FederatedType(), fedObjectName, err)
				}
				checkForDeletion(tl, adapter.FederatedType(), fedObjectName, func() error {
					_, err := adapter.GetFederatedObject(fedObjectName)
					return err
				})
				checkForDeletion(tl, adapter.TypeName(), versionName, func() error {
					err := client.Get(context.TODO(), adapter.NewObject(), versionName.Namespace, versionName.Name)
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

func waitForPropVer(tl common.TestLogger, adapter testVersionAdapter, client genericclient.Client, qualifiedName util.QualifiedName, expectedStatus fedv1a1.PropagatedVersionStatus) {
	err := wait.PollImmediate(framework.PollInterval, framework.TestContext.SingleCallTimeout, func() (bool, error) {
		propVer := adapter.NewObject()
		err := client.Get(context.TODO(), propVer, qualifiedName.Namespace, qualifiedName.Name)
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
	if !namespaced && framework.TestContext.LimitedScope {
		// Validation of cluster-scoped versioning is not supported for namespaced federation
		PIt(description, f)
	} else {
		It(description, f)
	}
}
