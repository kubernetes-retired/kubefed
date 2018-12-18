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
	"flag"
	"fmt"
	"os"

	"github.com/golang/glog"
	configlib "github.com/kubernetes-sigs/kubebuilder/pkg/config"
	"github.com/kubernetes-sigs/kubebuilder/pkg/install"
	"github.com/kubernetes-sigs/kubebuilder/pkg/signals"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/api/core/v1"
	extv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/apiserver/pkg/util/logs"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"

	"github.com/kubernetes-sigs/federation-v2/cmd/controller-manager/app/options"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/dnsendpoint"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/federatedcluster"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/federatedtypeconfig"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/ingressdns"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/schedulingmanager"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/servicedns"
	"github.com/kubernetes-sigs/federation-v2/pkg/features"
	"github.com/kubernetes-sigs/federation-v2/pkg/inject"
	"github.com/kubernetes-sigs/federation-v2/pkg/version"
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

	return cmd
}

// Run runs the controller-manager with options. This should never exit.
func Run(opts *options.Options) error {
	logs.InitLogs()
	defer logs.FlushLogs()

	if err := utilfeature.DefaultFeatureGate.SetFromMap(opts.FeatureGates); err != nil {
		glog.Fatalf("Invalid Feature Gate: %v", err)
	}

	stopChan := signals.SetupSignalHandler()

	opts.Config.KubeConfig = configlib.GetConfigOrDie()

	if opts.InstallCRDs {
		if err := install.NewInstaller(opts.Config.KubeConfig).Install(&InstallStrategy{crds: inject.Injector.CRDs}); err != nil {
			glog.Fatalf("Could not create CRDs: %v", err)
		}
	}

	if opts.LimitedScope {
		opts.Config.TargetNamespace = opts.Config.FederationNamespace
		glog.Infof("Federation will be limited to the %q namespace", opts.Config.FederationNamespace)
	} else {
		opts.Config.TargetNamespace = metav1.NamespaceAll
		glog.Info("Federation will target all namespaces")
	}

	run := func(ctx context.Context) {
		federatedcluster.StartClusterController(opts.Config, ctx.Done(), opts.ClusterMonitorPeriod)

		if utilfeature.DefaultFeatureGate.Enabled(features.SchedulerPreferences) {
			schedulingmanager.StartSchedulerController(opts.Config, ctx.Done())
		}

		if utilfeature.DefaultFeatureGate.Enabled(features.CrossClusterServiceDiscovery) {
			if err := servicedns.StartController(opts.Config, ctx.Done()); err != nil {
				glog.Fatalf("Error starting dns controller: %v", err)
			}

			if err := dnsendpoint.StartServiceDNSEndpointController(opts.Config, ctx.Done()); err != nil {
				glog.Fatalf("Error starting dns endpoint controller: %v", err)
			}
		}

		if utilfeature.DefaultFeatureGate.Enabled(features.FederatedIngress) {
			if err := ingressdns.StartController(opts.Config, ctx.Done()); err != nil {
				glog.Fatalf("Error starting ingress dns controller: %v", err)
			}

			if err := dnsendpoint.StartIngressDNSEndpointController(opts.Config, ctx.Done()); err != nil {
				glog.Fatalf("Error starting ingress dns endpoint controller: %v", err)
			}
		}

		if utilfeature.DefaultFeatureGate.Enabled(features.PushReconciler) {
			federatedtypeconfig.StartController(opts.Config, ctx.Done())
		}

		// Blockforever
		select {}
	}

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	go func() {
		select {
		case <-stopChan:
			cancel()
		case <-ctx.Done():
		}
	}()

	if !opts.LeaderElection.LeaderElect {
		run(ctx)
		glog.Fatalf("finished without leader elect")
	}
	restclient.AddUserAgent(opts.Config.KubeConfig, "leader-election")
	leaderElectionClient := clientset.NewForConfigOrDie(opts.Config.KubeConfig)

	hostname, err := os.Hostname()
	if err != nil {
		glog.Fatalf("unable to get hostname: %v", err)
	}

	// Prepare event clients.
	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&corev1.EventSinkImpl{Interface: leaderElectionClient.CoreV1().Events(opts.Config.FederationNamespace)})
	eventRecorder := broadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: "federation-controller"})

	// add a uniquifier so that two processes on the same host don't accidentally both become active
	id := hostname + "_" + string(uuid.NewUUID())
	rl, err := resourcelock.New(opts.LeaderElection.ResourceLock,
		opts.Config.FederationNamespace,
		"federation-controller",
		leaderElectionClient.CoreV1(),
		resourcelock.ResourceLockConfig{
			Identity:      id,
			EventRecorder: eventRecorder,
		})
	if err != nil {
		glog.Fatalf("couldn't create resource lock: %v", err)
	}

	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: opts.LeaderElection.LeaseDuration,
		RenewDeadline: opts.LeaderElection.RenewDeadline,
		RetryPeriod:   opts.LeaderElection.RetryPeriod,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: run,
			OnStoppedLeading: func() {
				glog.Fatalf("leaderelection lost")
			},
		},
	})
	glog.Errorf("lost lease")

	return fmt.Errorf("lost lease")
}

type InstallStrategy struct {
	install.EmptyInstallStrategy
	crds []*extv1b1.CustomResourceDefinition
}

func (s *InstallStrategy) GetCRDs() []*extv1b1.CustomResourceDefinition {
	return s.crds
}

// PrintFlags logs the flags in the flagset
func PrintFlags(flags *pflag.FlagSet) {
	flags.VisitAll(func(flag *pflag.Flag) {
		glog.V(1).Infof("FLAG: --%s=%q", flag.Name, flag.Value)
	})
}
