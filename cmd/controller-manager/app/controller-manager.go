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
	corev1b1 "sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	genericclient "sigs.k8s.io/kubefed/pkg/client/generic"
	"sigs.k8s.io/kubefed/pkg/controller/dnsendpoint"
	"sigs.k8s.io/kubefed/pkg/controller/federatedtypeconfig"
	"sigs.k8s.io/kubefed/pkg/controller/ingressdns"
	"sigs.k8s.io/kubefed/pkg/controller/kubefedcluster"
	"sigs.k8s.io/kubefed/pkg/controller/schedulingmanager"
	"sigs.k8s.io/kubefed/pkg/controller/servicedns"
	"sigs.k8s.io/kubefed/pkg/controller/util"
	"sigs.k8s.io/kubefed/pkg/features"
	"sigs.k8s.io/kubefed/pkg/version"
)

var (
	kubeconfig, kubeFedConfig, masterURL string
)

// NewControllerManagerCommand creates a *cobra.Command object with default parameters
func NewControllerManagerCommand(stopChan <-chan struct{}) *cobra.Command {
	verFlag := false
	opts := options.NewOptions()

	cmd := &cobra.Command{
		Use: "controller-manager",
		Long: `The KubeFed controller manager runs a bunch of controllers
which watches KubeFed CRD's and the corresponding resources in
member clusters and does the necessary reconciliation`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(os.Stdout, "KubeFed controller-manager version: %s\n", fmt.Sprintf("%#v", version.Get()))
			if verFlag {
				os.Exit(0)
			}
			PrintFlags(cmd.Flags())

			if err := Run(opts, stopChan); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	}

	// Add the command line flags from other dependencies(klog, kubebuilder, etc.)
	cmd.Flags().AddGoFlagSet(flag.CommandLine)

	opts.AddFlags(cmd.Flags())
	cmd.Flags().BoolVar(&verFlag, "version", false, "Prints the Version info of controller-manager")
	cmd.Flags().StringVar(&kubeFedConfig, "kubefed-config", "", "Path to a KubeFedConfig yaml file. Test only.")
	cmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	cmd.Flags().StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")

	return cmd
}

// Run runs the controller-manager with options. This should never exit.
func Run(opts *options.Options, stopChan <-chan struct{}) error {
	logs.InitLogs()
	defer logs.FlushLogs()

	// TODO: Make healthz endpoint configurable
	go serveHealthz(":8080")

	var err error
	opts.Config.KubeConfig, err = clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		panic(err)
	}

	setOptionsByKubeFedConfig(opts)

	if err := utilfeature.DefaultFeatureGate.SetFromMap(opts.FeatureGates); err != nil {
		klog.Fatalf("Invalid Feature Gate: %v", err)
	}

	if opts.Scope == apiextv1b1.NamespaceScoped {
		opts.Config.TargetNamespace = opts.Config.KubeFedNamespace
		klog.Infof("KubeFed will be limited to the %q namespace", opts.Config.KubeFedNamespace)
	} else {
		opts.Config.TargetNamespace = metav1.NamespaceAll
		klog.Info("KubeFed will target all namespaces")
	}

	elector, err := leaderelection.NewKubeFedLeaderElector(opts, startControllers)
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

func getKubeFedConfig(opts *options.Options) *corev1b1.KubeFedConfig {
	fedConfig := &corev1b1.KubeFedConfig{}
	if kubeFedConfig == "" {
		// there is no --kubefed-config specified, get `kubefed` KubeFedConfig from the cluster
		client := genericclient.NewForConfigOrDieWithUserAgent(opts.Config.KubeConfig, "kubefedconfig")

		name := util.KubeFedConfigName
		namespace := opts.Config.KubeFedNamespace
		qualifiedName := util.QualifiedName{
			Namespace: namespace,
			Name:      name,
		}

		err := client.Get(context.Background(), fedConfig, namespace, name)
		if apierrors.IsNotFound(err) {
			klog.Infof("Cannot retrieve KubeFedConfig %q: %v. Default options are used.", qualifiedName.String(), err)
			return nil
		}
		if err != nil {
			klog.Fatalf("Error retrieving KubeFedConfig %q: %v.", qualifiedName.String(), err)
		}

		klog.Infof("Setting Options with KubeFedConfig %q", qualifiedName.String())
		return fedConfig
	}

	file, err := os.Open(kubeFedConfig)
	if err != nil {
		klog.Fatalf("Cannot open KubeFedConfig from file %q: %v", kubeFedConfig, err)
	}
	defer file.Close()

	decoder := yaml.NewYAMLToJSONDecoder(file)
	err = decoder.Decode(fedConfig)
	if err != nil {
		klog.Fatalf("Cannot decode KubeFedConfig from file %q: %v", kubeFedConfig, err)
	}

	// set to current namespace to make sure `KubeFedConfig` is updated in correct namespace
	fedConfig.Namespace = opts.Config.KubeFedNamespace
	klog.Infof("Setting Options with KubeFedConfig from file %q: %v", kubeFedConfig, fedConfig.Spec)
	return fedConfig
}

