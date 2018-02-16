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

package util

import (
	"fmt"

	fedclient "github.com/marun/fnord/pkg/client/clientset_generated/clientset"
	"github.com/spf13/pflag"
	client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	crclient "k8s.io/cluster-registry/pkg/client/clientset_generated/clientset"
)

const (
	// DefaultFederationNamespace is the namespace in which
	// federation system components are hosted.
	DefaultFederationNamespace = "federation"
)

// FedConfig provides a filesystem based kubeconfig (via
// `PathOptions()`) and a mechanism to talk to the federation
// host cluster and the federation control plane api server.
type FedConfig interface {
	// PathOptions provides filesystem based kubeconfig access.
	PathOptions() *clientcmd.PathOptions
	// HostClientSet provides a kubernetes API compliant clientset
	// to communicate with the kubernetes API server.
	HostClientset(context, kubeconfigPath string) (*client.Clientset, error)
	// ClusterClientset provides a mechanism to communicate with the
	// cluster derived from the context and the kubeconfig.
	ClusterClientset(context,
		kubeconfigPath string) (*client.Clientset, *rest.Config, error)
	// ClusterRegistryClientset provides a clientset dervied from the context
	// and the kubeconfig to communicate with the cluster registry.
	ClusterRegistryClientset(context,
		kubeconcifgPath string) (*crclient.Clientset, *rest.Config, error)
	// FedClientSet provides a federation API compliant clientset
	// to communicate with the federation API server.
	FedClientset(context, kubeconfigPath string) (*fedclient.Clientset, error)
}

// fedConfig implements the FedConfig interface.
type fedConfig struct {
	pathOptions      *clientcmd.PathOptions
	hostClientset    *client.Clientset
	clusterClientset *client.Clientset
	crClientset      *crclient.Clientset
	fedClientset     *fedclient.Clientset
}

// NewFedConfig creates a fedConfig for `kubefnord` commands.
func NewFedConfig(pathOptions *clientcmd.PathOptions) FedConfig {
	return &fedConfig{
		pathOptions: pathOptions,
	}
}

// PathOptions returns the pathOptions in the fedConfig.
func (a *fedConfig) PathOptions() *clientcmd.PathOptions {
	return a.pathOptions
}

// HostClientset provides a kubernetes API compliant clientset to communicate
// with the kubernetes API server.
func (a *fedConfig) HostClientset(context, kubeconfigPath string) (*client.Clientset, error) {
	hostConfig := a.getClientConfig(context, kubeconfigPath)
	hostClientConfig, err := hostConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	a.hostClientset = client.NewForConfigOrDie(hostClientConfig)
	return a.hostClientset, nil
}

// ClusterClientset provides a mechanism to communicate with the cluster
// derived from the context and the kubeconfig.
func (a *fedConfig) ClusterClientset(context,
	kubeconfigPath string) (*client.Clientset, *rest.Config, error) {
	clusterConfig := a.getClientConfig(context, kubeconfigPath)
	clusterClientConfig, err := clusterConfig.ClientConfig()
	if err != nil {
		return nil, nil, err
	}

	a.clusterClientset = client.NewForConfigOrDie(clusterClientConfig)
	return a.clusterClientset, clusterClientConfig, nil
}

// ClusterRegistryClientset provides a clientset dervied from the context and
// the kubeconfig to communicate with the cluster registry.
func (a *fedConfig) ClusterRegistryClientset(context,
	kubeconfigPath string) (*crclient.Clientset, *rest.Config, error) {
	crConfig := a.getClientConfig(context, kubeconfigPath)
	crClientConfig, err := crConfig.ClientConfig()
	if err != nil {
		return nil, nil, err
	}

	a.crClientset = crclient.NewForConfigOrDie(crClientConfig)
	return a.crClientset, crClientConfig, nil
}

// FedClientSet provides a federation API compliant clientset to communicate
// with the federation API server.
func (a *fedConfig) FedClientset(context, kubeconfigPath string) (*fedclient.Clientset, error) {
	fedConfig := a.getClientConfig(context, kubeconfigPath)
	fedClientConfig, err := fedConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	a.fedClientset = fedclient.NewForConfigOrDie(fedClientConfig)
	return a.fedClientset, nil
}

// getClientConfig is a helper method to create a client config from the
// context and kubeconfig passed as arguments.
func (a *fedConfig) getClientConfig(context, kubeconfigPath string) clientcmd.ClientConfig {
	loadingRules := *a.pathOptions.LoadingRules
	loadingRules.Precedence = a.pathOptions.GetLoadingPrecedence()
	loadingRules.ExplicitPath = kubeconfigPath
	overrides := &clientcmd.ConfigOverrides{
		CurrentContext: context,
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(&loadingRules, overrides)
}

// SubcommandOptions holds the configuration required by the subcommands of
// `kubefnord`.
type SubcommandOptions struct {
	Name                string
	Host                string
	FederationNamespace string
	Kubeconfig          string
	DryRun              bool
}

// CommonBind adds the common flags to the flagset passed in.
func (o *SubcommandOptions) CommonBind(flags *pflag.FlagSet) {
	flags.StringVar(&o.Kubeconfig, "kubeconfig", "", "Path to the kubeconfig file to use for CLI requests.")
	flags.StringVar(&o.Host, "host-cluster-context", "", "Host cluster context")
	flags.StringVar(&o.FederationNamespace, "federation-namespace", DefaultFederationNamespace, "Namespace in the host cluster where the federation system components are installed")
	flags.BoolVar(&o.DryRun, "dry-run", false,
		"Run the command in dry-run mode, without making any server requests.")
}

// SetName sets the name from the args passed in for the required positional
// argument.
func (o *SubcommandOptions) SetName(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("NAME is required")
	}

	o.Name = args[0]
	return nil
}

// ClusterServiceAccountName returns the name of a service account whose
// credentials are used by the host cluster to access the client cluster.
func ClusterServiceAccountName(joiningClusterName, hostClusterName string) string {
	return fmt.Sprintf("%s-%s", joiningClusterName, hostClusterName)
}

// ClusterRoleName returns the name of a ClusterRole and its associated
// ClusterRoleBinding that are used to allow the service account to
// access necessary resources on the cluster.
func ClusterRoleName(federationName, serviceAccountName string) string {
	return fmt.Sprintf("federation-controller-manager:%s-%s", federationName, serviceAccountName)
}
