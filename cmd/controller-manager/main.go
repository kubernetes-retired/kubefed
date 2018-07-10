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

	// Import auth/gcp to connect to GKE clusters remotely
	"k8s.io/apiserver/pkg/util/logs"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	configlib "github.com/kubernetes-sigs/kubebuilder/pkg/config"
	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
	"github.com/kubernetes-sigs/kubebuilder/pkg/install"
	"github.com/kubernetes-sigs/kubebuilder/pkg/signals"
	extensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"

	"github.com/kubernetes-sigs/federation-v2/pkg/controller/federatedcluster"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/federatedtypeconfig"
	rspcontroller "github.com/kubernetes-sigs/federation-v2/pkg/controller/replicaschedulingpreference"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/servicedns"
	"github.com/kubernetes-sigs/federation-v2/pkg/features"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	flagutil "k8s.io/apiserver/pkg/util/flag"

	"github.com/kubernetes-sigs/federation-v2/pkg/inject"
	"github.com/kubernetes-sigs/federation-v2/pkg/inject/args"
)

var featureGates map[string]bool
var installCRDs = flag.Bool("install-crds", true, "install the CRDs used by the controller as part of startup")

// Controller-manager main.
func main() {
	flag.Var(flagutil.NewMapStringBool(&featureGates), "feature-gates", "A set of key=value pairs that describe feature gates for alpha/experimental features. "+
		"Options are:\n"+strings.Join(utilfeature.DefaultFeatureGate.KnownFeatures(), "\n"))

	flag.Parse()

	logs.InitLogs()
	defer logs.FlushLogs()

	err := utilfeature.DefaultFeatureGate.SetFromMap(featureGates)
	if err != nil {
		log.Fatalf("Invalid Feature Gate: %v", err)
	}

	stopChan := signals.SetupSignalHandler()

	config := configlib.GetConfigOrDie()

	if *installCRDs {
		if err := install.NewInstaller(config).Install(&InstallStrategy{crds: inject.Injector.CRDs}); err != nil {
			log.Fatalf("Could not create CRDs: %v", err)
		}
	}

	// TODO(marun) Make the monitor period configurable
	clusterMonitorPeriod := time.Second * 40
	federatedcluster.StartClusterController(config, stopChan, clusterMonitorPeriod)

	if utilfeature.DefaultFeatureGate.Enabled(features.SchedulerPreferences) {
		err = rspcontroller.StartReplicaSchedulingPreferenceController(config, stopChan, true)
		if err != nil {
			log.Fatalf("Error starting replicaschedulingpreference controller: %v", err)
		}
	}

	if utilfeature.DefaultFeatureGate.Enabled(features.CrossClusterServiceDiscovery) {
		err = servicedns.StartController(config, stopChan, false)
		if err != nil {
			log.Fatalf("Error starting dns controller: %v", err)
		}
	}

	if utilfeature.DefaultFeatureGate.Enabled(features.PushReconciler) {
		// TODO(marun) Reconsider using kubebuilder framework to start
		// the controller.  It's not a good fit.
		inject.Inject = append(inject.Inject, func(arguments args.InjectArgs) error {
			if c, err := federatedtypeconfig.ProvideController(arguments, stopChan); err != nil {
				return err
			} else {
				arguments.ControllerManager.AddController(c)
			}
			return nil
		})
		// RunAll will never return - wrap in goroutine to avoid blocking
		go func() {
			if err := inject.RunAll(run.RunArguments{Stop: stopChan}, args.CreateInjectArgs(config)); err != nil {
				log.Fatalf("%v", err)
			}
		}()
	}

	// Blockforever
	select {}
}

type InstallStrategy struct {
	install.EmptyInstallStrategy
	crds []*extensionsv1beta1.CustomResourceDefinition
}

func (s *InstallStrategy) GetCRDs() []*extensionsv1beta1.CustomResourceDefinition {
	return s.crds
}
