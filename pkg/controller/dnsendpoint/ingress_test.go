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
	"testing"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	feddnsv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/multiclusterdns/v1alpha1"
)

func TestGetEndpointsForIngressDNSObject(t *testing.T) {
	// Fake out the internet
	netmock := &NetWrapperMock{}
	netmock.AddHost("a9.us-west-2.elb.amazonaws.test", []string{lb3})
	netWrapper = netmock

	testCases := map[string]struct {
		dnsObject       feddnsv1a1.IngressDNSRecord
		expectEndpoints []*feddnsv1a1.Endpoint
		expectError     bool
	}{
		"NoClusters": {
			dnsObject: feddnsv1a1.IngressDNSRecord{
				Spec: feddnsv1a1.IngressDNSRecordSpec{
					Hosts: []string{"foo.bar.test"},
				},
				Status: feddnsv1a1.IngressDNSRecordStatus{},
			},
			expectEndpoints: nil,
			expectError:     false,
		},
		"SingleLBInSingleCluster": {
			dnsObject: feddnsv1a1.IngressDNSRecord{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: feddnsv1a1.IngressDNSRecordSpec{
					Hosts: []string{"foo.bar.test"},
				},
				Status: feddnsv1a1.IngressDNSRecordStatus{
					DNS: []feddnsv1a1.ClusterIngressDNS{
						{
							Cluster:      c1,
							LoadBalancer: v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{{IP: lb1}}},
						},
					},
				},
			},
			expectEndpoints: []*feddnsv1a1.Endpoint{
				{DNSName: "foo.bar.test", Targets: []string{lb1}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL},
			},
			expectError: false,
		},
		"LBsInBothClusters": {
			dnsObject: feddnsv1a1.IngressDNSRecord{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: feddnsv1a1.IngressDNSRecordSpec{
					Hosts: []string{"foo.bar.test"},
				},
				Status: feddnsv1a1.IngressDNSRecordStatus{
					DNS: []feddnsv1a1.ClusterIngressDNS{
						{
							Cluster:      c1,
							LoadBalancer: v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{{IP: lb1}}},
						},
						{
							Cluster:      c2,
							LoadBalancer: v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{{IP: lb2}}},
						},
					},
				},
			},
			expectEndpoints: []*feddnsv1a1.Endpoint{
				{DNSName: "foo.bar.test", Targets: []string{lb1, lb2}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL},
			},
			expectError: false,
		},
		"HostnameInLB": {
			dnsObject: feddnsv1a1.IngressDNSRecord{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: feddnsv1a1.IngressDNSRecordSpec{
					Hosts: []string{"foo.bar.test"},
				},
				Status: feddnsv1a1.IngressDNSRecordStatus{
					DNS: []feddnsv1a1.ClusterIngressDNS{
						{
							Cluster:      c1,
							LoadBalancer: v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{{Hostname: "a9.us-west-2.elb.amazonaws.test"}}},
						},
						{
							Cluster:      c2,
							LoadBalancer: v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{{IP: lb2}}},
						},
					},
				},
			},
			expectEndpoints: []*feddnsv1a1.Endpoint{
				{DNSName: "foo.bar.test", Targets: []string{lb2, lb3}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL},
			},
			expectError: false,
		},
		"MultipleHosts": {
			dnsObject: feddnsv1a1.IngressDNSRecord{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: feddnsv1a1.IngressDNSRecordSpec{
					Hosts: []string{"foo.bar.test", "jane.goodall.test"},
				},
				Status: feddnsv1a1.IngressDNSRecordStatus{
					DNS: []feddnsv1a1.ClusterIngressDNS{
						{
							Cluster:      c1,
							LoadBalancer: v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{{IP: lb1}}},
						},
						{
							Cluster:      c2,
							LoadBalancer: v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{{IP: lb2}}},
						},
					},
				},
			},
			expectEndpoints: []*feddnsv1a1.Endpoint{
				{DNSName: "foo.bar.test", Targets: []string{lb1, lb2}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL},
				{DNSName: "jane.goodall.test", Targets: []string{lb1, lb2}, RecordType: RecordTypeA, RecordTTL: defaultDNSTTL},
			},
			expectError: false,
		},
		"UserConfiguredDNSRecordTTL": {
			dnsObject: feddnsv1a1.IngressDNSRecord{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: feddnsv1a1.IngressDNSRecordSpec{
					Hosts:     []string{"foo.bar.test"},
					RecordTTL: userConfiguredTTL,
				},
				Status: feddnsv1a1.IngressDNSRecordStatus{
					DNS: []feddnsv1a1.ClusterIngressDNS{
						{
							Cluster:      c1,
							LoadBalancer: v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{{IP: lb1}}},
						},
					},
				},
			},
			expectEndpoints: []*feddnsv1a1.Endpoint{
				{DNSName: "foo.bar.test", Targets: []string{lb1}, RecordType: RecordTypeA, RecordTTL: userConfiguredTTL},
			},
			expectError: false,
		},
	}

	for testName, tc := range testCases {
		t.Run(testName, func(t *testing.T) {
			endpoints, err := getIngressDNSEndpoints(&tc.dnsObject)
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
