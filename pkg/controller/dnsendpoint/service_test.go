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

package dnsendpoint

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	feddnsv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/multiclusterdns/v1alpha1"
)

const (
	dnsZone    = "example.com"
	federation = "galactic"

	c1Region = "us"
	c2Region = "eu"
	c1Zone   = "us1"
	c2Zone   = "eu1"
)

func TestGetEndpointsForServiceDNSObject(t *testing.T) {
	// Fake out the internet
	netmock := &NetWrapperMock{}
	netmock.AddHost("a9.us-west-2.elb.amazonaws.test", []string{lb3})

	netWrapper = netmock

	globalDNSPrefix := strings.Join([]string{namespace, federation, "svc", dnsZone}, ".")
	c1RegionDNSPrefix := strings.Join([]string{namespace, federation, "svc", c1Region, dnsZone}, ".")
	c1ZoneDNSPrefix := strings.Join([]string{namespace, federation, "svc", c1Zone, c1Region, dnsZone}, ".")
	c2RegionDNSPrefix := strings.Join([]string{namespace, federation, "svc", c2Region, dnsZone}, ".")
	c2ZoneDNSPrefix := strings.Join([]string{namespace, federation, "svc", c2Zone, c2Region, dnsZone}, ".")
	globalDNSName := strings.Join([]string{name, globalDNSPrefix}, ".")
	c1RegionDNSName := strings.Join([]string{name, c1RegionDNSPrefix}, ".")
	c1ZoneDNSName := strings.Join([]string{name, c1ZoneDNSPrefix}, ".")
	c2RegionDNSName := strings.Join([]string{name, c2RegionDNSPrefix}, ".")
	c2ZoneDNSName := strings.Join([]string{name, c2ZoneDNSPrefix}, ".")

	labels := map[string]string{"serviceName": name}

	testCases := map[string]struct {
		dnsObject       feddnsv1a1.ServiceDNSRecord
		expectEndpoints []*feddnsv1a1.Endpoint
		expectError     bool
	}{
		"NoClusters": {
			dnsObject: feddnsv1a1.ServiceDNSRecord{
				Spec: feddnsv1a1.ServiceDNSRecordSpec{
					DomainRef: federation,
				},
				Status: feddnsv1a1.ServiceDNSRecordStatus{
					Domain: dnsZone,
				},
			},
			expectEndpoints: nil,
			expectError:     false,
		},
		"SingleLBInSingleCluster": {
			dnsObject: feddnsv1a1.ServiceDNSRecord{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: feddnsv1a1.ServiceDNSRecordSpec{
					DomainRef: federation,
				},
				Status: feddnsv1a1.ServiceDNSRecordStatus{
					Domain: dnsZone,
					DNS: []feddnsv1a1.ClusterDNS{
						{
							Cluster: c1, Zone: c1Zone, Region: c1Region,
							LoadBalancer: v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{{IP: lb1}}},
						},
					},
				},
			},
			expectEndpoints: []*feddnsv1a1.Endpoint{
				{DNSName: globalDNSName, Targets: []string{lb1}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL},
				{DNSName: c1RegionDNSName, Targets: []string{lb1}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL},
				{DNSName: c1ZoneDNSName, Targets: []string{lb1}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL},
			},
			expectError: false,
		},
		"LBsInBothClusters": {
			dnsObject: feddnsv1a1.ServiceDNSRecord{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: feddnsv1a1.ServiceDNSRecordSpec{
					DomainRef: federation,
				},
				Status: feddnsv1a1.ServiceDNSRecordStatus{
					Domain: dnsZone,
					DNS: []feddnsv1a1.ClusterDNS{
						{
							Cluster: c1, Zone: c1Zone, Region: c1Region,
							LoadBalancer: v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{{IP: lb1}}},
						},
						{
							Cluster: c2, Zone: c2Zone, Region: c2Region,
							LoadBalancer: v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{{IP: lb2}}},
						},
					},
				},
			},
			expectEndpoints: []*feddnsv1a1.Endpoint{
				{DNSName: globalDNSName, Targets: []string{lb1, lb2}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL},
				{DNSName: c1RegionDNSName, Targets: []string{lb1}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL},
				{DNSName: c1ZoneDNSName, Targets: []string{lb1}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL},
				{DNSName: c2RegionDNSName, Targets: []string{lb2}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL},
				{DNSName: c2ZoneDNSName, Targets: []string{lb2}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL},
			},
			expectError: false,
		},
		"NoLBInOneCluster": {
			dnsObject: feddnsv1a1.ServiceDNSRecord{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: feddnsv1a1.ServiceDNSRecordSpec{
					DomainRef: federation,
				},
				Status: feddnsv1a1.ServiceDNSRecordStatus{
					Domain: dnsZone,
					DNS: []feddnsv1a1.ClusterDNS{
						{
							Cluster: c1, Zone: c1Zone, Region: c1Region,
							LoadBalancer: v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{{IP: lb1}}},
						},
						{
							Cluster: c2, Zone: c2Zone, Region: c2Region,
						},
					},
				},
			},
			expectEndpoints: []*feddnsv1a1.Endpoint{
				{DNSName: globalDNSName, Targets: []string{lb1}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL},
				{DNSName: c1RegionDNSName, Targets: []string{lb1}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL},
				{DNSName: c1ZoneDNSName, Targets: []string{lb1}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL},
				{DNSName: c2RegionDNSName, Targets: []string{globalDNSName}, RecordType: RecordTypeCNAME, RecordTTL: defaultDNSTTL},
				{DNSName: c2ZoneDNSName, Targets: []string{c2RegionDNSName}, RecordType: RecordTypeCNAME, RecordTTL: defaultDNSTTL},
			},
			expectError: false,
		},
		"NoLBInBothClusters": {
			dnsObject: feddnsv1a1.ServiceDNSRecord{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: feddnsv1a1.ServiceDNSRecordSpec{
					DomainRef: federation,
				},
				Status: feddnsv1a1.ServiceDNSRecordStatus{
					Domain: dnsZone,
					DNS: []feddnsv1a1.ClusterDNS{
						{
							Cluster: c1, Zone: c1Zone, Region: c1Region,
						},
						{
							Cluster: c2, Zone: c2Zone, Region: c2Region,
						},
					},
				},
			},
			expectEndpoints: []*feddnsv1a1.Endpoint{
				{DNSName: c1RegionDNSName, Targets: []string{globalDNSName}, RecordType: RecordTypeCNAME, RecordTTL: defaultDNSTTL},
				{DNSName: c1ZoneDNSName, Targets: []string{c1RegionDNSName}, RecordType: RecordTypeCNAME, RecordTTL: defaultDNSTTL},
				{DNSName: c2RegionDNSName, Targets: []string{globalDNSName}, RecordType: RecordTypeCNAME, RecordTTL: defaultDNSTTL},
				{DNSName: c2ZoneDNSName, Targets: []string{c2RegionDNSName}, RecordType: RecordTypeCNAME, RecordTTL: defaultDNSTTL},
			},
			expectError: false,
		},
		"HostnameInLB": {
			dnsObject: feddnsv1a1.ServiceDNSRecord{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: feddnsv1a1.ServiceDNSRecordSpec{
					DomainRef: federation,
				},
				Status: feddnsv1a1.ServiceDNSRecordStatus{
					Domain: dnsZone,
					DNS: []feddnsv1a1.ClusterDNS{
						{
							Cluster: c1, Zone: c1Zone, Region: c1Region,
							LoadBalancer: v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{{Hostname: "a9.us-west-2.elb.amazonaws.test"}}},
						},
						{
							Cluster: c2, Zone: c2Zone, Region: c2Region,
							LoadBalancer: v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{{IP: lb2}}},
						},
					},
				},
			},
			expectEndpoints: []*feddnsv1a1.Endpoint{
				{DNSName: globalDNSName, Targets: []string{lb2, lb3}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL},
				{DNSName: c1RegionDNSName, Targets: []string{lb3}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL},
				{DNSName: c1ZoneDNSName, Targets: []string{lb3}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL},
				{DNSName: c2RegionDNSName, Targets: []string{lb2}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL},
				{DNSName: c2ZoneDNSName, Targets: []string{lb2}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL},
			},
			expectError: false,
		},
		"UserConfiguredDNSRecordTTL": {
			dnsObject: feddnsv1a1.ServiceDNSRecord{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: feddnsv1a1.ServiceDNSRecordSpec{
					DomainRef: federation,
					RecordTTL: userConfiguredTTL,
				},
				Status: feddnsv1a1.ServiceDNSRecordStatus{
					Domain: dnsZone,
					DNS: []feddnsv1a1.ClusterDNS{
						{
							Cluster: c1, Zone: c1Zone, Region: c1Region,
							LoadBalancer: v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{{IP: lb1}}},
						},
					},
				},
			},
			expectEndpoints: []*feddnsv1a1.Endpoint{
				{DNSName: globalDNSName, Targets: []string{lb1}, RecordType: RecordTypeA, RecordTTL: userConfiguredTTL},
				{DNSName: c1RegionDNSName, Targets: []string{lb1}, RecordType: RecordTypeA, RecordTTL: userConfiguredTTL},
				{DNSName: c1ZoneDNSName, Targets: []string{lb1}, RecordType: RecordTypeA, RecordTTL: userConfiguredTTL},
			},
			expectError: false,
		},
		"UserConfiguredDNSPrefix": {
			dnsObject: feddnsv1a1.ServiceDNSRecord{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: feddnsv1a1.ServiceDNSRecordSpec{
					DomainRef: federation,
					DNSPrefix: "foo",
				},
				Status: feddnsv1a1.ServiceDNSRecordStatus{
					Domain: dnsZone,
					DNS: []feddnsv1a1.ClusterDNS{
						{
							Cluster: c1, Zone: c1Zone, Region: c1Region,
							LoadBalancer: v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{{IP: lb1}}},
						},
						{
							Cluster: c2, Zone: c2Zone, Region: c2Region,
							LoadBalancer: v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{{IP: lb2}}},
						},
					},
				},
			},
			expectEndpoints: []*feddnsv1a1.Endpoint{
				{DNSName: globalDNSName, Targets: []string{lb1, lb2}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL},
				{DNSName: c1RegionDNSName, Targets: []string{lb1}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL},
				{DNSName: c1ZoneDNSName, Targets: []string{lb1}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL},
				{DNSName: c2RegionDNSName, Targets: []string{lb2}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL},
				{DNSName: c2ZoneDNSName, Targets: []string{lb2}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL},
				{DNSName: "foo" + "." + dnsZone, Targets: []string{globalDNSName}, RecordType: RecordTypeCNAME, RecordTTL: defaultDNSTTL},
			},
			expectError: false,
		},
		"UserConfiguredExternalName": {
			dnsObject: feddnsv1a1.ServiceDNSRecord{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: feddnsv1a1.ServiceDNSRecordSpec{
					DomainRef:    federation,
					ExternalName: "foo",
				},
				Status: feddnsv1a1.ServiceDNSRecordStatus{
					Domain: dnsZone,
					DNS: []feddnsv1a1.ClusterDNS{
						{
							Cluster: c1, Zone: c1Zone, Region: c1Region,
							LoadBalancer: v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{{IP: lb1}}},
						},
						{
							Cluster: c2, Zone: c2Zone, Region: c2Region,
							LoadBalancer: v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{{IP: lb2}}},
						},
					},
				},
			},
			expectEndpoints: []*feddnsv1a1.Endpoint{
				{DNSName: "foo" + "." + globalDNSPrefix, Targets: []string{lb1, lb2}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL, Labels: labels},
				{DNSName: "foo" + "." + c1RegionDNSPrefix, Targets: []string{lb1}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL, Labels: labels},
				{DNSName: "foo" + "." + c1ZoneDNSPrefix, Targets: []string{lb1}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL, Labels: labels},
				{DNSName: "foo" + "." + c2RegionDNSPrefix, Targets: []string{lb2}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL, Labels: labels},
				{DNSName: "foo" + "." + c2ZoneDNSPrefix, Targets: []string{lb2}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL, Labels: labels},
			},
			expectError: false,
		},
	}

	for testName, tc := range testCases {
		t.Run(testName, func(t *testing.T) {
			endpoints, err := getServiceDNSEndpoints(&tc.dnsObject)
			if tc.expectError == false && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			} else if tc.expectError == true && err == nil {
				t.Fatalf("Expected to fail, but got success")
			}
			sort.Slice(tc.expectEndpoints, func(i, j int) bool {
				return tc.expectEndpoints[i].DNSName < tc.expectEndpoints[j].DNSName
			})
			if !reflect.DeepEqual(endpoints, tc.expectEndpoints) {
				t.Logf("Expected endpoints: %#v", tc.expectEndpoints)
				for _, ep := range tc.expectEndpoints {
					t.Logf("%+v", ep)
				}
				t.Logf("Actual endpoints: %#v", endpoints)
				for _, ep := range endpoints {
					t.Logf("%+v", ep)
				}
				t.Fatalf("Does not match expected endpoints")
			}
		})
	}
}
