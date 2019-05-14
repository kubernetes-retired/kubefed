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
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	apiextv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/apiserver/pkg/util/logs"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	"sigs.k8s.io/kubefed/cmd/controller-manager/app/leaderelection"
	"sigs.k8s.io/kubefed/cmd/controller-manager/app/options"
	corev1a1 "sigs.k8s.io/kubefed/pkg/apis/core/v1alpha1"
	genericclient "sigs.k8s.io/kubefed/pkg/client/generic"
	"sigs.k8s.io/kubefed/pkg/controller/dnsendpoint"
	"sigs.k8s.io/kubefed/pkg/controller/kubefedcluster"
	"sigs.k8s.io/kubefed/pkg/controller/federatedtypeconfig"
	"sigs.k8s.io/kubefed/pkg/controller/ingressdns"
	"sigs.k8s.io/kubefed/pkg/controller/schedulingmanager"
	"sigs.k8s.io/kubefed/pkg/controller/servicedns"
	"sigs.k8s.io/kubefed/pkg/controller/util"
	"sigs.k8s.io/kubefed/pkg/features"
	"sigs.k8s.io/kubefed/pkg/version"
)

var (
	kubeconfig, kubefedConfig, masterURL string
)

// NewControllerManagerCommand creates a *cobra.Command object with default parameters
func NewControllerManagerCommand() *cobra.Command {
	verFlag := false
	opts := options.NewOptions()

	cmd := &cobra.Command{
		Use: "controller-manager",
		Long: `The Kubefed controller manager runs a bunch of controllers
which watches federation CRD's and the corresponding resources in federation
member clusters and does the necessary reconciliation`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(os.Stdout, "Kubefed controller-manager version: %s\n", fmt.Sprintf("%#v", version.Get()))
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

	// Add the command line flags from other dependencies(klog, kubebuilder, etc.)
	cmd.Flags().AddGoFlagSet(flag.CommandLine)

	opts.AddFlags(cmd.Flags())
	cmd.Flags().BoolVar(&verFlag, "version", false, "Prints the Version info of controller-manager")
	cmd.Flags().StringVar(&kubefedConfig, "kubefed-config", "", "Path to a kubefed config yaml file. Test only.")
	cmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	cmd.Flags().StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")

	return cmd
}

// Run runs the controller-manager with options. This should never exit.
func Run(opts *options.Options) error {
	logs.InitLogs()
	defer logs.FlushLogs()

	// TODO: Make healthz endpoint configurable
	go serveHealthz(":8080")

	stopChan := setupSignalHandler()

	var err error
	opts.Config.KubeConfig, err = clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		panic(err)
	}

	setOptionsByKubefedConfig(opts)

	if err := utilfeature.DefaultFeatureGate.SetFromMap(opts.FeatureGates); err != nil {
		klog.Fatalf("Invalid Feature Gate: %v", err)
	}

	if opts.Scope == apiextv1b1.NamespaceScoped {
		opts.Config.TargetNamespace = opts.Config.KubefedNamespace
		klog.Infof("Federation will be limited to the %q namespace", opts.Config.KubefedNamespace)
	} else {
		opts.Config.TargetNamespace = metav1.NamespaceAll
		klog.Info("Federation will target all namespaces")
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

	klog.Errorf("lost lease")
	return errors.New("lost lease")
}

func startControllers(opts *options.Options, stopChan <-chan struct{}) {
	if err := kubefedcluster.StartClusterController(opts.Config, opts.ClusterHealthCheckConfig, stopChan); err != nil {
		klog.Fatalf("Error starting cluster controller: %v", err)
	}

	if utilfeature.DefaultFeatureGate.Enabled(features.SchedulerPreferences) {
		if _, err := schedulingmanager.StartSchedulingManager(opts.Config, stopChan); err != nil {
			klog.Fatalf("Error starting scheduling manager: %v", err)
		}
	}

	if utilfeature.DefaultFeatureGate.Enabled(features.CrossClusterServiceDiscovery) {
		if err := servicedns.StartController(opts.Config, stopChan); err != nil {
			klog.Fatalf("Error starting dns controller: %v", err)
		}

		if err := dnsendpoint.StartServiceDNSEndpointController(opts.Config, stopChan); err != nil {
			klog.Fatalf("Error starting dns endpoint controller: %v", err)
		}
	}

	if utilfeature.DefaultFeatureGate.Enabled(features.FederatedIngress) {
		if err := ingressdns.StartController(opts.Config, stopChan); err != nil {
			klog.Fatalf("Error starting ingress dns controller: %v", err)
		}

		if err := dnsendpoint.StartIngressDNSEndpointController(opts.Config, stopChan); err != nil {
			klog.Fatalf("Error starting ingress dns endpoint controller: %v", err)
		}
	}

	if utilfeature.DefaultFeatureGate.Enabled(features.PushReconciler) {
		if err := federatedtypeconfig.StartController(opts.Config, stopChan); err != nil {
			klog.Fatalf("Error starting federated type config controller: %v", err)
		}
	}
}

func getKubefedConfig(opts *options.Options) *corev1a1.KubefedConfig {
	fedConfig := &corev1a1.KubefedConfig{}
	if kubefedConfig == "" {
		// there is no --kubefed-config specified, get `kubefed` KubefedConfig from the cluster
		client := genericclient.NewForConfigOrDieWithUserAgent(opts.Config.KubeConfig, "kubefedconfig")

		name := util.KubefedConfigName
		namespace := opts.Config.KubefedNamespace
		qualifiedName := util.QualifiedName{
			Namespace: namespace,
			Name:      name,
		}

		err := client.Get(context.Background(), fedConfig, namespace, name)
		if apierrors.IsNotFound(err) {
			klog.Infof("Cannot retrieve KubefedConfig %q: %v. Default options are used.", qualifiedName.String(), err)
			return nil
		}
		if err != nil {
			klog.Fatalf("Error retrieving KubefedConfig %q: %v.", qualifiedName.String(), err)
		}

		klog.Infof("Setting Options with KubefedConfig %q", qualifiedName.String())
		return fedConfig
	}

	file, err := os.Open(kubefedConfig)
	if err != nil {
		// when kubefed config file is specified, it should be fatal error if the file does not valid
		klog.Fatalf("Cannot open kubefed config file %q: %v", kubefedConfig, err)
	}
	defer file.Close()

	decoder := yaml.NewYAMLToJSONDecoder(file)
	err = decoder.Decode(fedConfig)
	if err != nil {
		klog.Fatalf("Cannot decode KubefedConfig from file %q: %v", kubefedConfig, err)
	}

	// set to current namespace to make sure `KubefedConfig` is updated in correct namespace
	fedConfig.Namespace = opts.Config.KubefedNamespace
	klog.Infof("Setting Options with KubefedConfig from file %q: %v", kubefedConfig, fedConfig.Spec)
	return fedConfig
}

func setDuration(target *metav1.Duration, defaultValue time.Duration) {
	if target.Duration == 0 {
		target.Duration = defaultValue
	}
}

func setString(target *string, defaultValue string) {
	if *target == "" {
		*target = defaultValue
	}
}

func setInt(target *int, defaultValue int) {
	if *target == 0 {
		*target = defaultValue
	}
}

func setDefaultKubefedConfig(fedConfig *corev1a1.KubefedConfig) {
	spec := &fedConfig.Spec

	if len(spec.Scope) == 0 {
		// TODO(sohankunkerkar) Remove when no longer necessary.
		// This Environment variable is a temporary addition to support Red Hat's downstream testing efforts.
		// Its continued existence should not be relied upon.
		const defaultScopeEnv = "DEFAULT_FEDERATION_SCOPE"
		defaultScope := os.Getenv(defaultScopeEnv)
		if len(defaultScope) != 0 {
			if defaultScope != string(apiextv1b1.ClusterScoped) && defaultScope != string(apiextv1b1.NamespaceScoped) {
				klog.Fatalf("%s must be Cluster or Namespaced; got %q", defaultScopeEnv, defaultScope)
			}
			spec.Scope = apiextv1b1.ResourceScope(defaultScope)
		}
	}

	duration := &spec.ControllerDuration
	setDuration(&duration.AvailableDelay, util.DefaultClusterAvailableDelay)
	setDuration(&duration.UnavailableDelay, util.DefaultClusterUnavailableDelay)

	election := &spec.LeaderElect
	setString(&election.ResourceLock, util.DefaultLeaderElectionResourceLock)
	setDuration(&election.RetryPeriod, util.DefaultLeaderElectionRetryPeriod)
	setDuration(&election.RenewDeadline, util.DefaultLeaderElectionRenewDeadline)
	setDuration(&election.LeaseDuration, util.DefaultLeaderElectionLeaseDuration)

	healthCheck := &spec.ClusterHealthCheck
	setInt(&healthCheck.PeriodSeconds, util.DefaultClusterHealthCheckPeriod)
	setInt(&healthCheck.TimeoutSeconds, util.DefaultClusterHealthCheckTimeout)
	setInt(&healthCheck.FailureThreshold, util.DefaultClusterHealthCheckFailureThreshold)
	setInt(&healthCheck.SuccessThreshold, util.DefaultClusterHealthCheckSuccessThreshold)

}

func updateKubefedConfig(config *rest.Config, fedConfig *corev1a1.KubefedConfig) {
	name := fedConfig.Name
	namespace := fedConfig.Namespace
	qualifiedName := util.QualifiedName{
		Namespace: namespace,
		Name:      name,
	}

	configResource := &corev1a1.KubefedConfig{}
	client := genericclient.NewForConfigOrDieWithUserAgent(config, "kubefedconfig")
	err := client.Get(context.Background(), configResource, namespace, name)
	if err != nil && !apierrors.IsNotFound(err) {
		klog.Fatalf("Error retrieving KubefedConfig %q: %v", qualifiedName, err)
	}
	if apierrors.IsNotFound(err) {
		// if `--kubefed-config` is specifed but there is not KubefedConfig resource accordingly
		err = client.Create(context.Background(), fedConfig)
		if err != nil {
			klog.Fatalf("Error creating KubefedConfig %q: %v", qualifiedName, err)
		}
	} else {
		configResource.Spec = fedConfig.Spec
		err = client.Update(context.Background(), configResource)
		if err != nil {
			klog.Fatalf("Error updating KubefedConfig %q: %v", qualifiedName, err)
		}
	}
}

func setOptionsByKubefedConfig(opts *options.Options) {
	fedConfig := getKubefedConfig(opts)
	if fedConfig == nil {
		// KubefedConfig could not be sourced from --kubefed-config or from the API.
		qualifiedName := util.QualifiedName{
			Namespace: opts.Config.KubefedNamespace,
			Name:      util.KubefedConfigName,
		}

		klog.Infof("Creating KubefedConfig %q with default values", qualifiedName)

		fedConfig = &corev1a1.KubefedConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      qualifiedName.Name,
				Namespace: qualifiedName.Namespace,
			},
		}
	}

	setDefaultKubefedConfig(fedConfig)

	spec := fedConfig.Spec
	opts.Scope = spec.Scope

	opts.Config.ClusterAvailableDelay = spec.ControllerDuration.AvailableDelay.Duration
	opts.Config.ClusterUnavailableDelay = spec.ControllerDuration.UnavailableDelay.Duration

	opts.LeaderElection.ResourceLock = spec.LeaderElect.ResourceLock
	opts.LeaderElection.RetryPeriod = spec.LeaderElect.RetryPeriod.Duration
	opts.LeaderElection.RenewDeadline = spec.LeaderElect.RenewDeadline.Duration
	opts.LeaderElection.LeaseDuration = spec.LeaderElect.LeaseDuration.Duration

	opts.ClusterHealthCheckConfig.PeriodSeconds = spec.ClusterHealthCheck.PeriodSeconds
	opts.ClusterHealthCheckConfig.TimeoutSeconds = spec.ClusterHealthCheck.TimeoutSeconds
	opts.ClusterHealthCheckConfig.FailureThreshold = spec.ClusterHealthCheck.FailureThreshold
	opts.ClusterHealthCheckConfig.SuccessThreshold = spec.ClusterHealthCheck.SuccessThreshold

	opts.Config.SkipAdoptingResources = spec.SyncController.SkipAdoptingResources

	updateKubefedConfig(opts.Config.KubeConfig, fedConfig)

	var featureGates = make(map[string]bool)
	for _, v := range fedConfig.Spec.FeatureGates {
		featureGates[v.Name] = v.Enabled
	}
	if len(featureGates) == 0 {
		return
	}

	opts.FeatureGates = featureGates
	klog.V(1).Infof("\"feature-gates\" will be set to %v", featureGates)
}

// PrintFlags logs the flags in the flagset
func PrintFlags(flags *pflag.FlagSet) {
	flags.VisitAll(func(flag *pflag.Flag) {
		klog.V(1).Infof("FLAG: --%s=%q", flag.Name, flag.Value)
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

func serveHealthz(address string) {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	klog.Fatal(http.ListenAndServe(address, nil))
}
