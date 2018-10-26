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
	"github.com/kubernetes-sigs/kubebuilder/pkg/install"
	"github.com/kubernetes-sigs/kubebuilder/pkg/signals"
	extensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubernetes-sigs/federation-v2/pkg/controller/dnsendpoint"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/federatedcluster"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/federatedtypeconfig"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/ingressdns"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/schedulingpreference"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/servicedns"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/features"
	"github.com/kubernetes-sigs/federation-v2/pkg/version"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	flagutil "k8s.io/apiserver/pkg/util/flag"

	"github.com/kubernetes-sigs/federation-v2/pkg/inject"
)

var featureGates map[string]bool
var fedNamespace string
var clusterNamespace string
var limitedScope bool
var installCRDs = flag.Bool("install-crds", true, "install the CRDs used by the controller as part of startup")

// Controller-manager main.
func main() {
	verFlag := flag.Bool("version", false, "Prints the version info of controller-manager")
	flag.Var(flagutil.NewMapStringBool(&featureGates), "feature-gates", "A set of key=value pairs that describe feature gates for alpha/experimental features. "+
		"Options are:\n"+strings.Join(utilfeature.DefaultFeatureGate.KnownFeatures(), "\n"))
	flag.StringVar(&fedNamespace, "federation-namespace", util.DefaultFederationSystemNamespace, "The namespace the federation control plane is deployed in.")
	flag.StringVar(&clusterNamespace, "registry-namespace", util.MulticlusterPublicNamespace, "The cluster registry namespace.")
	flag.BoolVar(&limitedScope, "limited-scope", false, "Whether the federation namespace will be the only target for federation.")

	flag.Parse()
	if *verFlag {
		fmt.Fprintf(os.Stdout, "Federation v2 controller-manager version: %s\n", fmt.Sprintf("%#v", version.Get()))
		os.Exit(0)
	}

	// To help debugging, immediately log version.
	glog.Infof("Version: %+v", version.Get())

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

	glog.Infof("Federation namespace: %s", fedNamespace)
	glog.Infof("Cluster registry namespace: %s", clusterNamespace)

	targetNamespace := metav1.NamespaceAll
	if limitedScope {
		targetNamespace = fedNamespace
		glog.Infof("Federation will be limited to the %q namespace", fedNamespace)
	} else {
		glog.Info("Federation will target all namespaces")
	}

	// TODO(marun) Make the monitor period configurable
	clusterMonitorPeriod := time.Second * 40
	federatedcluster.StartClusterController(config, fedNamespace, clusterNamespace, stopChan, clusterMonitorPeriod)

	if utilfeature.DefaultFeatureGate.Enabled(features.SchedulerPreferences) {
		err = schedulingpreference.StartSchedulerController(config, fedNamespace, clusterNamespace, targetNamespace, stopChan, true)
		if err != nil {
			glog.Fatalf("Error starting scheduler controller: %v", err)
		}
	}

	if utilfeature.DefaultFeatureGate.Enabled(features.CrossClusterServiceDiscovery) {
		err = servicedns.StartController(config, fedNamespace, clusterNamespace, targetNamespace, stopChan, false)
		if err != nil {
			glog.Fatalf("Error starting dns controller: %v", err)
		}

		err = dnsendpoint.StartServiceDNSEndpointController(config, targetNamespace, stopChan, false)
		if err != nil {
			glog.Fatalf("Error starting dns endpoint controller: %v", err)
		}
	}

	if utilfeature.DefaultFeatureGate.Enabled(features.FederatedIngress) {
		err = ingressdns.StartController(config, fedNamespace, clusterNamespace, targetNamespace, stopChan, false)
		if err != nil {
			glog.Fatalf("Error starting ingress dns controller: %v", err)
		}

		err = dnsendpoint.StartIngressDNSEndpointController(config, targetNamespace, stopChan, false)
		if err != nil {
			glog.Fatalf("Error starting ingress dns endpoint controller: %v", err)
		}
	}

	if utilfeature.DefaultFeatureGate.Enabled(features.PushReconciler) {
		err = federatedtypeconfig.StartController(config, fedNamespace, clusterNamespace, targetNamespace, stopChan)
		if err != nil {
			glog.Fatalf("Error starting federated type config controller: %v", err)
		}
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
