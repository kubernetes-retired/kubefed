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
	"strings"
	"time"

	controllerlib "github.com/kubernetes-incubator/apiserver-builder/pkg/controller"
	"github.com/kubernetes-sigs/federation-v2/pkg/client/clientset_generated/clientset"
	"github.com/kubernetes-sigs/federation-v2/pkg/client/informers_generated/externalversions"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/federatedcluster"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/manager"
	rspcontroller "github.com/kubernetes-sigs/federation-v2/pkg/controller/replicaschedulingpreference"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/servicedns"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/sharedinformers"
	"github.com/kubernetes-sigs/federation-v2/pkg/features"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	flagutil "k8s.io/apiserver/pkg/util/flag"
)

var kubeconfig = flag.String("kubeconfig", "", "path to kubeconfig")
var featureGates map[string]bool

func main() {
	flag.Var(flagutil.NewMapStringBool(&featureGates), "feature-gates", "A set of key=value pairs that describe feature gates for alpha/experimental features. "+
		"Options are:\n"+strings.Join(utilfeature.DefaultFeatureGate.KnownFeatures(), "\n"))

	flag.Parse()
	config, err := controllerlib.GetConfig(*kubeconfig)
	if err != nil {
		log.Fatalf("Could not create Config for talking to the apiserver: %v", err)
	}

	err = utilfeature.DefaultFeatureGate.SetFromMap(featureGates)
	if err != nil {
		log.Fatalf("Invalid Feature Gate: %v", err)
	}

	stopChan := make(chan struct{})

	// Configuration is passed in separately for the kube, federation
	// and cluster registry clients.  When deployed in an aggregated
	// configuration - as this controller manager is intended to run -
	// requires that all 3 clients receive the same configuration.

	// TODO(marun) Make the monitor period configurable
	clusterMonitorPeriod := time.Second * 40
	federatedcluster.StartClusterController(config, config, config, stopChan, clusterMonitorPeriod)

	if utilfeature.DefaultFeatureGate.Enabled(features.PushReconciler) {
		// Initialize shared informer to enable reuse of the controller factory.
		// TODO(marun) Shared informer doesn't makes sense for FederatedTypeConfig.
		si := &sharedinformers.SharedInformers{
			controllerlib.SharedInformersDefaults{},
			externalversions.NewSharedInformerFactory(clientset.NewForConfigOrDie(config), 10*time.Minute),
		}
		go si.Factory.Federation().V1alpha1().FederatedTypeConfigs().Informer().Run(stopChan)

		c := manager.NewFederatedTypeConfigController(config, si)
		c.Run(stopChan)
	}

	if utilfeature.DefaultFeatureGate.Enabled(features.SchedulerPreferences) {
		err = rspcontroller.StartReplicaSchedulingPreferenceController(config, config, config, stopChan, true)
		if err != nil {
			log.Fatalf("Error starting replicaschedulingpreference controller: %v", err)
		}
	}

	err = servicedns.StartController(config, config, config, stopChan, false)
	if err != nil {
		log.Fatalf("Error starting dns controller: %v", err)
	}

	// Blockforever
	select {}
}
