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

package framework

import (
	"time"

	"github.com/kubernetes-sigs/federation-v2/pkg/controller/sync"
	"github.com/kubernetes-sigs/federation-v2/pkg/federatedtypes"
	"github.com/kubernetes-sigs/federation-v2/test/common"

	"github.com/marun/fnord/pkg/controller/federatedcluster"
	restclient "k8s.io/client-go/rest"
)

// ControllerFixture manages a federation controller for testing.
type ControllerFixture struct {
	stopChan chan struct{}
}

// NewSyncControllerFixture initializes a new sync controller fixture.
func NewSyncControllerFixture(tl common.TestLogger, kind string, adapterFactory federatedtypes.AdapterFactory, fedConfig, kubeConfig, crConfig *restclient.Config) *ControllerFixture {
	f := &ControllerFixture{
		stopChan: make(chan struct{}),
	}
	sync.StartFederationSyncController(kind, adapterFactory, fedConfig, kubeConfig, crConfig, f.stopChan, true)
	return f
}

// NewClusterControllerFixture initializes a new cluster controller fixture.
func NewClusterControllerFixture(fedConfig, kubeConfig, crConfig *restclient.Config) *ControllerFixture {
	f := &ControllerFixture{
		stopChan: make(chan struct{}),
	}
	monitorPeriod := 1 * time.Second
	federatedcluster.StartClusterController(fedConfig, kubeConfig, crConfig,
		f.stopChan, monitorPeriod)
	return f
}

func (f *ControllerFixture) TearDown(tl common.TestLogger) {
	if f.stopChan != nil {
		close(f.stopChan)
		f.stopChan = nil
	}
}
