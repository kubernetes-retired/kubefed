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
	"strings"

	"github.com/pkg/errors"

	restclient "k8s.io/client-go/rest"

	feddnsv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/multiclusterdns/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
)

func StartServiceDNSEndpointController(config *util.ControllerConfig, stopChan <-chan struct{}) error {
	restclient.AddUserAgent(config.KubeConfig, "Service DNSEndpoint")
	controller, err := newDNSEndpointController(config, &feddnsv1a1.ServiceDNSRecord{}, "service",
		getServiceDNSEndpoints, config.MinimizeLatency)
	if err != nil {
		return err
	}

	go controller.Run(stopChan)
	return nil
}

// getServiceDNSEndpoints returns endpoint objects for each ServiceDNSRecord object that should be processed.
func getServiceDNSEndpoints(obj interface{}) ([]*feddnsv1a1.Endpoint, error) {
	var endpoints []*feddnsv1a1.Endpoint
	var commonPrefix string
	labels := make(map[string]string)

	dnsObject, ok := obj.(*feddnsv1a1.ServiceDNSRecord)
	if !ok {
		return nil, errors.Errorf("received event for unknown object %v", obj)
	}

	if dnsObject.Spec.ExternalName != "" {
		commonPrefix = strings.Join([]string{dnsObject.Spec.ExternalName, dnsObject.Namespace, dnsObject.Spec.DomainRef,
			"svc"}, ".")
		labels["serviceName"] = dnsObject.Name
	} else {
		commonPrefix = strings.Join([]string{dnsObject.Name, dnsObject.Namespace, dnsObject.Spec.DomainRef, "svc"}, ".")
	}

	ttl := dnsObject.Spec.RecordTTL
	if ttl == 0 {
		ttl = defaultDNSTTL
	}

	for _, clusterDNS := range dnsObject.Status.DNS {
		var zoneDNSName string
		regionDNSName := strings.Join([]string{commonPrefix, clusterDNS.Region, dnsObject.Status.Domain}, ".") // region level, one up from zone level
		globalDNSName := strings.Join([]string{commonPrefix, dnsObject.Status.Domain}, ".")                    // global level, one up from region level

		// Zone endpoints
		for _, zone := range clusterDNS.Zones {
			zoneDNSName = strings.Join([]string{commonPrefix, zone, clusterDNS.Region, dnsObject.Status.Domain}, ".")
			zoneTargets := ExtractLoadBalancerTargets(clusterDNS.LoadBalancer)
			zoneEndpoint, err := generateEndpointForServiceDNSObject(zoneDNSName, zoneTargets, regionDNSName, ttl, labels)
			if err != nil {
				return nil, err
			}
			endpoints = append(endpoints, zoneEndpoint)
		}

		// Region endpoints
		regionTargets := ExtractLoadBalancerTargets(clusterDNS.LoadBalancer)
		regionEndpoint, err := generateEndpointForServiceDNSObject(regionDNSName, regionTargets, globalDNSName, ttl, labels)
		if err != nil {
			return nil, err
		}
		endpoints = append(endpoints, regionEndpoint)

		// Global endpoints
		globalTargets := ExtractLoadBalancerTargets(clusterDNS.LoadBalancer)
		globalEndpoint, err := generateEndpointForServiceDNSObject(globalDNSName, globalTargets, "", ttl, labels)
		if err != nil {
			return nil, err
		}
		endpoints = append(endpoints, globalEndpoint)
	}

	if dnsObject.Spec.DNSPrefix != "" {
		endpoint := &feddnsv1a1.Endpoint{
			DNSName:    dnsObject.Spec.DNSPrefix + "." + dnsObject.Status.Domain,
			RecordTTL:  ttl,
			RecordType: RecordTypeCNAME,
		}
		endpoint.Targets = []string{strings.Join([]string{commonPrefix, dnsObject.Status.Domain}, ".")}
		endpoints = append(endpoints, endpoint)
	}

	return DedupeAndMergeEndpoints(endpoints), nil
}

func generateEndpointForServiceDNSObject(name string, targets feddnsv1a1.Targets, uplevelCname string,
	ttl feddnsv1a1.TTL, labels map[string]string) (ep *feddnsv1a1.Endpoint, err error) {
	ep = &feddnsv1a1.Endpoint{
		DNSName:   name,
		RecordTTL: ttl,
	}

	if len(labels) > 0 {
		ep.Labels = labels
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
