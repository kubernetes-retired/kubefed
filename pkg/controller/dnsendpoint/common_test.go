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
)

const (
	name      = "nginx"
	namespace = "test"

	c1 = "c1"
	c2 = "c2"

	lb1 = "10.20.30.1"
	lb2 = "10.20.30.2"
	lb3 = "10.20.30.3"

	userConfiguredTTL = 300
)

type NetWrapperMock struct {
	result map[string][]string
}

func (mock *NetWrapperMock) LookupHost(host string) (addrs []string, err error) {

	// If nothing to return, return empty list
	if mock.result == nil || len(mock.result) == 0 {
		return make([]string, 0), fmt.Errorf("Mock error response")
	}

	return mock.result[host], nil
}

func (mock *NetWrapperMock) AddHost(host string, addrs []string) {
	// Initialise if null
	if mock.result == nil {
		mock.result = make(map[string][]string)
	}

	mock.result[host] = addrs
}
