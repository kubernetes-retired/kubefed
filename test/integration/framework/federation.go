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
	"fmt"
	"testing"

	fedcommon "github.com/marun/fnord/pkg/apis/federation/common"
	fedv1a1 "github.com/marun/fnord/pkg/apis/federation/v1alpha1"
	"github.com/marun/fnord/pkg/controller/util"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	crv1a1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

// TODO(marun) In fedv1 namespace cleanup required that a kube api
// fixture run a namespace controller to ensure cleanup on deletion.
// Will this be required?

const userAgent = "federation-framework"

// FederationFixture manages servers for kube, cluster registry and
// federation along with a set of member clusters.
type FederationFixture struct {
	KubeApi *KubernetesApiFixture
	CrApi   *ClusterRegistryApiFixture
	FedApi  *FederationApiFixture

	Clusters map[string]*KubernetesApiFixture
}

func SetUpFederationFixture(t *testing.T, clusterCount int) *FederationFixture {
	if clusterCount < 1 {
		t.Fatal("Cluster count must be greater than 0")
	}
	t.Logf("Starting a federation of %d clusters...", clusterCount)
	f := &FederationFixture{}
	f.setUp(t, clusterCount)
	return f
}

func (f *FederationFixture) setUp(t *testing.T, clusterCount int) {
	defer TearDownOnPanic(t, f)

	f.CrApi = SetUpClusterRegistryApiFixture(t)
	f.FedApi = SetUpFederationApiFixture(t)

	f.Clusters = make(map[string]*KubernetesApiFixture)
	for i := 0; i < clusterCount; i++ {
		clusterName := f.AddMemberCluster(t)
		t.Logf("Added cluster %s to the federation", clusterName)
	}
	t.Log("Federation started.")
}

func (f *FederationFixture) TearDown(t *testing.T) {
	fixtures := []TestFixture{
		// KubeApi will be torn down via f.Clusters
		f.CrApi,
		f.FedApi,
	}
	for _, cluster := range f.Clusters {
		fixtures = append(fixtures, cluster)
	}
	for _, fixture := range fixtures {
		fixture.TearDown(t)
	}
}

// AddCluster adds a new member cluster to the federation.
func (f *FederationFixture) AddMemberCluster(t *testing.T) string {
	kubeApi := SetUpKubernetesApiFixture(t)

	// Pick the first added cluster to be the primary
	if f.KubeApi == nil {
		f.KubeApi = kubeApi
	}

	clusterName := f.registerCluster(t, kubeApi.Host)
	secretName := f.createSecret(t, kubeApi, clusterName)
	f.createFederatedCluster(t, clusterName, secretName)

	// Track clusters by name
	f.Clusters[clusterName] = kubeApi

	return clusterName
}

// registerCluster registers a cluster with the cluster registry
func (f *FederationFixture) registerCluster(t *testing.T, host string) string {
	// Registry the kube api with the cluster registry
	crClient := f.CrApi.NewClient(t, userAgent)
	cluster, err := crClient.ClusterregistryV1alpha1().Clusters().Create(&crv1a1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-cluster-",
		},
		Spec: crv1a1.ClusterSpec{
			KubernetesAPIEndpoints: crv1a1.KubernetesAPIEndpoints{
				ServerEndpoints: []crv1a1.ServerAddressByClientCIDR{
					{
						ClientCIDR:    "0.0.0.0/0",
						ServerAddress: host,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return cluster.Name
}

// createSecret creates a secret resource containing the credentials
// necessary to access the fixture-managed cluster.
func (f *FederationFixture) createSecret(t *testing.T, clusterFixture *KubernetesApiFixture, clusterName string) string {
	// Do not include the host - it will need to be sourced from the
	// Cluster resource.
	config := clusterFixture.SecureConfigFixture.NewClientConfig(t, "", userAgent)
	kubeConfig := CreateKubeConfig(config)

	// Flatten the kubeconfig to ensure that all the referenced file
	// contents are inlined.
	err := clientcmdapi.FlattenConfig(kubeConfig)
	if err != nil {
		t.Fatal(err)
	}
	configBytes, err := clientcmd.Write(*kubeConfig)
	if err != nil {
		t.Fatal(err)
	}

	// Build the secret object with the flattened kubeconfig content.
	// TODO(marun) enforce some kind of relationship between federated cluster and secret?
	kubeClient := f.KubeApi.NewClient(t, userAgent)
	secret, err := kubeClient.CoreV1().Secrets(util.FederationSystemNamespace).Create(&apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-credentials", clusterName),
			Namespace:    util.FederationSystemNamespace,
		},
		Data: map[string][]byte{
			util.KubeconfigSecretDataKey: configBytes,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return secret.Name
}

// createFederatedCluster create a federated cluster resource that
// associates the cluster and secret.
func (f *FederationFixture) createFederatedCluster(t *testing.T, clusterName, secretName string) {
	fedClient := f.FedApi.NewClient(t, userAgent)
	cluster, err := fedClient.FederationV1alpha1().FederatedClusters().Create(&fedv1a1.FederatedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
		},
		Spec: fedv1a1.FederatedClusterSpec{
			ClusterRef: apiv1.LocalObjectReference{
				Name: clusterName,
			},
			SecretRef: &apiv1.LocalObjectReference{
				Name: secretName,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// TODO(marun) Rely on the cluster controller to set status rather than setting it manually.
	currentTime := metav1.Now()
	cluster.Status = fedv1a1.FederatedClusterStatus{
		Conditions: []fedv1a1.ClusterCondition{
			fedv1a1.ClusterCondition{
				Type:               fedcommon.ClusterReady,
				Status:             apiv1.ConditionTrue,
				Reason:             "ClusterReady",
				Message:            "/healthz responded with ok",
				LastProbeTime:      currentTime,
				LastTransitionTime: currentTime,
			},
		},
	}
	_, err = fedClient.FederationV1alpha1().FederatedClusters().UpdateStatus(cluster)
	if err != nil {
		t.Fatal(err)
	}
}

func (f *FederationFixture) ClusterClients(t *testing.T, userAgent string) []clientset.Interface {
	clients := []clientset.Interface{}
	for _, cluster := range f.Clusters {
		client := cluster.NewClient(t, userAgent)
		clients = append(clients, client)
	}
	return clients
}
