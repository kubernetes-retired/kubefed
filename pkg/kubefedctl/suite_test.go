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
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
)

var (
	cfg          *rest.Config
	k8sClient    client.Client
	clientScheme *runtime.Scheme
	testEnv      *envtest.Environment
)

func TestKubefedCTL(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Kubefedctl Suite")
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	testEnv = &envtest.Environment{
		CRDInstallOptions: envtest.CRDInstallOptions{
			ErrorIfPathMissing: true,
			Paths: []string{
				filepath.Join("..", "..", "charts", "kubefed", "charts", "controllermanager", "crds"),
			},
		},
	}

	clientScheme = runtime.NewScheme()

	Expect(v1beta1.AddToScheme(clientScheme)).To(Succeed())
	Expect(corev1.AddToScheme(clientScheme)).To(Succeed())

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	k8sClient, err = client.New(cfg, client.Options{Scheme: clientScheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})
