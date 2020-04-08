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

package federate_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/kubefed/pkg/kubefedctl/federate"
)

func TestFederateResources(t *testing.T) {
	var resource = &unstructured.Unstructured{}
	resource.Object = map[string]interface{}{
		"name": "name",
		"spec": map[string]interface{}{
			"replicas": "2",
			"selector": map[string]interface{}{
				"matchLabels": map[string]interface{}{
					"foo": "bar",
				},
			},
			"template": map[string]interface{}{
				"labels": map[string]interface{}{
					"foo": "bar",
				},
			},
		},
	}
	resource.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apps",
		Kind:    "Deployment",
		Version: "v1",
	})

	t.Run("TestNonNameSpacedDeployment", func(t *testing.T) {
		federatedResources, err := federate.FederateResources([]*unstructured.Unstructured{resource})
		assert.NoError(t, err, "Should not expect any errorr")
		assert.Len(t, federatedResources, 1, "Should return a federated resource")

		federatedResource := federatedResources[0]

		assert.Empty(t, federatedResource.GetNamespace(), "Should not return a namespaces if not set")
		assert.Equal(t, "FederatedDeployment", federatedResource.GetKind(), "Federated Resources should return a federated crd")
		assert.Equal(t, "types.kubefed.io/v1beta1", federatedResource.GetAPIVersion(), "federated resourece should return correct api-version")
		federatedSpec := federatedResource.Object["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"]
		assert.Equal(t, resource.Object["spec"], federatedSpec)
	})

	t.Run("TestNameSpacedDeployment", func(t *testing.T) {
		testNS := "testNS"
		resource.SetNamespace(testNS)
		federatedResources, err := federate.FederateResources([]*unstructured.Unstructured{resource})
		assert.NoError(t, err, "Should not expect any errorr")
		assert.Len(t, federatedResources, 1, "Should return a federated resource")

		federatedResource := federatedResources[0]
		assert.Equal(t, testNS, federatedResource.GetNamespace(), "A namespaces should be set and match orignal resource")
		assert.Equal(t, "FederatedDeployment", federatedResource.GetKind(), "Federated Resources should return a federated crd")
		assert.Equal(t, "types.kubefed.io/v1beta1", federatedResource.GetAPIVersion(), "federated resourece should return correct api-version")
		federatedSpec := federatedResource.Object["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"]
		assert.Equal(t, resource.Object["spec"], federatedSpec)
	})

}
