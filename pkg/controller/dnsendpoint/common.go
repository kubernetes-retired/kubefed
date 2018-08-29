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
	"net"
	"sort"

	"github.com/golang/glog"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	feddnsv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/multiclusterdns/v1alpha1"
)

const (
	// defaultDNSTTL is the default DNS record TTL value to use (in seconds) when user has not provided specific value.
	defaultDNSTTL = 180

	// RecordTypeA is a RecordType enum value
	RecordTypeA = "A"
	// RecordTypeCNAME is a RecordType enum value
	RecordTypeCNAME = "CNAME"
)

// Abstracting away the internet for testing purposes
type NetWrapper interface {
	LookupHost(host string) (addrs []string, err error)
}

type NetWrapperDefaultImplementation struct{}

func (r *NetWrapperDefaultImplementation) LookupHost(host string) (addrs []string, err error) {
	return net.LookupHost(host)
}

var netWrapper NetWrapper

func init() {
	netWrapper = &NetWrapperDefaultImplementation{}
}

// getResolvedTargets performs DNS resolution on the provided slice of endpoints (which might be DNS names
// or IPv4 addresses) and returns a list of IPv4 addresses.  If any of the endpoints are neither valid IPv4
// addresses nor resolvable DNS names, non-nil error is also returned (possibly along with a partially
// complete list of resolved endpoints.
func getResolvedTargets(targets feddnsv1a1.Targets, netWrapper NetWrapper) (feddnsv1a1.Targets, error) {
	resolvedTargets := sets.String{}
	for _, target := range targets {
		if net.ParseIP(target) == nil {
			// It's not a valid IP address, so assume it's a DNS name, and try to resolve it,
			// replacing its DNS name with its IP addresses in expandedEndpoints
			// through an interface abstracting the internet
			ipAddrs, err := netWrapper.LookupHost(target)
			if err != nil {
				glog.Errorf("Failed to resolve %s, err: %v", target, err)
				return resolvedTargets.List(), err
			}
			for _, ip := range ipAddrs {
				resolvedTargets.Insert(ip)
			}
		} else {
			resolvedTargets.Insert(target)
		}
	}
	return resolvedTargets.List(), nil
}

func ExtractLoadBalancerTargets(lbStatus corev1.LoadBalancerStatus) feddnsv1a1.Targets {
	var targets feddnsv1a1.Targets

	for _, lb := range lbStatus.Ingress {
		if lb.IP != "" {
			targets = append(targets, lb.IP)
		}
		if lb.Hostname != "" {
			targets = append(targets, lb.Hostname)
		}
	}

	return targets
}

// Merge and remove duplicate endpoints
func DedupeAndMergeEndpoints(endpoints []*feddnsv1a1.Endpoint) (result []*feddnsv1a1.Endpoint) {
	// Sort endpoints by DNSName
	sort.Slice(endpoints, func(i, j int) bool {
		return endpoints[i].DNSName < endpoints[j].DNSName
	})

	// Remove the endpoint with no targets/ empty targets
	for i := 0; i < len(endpoints); {
		for j := 0; j < len(endpoints[i].Targets); {
			if endpoints[i].Targets[j] == "" {
				endpoints[i].Targets = append(endpoints[i].Targets[:j], endpoints[i].Targets[j+1:]...)
				continue
			}
			j++
		}
		if len(endpoints[i].Targets) == 0 {
			endpoints = append(endpoints[:i], endpoints[i+1:]...)
			continue
		}
		i++
	}

	// Merge endpoints with same DNSName
	for i := 1; i < len(endpoints); {
		if endpoints[i].DNSName == endpoints[i-1].DNSName {
			// Merge targets
			endpoints[i-1].Targets = append(endpoints[i-1].Targets, endpoints[i].Targets...)
			endpoints[i-1].Targets = sortAndRemoveDuplicateTargets(endpoints[i-1].Targets)

			// Remove the duplicate endpoint
			endpoints = append(endpoints[:i], endpoints[i+1:]...)
			continue
		}
		i++
	}

	return endpoints
}

func sortAndRemoveDuplicateTargets(targets []string) []string {
	sort.Slice(targets, func(i, j int) bool {
		return targets[i] < targets[j]
	})
	for i := 1; i < len(targets); {
		if targets[i] == targets[i-1] {
			// Remove duplicate target
			targets = append(targets[:i], targets[i+1:]...)
			continue
		}
		i++
	}
	return targets
}