func setDuration(target *metav1.Duration, defaultValue time.Duration) {
	if target.Duration == 0 {
		target.Duration = defaultValue
	}
}

func setInt64(target *int64, defaultValue int64) {
	if *target == 0 {
		*target = defaultValue
	}
}

func setDefaultKubeFedConfig(fedConfig *corev1b1.KubeFedConfig) {
	spec := &fedConfig.Spec

	if len(spec.Scope) == 0 {
		// TODO(sohankunkerkar) Remove when no longer necessary.
		// This Environment variable is a temporary addition to support Red Hat's downstream testing efforts.
		// Its continued existence should not be relied upon.
		const defaultScopeEnv = "DEFAULT_KUBEFED_SCOPE"
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
	if len(election.ResourceLock) == 0 {
		election.ResourceLock = util.DefaultLeaderElectionResourceLock
	}
	setDuration(&election.RetryPeriod, util.DefaultLeaderElectionRetryPeriod)
	setDuration(&election.RenewDeadline, util.DefaultLeaderElectionRenewDeadline)
	setDuration(&election.LeaseDuration, util.DefaultLeaderElectionLeaseDuration)

	healthCheck := &spec.ClusterHealthCheck
	setInt64(&healthCheck.PeriodSeconds, util.DefaultClusterHealthCheckPeriod)
	setInt64(&healthCheck.TimeoutSeconds, util.DefaultClusterHealthCheckTimeout)
	setInt64(&healthCheck.FailureThreshold, util.DefaultClusterHealthCheckFailureThreshold)
	setInt64(&healthCheck.SuccessThreshold, util.DefaultClusterHealthCheckSuccessThreshold)

	if len(spec.SyncController.AdoptResources) == 0 {
		spec.SyncController.AdoptResources = corev1b1.AdoptResourcesEnabled
	}
}

func updateKubeFedConfig(config *rest.Config, fedConfig *corev1b1.KubeFedConfig) {
	name := fedConfig.Name
	namespace := fedConfig.Namespace
	qualifiedName := util.QualifiedName{
		Namespace: namespace,
		Name:      name,
	}

	configResource := &corev1b1.KubeFedConfig{}
	client := genericclient.NewForConfigOrDieWithUserAgent(config, "kubefedconfig")
	err := client.Get(context.Background(), configResource, namespace, name)
	if err != nil && !apierrors.IsNotFound(err) {
		klog.Fatalf("Error retrieving KubeFedConfig %q: %v", qualifiedName, err)
	}
	if apierrors.IsNotFound(err) {
		// if `--kubefed-config` is specifed but there is not KubeFedConfig resource accordingly
		err = client.Create(context.Background(), fedConfig)
		if err != nil {
			klog.Fatalf("Error creating KubeFedConfig %q: %v", qualifiedName, err)
		}
	} else {
		configResource.Spec = fedConfig.Spec
		err = client.Update(context.Background(), configResource)
		if err != nil {
			klog.Fatalf("Error updating KubeFedConfig %q: %v", qualifiedName, err)
		}
	}
}

func setOptionsByKubeFedConfig(opts *options.Options) {
	fedConfig := getKubeFedConfig(opts)
	if fedConfig == nil {
		// KubeFedConfig could not be sourced from --kubefed-config or from the API.
		qualifiedName := util.QualifiedName{
			Namespace: opts.Config.KubeFedNamespace,
			Name:      util.KubeFedConfigName,
		}

		klog.Infof("Creating KubeFedConfig %q with default values", qualifiedName)

		fedConfig = &corev1b1.KubeFedConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      qualifiedName.Name,
				Namespace: qualifiedName.Namespace,
			},
		}
	}

	setDefaultKubeFedConfig(fedConfig)

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

	opts.Config.SkipAdoptingResources = spec.SyncController.AdoptResources == corev1b1.AdoptResourcesDisabled

	updateKubeFedConfig(opts.Config.KubeConfig, fedConfig)

	var featureGates = make(map[string]bool)
	for _, v := range fedConfig.Spec.FeatureGates {
		featureGates[v.Name] = v.Configuration == corev1b1.ConfigurationEnabled
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

func serveHealthz(address string) {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	klog.Fatal(http.ListenAndServe(address, nil))
}
