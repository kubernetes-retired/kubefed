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

package v1alpha1_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	. "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned/typed/core/v1alpha1"
	"testing"
)

// EDIT THIS FILE!
// Created by "kubebuilder create resource" for you to implement the FederatedTypeConfig resource tests

var _ = Describe("FederatedTypeConfig", func() {
	var instance FederatedTypeConfig
	var expected FederatedTypeConfig
	var client FederatedTypeConfigInterface

	BeforeEach(func() {
		instance = FederatedTypeConfig{}
		instance.Name = "instance-1"

		expected = instance
	})

	AfterEach(func() {
		client.Delete(instance.Name, &metav1.DeleteOptions{})
	})

	// INSERT YOUR CODE HERE - add more "Describe" tests

	// Automatically created storage tests
	Describe("when sending a storage request", func() {
		Context("for a valid config", func() {
			It("should provide CRUD access to the object", func() {
				client = cs.CoreV1alpha1().FederatedTypeConfigs("default")

				By("returning success from the create request")
				actual, err := client.Create(&instance)
				Expect(err).ShouldNot(HaveOccurred())

				By("defaulting the expected fields")
				Expect(actual.Spec).To(Equal(expected.Spec))

				By("returning the item for list requests")
				result, err := client.List(metav1.ListOptions{})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(result.Items).To(HaveLen(1))
				Expect(result.Items[0].Spec).To(Equal(expected.Spec))

				By("returning the item for get requests")
				actual, err = client.Get(instance.Name, metav1.GetOptions{})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(actual.Spec).To(Equal(expected.Spec))

				By("deleting the item for delete requests")
				err = client.Delete(instance.Name, &metav1.DeleteOptions{})
				Expect(err).ShouldNot(HaveOccurred())
				result, err = client.List(metav1.ListOptions{})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(result.Items).To(HaveLen(0))
			})
		})
	})
})

func TestPluralName(t *testing.T) {
	var tests = []struct {
		name   string
		plural string
		expect bool
	}{
		{
			name:   "ingress",
			plural: "ingresses",
			expect: true,
		},
		{
			name:   "ingress",
			plural: "ingresss",
			expect: false,
		},
		{
			name:   "match",
			plural: "matches",
			expect: true,
		},
		{
			name:   "match",
			plural: "matchs",
			expect: false,
		},
		{
			name:   "mesh",
			plural: "meshes",
			expect: true,
		},
		{
			name:   "mesh",
			plural: "meshs",
			expect: false,
		},
		{
			name:   "box",
			plural: "boxes",
			expect: true,
		},
		{
			name:   "box",
			plural: "boxs",
			expect: false,
		},
		{
			name:   "match",
			plural: "matches",
			expect: true,
		},
		{
			name:   "match",
			plural: "matchs",
			expect: false,
		},
		{
			name:   "go",
			plural: "goes",
			expect: true,
		},
		{
			name:   "go",
			plural: "gos",
			expect: false,
		},
		{
			name:   "waltz",
			plural: "waltzes",
			expect: true,
		},
		{
			name:   "waltz",
			plural: "waltzs",
			expect: false,
		},
		{
			name:   "serviceentry",
			plural: "serviceentrys",
			expect: false,
		},
		{
			name:   "serviceentry",
			plural: "serviceentries",
			expect: true,
		},
	}

	for _, rt := range tests {
		actual := PluralName(rt.name)
		if rt.expect && actual != rt.plural {
			t.Errorf(
				"failed pluralizing:\n\texpected: %v\n\t  actual: %v",
				rt.name,
				actual,
			)
		}
		if !rt.expect && actual == rt.plural {
			t.Errorf(
				"pluralizing should have failed:\n\texpected: %v\n\t  actual: %v",
				rt.name,
				actual,
			)
		}
	}
}
