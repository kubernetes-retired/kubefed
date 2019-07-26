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
	"strings"

	"github.com/pkg/errors"

	apiv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	fedv1b1 "sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	dnsv1a1 "sigs.k8s.io/kubefed/pkg/apis/multiclusterdns/v1alpha1"
	genericclient "sigs.k8s.io/kubefed/pkg/client/generic"
	"sigs.k8s.io/kubefed/pkg/controller/dnsendpoint"
	"sigs.k8s.io/kubefed/test/common"
	"sigs.k8s.io/kubefed/test/e2e/framework"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("ServiceDNS", func() {
	f := framework.NewKubeFedFramework("multicluster-service-dns")
	tl := framework.NewE2ELogger()

	const userAgent = "test-service-dns"
	const baseName = "test-service-dns-"
	const federationPrefix = "galactic"
	const Domain = "example.com"

	var client genericclient.Client
	var clusterRegionZones map[string]fedv1b1.KubeFedClusterStatus
	var federation string
	var namespace string

	objectGetter := func(namespace, name string) (pkgruntime.Object, error) {
		serviceDNSRecords := &dnsv1a1.ServiceDNSRecord{}
		err := client.Get(context.TODO(), serviceDNSRecords, namespace, name)
		return serviceDNSRecords, err
	}

	BeforeEach(func() {
		client = f.Client(userAgent)
		namespace = f.TestNamespaceName()

		clusterRegionZones = ensureClustersHaveRegionZoneAttributes(tl, client, f.KubeFedSystemNamespace())
		if framework.TestContext.RunControllers() {
			fixture := framework.NewServiceDNSControllerFixture(tl, f.ControllerConfig())
			f.RegisterFixture(fixture)
		}
		f.EnsureTestNamespacePropagation()
		domainObj := common.NewDomainObject(federationPrefix, Domain)
		domainObj.Namespace = f.KubeFedSystemNamespace()
		err := client.Create(context.TODO(), domainObj)
		framework.ExpectNoError(err, "Error creating Domain object")
		federation = domainObj.Name
	})

	AfterEach(func() {
		domainObj := &dnsv1a1.Domain{}
		err := client.Delete(context.TODO(), domainObj, f.KubeFedSystemNamespace(), federation)
		framework.ExpectNoError(err, "Error deleting Domain object")
	})

	It("ServiceDNS object status should be updated correctly when there are no service and/or endpoint in member clusters", func() {
		By("Creating the ServiceDNS object")
		serviceDNS := common.NewServiceDNSObject(baseName, namespace)
		serviceDNS.Spec.DomainRef = federation
		err := client.Create(context.TODO(), serviceDNS)
		framework.ExpectNoError(err, "Error creating ServiceDNS object: %v", serviceDNS)

		serviceDNSStatus := dnsv1a1.ServiceDNSRecordStatus{Domain: Domain, DNS: []dnsv1a1.ClusterDNS{}}
		for _, clusterName := range f.ClusterNames(userAgent) {
			serviceDNSStatus.DNS = append(serviceDNSStatus.DNS, dnsv1a1.ClusterDNS{
				Cluster: clusterName,
				Region:  *clusterRegionZones[clusterName].Region,
				Zones:   clusterRegionZones[clusterName].Zones,
			})
		}
		sort.Slice(serviceDNSStatus.DNS, func(i, j int) bool {
			return serviceDNSStatus.DNS[i].Cluster < serviceDNSStatus.DNS[j].Cluster
		})

		serviceDNS.Status = serviceDNSStatus
		By("Waiting for the ServiceDNS object to have correct status")
		framework.WaitForObject(tl, namespace, serviceDNS.Name, objectGetter, serviceDNS, common.Equivalent)
	})

	Context("When ServiceDNS object is created", func() {
		const (
			RecordTypeA = "A"
			RecordTTL   = 300
		)

		It("DNSEndpoint object should be created with correct status when ServiceDNS object is created", func() {
			By("Creating the ServiceDNS object")
			serviceDNS := common.NewServiceDNSObject(baseName, namespace)
			serviceDNS.Spec.DomainRef = federation
			serviceDNS.Spec.RecordTTL = RecordTTL
			err := client.Create(context.TODO(), serviceDNS)
			framework.ExpectNoError(err, "Error creating ServiceDNS object %v", serviceDNS)
			name := serviceDNS.Name

			By("Creating corresponding service and endpoint for the ServiceDNS object in member clusters")
			serviceDNSStatus := createClusterServiceAndEndpoints(f, name, namespace, Domain, clusterRegionZones)
			serviceDNS.Status = *serviceDNSStatus

			By("Waiting for the ServiceDNS object to have correct status")
			framework.WaitForObject(tl, namespace, name, objectGetter, serviceDNS, common.Equivalent)

			By("Waiting for the DNSEndpoint object to be created")
			endpointObjectGetter := func(namespace, name string) (pkgruntime.Object, error) {
				dnsEndpoint := &dnsv1a1.DNSEndpoint{}
				err := client.Get(context.TODO(), dnsEndpoint, namespace, name)
				return dnsEndpoint, err
			}

			endpoints := []*dnsv1a1.Endpoint{}
			for _, cluster := range serviceDNS.Status.DNS {
				zones := clusterRegionZones[cluster.Cluster].Zones
				region := *clusterRegionZones[cluster.Cluster].Region
				lbs := dnsendpoint.ExtractLoadBalancerTargets(cluster.LoadBalancer)

				endpoint := common.NewDNSEndpoint(
					strings.Join([]string{name, namespace, federation, "svc", zones[0], region, Domain}, "."),
					lbs, RecordTypeA, RecordTTL)
				endpoints = append(endpoints, endpoint)
				endpoint = common.NewDNSEndpoint(
					strings.Join([]string{name, namespace, federation, "svc", region, Domain}, "."),
					lbs, RecordTypeA, RecordTTL)
				endpoints = append(endpoints, endpoint)
				endpoint = common.NewDNSEndpoint(
					strings.Join([]string{name, namespace, federation, "svc", Domain}, "."),
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

			framework.WaitForObject(tl, namespace, "service-"+name, endpointObjectGetter, desiredDNSEndpoint, common.Equivalent)
		})
	})
})

func createClusterServiceAndEndpoints(f framework.KubeFedFramework, name, namespace string, domain string,
	clusterRegionZones map[string]fedv1b1.KubeFedClusterStatus) *dnsv1a1.ServiceDNSRecordStatus {

	const userAgent = "test-service-dns"

	service := common.NewServiceObject(name, namespace)
	endpoint := common.NewEndpointObject(name, namespace)
	lbsuffix := 1

	serviceDNSStatus := &dnsv1a1.ServiceDNSRecordStatus{Domain: domain, DNS: []dnsv1a1.ClusterDNS{}}
	for clusterName, client := range f.ClusterKubeClients(userAgent) {
		clusterLb := fmt.Sprintf("10.20.30.%d", lbsuffix)
		lbsuffix++

		loadbalancerStatus := apiv1.LoadBalancerStatus{Ingress: []apiv1.LoadBalancerIngress{{IP: clusterLb}}}
		serviceDNSStatus.DNS = append(serviceDNSStatus.DNS, dnsv1a1.ClusterDNS{
			Cluster:      clusterName,
			LoadBalancer: loadbalancerStatus,
			Region:       *clusterRegionZones[clusterName].Region,
			Zones:        clusterRegionZones[clusterName].Zones,
		})

		common.WaitForNamespaceOrDie(framework.NewE2ELogger(), client, clusterName, namespace,
			framework.PollInterval, framework.TestContext.SingleCallTimeout)

		createdService, err := client.CoreV1().Services(namespace).Create(service)
		framework.ExpectNoError(err, "Error creating service in cluster %q", clusterName)

		createdService.Status = apiv1.ServiceStatus{
			LoadBalancer: loadbalancerStatus,
		}

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

func ensureClustersHaveRegionZoneAttributes(tl common.TestLogger, client genericclient.Client, clusterNamespace string) map[string]fedv1b1.KubeFedClusterStatus {
	clusters := &fedv1b1.KubeFedClusterList{}
	err := client.List(context.TODO(), clusters, clusterNamespace)
	framework.ExpectNoError(err, "Error listing federated clusters")

	clusterRegionZones := make(map[string]fedv1b1.KubeFedClusterStatus)
	for i, cluster := range clusters.Items {
		err := wait.PollImmediate(framework.PollInterval, framework.TestContext.SingleCallTimeout, func() (bool, error) {
			region := fmt.Sprintf("r%d", i)
			cluster.Status.Region = &region
			cluster.Status.Zones = []string{fmt.Sprintf("z%d", i)}

			err := client.UpdateStatus(context.TODO(), &cluster)
			if apierrors.IsConflict(err) {
				clusterName := cluster.Name
				tl.Logf("Failed to update status for federated cluster %q: %v", clusterName, err)
				err = client.Get(context.TODO(), &cluster, clusterNamespace, clusterName)
				if err != nil {
					return false, errors.Wrapf(err, "failed to retrieve cluster object")
				}
				return false, nil
			}
			if err != nil {
				return false, err
			}
			return true, nil
		})
		framework.ExpectNoError(err, "Error updating federated cluster status")

		clusterRegionZones[cluster.Name] = fedv1b1.KubeFedClusterStatus{
			Region: cluster.Status.Region,
			Zones:  cluster.Status.Zones,
		}
	}

	return clusterRegionZones
}
