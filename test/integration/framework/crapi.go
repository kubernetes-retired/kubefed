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
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextcs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
	crv1alpha1 "k8s.io/cluster-registry/pkg/client/clientset/versioned/typed/clusterregistry/v1alpha1"
)

// ClusterRegistryApiFixture manages a api registry apiserver
type ClusterRegistryApiFixture struct {
	KubeAPIFixture *KubernetesApiFixture
	CrdClient      *crv1alpha1.ClusterregistryV1alpha1Client
	CRD            []*v1beta1.CustomResourceDefinition
}

func SetUpClusterRegistryApiFixture(tl common.TestLogger) *ClusterRegistryApiFixture {
	f := &ClusterRegistryApiFixture{}
	f.setUp(tl)
	return f
}

func (f *ClusterRegistryApiFixture) setUp(tl common.TestLogger) {
	defer TearDownOnPanic(tl, f)
	f.KubeAPIFixture = SetUpKubernetesApiFixture(tl)
	f.CRD = []*v1beta1.CustomResourceDefinition{&v1alpha1.ClusterCRD}
}

func (f *ClusterRegistryApiFixture) TearDown(tl common.TestLogger) {
	if f.KubeAPIFixture != nil {
		f.KubeAPIFixture.TearDown(tl)
		f.KubeAPIFixture = nil
	}
}

func (f *ClusterRegistryApiFixture) NewClient(tl common.TestLogger, userAgent string) *crv1alpha1.ClusterregistryV1alpha1Client {
	kubeConfig := f.NewConfig(tl)
	rest.AddUserAgent(kubeConfig, userAgent)

	clientset, err := apiextcs.NewForConfig(kubeConfig)
	_, err = clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(f.CRD[0])
	client, err := crv1alpha1.NewForConfig(kubeConfig)
	if err != nil {
		tl.Fatalf("Error creating crd: %v", err)
	}

	err = wait.PollImmediate(DefaultWaitInterval, wait.ForeverTestTimeout, func() (bool, error) {
		_, err := client.Clusters("invalid").Get("invalid", metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return true, nil
		}
		return (err == nil), err
	})
	if err != nil {
		tl.Fatalf("Error waiting for cluster-registry crd to become established: %v", err)
	}

	return client
}

func (f *ClusterRegistryApiFixture) NewConfig(tl common.TestLogger) *rest.Config {
	return f.KubeAPIFixture.NewConfig(tl)
}
