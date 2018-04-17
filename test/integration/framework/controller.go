/*
Copyright 2018 The Federation v2 Authors.

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

package framework

import (
	"github.com/marun/federation-v2/pkg/controller/sync"
	"github.com/marun/federation-v2/pkg/federatedtypes"
	"github.com/marun/federation-v2/test/common"
	restclient "k8s.io/client-go/rest"
)

// ControllerFixture manages a federation controller for testing.
type ControllerFixture struct {
	stopChan chan struct{}
}

// NewControllerFixture initializes a new controller fixture
func NewControllerFixture(tl common.TestLogger, kind string, adapterFactory federatedtypes.AdapterFactory, fedConfig, kubeConfig, crConfig *restclient.Config) *ControllerFixture {
	f := &ControllerFixture{
		stopChan: make(chan struct{}),
	}
	err := sync.StartFederationSyncController(kind, adapterFactory, fedConfig, kubeConfig, crConfig, f.stopChan, true)
	if err != nil {
		tl.Fatalf("Error starting sync controller: %v", err)
	}
	return f
}

func (f *ControllerFixture) TearDown(tl common.TestLogger) {
	if f.stopChan != nil {
		close(f.stopChan)
		f.stopChan = nil
	}
}
