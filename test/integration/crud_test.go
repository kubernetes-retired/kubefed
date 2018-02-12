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
	"fmt"
	"testing"

	fedv1a1 "github.com/marun/fnord/pkg/apis/federation/v1alpha1"
	"github.com/marun/fnord/test/integration/framework"
	apiv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
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
	_, err = fedClient.FederationV1alpha1().FederatedSecrets(namespace).Create(&fedv1a1.FederatedSecret{
		ObjectMeta: metav1.ObjectMeta{Name: "my-fed-secret"},
		Spec: fedv1a1.FederatedSecretSpec{
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

//  TestClusterRegistration validates registration of a kube api and
//  use of the registration details to access the cluster.
func TestClusterRegistration(t *testing.T) {
	userAgent := "crud-test"
	// Create an api registry
	apiRegistryFixture := framework.SetUpApiRegistryFixture(t)
	defer apiRegistryFixture.TearDown(t)
	crClient := apiRegistryFixture.NewClient(t, userAgent)

	// Create a kube api
	kubeApiFixture := framework.SetUpKubernetesApiFixture(t)
	defer kubeApiFixture.TearDown(t)
	kubeClient := kubeApiFixture.NewClient(t, userAgent)

	// Create a federation api
	fedApiFixture := framework.SetUpFederationApiFixture(t)
	defer fedApiFixture.TearDown(t)
	fedClient := fedApiFixture.NewClient(t, userAgent)

	// Registry the kube api with the api registry
	cluster, err := crClient.ClusterregistryV1alpha1().Clusters().Create(&crv1a1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-cluster-",
		},
		Spec: crv1a1.ClusterSpec{
			KubernetesAPIEndpoints: crv1a1.KubernetesAPIEndpoints{
				ServerEndpoints: []crv1a1.ServerAddressByClientCIDR{
					{
						ClientCIDR: "0.0.0.0",
						// TODO(marun) can this include https:// prefix?
						ServerAddress: kubeApiFixture.Host,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	clusterName := cluster.Name
	namespace := "federation-system"
	kubeConfigKey := "kubeconfig"

	// Create a secret containing the credentials necessary to access the cluster
	// Do not include the host - it will be sourced from the Cluster resource
	config := kubeApiFixture.SecureConfigFixture.NewClientConfig(t, "", userAgent)
	kubeConfig := createKubeConfig(config)

	// Flatten the kubeconfig to ensure that all the referenced file
	// contents are inlined.
	err = clientcmdapi.FlattenConfig(kubeConfig)
	if err != nil {
		t.Fatal(err)
	}
	configBytes, err := clientcmd.Write(*kubeConfig)
	if err != nil {
		t.Fatal(err)
	}

	// Build the secret object with the flattened kubeconfig content.
	// TODO(marun) enforce some kind of relationship between federated cluster and secret?
	secret, err := kubeClient.CoreV1().Secrets(namespace).Create(&apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-credentials", clusterName),
			Namespace:    namespace,
		},
		Data: map[string][]byte{
			kubeConfigKey: configBytes,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	secretName := secret.Name

	// Create a federated cluster resource that associates the cluster and secret
	_, err = fedClient.FederationV1alpha1().FederatedClusters().Create(&fedv1a1.FederatedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
		},
		Spec: fedv1a1.FederatedClusterSpec{
			ClusterRef: apiv1.LocalObjectReference{
				Name: clusterName,
			},
			SecretRef: &apiv1.LocalObjectReference{
				Name: secret.Name,
			},
		},
	})

	// Create a client from the registered details
	kubeConfigFromSecret, ok := secret.Data[kubeConfigKey]
	if !ok || len(kubeConfigFromSecret) == 0 {
		t.Fatalf("Secret \"%s/%s\" for cluster %q has no value for key %q", namespace, secretName, clusterName, kubeConfigKey)
	}
	clientConfig, err := clientcmd.Load(kubeConfigFromSecret)
	if err != nil {
		t.Fatal(err)
	}
	// Set the host here to reflect that it will be sourced from the
	// cluster registry resource rather than the federated cluster
	// resource.
	overrides := &clientcmd.ConfigOverrides{
		ClusterInfo: clientcmdapi.Cluster{
			Server: kubeApiFixture.Host,
		},
	}
	restConfig, err := clientcmd.NewDefaultClientConfig(*clientConfig, overrides).ClientConfig()
	if err != nil {
		t.Fatal(err)
	}

	clusterClient := kubeclientset.NewForConfigOrDie(restConfig)

	// Test the client
	_, err = clusterClient.CoreV1().Secrets(namespace).Create(&apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-secret-",
			Namespace:    namespace,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func createKubeConfig(clientCfg *rest.Config) *clientcmdapi.Config {
	clusterNick := "cluster"
	userNick := "user"
	contextNick := "context"

	config := clientcmdapi.NewConfig()

	credentials := clientcmdapi.NewAuthInfo()
	credentials.Token = clientCfg.BearerToken
	credentials.ClientCertificate = clientCfg.TLSClientConfig.CertFile
	if len(credentials.ClientCertificate) == 0 {
		credentials.ClientCertificateData = clientCfg.TLSClientConfig.CertData
	}
	credentials.ClientKey = clientCfg.TLSClientConfig.KeyFile
	if len(credentials.ClientKey) == 0 {
		credentials.ClientKeyData = clientCfg.TLSClientConfig.KeyData
	}
	config.AuthInfos[userNick] = credentials

	cluster := clientcmdapi.NewCluster()
	cluster.Server = clientCfg.Host
	cluster.CertificateAuthority = clientCfg.CAFile
	if len(cluster.CertificateAuthority) == 0 {
		cluster.CertificateAuthorityData = clientCfg.CAData
	}
	cluster.InsecureSkipTLSVerify = clientCfg.Insecure
	config.Clusters[clusterNick] = cluster

	context := clientcmdapi.NewContext()
	context.Cluster = clusterNick
	context.AuthInfo = userNick
	config.Contexts[contextNick] = context
	config.CurrentContext = contextNick

	return config
}
