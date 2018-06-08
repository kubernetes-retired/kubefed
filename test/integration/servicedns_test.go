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

package integration

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"

	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	dnsv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/multiclusterdns/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/servicedns"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/integration/framework"
)

var TestMultiClusterServiceDNS = func(t *testing.T) {
	t.Parallel()
	tl := framework.NewIntegrationLogger(t)
	fixture := newServiceDNSTestFixture(tl, FedFixture)
	defer fixture.TearDown(tl)

	testCases := map[string]struct {
		name             string
		createService    bool
		createEndpoint   bool
		desiredDNSStatus []dnsv1a1.ClusterDNS
	}{
		"MultiClusterServiceDNS object in federation with no service and endpoint in cluster": {
			name:             "test-dns1",
			createService:    false,
			createEndpoint:   false,
			desiredDNSStatus: fixture.serviceDNSStatusWithoutLoadbalancer,
		},
		"MultiClusterServiceDNS object in federation with service but no endpoint in cluster": {
			name:             "test-dns2",
			createService:    true,
			createEndpoint:   false,
			desiredDNSStatus: fixture.serviceDNSStatusWithoutLoadbalancer,
		},
		"MultiClusterServiceDNS object in federation with service and endpoint in cluster": {
			name:             "test-dns3",
			createService:    true,
			createEndpoint:   true,
			desiredDNSStatus: fixture.serviceDNSStatus,
		},
	}

	for testName, tc := range testCases {
		t.Run(testName, func(t *testing.T) {
			tl := framework.NewIntegrationLogger(t)
			namespace := "default"
			key := fmt.Sprintf("%s/%s", namespace, tc.name)

			objectGetter := func(namespace, name string) (pkgruntime.Object, error) {
				return fixture.client.MulticlusterdnsV1alpha1().MultiClusterServiceDNSRecords(namespace).Get(name, metav1.GetOptions{})
			}

			serviceDNS := common.NewServiceDNSObject(tc.name, namespace)
			tl.Logf("Create serviceDNSObj: %s", key)
			serviceDNSObj, err := fixture.client.MulticlusterdnsV1alpha1().MultiClusterServiceDNSRecords(namespace).Create(serviceDNS)
			name := serviceDNSObj.Name
			if err != nil {
				tl.Fatalf("Error creating MultiClusterServiceDNS object %q: %v", key, err)
			}
			tl.Logf("Create MultiClusterServiceDNS object success: %s", key)

			for clusterName, clusterClient := range fixture.clusterClients {
				if tc.createService {
					service := common.NewServiceObject(name, namespace)
					tl.Logf("Create service %q, in cluster: %s", key, clusterName)
					service, err = clusterClient.CoreV1().Services(namespace).Create(service)
					if err != nil {
						tl.Fatalf("Error creating service %q in cluster %q: %v", key, clusterName, err)
					}

					service.Status = apiv1.ServiceStatus{LoadBalancer: apiv1.LoadBalancerStatus{
						Ingress: []apiv1.LoadBalancerIngress{{IP: fixture.clusterLbs[clusterName]}}}}
					service, err = clusterClient.CoreV1().Services(namespace).UpdateStatus(service)
					if err != nil {
						tl.Fatalf("Error updating service status %q in cluster %q: %v", key, clusterName, err)
					}
					tl.Logf("Created service: %q", key)
				}
				if tc.createEndpoint {
					endpoint := common.NewEndpointObject(name, namespace)
					tl.Logf("Create endpoint %q, in cluster: %s", key, clusterName)
					endpoint, err = clusterClient.CoreV1().Endpoints(namespace).Create(endpoint)
					if err != nil {
						tl.Fatalf("Error creating endpoint %q in cluster %q: %v", key, clusterName, err)
					}
					tl.Logf("Created endpoint: %q", key)
				}
			}

			serviceDNSObj.Status.DNS = tc.desiredDNSStatus

			tl.Logf("Wait for MultiClusterServiceDNS object status update")
			common.WaitForObject(tl, namespace, name, objectGetter, serviceDNSObj, framework.DefaultWaitInterval, wait.ForeverTestTimeout)
			tl.Logf("MultiClusterServiceDNS object status is updated successfully: %s", key)
		})
	}
}

