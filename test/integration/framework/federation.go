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
	"time"

	"github.com/kubernetes-sigs/kubebuilder/pkg/install"

	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/inject"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	apiv1 "k8s.io/api/core/v1"
	apiextv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	crv1a1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
	crclientset "k8s.io/cluster-registry/pkg/client/clientset/versioned"
)

// TODO(marun) In fedv1 namespace cleanup required that a kube api
// fixture run a namespace controller to ensure cleanup on deletion.
// Will this be required?

const userAgent = "federation-framework"

type InstallStrategy struct {
	install.EmptyInstallStrategy
	crds []*extensionsv1beta1.CustomResourceDefinition
}

func (s *InstallStrategy) GetCRDs() []*extensionsv1beta1.CustomResourceDefinition {
	return s.crds
}

// FederationFixture manages servers for kube, cluster registry and
// federation along with a set of member clusters.
type FederationFixture struct {
	KubeApi           *KubernetesApiFixture
	Clusters          map[string]*KubernetesApiFixture
	ClusterController *ControllerFixture
	SystemNamespace   string
}

func SetUpFederationFixture(tl common.TestLogger, clusterCount int) *FederationFixture {
	if clusterCount < 1 {
		tl.Fatal("Cluster count must be greater than 0")
	}
	tl.Logf("Starting a federation of %d clusters...", clusterCount)
	f := &FederationFixture{}
	f.setUp(tl, clusterCount)
	return f
}

func (f *FederationFixture) setUp(tl common.TestLogger, clusterCount int) {
	defer TearDownOnPanic(tl, f)

	f.Clusters = make(map[string]*KubernetesApiFixture)
	for i := 0; i < clusterCount; i++ {
		clusterName := f.AddMemberCluster(tl)
		tl.Logf("Added cluster %s to the federation", clusterName)
	}

	config := f.KubeApi.NewConfig(tl)

	// TODO(marun) Consider running the cluster controller as soon as
	// the kube api is available to speed up setting cluster status.
	tl.Logf("Starting cluster controller")
	f.ClusterController = NewClusterControllerFixture(config, f.SystemNamespace, f.SystemNamespace)
	tl.Log("Federation started.")
}

func (f *FederationFixture) TearDown(tl common.TestLogger) {
	// Stop the cluster controller first to avoid spurious connection
	// errors when the target urls become unavailable.
	fixtures := []TestFixture{
		f.ClusterController,
		// KubeApi will be torn down via f.Clusters
	}
	for _, cluster := range f.Clusters {
		fixtures = append(fixtures, cluster)
	}
	for _, fixture := range fixtures {
		fixture.TearDown(tl)
		// Blocking IO to give cluster controller go routine the opportunity to
		// shut down after closing its stop channel before API server is shut
		// down. This helps avoid spurious connection errors when the target
		// URLs become unavailable in API server.
		time.Sleep(100 * time.Millisecond)
	}
}

// AddCluster adds a new member cluster to the federation.
func (f *FederationFixture) AddMemberCluster(tl common.TestLogger) string {
	kubeApi := SetUpKubernetesApiFixture(tl)

	// Pick the first added cluster to be the primary
	if f.KubeApi == nil {
		f.KubeApi = kubeApi
		f.KubeApi.IsPrimary = true
		f.ensureNamespace(tl)
		f.installCrds(tl)
	}

	clusterName := f.registerCluster(tl, kubeApi.Host)

	secretName := f.createSecret(tl, kubeApi, clusterName)
	f.createFederatedCluster(tl, clusterName, secretName)

	// Track clusters by name
	f.Clusters[clusterName] = kubeApi

	return clusterName
}

// registerCluster registers a cluster with the cluster registry
func (f *FederationFixture) registerCluster(tl common.TestLogger, host string) string {
	// Registry the kube api with the cluster registry
	crClient := f.NewCrClient(tl, userAgent)
	cluster, err := crClient.ClusterregistryV1alpha1().Clusters(f.SystemNamespace).Create(&crv1a1.Cluster{
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
		tl.Fatal(err)
	}
	return cluster.Name
}

// createSecret creates a secret resource containing the credentials
// necessary to access the fixture-managed cluster.
func (f *FederationFixture) createSecret(tl common.TestLogger, clusterFixture *KubernetesApiFixture, clusterName string) string {
	// Do not include the host - it will need to be sourced from the
	// Cluster resource.
	config := &rest.Config{}
	kubeConfig := CreateKubeConfig(config)

	// Flatten the kubeconfig to ensure that all the referenced file
	// contents are inlined.
	err := clientcmdapi.FlattenConfig(kubeConfig)
	if err != nil {
		tl.Fatal(err)
	}
	configBytes, err := clientcmd.Write(*kubeConfig)
	if err != nil {
		tl.Fatal(err)
	}

	// Build the secret object with the flattened kubeconfig content.
	// TODO(marun) enforce some kind of relationship between federated cluster and secret?
	kubeClient := f.KubeApi.NewClient(tl, userAgent)
	secret, err := kubeClient.CoreV1().Secrets(f.SystemNamespace).Create(&apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-credentials", clusterName),
			Namespace:    f.SystemNamespace,
		},
		Data: map[string][]byte{
			util.KubeconfigSecretDataKey: configBytes,
		},
	})
	if err != nil {
		tl.Fatal(err)
	}
	return secret.Name
}

