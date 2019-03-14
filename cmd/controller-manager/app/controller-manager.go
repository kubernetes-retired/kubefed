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
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/kubernetes-sigs/kubebuilder/pkg/install"
	"github.com/kubernetes-sigs/kubebuilder/pkg/signals"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/api/core/v1"
	extv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	flagutil "k8s.io/apiserver/pkg/util/flag"
	"k8s.io/apiserver/pkg/util/logs"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubernetes-sigs/federation-v2/cmd/controller-manager/app/leaderelection"
	"github.com/kubernetes-sigs/federation-v2/cmd/controller-manager/app/options"
	genericclient "github.com/kubernetes-sigs/federation-v2/pkg/client/generic"
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

	stopChan := signals.SetupSignalHandler()

	var err error
	opts.Config.KubeConfig, err = clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		panic(err)
	}

	setOptionsByConfigMap(opts)

	if err := utilfeature.DefaultFeatureGate.SetFromMap(opts.FeatureGates); err != nil {
		glog.Fatalf("Invalid Feature Gate: %v", err)
	}

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

	if !opts.LeaderElection.LeaderElect {
		// Leader election is disabled, so run inline until done.
		startControllers(opts, stopChan)
		<-stopChan
		return errors.New("finished without leader election")
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
		if _, err := schedulingmanager.StartSchedulerController(opts.Config, stopChan); err != nil {
			glog.Fatalf("Error starting scheduler controller: %v", err)
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

func setOptionsByConfigMap(opts *options.Options) {
	client := genericclient.NewForConfigOrDieWithUserAgent(opts.Config.KubeConfig, "federation-v2-configmap")

	name := "federation-v2"
	namespace := os.Getenv("FEDERATION_NAMESPACE")
	configMap := &v1.ConfigMap{}
	err := client.Get(context.Background(), configMap, namespace, name)
	if err != nil {
		glog.V(1).Infof("Cannot retrieve configmap %q in namespace %q: %v. Command line options are used.", name, namespace, err)
		return
	}

	glog.V(1).Infof("Options are setting by configmap %q in namespace %q", name, namespace)

	setBool(&opts.InstallCRDs, configMap.Data, "install-crds")
	setBool(&opts.LimitedScope, configMap.Data, "limited-scope")
	setDuration(&opts.ClusterMonitorPeriod, configMap.Data, "cluster-monitor-period")

	setString(&opts.Config.ClusterNamespace, configMap.Data, "registry-namespace")
	setString(&opts.Config.FederationNamespace, configMap.Data, "federation-namespace")
	setDuration(&opts.Config.ClusterAvailableDelay, configMap.Data, "cluster-available-delay")
	setDuration(&opts.Config.ClusterUnavailableDelay, configMap.Data, "cluster-unavailable-delay")

	setBool(&opts.LeaderElection.LeaderElect, configMap.Data, "leader-elect")
	setString(&opts.LeaderElection.ResourceLock, configMap.Data, "leader-elect-resource-lock")
	setDuration(&opts.LeaderElection.RetryPeriod, configMap.Data, "leader-elect-retry-period")
	setDuration(&opts.LeaderElection.RenewDeadline, configMap.Data, "leader-elect-renew-deadline")
	setDuration(&opts.LeaderElection.LeaseDuration, configMap.Data, "leader-elect-lease-duration")

	featureList, ok := configMap.Data["feature-gates"]
	if !ok && len(featureList) <= 0 {
		return
	}
	featureList = strings.Replace(featureList, "\n", ",", -1)
	featureMap := flagutil.NewMapStringBool(new(map[string]bool))
	featureMap.Set(featureList)
	opts.FeatureGates = *featureMap.Map
	glog.V(1).Infof("\"feature-gates\" are setting with %q", featureList)
}

func setDuration(target *time.Duration, data map[string]string, key string) {
	value, ok := data[key]
	if !ok || len(value) <= 0 {
		return
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		glog.Warningf("Failed in parsing %q to duration. Ignored.", value)
		return
	}
	if *target == duration {
		return
	}

	glog.V(1).Infof("Option %q is changed from %q to %q by configmap", key, *target, duration)
	*target = duration
}

func setString(target *string, data map[string]string, key string) {
	value, ok := data[key]
	if !ok || len(value) <= 0 {
		return
	}
	if *target == value {
		return
	}

	glog.V(1).Infof("Option %q is changed from %q to %q by configmap", key, *target, value)
	*target = value
}

func setBool(target *bool, data map[string]string, key string) {
	value, ok := data[key]
	if !ok || len(value) <= 0 {
		return
	}

	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		glog.Warningf("Failed in parsing \"%v\" to bool. Ignored.", boolValue)
		return
	}
	if *target == boolValue {
		return
	}

	glog.V(1).Infof("Option %q is changed from \"%v\" to \"%v\" by configmap", key, *target, boolValue)
	*target = boolValue
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
