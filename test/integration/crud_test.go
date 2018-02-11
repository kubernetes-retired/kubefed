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

package integration

import (
	"testing"

	fedapiv1a1 "github.com/marun/fnord/pkg/apis/federation/v1alpha1"
	"github.com/marun/fnord/test/integration/framework"
	apiv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	crv1a1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

// TestCrud validates create/read/update/delete operations for federated types.
func TestCrud(t *testing.T) {
	namespace := "foo"

	kubeApiFixture := framework.SetUpKubernetesApiFixture(t)
	defer kubeApiFixture.TearDown(t)

	kubeClient := kubeApiFixture.NewClient(t, "crud-test")
	_, err := kubeClient.CoreV1().Secrets(namespace).Create(&apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-secret-",
			Namespace:    namespace,
		},
		Data: map[string][]byte{
			"A": []byte("ala ma kota"),
		},
		Type: apiv1.SecretTypeOpaque,
	})
	if err != nil {
		t.Fatal(err)
	}

	fedApiFixture := framework.SetUpFederationApiFixture(t)
	defer fedApiFixture.TearDown(t)

	fedClient := fedApiFixture.NewClient(t, "crud-test")
	_, err = fedClient.FederationV1alpha1().FederatedSecrets(namespace).Create(&fedapiv1a1.FederatedSecret{
		ObjectMeta: metav1.ObjectMeta{Name: "my-fed-secret"},
		Spec: fedapiv1a1.FederatedSecretSpec{
			Template: corev1.Secret{},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

}

//  TestClusterRegisterApi exercises the cluster registry api.
func TestClusterRegistryApi(t *testing.T) {
	apiRegistryFixture := framework.SetUpApiRegistryFixture(t)
	defer apiRegistryFixture.TearDown(t)

	crClient := apiRegistryFixture.NewClient(t, "crud-test")
	_, err := crClient.ClusterregistryV1alpha1().Clusters().Create(&crv1a1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-cluster-",
		},
		Spec: crv1a1.ClusterSpec{
			KubernetesAPIEndpoints: crv1a1.KubernetesAPIEndpoints{
				ServerEndpoints: []crv1a1.ServerAddressByClientCIDR{
					{
						ClientCIDR:    "0.0.0.0",
						ServerAddress: "192.168.0.1:53332",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}
