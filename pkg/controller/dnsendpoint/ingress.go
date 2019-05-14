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
	"github.com/pkg/errors"

	restclient "k8s.io/client-go/rest"

	feddnsv1a1 "sigs.k8s.io/kubefed/pkg/apis/multiclusterdns/v1alpha1"
	"sigs.k8s.io/kubefed/pkg/controller/util"
)

func StartIngressDNSEndpointController(config *util.ControllerConfig, stopChan <-chan struct{}) error {
	restclient.AddUserAgent(config.KubeConfig, "Ingress DNSEndpoint")
	controller, err := newDNSEndpointController(config, &feddnsv1a1.IngressDNSRecord{}, "ingress",
		getIngressDNSEndpoints, config.MinimizeLatency)
	if err != nil {
		return err
	}

	go controller.Run(stopChan)
	return nil
}

// getIngressDNSEndpoints returns endpoint objects for each IngressDNSRecord object that should be processed.
func getIngressDNSEndpoints(obj interface{}) ([]*feddnsv1a1.Endpoint, error) {
	var endpoints []*feddnsv1a1.Endpoint

	dnsObject, ok := obj.(*feddnsv1a1.IngressDNSRecord)
	if !ok {
		return nil, errors.Errorf("received event for unknown object %v", obj)
	}

	ttl := dnsObject.Spec.RecordTTL
	if ttl == 0 {
		ttl = defaultDNSTTL
	}
	for _, host := range dnsObject.Spec.Hosts {
		var targets feddnsv1a1.Targets
		for _, clusterDNS := range dnsObject.Status.DNS {
			targets = append(targets, ExtractLoadBalancerTargets(clusterDNS.LoadBalancer)...)
		}
		endpoint, err := generateEndpointForIngressDNSObject(host, targets, ttl)
		if err != nil {
			return nil, err
		}
		if endpoint != nil {
			endpoints = append(endpoints, endpoint)
		}
	}

	return DedupeAndMergeEndpoints(endpoints), nil
}

func generateEndpointForIngressDNSObject(name string, targets feddnsv1a1.Targets, ttl feddnsv1a1.TTL) (ep *feddnsv1a1.Endpoint, err error) {
	if len(targets) == 0 {
		return nil, nil
	}

	ep = &feddnsv1a1.Endpoint{
		DNSName:   name,
		RecordTTL: ttl,
	}

	targets, err = getResolvedTargets(targets, netWrapper)
	if err != nil {
		return nil, err
	}
	ep.Targets = targets
	ep.RecordType = RecordTypeA
	return ep, nil
}
