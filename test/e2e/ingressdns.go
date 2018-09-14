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

package e2e

import (
	"fmt"
	"sort"

	apiv1 "k8s.io/api/core/v1"
	extv1b1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"

	dnsv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/multiclusterdns/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	dnsv1a1client "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned/typed/multiclusterdns/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/dnsendpoint"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/e2e/framework"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("MultiClusterIngressDNS", func() {
	f := framework.NewFederationFramework("multicluster-ingress-dns")
	tl := framework.NewE2ELogger()

	const userAgent = "test-ingress-dns"
	const baseName = "test-ingress-dns-"

	var fedClient fedclientset.Interface
	var namespace string
	var dnsClient dnsv1a1client.MultiClusterIngressDNSRecordInterface

	objectGetter := func(namespace, name string) (pkgruntime.Object, error) {
		dnsClient := fedClient.MulticlusterdnsV1alpha1().MultiClusterIngressDNSRecords(namespace)
		return dnsClient.Get(name, metav1.GetOptions{})
	}

	BeforeEach(func() {
		fedClient = f.FedClient(userAgent)
		f.SetUpIngressDNSControllerFixture()
		namespace = f.TestNamespaceName()
		dnsClient = fedClient.MulticlusterdnsV1alpha1().MultiClusterIngressDNSRecords(namespace)
	})

	Context("When IngressDNS is created", func() {
		It("IngressDNS status should be updated correctly when there are no corresponding ingresses in member clusters", func() {
			By("Creating the IngressDNS object")
			ingressDNSObj := common.NewIngressDNSObject(baseName, namespace)
			ingressDNS, err := dnsClient.Create(ingressDNSObj)
			framework.ExpectNoError(err, "Error creating MultiClusterIngressDNS object: %v", ingressDNS)

			ingressDNSStatus := dnsv1a1.MultiClusterIngressDNSRecordStatus{DNS: []dnsv1a1.ClusterIngressDNS{}}
			for _, clusterName := range f.ClusterNames(userAgent) {
				ingressDNSStatus.DNS = append(ingressDNSStatus.DNS, dnsv1a1.ClusterIngressDNS{
					Cluster: clusterName,
				})
			}
			sort.Slice(ingressDNSStatus.DNS, func(i, j int) bool {
				return ingressDNSStatus.DNS[i].Cluster < ingressDNSStatus.DNS[j].Cluster
			})

			ingressDNS.Status = ingressDNSStatus
			By("Waiting for the IngressDNS object to have correct status")
			common.WaitForObject(tl, namespace, ingressDNS.Name, objectGetter, ingressDNS, framework.PollInterval, framework.TestContext.SingleCallTimeout)
		})

		It("IngressDNS status should be updated correctly when there are corresponding ingresses in member clusters", func() {
			const (
				RecordTypeA = "A"
				RecordTTL   = 300
			)
			hosts := []string{"foo.bar.test"}

			By("Creating the IngressDNS object")
			ingressDNSObj := common.NewIngressDNSObject(baseName, namespace)
			ingressDNSObj.Spec.Hosts = hosts
			ingressDNSObj.Spec.RecordTTL = RecordTTL
			ingressDNS, err := dnsClient.Create(ingressDNSObj)
			framework.ExpectNoError(err, "Error creating MultiClusterIngressDNS object %v", ingressDNS)
			name := ingressDNS.Name

			ingressDNSStatus := &dnsv1a1.MultiClusterIngressDNSRecordStatus{DNS: []dnsv1a1.ClusterIngressDNS{}}

			By("Creating corresponding ingress for the IngressDNS object in member clusters")
			ingressDNSStatus = createClusterIngress(f, name, namespace, ingressDNSStatus)

			ingressDNS.Status = *ingressDNSStatus

			By("Waiting for the IngressDNS object to have correct status")
			common.WaitForObject(tl, namespace, name, objectGetter, ingressDNS, framework.PollInterval, framework.TestContext.SingleCallTimeout)

			By("Waiting for the DNSEndpoint object to be created")
			endpointObjectGetter := func(namespace, name string) (pkgruntime.Object, error) {
				return fedClient.MulticlusterdnsV1alpha1().DNSEndpoints(namespace).Get(name, metav1.GetOptions{})
			}

			endpoints := []*dnsv1a1.Endpoint{}
			for _, cluster := range ingressDNS.Status.DNS {
				lbs := dnsendpoint.ExtractLoadBalancerTargets(cluster.LoadBalancer)

				endpoint := common.NewDNSEndpoint(
					"foo.bar.test",
					lbs, RecordTypeA, RecordTTL)
				endpoints = append(endpoints, endpoint)
			}
			desiredDNSEndpoint := &dnsv1a1.DNSEndpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress-" + name,
					Namespace: namespace,
				},
				Spec: dnsv1a1.DNSEndpointSpec{
					Endpoints: dnsendpoint.DedupeAndMergeEndpoints(endpoints),
				},
			}

			common.WaitForObject(tl, namespace, "ingress-"+name, endpointObjectGetter, desiredDNSEndpoint, framework.PollInterval, framework.TestContext.SingleCallTimeout)

			if !framework.TestContext.LimitedScope {
				By("Deleting test namespace in member clusters")
				cleanup(f, namespace)
			}
		})
	})
})