// serviceDNSControllerFixture manages the MultiClusterServiceDNS controller for testing.
type serviceDNSControllerFixture struct {
	stopChan                            chan struct{}
	client                              fedclientset.Interface
	clusterClients                      map[string]clientset.Interface
	clusterLbs                          map[string]string
	clusterRegionZones                  map[string]fedv1a1.FederatedClusterStatus
	serviceDNSStatusWithoutLoadbalancer []dnsv1a1.ClusterDNS
	serviceDNSStatus                    []dnsv1a1.ClusterDNS
}

func (f *serviceDNSControllerFixture) TearDown(tl common.TestLogger) {
	if f.stopChan != nil {
		close(f.stopChan)
		f.stopChan = nil
	}
}

func newServiceDNSTestFixture(tl common.TestLogger, fedFixture *framework.FederationFixture) *serviceDNSControllerFixture {
	const (
		userAgent = "test-service-dns"
	)

	fedConfig := fedFixture.FedApi.NewConfig(tl)
	kubeConfig := fedFixture.KubeApi.NewConfig(tl)
	crConfig := fedFixture.CrApi.NewConfig(tl)

	f := &serviceDNSControllerFixture{
		stopChan: make(chan struct{}),
	}
	err := servicedns.StartController(fedConfig, kubeConfig, crConfig, f.stopChan, true)
	if err != nil {
		tl.Fatalf("Error starting service-dns controller: %v", err)
	}
	f.client = fedFixture.FedApi.NewClient(tl, userAgent)

	lbsuffix := 1
	f.clusterClients = FedFixture.ClusterKubeClients(tl, userAgent)
	f.clusterLbs = map[string]string{}
	f.clusterRegionZones = map[string]fedv1a1.FederatedClusterStatus{}
	for clusterName, _ := range FedFixture.Clusters {
		f.clusterLbs[clusterName] = fmt.Sprintf("10.20.30.%d", lbsuffix)
		lbsuffix++
		suffix := strings.TrimPrefix(clusterName, "test-cluster-")
		regionZones := fedv1a1.FederatedClusterStatus{
			Region: "region" + suffix,
			Zone:   "zone" + suffix,
		}
		f.clusterRegionZones[clusterName] = regionZones

		federatedCluster, err := f.client.FederationV1alpha1().FederatedClusters().Get(clusterName, metav1.GetOptions{})
		if err != nil {
			tl.Fatal("Error retrieving federated cluster %q: %v", clusterName, err)
		}
		federatedCluster.Status = f.clusterRegionZones[clusterName]
		_, err = f.client.FederationV1alpha1().FederatedClusters().UpdateStatus(federatedCluster)
		if err != nil {
			tl.Fatal("Error updating federated cluster status %q: %v", clusterName, err)
		}

		clusterDNS := dnsv1a1.ClusterDNS{
			Cluster: clusterName,
			Zone:    f.clusterRegionZones[clusterName].Zone,
			Region:  f.clusterRegionZones[clusterName].Region,
		}

		f.serviceDNSStatusWithoutLoadbalancer = append(f.serviceDNSStatusWithoutLoadbalancer, clusterDNS)

		clusterDNS.LoadBalancer = apiv1.LoadBalancerStatus{
			Ingress: []apiv1.LoadBalancerIngress{{IP: f.clusterLbs[clusterName]}}}
		f.serviceDNSStatus = append(f.serviceDNSStatus, clusterDNS)
	}
	sort.Slice(f.serviceDNSStatusWithoutLoadbalancer, func(i, j int) bool {
		return f.serviceDNSStatusWithoutLoadbalancer[i].Cluster < f.serviceDNSStatusWithoutLoadbalancer[j].Cluster
	})
	sort.Slice(f.serviceDNSStatus, func(i, j int) bool {
		return f.serviceDNSStatus[i].Cluster < f.serviceDNSStatus[j].Cluster
	})

	return f
}
