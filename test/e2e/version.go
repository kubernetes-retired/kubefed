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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeclientset "k8s.io/client-go/kubernetes"

	apicommon "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/common"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/sync/version"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	testcommon "github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/e2e/framework"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("VersionManager", func() {
	userAgent := "test-version-manager"

	f := framework.NewFederationFramework(userAgent)

	tl := framework.NewE2ELogger()

	templateKind := "FederatedFoo"

	configMapYAML := `
apiVersion: v1
kind: ConfigMap
metadata:
  generateName: test-version-manager
data:
  foo: bar
`

	var targetKind string
	var namespace string
	var template, override *unstructured.Unstructured
	var versionName string
	var versionManager *version.VersionManager
	var expectedStatus fedv1a1.PropagatedVersionStatus
	var clusterNames []string
	var versionMap map[string]string
	var client fedclientset.Interface
	var kubeClient kubeclientset.Interface
	var qualifiedName util.QualifiedName

	var stopChan chan struct{}

	Context("NamespacedVersionManager", func() {
		BeforeEach(func() {
			// Use a random string for the target kind to ensure test
			// isolation for any test that depends on the state of the
			// API (like the sync test).  The target kind is used by
			// the version manager to prefix version names written to
			// the API.  Prefixing the uuid with an alphanumeric
			// ensures the prefix will be a valid kube name.
			targetKind = fmt.Sprintf("v%s", uuid.New())

			namespace = f.TestNamespaceName()

			clusterNames = []string{"cluster1", "cluster2", "cluster3"}

			// Use a non-template type as the template to avoid having
			// sync controllers add a deletion finalizer to the
			// created template that would complicate validating
			// garbage collection.
			var err error
			template, err = testcommon.ReaderToObj(strings.NewReader(configMapYAML))
			if err != nil {
				tl.Fatalf("Failed to parse template yaml: %v", err)
			}
			template.SetNamespace(namespace)
			content, err := template.MarshalJSON()
			if err != nil {
				tl.Fatalf("Failed to marshall template yaml: %v", err)
			}
			configMap := &corev1.ConfigMap{}
			err = json.Unmarshal(content, configMap)
			if err != nil {
				tl.Fatalf("Failed to unmarshall template: %v", err)
			}

			kubeClient = f.KubeClient("user-agent")

			// Create the template to ensure that created versions
			// will have a valid owner and not be subject to immediate
			// garbage collection.
			createdConfigMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Create(configMap)
			if err != nil {
				tl.Fatalf("Error creating template: %v", err)
			}
			template.SetName(createdConfigMap.Name)
			template.SetUID(createdConfigMap.UID)
			template.SetResourceVersion(createdConfigMap.ResourceVersion)

			versionName = apicommon.PropagatedVersionName(targetKind, template.GetName())
			qualifiedName = util.QualifiedName{Namespace: namespace, Name: versionName}

			versionMap = make(map[string]string)
			for _, name := range clusterNames {
				versionMap[name] = name
			}

			expectedStatus = fedv1a1.PropagatedVersionStatus{
				TemplateVersion: template.GetResourceVersion(),
				OverrideVersion: "",
				ClusterVersions: version.VersionMapToClusterVersions(versionMap),
			}

			client = f.FedClient(userAgent)
			versionManager = version.NewNamespacedVersionManager(client, templateKind, targetKind, namespace)
			stopChan = make(chan struct{})
			// There shouldn't be any api objects to load, but Sync
			// also starts the worker that will write to the API.
			versionManager.Sync(stopChan)
		})

		AfterEach(func() {
			close(stopChan)
			// Ensure removal of the template, which will prompt the
			// removal of owned versions by the garbage collector.
			if template != nil {
				err := kubeClient.CoreV1().ConfigMaps(namespace).Delete(template.GetName(), nil)
				if err != nil && !errors.IsNotFound(err) {
					tl.Errorf("Error deleting ConfigMap %q: %v", qualifiedName, err)
				}
			}
			// Managed fixture doesn't run the garbage collector, so
			// manual cleanup of propagated versions is required.
			if framework.TestContext.TestManagedFederation {
				err := client.CoreV1alpha1().PropagatedVersions(namespace).Delete(versionName, nil)
				if err != nil && !errors.IsNotFound(err) {
					tl.Errorf("Error deleting PropagatedVersion %q: %v", versionName, err)
				}
			}
		})

		It("should create a new version in the API", func() {
			versionManager.Update(template, override, clusterNames, versionMap)
			waitForPropVer(tl, client, qualifiedName, expectedStatus)
		})

		It("should load versions from the API on sync", func() {
			versionManager.Update(template, override, clusterNames, versionMap)
			waitForPropVer(tl, client, qualifiedName, expectedStatus)

			// Create a second manager and sync it
			otherManager := version.NewNamespacedVersionManager(client, templateKind, targetKind, namespace)
			otherManager.Sync(stopChan)

			// Ensure that the second manager loaded the version
			// written by the first manager.
			retrievedVersionMap := otherManager.Get(template, override)
			if !reflect.DeepEqual(versionMap, retrievedVersionMap) {
				tl.Fatalf("Sync failed to load version from api")
			}
		})

		It("should refresh and update after out-of-band creation", func() {
			// Create a second manager and use it to write a version to the api
			otherManager := version.NewNamespacedVersionManager(client, templateKind, targetKind, namespace)
			otherManager.Sync(stopChan)
			otherManager.Update(template, override, clusterNames, versionMap)
			waitForPropVer(tl, client, qualifiedName, expectedStatus)

			// Ensure that an updated version is written successfully
			clusterNames, versionMap = removeOneCluster(clusterNames, versionMap)
			versionManager.Update(template, override, clusterNames, versionMap)
			expectedStatus.ClusterVersions = version.VersionMapToClusterVersions(versionMap)
			waitForPropVer(tl, client, qualifiedName, expectedStatus)
		})

		It("should refresh and update after out-of-band update", func() {
			versionManager.Update(template, override, clusterNames, versionMap)
			waitForPropVer(tl, client, qualifiedName, expectedStatus)

			// Update an annotation out-of-band to ensure a conflict
			annotations := map[string]string{"foo": "bar"}
			propVer, err := client.CoreV1alpha1().PropagatedVersions(namespace).Get(versionName, metav1.GetOptions{})
			if err != nil {
				tl.Errorf("Error retrieving propagated version: %v", err)
			}
			propVer.SetAnnotations(annotations)
			_, err = client.CoreV1alpha1().PropagatedVersions(namespace).Update(propVer)
			if err != nil {
				tl.Errorf("Error updating propagated version: %v", err)
			}

			// Ensure that an updated version is written successfully
			clusterNames, versionMap = removeOneCluster(clusterNames, versionMap)
			versionManager.Update(template, override, clusterNames, versionMap)
			expectedStatus.ClusterVersions = version.VersionMapToClusterVersions(versionMap)
			waitForPropVer(tl, client, qualifiedName, expectedStatus)
		})

		It("should recreate after out-of-band deletion", func() {
			versionManager.Update(template, override, clusterNames, versionMap)
			waitForPropVer(tl, client, qualifiedName, expectedStatus)

			// Delete the written version out-of-band
			err := client.CoreV1alpha1().PropagatedVersions(namespace).Delete(versionName, nil)
			if err != nil {
				tl.Fatalf("Error deleting propagated version: %v", err)
			}

			// Ensure a modified version is written successfully
			clusterNames, versionMap = removeOneCluster(clusterNames, versionMap)
			versionManager.Update(template, override, clusterNames, versionMap)
			expectedStatus.ClusterVersions = version.VersionMapToClusterVersions(versionMap)
			waitForPropVer(tl, client, qualifiedName, expectedStatus)
		})

		It("should add owner reference that ensures garbage collection when template is deleted", func() {
			if framework.TestContext.TestManagedFederation {
				// Test-managed fixture does not run the garbage collector.
				framework.Skipf("Validation of garbage collection is not supported for test-managed federation.")
			}

			versionManager.Update(template, override, clusterNames, versionMap)
			waitForPropVer(tl, client, qualifiedName, expectedStatus)

			// Removal of the template should prompt the garbage
			// collection of the associated version resource due to
			// the owner reference added by the version manager.
			err := kubeClient.CoreV1().ConfigMaps(namespace).Delete(template.GetName(), nil)
			if err != nil {
				tl.Fatalf("Error deleting ConfigMap %q: %v", qualifiedName, err)
			}

			err = wait.PollImmediate(framework.PollInterval, framework.TestContext.SingleCallTimeout, func() (bool, error) {
				_, err = client.CoreV1alpha1().PropagatedVersions(namespace).Get(versionName, metav1.GetOptions{})
				if err == nil {
					return false, nil
				}
				if err != nil && !errors.IsNotFound(err) {
					tl.Errorf("Error retrieving PropagatedVersion %q: %v", qualifiedName, err)
					return false, nil
				}
				return true, nil
			})
			if err != nil {
				tl.Fatalf("Timed out waiting for PropagatedVersion %q to be deleted.", qualifiedName)
			}
		})
	})
})

func waitForPropVer(tl common.TestLogger, client fedclientset.Interface, qualifiedName util.QualifiedName, expectedStatus fedv1a1.PropagatedVersionStatus) {
	err := wait.PollImmediate(framework.PollInterval, framework.TestContext.SingleCallTimeout, func() (bool, error) {
		propVer, err := client.CoreV1alpha1().PropagatedVersions(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return false, nil
		}
		if err != nil {
			tl.Fatalf("Error retrieving status of propagated version %q: %v", qualifiedName, err)
		}
		return util.PropagatedVersionStatusEquivalent(&expectedStatus, &propVer.Status), nil
	})
	if err != nil {
		tl.Fatalf("Timed out waiting for propagated version: %v", err)
	}
}

func removeOneCluster(clusterNames []string, versionMap map[string]string) ([]string, map[string]string) {
	targetName := clusterNames[0]
	clusterNames[0] = clusterNames[len(clusterNames)-1]
	clusterNames = clusterNames[:len(clusterNames)-1]
	delete(versionMap, targetName)
	return clusterNames, versionMap
}
