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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	dnsv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/multiclusterdns/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/e2e/framework"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("MultiClusterServiceDNS", func() {
	f := framework.NewFederationFramework("multicluster-service-dns")
	tl := framework.NewE2ELogger()

	const userAgent = "test-service-dns"
	const baseName = "test-service-dns-"

	var fedClient fedclientset.Interface
	var clusterRegionZones map[string]fedv1a1.FederatedClusterStatus
	var objectGetter func(namespace, name string) (pkgruntime.Object, error)

	BeforeEach(func() {
		fedClient = f.FedClient(userAgent)
		federatedClusters, err := fedClient.CoreV1alpha1().FederatedClusters(util.FederationSystemNamespace).List(metav1.ListOptions{})
		framework.ExpectNoError(err, "Error listing federated clusters")
		clusterRegionZones = make(map[string]fedv1a1.FederatedClusterStatus)
		for _, cluster := range federatedClusters.Items {
			clusterRegionZones[cluster.Name] = fedv1a1.FederatedClusterStatus{
				Region: cluster.Status.Region,
				Zone:   cluster.Status.Zone,
			}
		}
		f.SetUpServiceDNSControllerFixture()

		objectGetter = func(namespace, name string) (pkgruntime.Object, error) {
			return fedClient.MulticlusterdnsV1alpha1().MultiClusterServiceDNSRecords(namespace).Get(name, metav1.GetOptions{})
		}
	})

	It("ServiceDNS object status should be updated correctly when there are no service and/or endpoint in member clusters", func() {
		namespace := f.TestNamespaceName()

		By("Create the ServiceDNS object")
		serviceDNSObj := common.NewServiceDNSObject(baseName, namespace)
		serviceDNS, err := fedClient.MulticlusterdnsV1alpha1().MultiClusterServiceDNSRecords(namespace).Create(serviceDNSObj)
		framework.ExpectNoError(err, "Error creating MultiClusterServiceDNS object: %v", serviceDNS)

		serviceDNSStatus := dnsv1a1.MultiClusterServiceDNSRecordStatus{DNS: []dnsv1a1.ClusterDNS{}}
		for clusterName, _ := range f.ClusterKubeClients(userAgent) {
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
		By("Wait for the ServiceDNS object to have correct status")
		common.WaitForObject(tl, namespace, serviceDNS.Name, objectGetter, serviceDNS, framework.PollInterval, wait.ForeverTestTimeout)
	})

	It("ServiceDNS object status should be updated, when service and endpoint are created in member cluster", func() {
		namespace := f.TestNamespaceName()

		By("Create the ServiceDNS object")
		serviceDNSObj := common.NewServiceDNSObject(baseName, namespace)
		serviceDNS, err := fedClient.MulticlusterdnsV1alpha1().MultiClusterServiceDNSRecords(namespace).Create(serviceDNSObj)
		framework.ExpectNoError(err, "Error creating MultiClusterServiceDNS object %v", serviceDNS)
		name := serviceDNS.Name

		serviceDNSStatus := dnsv1a1.MultiClusterServiceDNSRecordStatus{DNS: []dnsv1a1.ClusterDNS{}}

		service := common.NewServiceObject(name, namespace)
		endpoint := common.NewEndpointObject(name, namespace)
		lbsuffix := 1

		By("Create corresponding service and endpoint for the ServiceDNS object in member clusters")
		for clusterName, client := range f.ClusterKubeClients(userAgent) {
			clusterLb := fmt.Sprintf("10.20.30.%d", lbsuffix)
			lbsuffix++

			loadbalancerStatus := apiv1.LoadBalancerStatus{Ingress: []apiv1.LoadBalancerIngress{{IP: clusterLb}}}
			serviceDNSStatus.DNS = append(serviceDNSStatus.DNS, dnsv1a1.ClusterDNS{Cluster: clusterName, LoadBalancer: loadbalancerStatus})

			// Ensure the test namespace exists in the target cluster.
			_, err = client.CoreV1().Namespaces().Create(&apiv1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			})
			if !errors.IsAlreadyExists(err) {
				framework.ExpectNoError(err, "Error creating namespace in cluster %q", clusterName)
			}

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

		serviceDNS.Status = serviceDNSStatus

		By("Wait for the ServiceDNS object to have correct status")
		common.WaitForObject(tl, namespace, name, objectGetter, serviceDNS, framework.PollInterval, wait.ForeverTestTimeout)
	})
})
