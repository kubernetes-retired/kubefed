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

package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/apiserver/pkg/util/logs"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubernetes-sigs/federation-v2/cmd/controller-manager/app/leaderelection"
	"github.com/kubernetes-sigs/federation-v2/cmd/controller-manager/app/options"
	corev1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	genericclient "github.com/kubernetes-sigs/federation-v2/pkg/client/generic"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/dnsendpoint"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/federatedcluster"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/federatedtypeconfig"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/ingressdns"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/schedulingmanager"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/servicedns"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/features"
	"github.com/kubernetes-sigs/federation-v2/pkg/version"
)

var (
	kubeconfig, masterURL string
)

// NewControllerManagerCommand creates a *cobra.Command object with default parameters
func NewControllerManagerCommand() *cobra.Command {
	verFlag := false
	opts := options.NewOptions()

	cmd := &cobra.Command{
		Use: "controller-manager",
		Long: `The Federation controller manager runs a bunch of controllers
which watches federation CRD's and the corresponding resources in federation
member clusters and does the necessary reconciliation`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(os.Stdout, "Federation v2 controller-manager version: %s\n", fmt.Sprintf("%#v", version.Get()))
			if verFlag {
				os.Exit(0)
			}
			PrintFlags(cmd.Flags())

			if err := Run(opts); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	}

	// Add the command line flags from other dependencies(glog, kubebuilder, etc.)
	cmd.Flags().AddGoFlagSet(flag.CommandLine)

	opts.AddFlags(cmd.Flags())
	cmd.Flags().BoolVar(&verFlag, "version", false, "Prints the Version info of controller-manager")
	cmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	cmd.Flags().StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")

	return cmd
}

// Run runs the controller-manager with options. This should never exit.
func Run(opts *options.Options) error {
	logs.InitLogs()
	defer logs.FlushLogs()

	stopChan := setupSignalHandler()

	var err error
	opts.Config.KubeConfig, err = clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		panic(err)
	}

	setOptionsByFederationConfig(opts)

	if err := utilfeature.DefaultFeatureGate.SetFromMap(opts.FeatureGates); err != nil {
		glog.Fatalf("Invalid Feature Gate: %v", err)
	}

	if opts.LimitedScope {
		opts.Config.TargetNamespace = opts.Config.FederationNamespace
		glog.Infof("Federation will be limited to the %q namespace", opts.Config.FederationNamespace)
	} else {
		opts.Config.TargetNamespace = metav1.NamespaceAll
		glog.Info("Federation will target all namespaces")
	}

	elector, err := leaderelection.NewFederationLeaderElector(opts, startControllers)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		select {
		case <-stopChan:
			cancel()
		case <-ctx.Done():
		}
	}()

	elector.Run(ctx)

	glog.Errorf("lost lease")
	return errors.New("lost lease")
}

func startControllers(opts *options.Options, stopChan <-chan struct{}) {
	if err := federatedcluster.StartClusterController(opts.Config, stopChan, opts.ClusterMonitorPeriod); err != nil {
		glog.Fatalf("Error starting cluster controller: %v", err)
	}

	if utilfeature.DefaultFeatureGate.Enabled(features.SchedulerPreferences) {
		if _, err := schedulingmanager.StartSchedulingManager(opts.Config, stopChan); err != nil {
			glog.Fatalf("Error starting scheduling manager: %v", err)
		}
	}

	if utilfeature.DefaultFeatureGate.Enabled(features.CrossClusterServiceDiscovery) {
		if err := servicedns.StartController(opts.Config, stopChan); err != nil {
			glog.Fatalf("Error starting dns controller: %v", err)
		}

		if err := dnsendpoint.StartServiceDNSEndpointController(opts.Config, stopChan); err != nil {
			glog.Fatalf("Error starting dns endpoint controller: %v", err)
		}
	}

	if utilfeature.DefaultFeatureGate.Enabled(features.FederatedIngress) {
		if err := ingressdns.StartController(opts.Config, stopChan); err != nil {
			glog.Fatalf("Error starting ingress dns controller: %v", err)
		}

		if err := dnsendpoint.StartIngressDNSEndpointController(opts.Config, stopChan); err != nil {
			glog.Fatalf("Error starting ingress dns endpoint controller: %v", err)
		}
	}

	if utilfeature.DefaultFeatureGate.Enabled(features.PushReconciler) {
		if err := federatedtypeconfig.StartController(opts.Config, stopChan); err != nil {
			glog.Fatalf("Error starting federated type config controller: %v", err)
		}
	}
}

func setOptionsByFederationConfig(opts *options.Options) {
	client := genericclient.NewForConfigOrDieWithUserAgent(opts.Config.KubeConfig, "federationconfig")

	name := util.FederationConfigName
	namespace := opts.Config.FederationNamespace
	fedConfig := &corev1a1.FederationConfig{}
	err := client.Get(context.Background(), fedConfig, namespace, name)
	if err != nil {
		glog.V(1).Infof("Cannot retrieve FederationConfig %q in namespace %q: %v. Command line options are used.", name, namespace, err)
		return
	}

	glog.Infof("Setting Options with FederationConfig %q in namespace %q", name, namespace)

	spec := fedConfig.Spec
	opts.LimitedScope = spec.LimitedScope
	opts.ClusterMonitorPeriod = spec.ControllerDuration.ClusterMonitorPeriod.Duration

	opts.Config.ClusterNamespace = spec.RegistryNamespace
	opts.Config.ClusterAvailableDelay = spec.ControllerDuration.AvailableDelay.Duration
	opts.Config.ClusterUnavailableDelay = spec.ControllerDuration.UnavailableDelay.Duration

	opts.LeaderElection.ResourceLock = spec.LeaderElect.ResourceLock
	opts.LeaderElection.RetryPeriod = spec.LeaderElect.RetryPeriod.Duration
	opts.LeaderElection.RenewDeadline = spec.LeaderElect.RenewDeadline.Duration
	opts.LeaderElection.LeaseDuration = spec.LeaderElect.LeaseDuration.Duration

	var featureGates = make(map[string]bool)
	for _, v := range fedConfig.Spec.FeatureGates {
		featureGates[v.Name] = v.Enabled
	}
	if len(featureGates) < 1 {
		return
	}

	opts.FeatureGates = featureGates
	glog.V(1).Infof("\"feature-gates\" are setting with %v", featureGates)
}

// PrintFlags logs the flags in the flagset
func PrintFlags(flags *pflag.FlagSet) {
	flags.VisitAll(func(flag *pflag.Flag) {
		glog.V(1).Infof("FLAG: --%s=%q", flag.Name, flag.Value)
	})
}

var onlyOneSignalHandler = make(chan struct{})
var shutdownSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}

// setupSignalHandler registered for SIGTERM and SIGINT. A stop channel is returned
// which is closed on one of these signals. If a second signal is caught, the program
// is terminated with exit code 1.
func setupSignalHandler() (stopCh <-chan struct{}) {
	close(onlyOneSignalHandler) // panics when called twice

	stop := make(chan struct{})
	c := make(chan os.Signal, 2)
	signal.Notify(c, shutdownSignals...)
	go func() {
		<-c
		close(stop)
		<-c
		os.Exit(1) // second signal. Exit directly.
	}()

	return stop
}
