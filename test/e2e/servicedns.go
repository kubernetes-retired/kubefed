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
	"strings"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"

	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	dnsv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/multiclusterdns/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	dnsv1a1client "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned/typed/multiclusterdns/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/dnsendpoint"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/e2e/framework"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("ServiceDNS", func() {
	f := framework.NewFederationFramework("multicluster-service-dns")
	tl := framework.NewE2ELogger()

	const userAgent = "test-service-dns"
	const baseName = "test-service-dns-"

	var fedClient fedclientset.Interface
	var clusterRegionZones map[string]fedv1a1.FederatedClusterStatus
	var namespace string
	var dnsClient dnsv1a1client.ServiceDNSRecordInterface

	objectGetter := func(namespace, name string) (pkgruntime.Object, error) {
		dnsClient := fedClient.MulticlusterdnsV1alpha1().ServiceDNSRecords(namespace)
		return dnsClient.Get(name, metav1.GetOptions{})
	}

	BeforeEach(func() {
		fedClient = f.FedClient(userAgent)
		namespace = f.TestNamespaceName()
		dnsClient = fedClient.MulticlusterdnsV1alpha1().ServiceDNSRecords(namespace)

		federatedClusters, err := fedClient.CoreV1alpha1().FederatedClusters(f.FederationSystemNamespace()).List(metav1.ListOptions{})
		framework.ExpectNoError(err, "Error listing federated clusters")
		clusterRegionZones = make(map[string]fedv1a1.FederatedClusterStatus)
		for _, cluster := range federatedClusters.Items {
			clusterRegionZones[cluster.Name] = fedv1a1.FederatedClusterStatus{
				Region: cluster.Status.Region,
				Zone:   cluster.Status.Zone,
			}
		}
		f.SetUpServiceDNSControllerFixture()
	})

	It("ServiceDNS object status should be updated correctly when there are no service and/or endpoint in member clusters", func() {
		By("Creating the ServiceDNS object")
		serviceDNSObj := common.NewServiceDNSObject(baseName, namespace)
		serviceDNS, err := dnsClient.Create(serviceDNSObj)
		framework.ExpectNoError(err, "Error creating ServiceDNS object: %v", serviceDNS)

		serviceDNSStatus := dnsv1a1.ServiceDNSRecordStatus{DNS: []dnsv1a1.ClusterDNS{}}
		for _, clusterName := range f.ClusterNames(userAgent) {
			serviceDNSStatus.DNS = append(serviceDNSStatus.DNS, dnsv1a1.ClusterDNS{
				Cluster: clusterName,
				Region:  clusterRegionZones[clusterName].Region,
				Zone:    clusterRegionZones[clusterName].Zone,
			})
		}
		sort.Slice(serviceDNSStatus.DNS, func(i, j int) bool {
			return serviceDNSStatus.DNS[i].Cluster < serviceDNSStatus.DNS[j].Cluster
		})

		serviceDNS.Status = serviceDNSStatus
		By("Waiting for the ServiceDNS object to have correct status")
		common.WaitForObject(tl, namespace, serviceDNS.Name, objectGetter, serviceDNS, framework.PollInterval, framework.TestContext.SingleCallTimeout)
	})

	Context("When ServiceDNS object is created", func() {
		const (
			RecordTypeA = "A"
			RecordTTL   = 300
		)
		federation := "galactic"
		dnsZone := "dzone.io"

		It("DNSEndpoint object should be created with correct status when ServiceDNS object is created", func() {
			By("Creating the ServiceDNS object")
			serviceDNSObj := common.NewServiceDNSObject(baseName, namespace)
			serviceDNSObj.Spec.FederationName = federation
			serviceDNSObj.Spec.DNSSuffix = dnsZone
			serviceDNSObj.Spec.RecordTTL = RecordTTL
			serviceDNS, err := dnsClient.Create(serviceDNSObj)
			framework.ExpectNoError(err, "Error creating ServiceDNS object %v", serviceDNS)
			name := serviceDNS.Name

			serviceDNSStatus := &dnsv1a1.ServiceDNSRecordStatus{DNS: []dnsv1a1.ClusterDNS{}}

			By("Creating corresponding service and endpoint for the ServiceDNS object in member clusters")
			serviceDNSStatus = createClusterServiceAndEndpoints(f, name, namespace, serviceDNSStatus)

			serviceDNS.Status = *serviceDNSStatus

			By("Waiting for the ServiceDNS object to have correct status")
			common.WaitForObject(tl, namespace, name, objectGetter, serviceDNS, framework.PollInterval, framework.TestContext.SingleCallTimeout)

			By("Waiting for the DNSEndpoint object to be created")
			endpointObjectGetter := func(namespace, name string) (pkgruntime.Object, error) {
				return fedClient.MulticlusterdnsV1alpha1().DNSEndpoints(namespace).Get(name, metav1.GetOptions{})
			}

			endpoints := []*dnsv1a1.Endpoint{}
			for _, cluster := range serviceDNS.Status.DNS {
				zone := clusterRegionZones[cluster.Cluster].Zone
				region := clusterRegionZones[cluster.Cluster].Region
				lbs := dnsendpoint.ExtractLoadBalancerTargets(cluster.LoadBalancer)

				endpoint := common.NewDNSEndpoint(
					strings.Join([]string{name, namespace, federation, "svc", zone, region, dnsZone}, "."),
					lbs, RecordTypeA, RecordTTL)
				endpoints = append(endpoints, endpoint)
				endpoint = common.NewDNSEndpoint(
					strings.Join([]string{name, namespace, federation, "svc", region, dnsZone}, "."),
					lbs, RecordTypeA, RecordTTL)
				endpoints = append(endpoints, endpoint)
				endpoint = common.NewDNSEndpoint(
					strings.Join([]string{name, namespace, federation, "svc", dnsZone}, "."),
					lbs, RecordTypeA, RecordTTL)
				endpoints = append(endpoints, endpoint)
			}
			desiredDNSEndpoint := &dnsv1a1.DNSEndpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "service-" + name,
					Namespace: namespace,
				},
				Spec: dnsv1a1.DNSEndpointSpec{
					Endpoints: dnsendpoint.DedupeAndMergeEndpoints(endpoints),
				},
			}

			common.WaitForObject(tl, namespace, "service-"+name, endpointObjectGetter, desiredDNSEndpoint, framework.PollInterval, framework.TestContext.SingleCallTimeout)
		})
	})
})

