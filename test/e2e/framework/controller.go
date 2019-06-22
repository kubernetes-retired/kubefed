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
	"io"
	"os/exec"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/kubefed/pkg/apis/core/typeconfig"
	"sigs.k8s.io/kubefed/pkg/controller/dnsendpoint"
	"sigs.k8s.io/kubefed/pkg/controller/federatedtypeconfig"
	"sigs.k8s.io/kubefed/pkg/controller/ingressdns"
	"sigs.k8s.io/kubefed/pkg/controller/kubefedcluster"
	"sigs.k8s.io/kubefed/pkg/controller/schedulingmanager"
	"sigs.k8s.io/kubefed/pkg/controller/servicedns"
	"sigs.k8s.io/kubefed/pkg/controller/status"
	"sigs.k8s.io/kubefed/pkg/controller/sync"
	"sigs.k8s.io/kubefed/pkg/controller/util"
	"sigs.k8s.io/kubefed/test/common"
)

// ControllerFixture manages a KubeFed controller for testing.
type ControllerFixture struct {
	stopChan chan struct{}
}

// NewSyncControllerFixture initializes a new sync controller fixture.
func NewSyncControllerFixture(tl common.TestLogger, controllerConfig *util.ControllerConfig, typeConfig typeconfig.Interface, namespacePlacement *metav1.APIResource) *ControllerFixture {
	f := &ControllerFixture{
		stopChan: make(chan struct{}),
	}
	err := sync.StartKubeFedSyncController(controllerConfig, f.stopChan, typeConfig, namespacePlacement)
	if err != nil {
		tl.Fatalf("Error starting sync controller: %v", err)
	}
	if typeConfig.GetStatusEnabled() {
		err := status.StartKubeFedStatusController(controllerConfig, f.stopChan, typeConfig)
		if err != nil {
			tl.Fatalf("Error starting status controller: %v", err)
		}
	}
	return f
}

// NewFederatedTypeConfigControllerFixure initializes a new federatedtypeconfig
// controller fixure.
func NewFederatedTypeConfigControllerFixture(tl common.TestLogger, config *util.ControllerConfig) *ControllerFixture {
	f := &ControllerFixture{
		stopChan: make(chan struct{}),
	}

	err := federatedtypeconfig.StartController(config, f.stopChan)
	if err != nil {
		tl.Fatalf("Error starting federatedtypeconfig controller: %v", err)
	}
	return f
}

// NewServiceDNSControllerFixture initializes a new service-dns controller fixture.
func NewServiceDNSControllerFixture(tl common.TestLogger, config *util.ControllerConfig) *ControllerFixture {
	f := &ControllerFixture{
		stopChan: make(chan struct{}),
	}
	err := servicedns.StartController(config, f.stopChan)
	if err != nil {
		tl.Fatalf("Error starting service dns controller: %v", err)
	}
	err = dnsendpoint.StartServiceDNSEndpointController(config, f.stopChan)
	if err != nil {
		tl.Fatalf("Error starting service dns endpoint controller: %v", err)
	}
	return f
}

// NewIngressDNSControllerFixture initializes a new ingress-dns controller fixture.
func NewIngressDNSControllerFixture(tl common.TestLogger, config *util.ControllerConfig) *ControllerFixture {
	f := &ControllerFixture{
		stopChan: make(chan struct{}),
	}
	err := ingressdns.StartController(config, f.stopChan)
	if err != nil {
		tl.Fatalf("Error starting ingress dns controller: %v", err)
	}
	err = dnsendpoint.StartIngressDNSEndpointController(config, f.stopChan)
	if err != nil {
		tl.Fatalf("Error starting ingress dns endpoint controller: %v", err)
	}
	return f
}

// NewClusterControllerFixture initializes a new cluster controller fixture.
func NewClusterControllerFixture(tl common.TestLogger, config *util.ControllerConfig) *ControllerFixture {
	f := &ControllerFixture{
		stopChan: make(chan struct{}),
	}
	clusterHealthCheckConfig := &util.ClusterHealthCheckConfig{Period: 1 * time.Second, FailureThreshold: 1}
	err := kubefedcluster.StartClusterController(config, clusterHealthCheckConfig, f.stopChan)
	if err != nil {
		tl.Fatalf("Error starting cluster controller: %v", err)
	}
	return f
}

func NewSchedulingManagerFixture(tl common.TestLogger, config *util.ControllerConfig) (*ControllerFixture, *schedulingmanager.SchedulingManager) {
	f := &ControllerFixture{
		stopChan: make(chan struct{}),
	}

	manager, err := schedulingmanager.StartSchedulingManager(config, f.stopChan)
	if err != nil {
		tl.Fatalf("Error starting scheduler controller: %v", err)
	}
	return f, manager
}

func StartControllerManager(args []string) (*exec.Cmd, io.ReadCloser, error) {
	cmd := exec.Command("controller-manager", args...)

	logStream, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	return cmd, logStream, nil
}

func (f *ControllerFixture) TearDown(tl common.TestLogger) {
	if f.stopChan != nil {
		close(f.stopChan)
		f.stopChan = nil
	}
}
