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

package framework

import (
	"testing"

	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

//  TestClusterRegistration validates registration of a kube api and
//  use of the registration details to access the cluster.
func TestClusterRegistration(t *testing.T) {
	f := SetUpFederationFixture(t, 1)
	defer f.TearDown(t)

	// Get the name of a cluster
	var clusterName string
	for clusterName == "" {
		for key, _ := range f.Clusters {
			clusterName = key
			break
		}
	}

	userAgent := "crud-test"

	// Retrieve the federated cluster resource
	fedClient := f.FedApi.NewClient(t, userAgent)
	federatedCluster, err := fedClient.FederationV1alpha1().FederatedClusters().Get(clusterName, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// Retrieve the secret resource
	secretRef := federatedCluster.Spec.SecretRef
	if secretRef == nil {
		t.Fatalf("Secret ref for cluster %s is unexpectedly nil", clusterName)
	}
	kubeClient := f.KubeApi.NewClient(t, userAgent)
	secret, err := kubeClient.CoreV1().Secrets(util.FederationSystemNamespace).Get(secretRef.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// Load the config from the secret
	kubeConfigFromSecret, ok := secret.Data[util.KubeconfigSecretDataKey]
	if !ok || len(kubeConfigFromSecret) == 0 {
		t.Fatalf("Secret \"%s/%s\" for cluster %q has no value for key %q", util.FederationSystemNamespace, secret.Name, clusterName, util.KubeconfigSecretDataKey)
	}
	clientConfig, err := clientcmd.Load(kubeConfigFromSecret)
	if err != nil {
		t.Fatal(err)
	}

	// Retrieve the cluster resource
	// TODO(marun) does it make sense to even have a cluster ref?  When will the names differ
	crClient := f.CrApi.NewClient(t, userAgent)
	cluster, err := crClient.ClusterregistryV1alpha1().Clusters().Get(clusterName, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	host := cluster.Spec.KubernetesAPIEndpoints.ServerEndpoints[0].ServerAddress

	// Set the host here to reflect that it will be sourced from the
	// cluster registry resource rather than the federated cluster
	// resource.
	overrides := &clientcmd.ConfigOverrides{
		ClusterInfo: clientcmdapi.Cluster{
			Server: host,
		},
	}
	restConfig, err := clientcmd.NewDefaultClientConfig(*clientConfig, overrides).ClientConfig()
	if err != nil {
		t.Fatal(err)
	}

	clusterClient := kubeclientset.NewForConfigOrDie(restConfig)

	// Test the client
	_, err = clusterClient.CoreV1().Secrets("foo").Create(&apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-secret-",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}
