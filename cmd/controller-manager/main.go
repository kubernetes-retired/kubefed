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
	"fmt"
	"os"
	"strings"
	"time"

	// Import auth/gcp to connect to GKE clusters remotely
	"k8s.io/apiserver/pkg/util/logs"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/golang/glog"
	configlib "github.com/kubernetes-sigs/kubebuilder/pkg/config"
	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
	"github.com/kubernetes-sigs/kubebuilder/pkg/install"
	"github.com/kubernetes-sigs/kubebuilder/pkg/signals"
	extensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubernetes-sigs/federation-v2/pkg/controller/federatedcluster"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/federatedtypeconfig"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/ingressdns"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/schedulingpreference"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/servicedns"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/servicednsendpoint"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/features"
	"github.com/kubernetes-sigs/federation-v2/pkg/schedulingtypes"
	"github.com/kubernetes-sigs/federation-v2/pkg/version"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	flagutil "k8s.io/apiserver/pkg/util/flag"

	"github.com/kubernetes-sigs/federation-v2/pkg/inject"
	"github.com/kubernetes-sigs/federation-v2/pkg/inject/args"
)

var featureGates map[string]bool
var installCRDs = flag.Bool("install-crds", true, "install the CRDs used by the controller as part of startup")

// Controller-manager main.
func main() {
	verFlag := flag.Bool("version", false, "Prints the version info of controller-manager")
	flag.Var(flagutil.NewMapStringBool(&featureGates), "feature-gates", "A set of key=value pairs that describe feature gates for alpha/experimental features. "+
		"Options are:\n"+strings.Join(utilfeature.DefaultFeatureGate.KnownFeatures(), "\n"))

	flag.Parse()
	if *verFlag {
		fmt.Fprintf(os.Stdout, "Federation v2 controller-manager version: %s\n", fmt.Sprintf("%#v", version.Get()))
		os.Exit(0)
	}

	logs.InitLogs()
	defer logs.FlushLogs()

	err := utilfeature.DefaultFeatureGate.SetFromMap(featureGates)
	if err != nil {
		glog.Fatalf("Invalid Feature Gate: %v", err)
	}

	stopChan := signals.SetupSignalHandler()

	config := configlib.GetConfigOrDie()

	if *installCRDs {
		if err := install.NewInstaller(config).Install(&InstallStrategy{crds: inject.Injector.CRDs}); err != nil {
			glog.Fatalf("Could not create CRDs: %v", err)
		}
	}

	// TODO(marun) Make configurable and default to a namespace provided via the downward api
	fedNamespace := util.DefaultFederationSystemNamespace
	clusterNamespace := util.MulticlusterPublicNamespace
	targetNamespace := metav1.NamespaceAll

	// TODO(marun) Make the monitor period configurable
	clusterMonitorPeriod := time.Second * 40
	federatedcluster.StartClusterController(config, fedNamespace, clusterNamespace, stopChan, clusterMonitorPeriod)

	if utilfeature.DefaultFeatureGate.Enabled(features.SchedulerPreferences) {
		for kind, schedulingType := range schedulingtypes.SchedulingTypes() {
			err = schedulingpreference.StartSchedulingPreferenceController(kind, schedulingType.SchedulerFactory, config, fedNamespace, clusterNamespace, targetNamespace, stopChan, true)
			if err != nil {
				glog.Fatalf("Error starting schedulingpreference controller for %q : %v", kind, err)
			}
		}
	}

	if utilfeature.DefaultFeatureGate.Enabled(features.CrossClusterServiceDiscovery) {
		err = servicedns.StartController(config, fedNamespace, clusterNamespace, targetNamespace, stopChan, false)
		if err != nil {
			glog.Fatalf("Error starting dns controller: %v", err)
		}

		err = servicednsendpoint.StartController(config, stopChan, false)
		if err != nil {
			glog.Fatalf("Error starting dns endpoint controller: %v", err)
		}
	}

	if utilfeature.DefaultFeatureGate.Enabled(features.FederatedIngress) {
		err = ingressdns.StartController(config, fedNamespace, clusterNamespace, targetNamespace, stopChan, false)
		if err != nil {
			glog.Fatalf("Error starting ingress dns controller: %v", err)
		}
	}

	if utilfeature.DefaultFeatureGate.Enabled(features.PushReconciler) {
		// TODO(marun) Reconsider using kubebuilder framework to start
		// the controller.  It's not a good fit.
		inject.Inject = append(inject.Inject, func(arguments args.InjectArgs) error {
			if c, err := federatedtypeconfig.ProvideController(arguments, fedNamespace, clusterNamespace, targetNamespace, stopChan); err != nil {
				return err
			} else {
				arguments.ControllerManager.AddController(c)
			}
			return nil
		})
		// RunAll will never return - wrap in goroutine to avoid blocking
		go func() {
			if err := inject.RunAll(run.RunArguments{Stop: stopChan}, args.CreateInjectArgs(config)); err != nil {
				glog.Fatalf("%v", err)
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