func createClusterIngress(f framework.FederationFramework, name, namespace string, ingressDNSStatus *dnsv1a1.MultiClusterIngressDNSRecordStatus) *dnsv1a1.MultiClusterIngressDNSRecordStatus {
	const userAgent = "test-ingress-dns"

	ingress := common.NewIngressObject(name, namespace)
	lbSuffix := 1

	for clusterName, client := range f.ClusterKubeClients(userAgent) {
		clusterLb := fmt.Sprintf("10.20.30.%d", lbSuffix)
		lbSuffix++

		lbStatus := apiv1.LoadBalancerStatus{Ingress: []apiv1.LoadBalancerIngress{{IP: clusterLb}}}
		ingressDNSStatus.DNS = append(ingressDNSStatus.DNS, dnsv1a1.ClusterIngressDNS{Cluster: clusterName, LoadBalancer: lbStatus})

		// Ensure the test namespace exists in the target cluster if
		// not running namespaced.  When namespaced, join will ensure
		// that the namespace exists.
		if !framework.TestContext.LimitedScope {
			_, err := client.CoreV1().Namespaces().Create(&apiv1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			})
			if !errors.IsAlreadyExists(err) {
				framework.ExpectNoError(err, "Error creating namespace in cluster %q", clusterName)
			}
		}

		createdIngress, err := client.ExtensionsV1beta1().Ingresses(namespace).Create(ingress)
		framework.ExpectNoError(err, "Error creating ingress in cluster %q", clusterName)

		createdIngress.Status = extv1b1.IngressStatus{lbStatus}

		// Fake out provisioning LoadBalancer by updating the ingress status in member cluster.
		_, err = client.ExtensionsV1beta1().Ingresses(namespace).UpdateStatus(createdIngress)
		framework.ExpectNoError(err, "Error updating ingress status in cluster %q", clusterName)
	}

	sort.Slice(ingressDNSStatus.DNS, func(i, j int) bool {
		return ingressDNSStatus.DNS[i].Cluster < ingressDNSStatus.DNS[j].Cluster
	})

	return ingressDNSStatus
}

func cleanup(f framework.FederationFramework, namespace string) {
	const userAgent = "test-ingress-dns"
	propogationPolicy := metav1.DeletePropagationBackground
	for _, client := range f.ClusterKubeClients(userAgent) {
		client.CoreV1().Namespaces().Delete(namespace, &metav1.DeleteOptions{PropagationPolicy: &propogationPolicy})
	}
}
