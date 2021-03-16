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

package v1beta1

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("KubefedCluster", func() {
	var (
		key              types.NamespacedName
		created, fetched *KubeFedCluster
	)

	BeforeEach(func() {

	})

	AfterEach(func() {

	})

	Context("Create API", func() {

		It("should create a kubefed cluster successfully", func() {

			created = &KubeFedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: metav1.NamespaceDefault,
				},
				Spec: KubeFedClusterSpec{
					APIEndpoint: "https://my.example.com:80/path/to/endpoint",
					ProxyURL:    "socks5://example.com",
					SecretRef: LocalSecretReference{
						Name: "foo",
					},
				},
			}

			key, _ = client.ObjectKeyFromObject(created)

			By("creating an API obj")
			Expect(k8sClient.Create(context.TODO(), created)).To(Succeed())

			fetched = &KubeFedCluster{}
			Expect(k8sClient.Get(context.TODO(), key, fetched)).To(Succeed())
			Expect(fetched).To(Equal(created))

			By("deleting the created object")
			Expect(k8sClient.Delete(context.TODO(), created)).To(Succeed())
			Expect(k8sClient.Get(context.TODO(), key, created)).ToNot(Succeed())
		})

	})

})
