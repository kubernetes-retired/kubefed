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

package kubefedctl

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	utiltesting "k8s.io/client-go/util/testing"

	"sigs.k8s.io/kubefed/pkg/kubefedctl/util"

	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
)

var (
	joiningClusterName = "joiningcluster"
	hostClusterName    = "hostnamecluster"
)

var _ = Describe("Kubefedctl", func() {

	Context("Join Cluster", func() {
		It("should create a kubefed cluster successfully with a proxyURL", func() {
			testServer, _, _ := testServerEnv(200)
			defer testServer.Close()

			clusterConfig := &rest.Config{
				Host: testServer.URL,
				ContentConfig: rest.ContentConfig{
					GroupVersion:         &v1.SchemeGroupVersion,
					NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
				},
			}

			expectedKubefedCluster := &v1beta1.KubeFedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      joiningClusterName,
					Namespace: metav1.NamespaceDefault,
				},
				Spec: v1beta1.KubeFedClusterSpec{
					APIEndpoint: testServer.URL,
					SecretRef: v1beta1.LocalSecretReference{
						Name: "secret",
					},
				},
			}

			kubefedCluster, err := joinClusterForNamespace(
				cfg,
				clusterConfig,
				metav1.NamespaceDefault,
				metav1.NamespaceDefault,
				hostClusterName,
				joiningClusterName,
				"secret",
				v1.ClusterScoped, true, false)

			Expect(err).NotTo(HaveOccurred())
			Expect(kubefedCluster).To(Equal(expectedKubefedCluster))

			By("Using a joining cluster behind a proxy-url")
			testProxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				to, err := url.Parse(req.RequestURI)
				Expect(err).To(BeNil())
				httputil.NewSingleHostReverseProxy(to).ServeHTTP(w, req)
			}))
			defer testProxyServer.Close()

			u, err := url.Parse(testProxyServer.URL)
			if err != nil {
				Expect(err).To(BeNil())
			}


			// Set a fakeProxyServer as URL
			clusterConfig.Proxy = http.ProxyURL(u)

			expectedKubefedClusterWithProxyURL := &v1beta1.KubeFedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      joiningClusterName,
					Namespace: metav1.NamespaceDefault,
				},
				Spec: v1beta1.KubeFedClusterSpec{
					APIEndpoint: testServer.URL,
					ProxyURL:    u.String(),
				},
			}

			kubefedClusterWithProxyURL, err := joinClusterForNamespace(
				cfg,
				clusterConfig,
				metav1.NamespaceDefault,
				metav1.NamespaceDefault,
				hostClusterName,
				joiningClusterName,
				"",
				v1.ClusterScoped, true, false)

			Expect(err).NotTo(HaveOccurred())
			Expect(kubefedClusterWithProxyURL).To(Equal(expectedKubefedClusterWithProxyURL))

		})
	})
})

func testServerEnv(statusCode int) (*httptest.Server, *utiltesting.FakeHandler, *corev1.ServiceAccount) {
	status := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.ClusterServiceAccountName(joiningClusterName, hostClusterName),
			Namespace: metav1.NamespaceDefault,
		},
	}
	expectedBody, _ := runtime.Encode(scheme.Codecs.LegacyCodec(corev1.SchemeGroupVersion), status)
	fakeHandler := utiltesting.FakeHandler{
		StatusCode:   statusCode,
		ResponseBody: string(expectedBody),
	}
	testServer := httptest.NewServer(&fakeHandler)
	return testServer, &fakeHandler, status
}
