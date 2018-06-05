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

package common

import (
	"reflect"
	"time"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	dnsv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/multiclusterdns/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
)

func NewServiceDNSObject(name, namespace string) *dnsv1a1.MultiClusterServiceDNSRecord {
	return &dnsv1a1.MultiClusterServiceDNSRecord{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func NewServiceObject(name, namespace string) *apiv1.Service {
	return &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: apiv1.ServiceSpec{
			Type: apiv1.ServiceTypeLoadBalancer,
			Ports: []apiv1.ServicePort{{
				Port: 80,
				Name: "http",
			}},
		},
	}
}

func NewEndpointObject(name, namespace string) *apiv1.Endpoints {
	return &apiv1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Subsets: []apiv1.EndpointSubset{{
			Addresses: []apiv1.EndpointAddress{{IP: "1.2.3.4"}},
			Ports:     []apiv1.EndpointPort{{Port: 80}},
		}},
	}
}

// WaitForObject waits for object to match the desired status.
func WaitForObject(tl TestLogger, namespace, name string, objectGetter func(namespace, name string) (pkgruntime.Object, error), desired pkgruntime.Object, interval, timeout time.Duration) {
	var actual pkgruntime.Object
	err := wait.PollImmediate(interval, timeout, func() (exist bool, err error) {
		actual, err = objectGetter(namespace, name)
		if err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}

		return equivalent(actual, desired), nil
	})
	if err != nil {
		tl.Fatalf("Timedout waiting for desired state, \ndesired:%#v\nactual :%#v", desired, actual)
	}
}

// WaitForObjectDeletion waits for the object to be deleted.
func WaitForObjectDeletion(tl TestLogger, namespace, name string, objectGetter func(namespace, name string) (pkgruntime.Object, error), interval, timeout time.Duration) {
	err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		_, err := objectGetter(namespace, name)
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
	if err != nil {
		tl.Fatalf("Timedout waiting for object %q/%q to be deleted", namespace, name)
	}
}

func equivalent(actual, desired pkgruntime.Object) bool {
	// Check for meta & spec equivalence
	if !util.ObjectMetaAndSpecEquivalent(actual, desired) {
		return false
	}

	// Check for status equivalence
	statusActual := reflect.ValueOf(actual).Elem().FieldByName("Status").Interface()
	statusDesired := reflect.ValueOf(desired).Elem().FieldByName("Status").Interface()
	return reflect.DeepEqual(statusActual, statusDesired)
}
