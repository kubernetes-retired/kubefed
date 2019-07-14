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
	"context"
	"fmt"
	"sort"

	apiv1 "k8s.io/api/core/v1"
	extv1b1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"

	dnsv1a1 "sigs.k8s.io/kubefed/pkg/apis/multiclusterdns/v1alpha1"
	genericclient "sigs.k8s.io/kubefed/pkg/client/generic"
	"sigs.k8s.io/kubefed/pkg/controller/dnsendpoint"
	"sigs.k8s.io/kubefed/test/common"
	"sigs.k8s.io/kubefed/test/e2e/framework"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("IngressDNS", func() {
	f := framework.NewKubeFedFramework("multicluster-ingress-dns")
	tl := framework.NewE2ELogger()

	const userAgent = "test-ingress-dns"
	const baseName = "test-ingress-dns-"

	var client genericclient.Client
	var namespace string

	objectGetter := func(namespace, name string) (pkgruntime.Object, error) {
		ingressDNSRecords := &dnsv1a1.IngressDNSRecord{}
		err := client.Get(context.TODO(), ingressDNSRecords, namespace, name)
		return ingressDNSRecords, err
	}

	BeforeEach(func() {
		client = f.Client(userAgent)
		if framework.TestContext.RunControllers() {
			fixture := framework.NewIngressDNSControllerFixture(tl, f.ControllerConfig())
			f.RegisterFixture(fixture)
		}
		f.EnsureTestNamespacePropagation()
		namespace = f.TestNamespaceName()
	})

	Context("When IngressDNS is created", func() {
		It("IngressDNS status should be updated correctly when there are no corresponding ingresses in member clusters", func() {
			By("Creating the IngressDNS object")
			ingressDNS := common.NewIngressDNSObject(baseName, namespace)
			err := client.Create(context.TODO(), ingressDNS)
			framework.ExpectNoError(err, "Error creating IngressDNS object: %v", ingressDNS)

			ingressDNSStatus := dnsv1a1.IngressDNSRecordStatus{DNS: []dnsv1a1.ClusterIngressDNS{}}
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
			framework.WaitForObject(tl, namespace, ingressDNS.Name, objectGetter, ingressDNS, common.Equivalent)
		})

		It("IngressDNS status should be updated correctly when there are corresponding ingresses in member clusters", func() {
			const (
				RecordTypeA = "A"
				RecordTTL   = 300
			)
			hosts := []string{"foo.bar.test"}

			By("Creating the IngressDNS object")
			ingressDNS := common.NewIngressDNSObject(baseName, namespace)
			ingressDNS.Spec.Hosts = hosts
			ingressDNS.Spec.RecordTTL = RecordTTL
			err := client.Create(context.TODO(), ingressDNS)
			framework.ExpectNoError(err, "Error creating IngressDNS object %v", ingressDNS)
			name := ingressDNS.Name

			ingressDNSStatus := &dnsv1a1.IngressDNSRecordStatus{DNS: []dnsv1a1.ClusterIngressDNS{}}

			By("Creating corresponding ingress for the IngressDNS object in member clusters")
			ingressDNSStatus = createClusterIngress(f, name, namespace, ingressDNSStatus)

			ingressDNS.Status = *ingressDNSStatus

			By("Waiting for the IngressDNS object to have correct status")
			framework.WaitForObject(tl, namespace, name, objectGetter, ingressDNS, common.Equivalent)

			By("Waiting for the DNSEndpoint object to be created")
			endpointObjectGetter := func(namespace, name string) (pkgruntime.Object, error) {
				dnsEndpoints := &dnsv1a1.DNSEndpoint{}
				err := client.Get(context.TODO(), dnsEndpoints, namespace, name)
				return dnsEndpoints, err
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

			framework.WaitForObject(tl, namespace, "ingress-"+name, endpointObjectGetter, desiredDNSEndpoint, common.Equivalent)
		})
	})
})

func createClusterIngress(f framework.KubeFedFramework, name, namespace string, ingressDNSStatus *dnsv1a1.IngressDNSRecordStatus) *dnsv1a1.IngressDNSRecordStatus {
	const userAgent = "test-ingress-dns"

	ingress := common.NewIngressObject(name, namespace)
	lbSuffix := 1

	for clusterName, client := range f.ClusterKubeClients(userAgent) {
		clusterLb := fmt.Sprintf("10.20.30.%d", lbSuffix)
		lbSuffix++

		lbStatus := apiv1.LoadBalancerStatus{Ingress: []apiv1.LoadBalancerIngress{{IP: clusterLb}}}
		ingressDNSStatus.DNS = append(ingressDNSStatus.DNS, dnsv1a1.ClusterIngressDNS{Cluster: clusterName, LoadBalancer: lbStatus})

		common.WaitForNamespaceOrDie(framework.NewE2ELogger(), client, clusterName, namespace,
			framework.PollInterval, framework.TestContext.SingleCallTimeout)

		createdIngress, err := client.ExtensionsV1beta1().Ingresses(namespace).Create(ingress)
		framework.ExpectNoError(err, "Error creating ingress in cluster %q", clusterName)

		createdIngress.Status = extv1b1.IngressStatus{
			LoadBalancer: lbStatus,
		}

		// Fake out provisioning LoadBalancer by updating the ingress status in member cluster.
		_, err = client.ExtensionsV1beta1().Ingresses(namespace).UpdateStatus(createdIngress)
		framework.ExpectNoError(err, "Error updating ingress status in cluster %q", clusterName)
	}

	sort.Slice(ingressDNSStatus.DNS, func(i, j int) bool {
		return ingressDNSStatus.DNS[i].Cluster < ingressDNSStatus.DNS[j].Cluster
	})

	return ingressDNSStatus
}
