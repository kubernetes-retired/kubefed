/*
Copyright 2019 The Kubernetes Authors.

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

package kubefedcluster

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	fedapis "sigs.k8s.io/kubefed/pkg/apis"
	fedv1b1 "sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	"sigs.k8s.io/kubefed/pkg/controller/util"
)

func TestKubefedClusterController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "KubefedCluster Controller Integration Suite")
}

var testenv *envtest.Environment
var cfg *rest.Config
var clientset *kubernetes.Clientset
var k8sClient client.Client
var cc *ClusterController
var stopControllerCh chan struct{}
var config *util.ClusterHealthCheckConfig

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	testenv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "testdata", "fixtures"),
		},
	}

	var err error
	cfg, err = testenv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	clientset, err = kubernetes.NewForConfig(cfg)
	Expect(err).NotTo(HaveOccurred())

	scheme := runtime.NewScheme()
	err = fedapis.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	_, err = clientset.CoreV1().Namespaces().Create(
		context.Background(),
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: util.DefaultKubeFedSystemNamespace,
			},
		},
		metav1.CreateOptions{},
	)
	Expect(err).ToNot(HaveOccurred())

	config = &util.ClusterHealthCheckConfig{
		Period:           10 * time.Second,
		FailureThreshold: 3,
		SuccessThreshold: 1,
		Timeout:          10 * time.Second,
	}

	controllerConfig := &util.ControllerConfig{
		KubeFedNamespaces: util.KubeFedNamespaces{
			KubeFedNamespace: util.DefaultKubeFedSystemNamespace,
		},
		KubeConfig:      cfg,
		MinimizeLatency: true,
	}

	stopControllerCh = make(chan struct{})
	cc, err = newClusterController(controllerConfig, config)
	Expect(err).ToNot(HaveOccurred())

	cc.Run(stopControllerCh)

	close(done)
}, 60)

var _ = AfterSuite(func() {
	close(stopControllerCh)
	Expect(testenv.Stop()).To(Succeed())
})

var _ = Describe("TestKubefedClusterController", func() {
	It("validate controller informer actions and cluster data functions", func() {
		ctx := context.TODO()
		kubefedClusterSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: util.DefaultKubeFedSystemNamespace,
				Name:      "test-cluster-data",
			},
			Data: map[string][]byte{
				util.TokenKey: []byte("xxxxx"),
			},
		}

		_, err := clientset.CoreV1().Secrets(util.DefaultKubeFedSystemNamespace).Create(
			context.Background(), kubefedClusterSecret, metav1.CreateOptions{},
		)
		Expect(err).ToNot(HaveOccurred())

		kc := &fedv1b1.KubeFedCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cluster1",
				Namespace: util.DefaultKubeFedSystemNamespace,
				Labels: map[string]string{
					"foo": "bar",
				},
			},
			Spec: fedv1b1.KubeFedClusterSpec{
				APIEndpoint: "https://my.example.com:80/path/to/endpoint",
				SecretRef: fedv1b1.LocalSecretReference{
					Name: kubefedClusterSecret.Name,
				},
			},
			Status: *clusterStatus(corev1.ConditionTrue, metav1.Time{Time: metav1.Now().Add(1 * time.Second)}, metav1.Time{Time: metav1.Now().Add(1 * time.Second)}),
		}

		err = k8sClient.Create(ctx, kc)
		Expect(err).ToNot(HaveOccurred())

		cc.addToClusterSet(kc)
		Expect(err).ToNot(HaveOccurred())
		_, found := cc.clusterDataMap[kc.Name]
		Expect(found).To(BeTrue())

		cc.delFromClusterSet(kc)
		_, found = cc.clusterDataMap[kc.Name]
		Expect(found).NotTo(BeTrue())

		fedCluster := &fedv1b1.KubeFedCluster{}
		err = k8sClient.Get(ctx, client.ObjectKey{Namespace: util.DefaultKubeFedSystemNamespace, Name: kc.Name}, fedCluster)
		Expect(err).ToNot(HaveOccurred())

		fedCluster.Spec.APIEndpoint = "https://my.example.com:80/path/to/newendpoint"
		err = k8sClient.Update(ctx, fedCluster)
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() bool {
			c, found := cc.clusterDataMap[kc.Name]
			if found && c.cachedObj.Spec.APIEndpoint == fedCluster.Spec.APIEndpoint {
				return true
			}
			return false
		}, 10*time.Second).Should(BeTrue())
	})
})
