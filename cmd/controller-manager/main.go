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

package main

import (
	"flag"
	"log"
	"time"

	controllerlib "github.com/kubernetes-incubator/apiserver-builder/pkg/controller"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/federatedcluster"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/sync"
	"github.com/kubernetes-sigs/federation-v2/pkg/federatedtypes"
)

var kubeconfig = flag.String("kubeconfig", "", "path to kubeconfig")

func main() {
	flag.Parse()
	config, err := controllerlib.GetConfig(*kubeconfig)
	if err != nil {
		log.Fatalf("Could not create Config for talking to the apiserver: %v", err)
	}

	stopChan := make(chan struct{})

	// Configuration is passed in separately for the kube, federation
	// and cluster registry clients.  When deployed in an aggregated
	// configuration - as this controller manager is intended to run -
	// requires that all 3 clients receive the same configuration.

	// TODO(marun) Make the monitor period configurable
	clusterMonitorPeriod := time.Second * 40
	federatedcluster.StartClusterController(config, config, config, stopChan, clusterMonitorPeriod)

	for kind, fedTypeConfig := range federatedtypes.FederatedTypeConfigs() {
		sync.StartFederationSyncController(kind, fedTypeConfig.AdapterFactory, config, config, config, stopChan, false)
	}

	// Blockforever
	select {}
}