// createFederatedCluster create a federated cluster resource that
// associates the cluster and secret.
func (f *FederationFixture) createFederatedCluster(tl common.TestLogger, clusterName, secretName string) {
	fedClient := f.NewFedClient(tl, userAgent)
	_, err := fedClient.CoreV1alpha1().FederatedClusters(f.SystemNamespace).Create(&fedv1a1.FederatedCluster{
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
		tl.Fatal(err)
	}
}

func (f *FederationFixture) NewFedClient(tl common.TestLogger, userAgent string) fedclientset.Interface {
	config := f.KubeApi.NewConfig(tl)
	rest.AddUserAgent(config, userAgent)
	return fedclientset.NewForConfigOrDie(config)
}

func (f *FederationFixture) NewCrClient(tl common.TestLogger, userAgent string) crclientset.Interface {
	config := f.KubeApi.NewConfig(tl)
	rest.AddUserAgent(config, userAgent)
	return crclientset.NewForConfigOrDie(config)
}

func (f *FederationFixture) ClusterDynamicClients(tl common.TestLogger, apiResource *metav1.APIResource, userAgent string) map[string]common.TestCluster {
	clientMap := make(map[string]common.TestCluster)
	for name, cluster := range f.Clusters {
		config := cluster.NewConfig(tl)
		rest.AddUserAgent(config, userAgent)
		client, err := util.NewResourceClientFromConfig(config, apiResource)
		if err != nil {
			tl.Fatalf("Error creating a resource client in cluster %q for kind %q: %v", name, apiResource.Kind, err)
		}
		clientMap[name] = common.TestCluster{
			client,
			cluster.IsPrimary,
		}
	}
	return clientMap
}

func (f *FederationFixture) ClusterKubeClients(tl common.TestLogger, userAgent string) map[string]kubeclientset.Interface {
	clientMap := make(map[string]kubeclientset.Interface)
	for name, cluster := range f.Clusters {
		config := cluster.NewConfig(tl)
		rest.AddUserAgent(config, userAgent)
		client, err := kubeclientset.NewForConfig(config)
		if err != nil {
			tl.Fatalf("Error creating a resource client in cluster %q: %v", name, err)
		}
		clientMap[name] = client
	}
	return clientMap
}

func (f *FederationFixture) ClusterNames() []string {
	clusterNames := []string{}
	for name, _ := range f.Clusters {
		clusterNames = append(clusterNames, name)
	}
	return clusterNames
}

func (f *FederationFixture) installCrds(tl common.TestLogger) {
	config := f.KubeApi.NewConfig(tl)
	installer := install.NewInstaller(config)

	tl.Logf("Creating Cluster Registry CRD")
	crds := []*apiextv1b1.CustomResourceDefinition{&crv1a1.ClusterCRD}
	err := installer.Install(&InstallStrategy{crds: crds})
	if err != nil {
		tl.Fatalf("Could not create Cluster Registry CRD: %v", err)
	}

	tl.Logf("Creating Federation CRDs")
	err = installer.Install(&InstallStrategy{crds: inject.Injector.CRDs})
	if err != nil {
		tl.Fatalf("Could not create Federation CRDs: %v", err)
	}

	crds = append(crds, inject.Injector.CRDs...)
	for _, crd := range inject.Injector.CRDs {
		waitForCrd(tl, config, crd)
	}
}

func (f *FederationFixture) ensureNamespace(tl common.TestLogger) {
	client := f.KubeApi.NewClient(tl, "federation-fixture")
	systemNamespace, err := client.Core().Namespaces().Create(&apiv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: util.DefaultFederationSystemNamespace + "-",
		},
	})
	if err != nil {
		tl.Fatalf("Error creating federation system namespace: %v", err)
	}
	f.SystemNamespace = systemNamespace.Name
}

func waitForCrd(tl common.TestLogger, config *rest.Config, crd *apiextv1b1.CustomResourceDefinition) {
	apiResource := &metav1.APIResource{
		Group:      crd.Spec.Group,
		Version:    crd.Spec.Version,
		Kind:       crd.Spec.Names.Kind,
		Name:       crd.Spec.Names.Plural,
		Namespaced: crd.Spec.Scope == apiextv1b1.NamespaceScoped,
	}

	client, err := util.NewResourceClientFromConfig(config, apiResource)
	if err != nil {
		tl.Fatalf("Error creating client for crd %q: %v", apiResource.Kind, err)
	}
	// Wait for crd api to become available
	err = wait.PollImmediate(DefaultWaitInterval, wait.ForeverTestTimeout, func() (bool, error) {
		_, err := client.Resources("invalid").Get("invalid", metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return true, nil
		}
		return (err == nil), err

	})
	if err != nil {
		tl.Fatalf("Error waiting for crd %q to become established: %v", apiResource.Kind, err)
	}
}
