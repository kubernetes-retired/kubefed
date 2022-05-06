/*
Copyright 2021 The Kubernetes Authors.

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
	. "github.com/onsi/ginkgo" //nolint:stylecheck
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/kubefed/test/common"
	"sigs.k8s.io/kubefed/test/e2e/framework"
)

var typeConfigName = "deployments.apps"

var _ = Describe("DeleteOptions", func() {
	f := framework.NewKubeFedFramework("delete-options")

	tl := framework.NewE2ELogger()

	typeConfigFixtures := common.TypeConfigFixturesOrDie(tl)

	fixture := typeConfigFixtures[typeConfigName]

	It("Deployment should be created and deleted successfully, but ReplicaSet that created by Deployment won't be deleted", func() {

		typeConfig, testObjectsFunc := getCrudTestInput(f, tl, typeConfigName, fixture)
		crudTester, targetObject, overrides := initCrudTest(f, tl, f.KubeFedSystemNamespace(), typeConfig, testObjectsFunc)
		fedObject := crudTester.CheckCreate(targetObject, overrides, nil)

		By("Set PropagationPolicy property as 'Orphan' on the DeleteOptions for Federated Deployment")
		orphan := metav1.DeletePropagationOrphan
		prop := client.PropagationPolicy(orphan)

		crudTester.SetDeleteOption(fedObject, prop)

		crudTester.CheckDelete(fedObject, false)

		By("Checking ReplicatSet stutus for every cluster")
		crudTester.CheckReplicaSet(targetObject)
	})
})
