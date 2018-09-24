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
	"reflect"

	"github.com/pborman/uuid"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"

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
	userAgent := "version-manager"

	f := framework.NewFederationFramework(userAgent)

	tl := framework.NewE2ELogger()

	// Picking the first item is likely to pick ConfigMap due to
	// alphabetical sorting, but any type config will do.
	typeConfig := testcommon.TypeConfigsOrDie(tl)[0]

	var templateKind string
	var targetKind string
	var namespace string
	var template, override *unstructured.Unstructured
	var versionName string
	var versionManager *version.VersionManager
	var expectedStatus fedv1a1.PropagatedVersionStatus
	var clusterNames []string
	var versionMap map[string]string
	var client fedclientset.Interface
	var qualifiedName util.QualifiedName

	var stopChan chan struct{}

	Context("NamespacedVersionManager", func() {
		BeforeEach(func() {
			templateKind = typeConfig.GetTemplate().Kind

			// Use a random string for the target kind to ensure test
			// isolation for any test that depends on the state of the
			// API (like the sync test).  The target kind is used by
			// the version manager to prefix version names written to
			// the API.  Prefixing the uuid with an alphanumeric
			// ensures the prefix will be a valid kube name.
			targetKind = fmt.Sprintf("v%s", uuid.New())

			namespace = f.TestNamespaceName()

			clusterNames = []string{"cluster1", "cluster2", "cluster3"}

			var err error
			template, _, override, err = testcommon.NewTestObjects(typeConfig, namespace, clusterNames)
			if err != nil {
				tl.Fatalf("Failed to retrieve test objects for %q: %v", typeConfig.GetTarget().Name, err)
			}
			template.SetName(uuid.New())
			template.SetResourceVersion("templateVer")
			override.SetResourceVersion("overrideVer")

			versionName = apicommon.PropagatedVersionName(targetKind, template.GetName())
			qualifiedName = util.QualifiedName{Namespace: namespace, Name: versionName}

			versionMap = make(map[string]string)
			for _, name := range clusterNames {
				versionMap[name] = name
			}

			expectedStatus = fedv1a1.PropagatedVersionStatus{
				TemplateVersion: template.GetResourceVersion(),
				OverrideVersion: override.GetResourceVersion(),
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
			err := client.CoreV1alpha1().PropagatedVersions(namespace).Delete(versionName, nil)
			if err != nil && !errors.IsNotFound(err) {
				tl.Fatalf("Error deleting %q: %v", versionName, err)
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

		It("should delete a version from the API", func() {
			versionManager.Update(template, override, clusterNames, versionMap)
			waitForPropVer(tl, client, qualifiedName, expectedStatus)

			// Deletion is performed synchronously, so it's not necessary to poll
			err := versionManager.Delete(util.QualifiedName{Namespace: namespace, Name: template.GetName()})
			if err != nil {
				tl.Fatalf("Error deleting propagated version: %v", err)
			}
			_, err = client.CoreV1alpha1().PropagatedVersions(namespace).Get(versionName, metav1.GetOptions{})
			if err == nil {
				tl.Fatalf("Version manager did not delete propagated version.")
			} else if err != nil && !errors.IsNotFound(err) {
				tl.Fatalf("Error retrieving deleted version: %v", err)
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
