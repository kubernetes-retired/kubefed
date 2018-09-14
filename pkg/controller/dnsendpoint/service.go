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
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	restclient "k8s.io/client-go/rest"

	feddnsv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/multiclusterdns/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
)

func StartServiceDNSEndpointController(config *restclient.Config, targetNamespace string, stopChan <-chan struct{}, minimizeLatency bool) error {
	restclient.AddUserAgent(config, "Service DNSEndpoint")
	client := fedclientset.NewForConfigOrDie(config)

	listFunc := func(options metav1.ListOptions) (pkgruntime.Object, error) {
		return client.MulticlusterdnsV1alpha1().MultiClusterServiceDNSRecords(targetNamespace).List(options)
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		return client.MulticlusterdnsV1alpha1().MultiClusterServiceDNSRecords(targetNamespace).Watch(options)
	}

	controller, err := newDNSEndpointController(client, &feddnsv1a1.MultiClusterServiceDNSRecord{}, "service",
		listFunc, watchFunc, getServiceDNSEndpoints, minimizeLatency)
	if err != nil {
		return err
	}

	go controller.Run(stopChan)
	return nil
}

// getServiceDNSEndpoints returns endpoint objects for each MultiClusterServiceDNSRecord object that should be processed.
func getServiceDNSEndpoints(obj interface{}) ([]*feddnsv1a1.Endpoint, error) {
	var endpoints []*feddnsv1a1.Endpoint

	dnsObject, ok := obj.(*feddnsv1a1.MultiClusterServiceDNSRecord)
	if !ok {
		return nil, fmt.Errorf("received event for unknown object %v", obj)
	}

	commonPrefix := strings.Join([]string{dnsObject.Name, dnsObject.Namespace, dnsObject.Spec.FederationName, "svc"}, ".")
	for _, clusterDNS := range dnsObject.Status.DNS {
		zone := clusterDNS.Zone
		region := clusterDNS.Region

		dnsNames := []string{
			strings.Join([]string{commonPrefix, zone, region, dnsObject.Spec.DNSSuffix}, "."), // zone level
			strings.Join([]string{commonPrefix, region, dnsObject.Spec.DNSSuffix}, "."),       // region level, one up from zone level
			strings.Join([]string{commonPrefix, dnsObject.Spec.DNSSuffix}, "."),               // global level, one up from region level
			"", // nowhere to go up from global level
		}

		var zoneTargets, regionTargets, globalTargets feddnsv1a1.Targets
		for _, clusterDNS := range dnsObject.Status.DNS {
			if clusterDNS.Zone == zone {
				zoneTargets = append(zoneTargets, ExtractLoadBalancerTargets(clusterDNS.LoadBalancer)...)
			}
		}

		for _, clusterDNS := range dnsObject.Status.DNS {
			if clusterDNS.Region == region {
				regionTargets = append(regionTargets, ExtractLoadBalancerTargets(clusterDNS.LoadBalancer)...)
			}
		}

		for _, clusterDNS := range dnsObject.Status.DNS {
			globalTargets = append(globalTargets, ExtractLoadBalancerTargets(clusterDNS.LoadBalancer)...)
		}

		targets := [][]string{zoneTargets, regionTargets, globalTargets}

		ttl := dnsObject.Spec.RecordTTL
		if ttl == 0 {
			ttl = defaultDNSTTL
		}
		for i, target := range targets {
			endpoint, err := generateEndpointForServiceDNSObject(dnsNames[i], target, dnsNames[i+1], ttl)
			if err != nil {
				return nil, err
			}
			endpoints = append(endpoints, endpoint)
		}
	}

	return DedupeAndMergeEndpoints(endpoints), nil
}

func generateEndpointForServiceDNSObject(name string, targets feddnsv1a1.Targets, uplevelCname string, ttl feddnsv1a1.TTL) (ep *feddnsv1a1.Endpoint, err error) {
	ep = &feddnsv1a1.Endpoint{
		DNSName:   name,
		RecordTTL: ttl,
	}

	if len(targets) > 0 {
		targets, err = getResolvedTargets(targets, netWrapper)
		if err != nil {
			return nil, err
		}
		ep.Targets = targets
		ep.RecordType = RecordTypeA
	} else {
		ep.Targets = []string{uplevelCname}
		ep.RecordType = RecordTypeCNAME
	}

	return ep, nil
}