func createClusterServiceAndEndpoints(f framework.FederationFramework, name, namespace string, serviceDNSStatus *dnsv1a1.ServiceDNSRecordStatus) *dnsv1a1.ServiceDNSRecordStatus {
	const userAgent = "test-service-dns"

	service := common.NewServiceObject(name, namespace)
	endpoint := common.NewEndpointObject(name, namespace)
	lbsuffix := 1

	for clusterName, client := range f.ClusterKubeClients(userAgent) {
		clusterLb := fmt.Sprintf("10.20.30.%d", lbsuffix)
		lbsuffix++

		loadbalancerStatus := apiv1.LoadBalancerStatus{Ingress: []apiv1.LoadBalancerIngress{{IP: clusterLb}}}
		serviceDNSStatus.DNS = append(serviceDNSStatus.DNS, dnsv1a1.ClusterDNS{Cluster: clusterName, LoadBalancer: loadbalancerStatus})

		common.WaitForNamespaceOrDie(framework.NewE2ELogger(), client, clusterName, namespace,
			framework.PollInterval, framework.TestContext.SingleCallTimeout)

		createdService, err := client.CoreV1().Services(namespace).Create(service)
		framework.ExpectNoError(err, "Error creating service in cluster %q", clusterName)

		createdService.Status = apiv1.ServiceStatus{loadbalancerStatus}

		// Fake out provisioning LoadBalancer by updating the service status in member cluster.
		_, err = client.CoreV1().Services(namespace).UpdateStatus(createdService)
		framework.ExpectNoError(err, "Error updating service status in cluster %q", clusterName)

		// Fake out pods backing service by creating endpoint in member cluster.
		_, err = client.CoreV1().Endpoints(namespace).Create(endpoint)
		framework.ExpectNoError(err, "Error creating endpoint in cluster %q", clusterName)
	}

	sort.Slice(serviceDNSStatus.DNS, func(i, j int) bool {
		return serviceDNSStatus.DNS[i].Cluster < serviceDNSStatus.DNS[j].Cluster
	})

	return serviceDNSStatus
}
