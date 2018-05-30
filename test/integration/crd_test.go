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
	"strings"
	"testing"

	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/integration/framework"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

func TestCrd(t *testing.T) {
	tl := framework.NewIntegrationLogger(t)
	kubeApi := framework.SetUpKubernetesApiFixture(tl)
	defer kubeApi.TearDown(tl)

	userAgent := "test-crd"

	kubeConfig := kubeApi.NewConfig(tl)
	rest.AddUserAgent(kubeConfig, userAgent)

	pool := dynamic.NewDynamicClientPool(kubeConfig)
	crdApiResource := &metav1.APIResource{
		Group:      "apiextensions.k8s.io",
		Version:    "v1beta1",
		Name:       "customresourcedefinitions",
		Namespaced: false,
	}
	crdClient, err := util.NewResourceClient(pool, crdApiResource)

	data := `
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: mycrds.example.com
spec:
  group: example.com
  version: v1alpha1
  scope: Namespaced
  names:
    plural: mycrds
    singular: mycrd
    kind: MyCrd
`
	crd, err := common.ReaderToObj(strings.NewReader(data))
	if err != nil {
		tl.Fatalf("Error loading test object: %v", err)
	}

	crd, err = crdClient.Resources("").Create(crd)
	if err != nil {
		tl.Fatalf("Error creating crd: %v", err)
	}

	apiResource := &metav1.APIResource{
		Group:      "example.com",
		Version:    "v1alpha1",
		Kind:       "MyCrd",
		Name:       "mycrds",
		Namespaced: true,
	}

	client, err := util.NewResourceClient(pool, apiResource)
	if err != nil {
		tl.Fatalf("Error creating client for crd %q: %v", apiResource.Kind, err)
	}

	// Wait for crd api to become available
	err = wait.PollImmediate(framework.DefaultWaitInterval, wait.ForeverTestTimeout, func() (bool, error) {
		_, err := client.Resources("invalid").Get("invalid", metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return true, nil
		}
		return (err == nil), err
	})
	if err != nil {
		tl.Fatalf("Error waiting for crd %q to become established: %v", apiResource.Kind, err)
	}

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       "MyCrd",
			"apiVersion": "example.com/v1alpha1",
			"metadata": map[string]interface{}{
				"namespace": "foo",
				"name":      "bar",
			},
		},
	}
	_, err = client.Resources("foo").Create(obj)
	if err != nil {
		tl.Fatalf("Error creating crd resource: %v", err)
	}
}
